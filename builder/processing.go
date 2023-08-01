package builder

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	_ "os"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/beacon/engine"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"

	"github.com/bsn-eng/pon-golang-types/builder"

	bundleTypes "github.com/bsn-eng/pon-golang-types/bundles"
	commonTypes "github.com/bsn-eng/pon-golang-types/common"
)

type blockProperties struct {
	block               *types.Block
	blockExecutableData *engine.ExecutableData
	payoutPoolTx        []byte
	blockValue          *big.Int
	triedBundles        []bundleTypes.BuilderBundle
	builtTime           time.Time
	bountyBid           bool
}

func (b *Builder) prepareBlock(slotCtx context.Context, slotTimeCutOff uint64, slot uint64, proposerPubkey commonTypes.PublicKey, submissionChan chan blockProperties, buildingErr *error) {

	timeForJob := time.Until(time.Unix(int64(slotTimeCutOff), 0))
	log.Info("block builder time for new job", "timeForJob", timeForJob)

	ctx, cancel := context.WithTimeout(slotCtx, timeForJob)
	defer cancel()

	var (
		readyBlock           blockProperties
		processingMu         sync.Mutex
		successfulSubmission bool
	)

	// Updates the readyBlock with the new block if it is different and better than the current block ready for submission
	sealedBlockCallback := func(block *types.Block, fees *big.Int, bidAmount uint64, payoutPoolTx []byte, triedBundles []bundleTypes.BuilderBundle, err error) {

		processingMu.Lock()
		defer processingMu.Unlock()

		if ctx.Err() != nil {
			return
		}

		b.slotSubmissionsLock.Lock()
		if !successfulSubmission && len(b.slotSubmissions[slot]) > 0 {
			successfulSubmission = true
		}
		b.slotSubmissionsLock.Unlock()

		if err != nil {
			log.Error("block builder error", "err", err)
			if !successfulSubmission {
				*buildingErr = fmt.Errorf("block builder error: %w", err)
			}
			return
		}

		if payoutPoolTx == nil {
			log.Error("could not create payout pool transaction")
			if !successfulSubmission {
				*buildingErr = errors.New("could not create payout pool transaction")
			}
			return
		}

		log.Info("block builder received new block", "blockValue", fees, "existingValue", readyBlock.blockValue)

		// Validate block
		err = b.blockValidator.ValidateBody(block)
		if err != nil {
			log.Error("block builder error", "err", err)
			if !successfulSubmission {
				*buildingErr = fmt.Errorf("block builder error: %w", err)
			}
			return
		}

		bountyBlock := false
		b.slotBountyMu.Lock()
		for _, bountyAttrs := range b.slotBountyAttrs[slot] {
			if bidAmount == bountyAttrs.BidAmount {
				bountyBlock = true
				break
			}
		}
		b.slotBountyMu.Unlock()

		if readyBlock.blockValue == nil {
			executionPayloadEnvelope := engine.BlockToExecutableData(block, fees)

			// check the length of transaction list is greater than 1, since the payout pool transaction counts as one
			// Handles the case where geth is still preparing the mempool after syncing
			if len(executionPayloadEnvelope.ExecutionPayload.Transactions) < 2 {
				log.Error("block builder error", "err", "no transactions in block aside from payout pool transaction. Ignoring block")
				if !successfulSubmission {
					*buildingErr = errors.New("no transactions to fill block aside from payout pool transaction. Ignoring block till transactions are available or mempool is ready")
				}
				return
			}

			readyBlock = blockProperties{
				block:               block,
				blockExecutableData: executionPayloadEnvelope.ExecutionPayload,
				blockValue:          executionPayloadEnvelope.BlockValue,
				payoutPoolTx:        payoutPoolTx,
				triedBundles:        triedBundles,
				builtTime:           time.Now(),
				bountyBid:           bountyBlock,
			}

			log.Info("block builder first block", "blockValue", fees)

			select {
			case submissionChan <- readyBlock:
			default:
			}

		} else if fees.Cmp(readyBlock.blockValue) >= 0 {
			// Allow greater than or equal to, so that we can submit the same block again if geth creates a new block with the same gasUsed
			// This can happen if the block builder is restarted, and the same block is built again
			// This is a workaround for cases where sometimes we build a block with same gas used, or builds a block with different
			// logs and extraData than the previous but with the same gasUsed
			executionPayloadEnvelope := engine.BlockToExecutableData(block, fees)

			// check the length of transaction list is greater than 1, since the payout pool transaction counts as one
			if len(executionPayloadEnvelope.ExecutionPayload.Transactions) < 2 {
				log.Error("block builder error", "err", "no transactions in block aside from payout pool transaction. Ignoring block")
				if !successfulSubmission {
					*buildingErr = errors.New("no transactions could be filled in block aside from payout pool transaction. Ignoring block, till transactions are available or mempool prepared")
				}
				return
			}

			readyBlock = blockProperties{
				block:               block,
				blockExecutableData: executionPayloadEnvelope.ExecutionPayload,
				blockValue:          executionPayloadEnvelope.BlockValue,
				payoutPoolTx:        payoutPoolTx,
				triedBundles:        triedBundles,
				builtTime:           time.Now(),
				bountyBid:           bountyBlock,
			}

			log.Info("block builder new best block", "blockValue", fees)

			select {
			case submissionChan <- readyBlock:
			default:
			}

		} else if !successfulSubmission {
			// Even if the block is not better than the current block ready for submission,
			// and there have been no block submissions yet or encountered an error in submission
			// trigger to submit again

			log.Info("block builder resubmitting first block", "blockValue", readyBlock.blockValue)

			select {
			case submissionChan <- readyBlock:
			default:
			}

		} else {
			log.Info("new block does not increase block value, skipping", "blockValue", fees, "existingValue", readyBlock.blockValue)
		}
		return
	}

	// resubmits block builder requests to workers to build blocks
	t := time.NewTicker(b.buildInterval)
	defer t.Stop()

	startBlockBuild := func() (ctxCancelled bool) {

		// check if context is cancelled
		if ctx.Err() != nil {
			return true
		}

		// Check if there have been 2 submissions, or if in the bounty window
		// if so, then attempt to retrieve bounty attributes
		// if no bounty attributes are found, then we should not build a block

		b.slotSubmissionsLock.Lock()
		submissionAmounts, ok := b.slotBidAmounts[slot]
		b.slotSubmissionsLock.Unlock()

		useBountyAttrs := false

		if ok && len(submissionAmounts) == 2 {
			b.slotBountyMu.Lock()
			bountyAttrs, ok := b.slotBountyAttrs[slot]
			b.slotBountyMu.Unlock()

			if !ok || len(bountyAttrs) == 0 {
				log.Info("block builder no bounty attributes found, skipping block build")
				return false
			}

			useBountyAttrs = true
		} else if time.Until(time.Unix(int64(slotTimeCutOff), 0)) <= 1*time.Second {
			b.slotBountyMu.Lock()
			bountyAttrs, ok := b.slotBountyAttrs[slot]
			b.slotBountyMu.Unlock()

			if !ok || len(bountyAttrs) == 0 {
				log.Info("block builder no bounty attributes found, skipping block build")
				return false
			}

			useBountyAttrs = true
		}

		var attrs builder.BuilderPayloadAttributes

		if useBountyAttrs {
			b.slotBountyMu.Lock()
			bountyAttrs, ok := b.slotBountyAttrs[slot]
			// If we have not received any block attributes for this slot,
			// then we should not trigger an engine build
			if !ok || len(bountyAttrs) == 0 {
				b.slotBountyMu.Unlock()
				return false
			}
			// Get most recent block bid attributes for this slot
			attrs = bountyAttrs[len(bountyAttrs)-1]
			b.slotBountyMu.Unlock()
		} else {
			b.slotMu.Lock()
			allAttrs, ok := b.slotAttrs[slot]
			// If we have not received any block attributes for this slot,
			// then we should not trigger an engine build
			if !ok || len(allAttrs) == 0 {
				b.slotMu.Unlock()
				return false
			}
			// Get most recent block bid attributes for this slot
			attrs = allAttrs[len(allAttrs)-1]
			b.slotMu.Unlock()
		}

		// Need to check if there has been a successful submission with the
		// most recent block attributes for this slot, if so, then no need to
		// further block build for these attributes (this bid value), until
		// we receive new block attributes for this slot
		b.slotSubmissionsLock.Lock()
		submissions, ok := b.slotSubmissions[attrs.Slot]
		b.slotSubmissionsLock.Unlock()

		if ok && len(submissions) > 0 {
			// check if we have already submitted a bid for this slot
			// with the selected block attributes
			submitted := false
			for _, submission := range submissions { // max size of submissions is 3 so this is fine
				if submission.BlockBid.Message.Value == attrs.BidAmount {
					submitted = true
				}
			}
			if submitted {
				log.Debug("already submitted bid for slot with provided block attributes", "slot", attrs.Slot, "bidAmount", attrs.BidAmount)
				return false
			}
		}

		// If recent attributes have not been submitted, then we can build (for best block) based on these attributes
		// for a new block bid

		log.Info("Building Block with Geth", "slot", attrs.Slot, "parent", attrs.HeadHash)

		b.slotSubmissionsLock.Lock()
		if !successfulSubmission && len(b.slotSubmissions[attrs.Slot]) > 0 {
			successfulSubmission = true
		}
		b.slotSubmissionsLock.Unlock()

		if b.BundlesEnabled {
			currentTime := time.Now()
			currentBlockNumber := b.eth.GetBlockChain().CurrentBlock().Number.Uint64()

			// check if we are submitting a bid for the current slot or next slot
			// if we are submitting a bid for the next slot, we want to include the bundles that are ready for the next slot
			// if we are submitting a bid for the current slot (within 2s of current slot), we want to include the bundles that are ready for the current slot

			slotInRequest := attrs.Slot
			b.beacon.BeaconData.Mu.Lock()
			currentSlot := b.beacon.BeaconData.CurrentSlot
			b.beacon.BeaconData.Mu.Unlock()

			var blockNumber uint64
			if currentSlot == slotInRequest {
				// if the current slot is the same as the slot in the request, we want to include the bundles that are ready for the current slot
				blockNumber = currentBlockNumber
			} else {
				// if the current slot is not the same as the slot in the request, we want to include the bundles that are ready for the next slot
				blockNumber = currentBlockNumber + 1
			}

			// Get the bundles that are ready to be included in the next block or at the current time
			readyBundles, err := b.bundles.GetReadyBundles(blockNumber, currentTime)
			if err != nil {
				log.Error("Failed to get ready bundles", "err", err)
			}
			attrs.Bundles = readyBundles
		}

		if b.BundlesEnabled {
			updatedBundles, err := b.bundles.SetBundlesAddingTrue(attrs.Bundles)
			if err != nil {
				log.Error("Failed to set bundles adding", "err", err)
			}
			attrs.Bundles = updatedBundles
		}

		err := b.eth.BuildBlock(&attrs, sealedBlockCallback)
		if err != nil {
			log.Warn("Failed to build block", "err", err)
			processingMu.Lock()
			if !successfulSubmission {
				*buildingErr = err
			}
			processingMu.Unlock()
		}

		return false
	}

	startBlockBuild()

blockBuilder:
	for {
		select {
		case <-ctx.Done():

			log.Info("Stopping Block Builder. Slot context done or cancelled. slot", "slot", slot)
			break blockBuilder

		case <-t.C:

			ctxCancelled := startBlockBuild()
			if ctxCancelled {
				break blockBuilder
			}

		}
	}

	b.slotSubmissionsLock.Lock()
	if !successfulSubmission && len(b.slotSubmissions[slot]) > 0 {
		successfulSubmission = true
	}
	b.slotSubmissionsLock.Unlock()

	if !successfulSubmission {
		processingMu.Lock()
		if *buildingErr == nil {
			// Then it means the context was cancelled or deadline reached before a block was submitted
			*buildingErr = fmt.Errorf("Failed to submit any block on time: context cancelled / slot submission deadline reached. slot: %d", slot)
		}
		processingMu.Unlock()

		log.Info("Failed to submit any block on time", "slot", slot)
	}

}
