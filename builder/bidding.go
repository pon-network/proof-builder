package builder

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"

	_ "os"
	"strconv"
	"time"

	"github.com/ethereum/go-ethereum/common"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"

	builderTypes "github.com/bsn-eng/pon-golang-types/builder"
	commonTypes "github.com/bsn-eng/pon-golang-types/common"
	bbTypes "github.com/ethereum/go-ethereum/builder/types"
)

func (b *Builder) ProcessBuilderBid(attrs *builderTypes.BuilderPayloadAttributes) ([]builderTypes.BlockBidResponse, error) {

	res := []builderTypes.BlockBidResponse{}
	startTime := time.Now()

	if attrs == nil {
		return res, fmt.Errorf("nil attributes")
	}

	if attrs.BidAmount == nil || attrs.BidAmount.Sign() <= 0 {
		return res, fmt.Errorf("invalid bid amount. Must provide a positive bid amount")
	}

	b.beacon.BeaconData.Mu.Lock()
	if b.beacon.Clients[0].SyncStatus == nil || b.beacon.Clients[0].SyncStatus.IsSyncing {
		b.beacon.BeaconData.Mu.Unlock()
		return res, errors.New("beacon not synced")
	}
	b.beacon.BeaconData.Mu.Unlock()

	if !b.eth.Synced() {
		return res, errors.New("geth backend not Synced")
	}

	b.beacon.BeaconData.Mu.Lock()
	currentSlot := b.beacon.BeaconData.CurrentSlot
	currentBlock := b.eth.GetBlockChain().CurrentBlock()
	b.beacon.BeaconData.Mu.Unlock()

	currentSlotStartTime := b.genesisInfo.GenesisTime + (currentSlot)*bbTypes.SLOT_DURATION
	slotTimeCutOff := currentSlotStartTime + 2 // 2s into the current slot

	// If attrs.Slot is 0 then use the next available slot if current time is past the current slot time cutt off
	// else use the current slot
	if attrs.Slot == 0 {
		if uint64(time.Now().Unix()) > slotTimeCutOff {
			attrs.Slot = currentSlot + 1
		} else {
			attrs.Slot = currentSlot
		}
	}

	// If attrs.Slot is not the next slot, return
	if attrs.Slot < currentSlot {
		log.Error("slot past and not available to bid on", "slot", attrs.Slot, "current slot", currentSlot, "next available slot for bid", currentSlot+1)
		return res, errors.New("slot " + strconv.Itoa(int(attrs.Slot)) + " is past and not available to bid on, current slot is " + strconv.Itoa(int(currentSlot)) + " and next available slot for bid is " + strconv.Itoa(int(currentSlot+1)))
	}

	if attrs.Slot > currentSlot+1 {
		log.Error("slot too far in the future", "slot", attrs.Slot, "current slot", currentSlot)
		return res, errors.New("slot " + strconv.Itoa(int(attrs.Slot)) + " is too far in the future, current slot is " + strconv.Itoa(int(currentSlot)) + " and next available slot for bid is " + strconv.Itoa(int(currentSlot+1)))
	}

	// If in current slot check if the time is past the cut off
	if attrs.Slot == currentSlot {
		if uint64(time.Now().Unix()) > slotTimeCutOff {
			log.Error("slot past and not available to bid on", "slot", attrs.Slot, "current slot", currentSlot, "next available slot for bid", currentSlot+1)
			return res, errors.New("slot " + strconv.Itoa(int(attrs.Slot)) + " is past and not available to bid on, current slot is " + strconv.Itoa(int(currentSlot)) + " and next available slot for bid is " + strconv.Itoa(int(currentSlot+1)))
		}
	} else if attrs.Slot == currentSlot+1 {
		// Bidding on next slot, thus the slot time cut off is 2s into the next slot
		slotTimeCutOff = slotTimeCutOff + bbTypes.SLOT_DURATION
	}

	slotBountyTimeCutOff := slotTimeCutOff + 1 // 1s from the open auction slot time cut off

	log.Info("Processing bid",
		"slot", attrs.Slot,
		"bid amount", attrs.BidAmount,
		"current slot", currentSlot,
		"current block number", currentBlock.Number,
		"current block time", currentBlock.Time,
		"current slot start time", currentSlotStartTime,
		"slot time cut off", slotTimeCutOff,
		"slot bounty time cut off", slotBountyTimeCutOff,
		"current time", time.Now().Unix(),
	)

	b.slotSubmissionsLock.Lock()
	if len(b.slotBidAmounts[attrs.Slot]) == 2 {
		b.slotSubmissionsLock.Unlock()
		log.Info("Already submitted 2 blocks for this slot", "slot", attrs.Slot)
		return res, fmt.Errorf("already submitted 2 blocks for slot %d", attrs.Slot)
	}

	if len(b.slotBidAmounts[attrs.Slot]) > 0 {
		lastSubmittedBidAmount := b.slotBidAmounts[attrs.Slot][len(b.slotBidAmounts[attrs.Slot])-1]
		if attrs.BidAmount != nil && attrs.BidAmount.Cmp(lastSubmittedBidAmount) < 0 {
			b.slotSubmissionsLock.Unlock()
			log.Info("Bid amount is not greater than last bid amount", "slot", attrs.Slot, "bid amount", attrs.BidAmount, "last bid amount", lastSubmittedBidAmount)
			return res, fmt.Errorf("bid amount %d is not greater than last bid amount %d", attrs.BidAmount, lastSubmittedBidAmount)
		}
	}
	b.slotSubmissionsLock.Unlock()

	payload_base_attributes, err := b.beacon.GetPayloadAttributesForSlot(attrs.Slot)
	if err != nil {
		log.Info("could not get payload attributes while submitting block", "err", err, "slot", attrs.Slot)
		return res, fmt.Errorf("could not get payload attributes while submitting block: %w", err)
	}

	withdrawal_list := types.Withdrawals{}
	for _, w := range payload_base_attributes.PayloadAttributes.Withdrawals {
		withdrawal_list = append(withdrawal_list, &types.Withdrawal{
			Index:     w.Index,
			Validator: w.ValidatorIndex,
			Address:   common.HexToAddress(w.Address),
			Amount:    w.Amount,
		})
	}

	attrs.Withdrawals = withdrawal_list

	vd, err := b.beacon.GetSlotProposer(attrs.Slot)
	if err != nil {
		log.Info("could not get validator while submitting block", "err", err, "slot", attrs.Slot)
		return res, fmt.Errorf("could not get validator while submitting block: %w", err)
	}
	proposerPubkey, err := commonTypes.HexToPubkey(string(vd.PubkeyHex))
	if err != nil {
		log.Error("could not parse pubkey", "err", err, "pubkey", vd.PubkeyHex)
		return res, fmt.Errorf("could not parse pubkey of validator: %w", err)
	}

	blockNumber := currentBlock.Number.Uint64() + (attrs.Slot - currentSlot)

	parentBlock := b.eth.GetBlockChain().GetBlockByNumber(blockNumber - 1)
	if parentBlock == nil {
		log.Debug("parent block not found, using payload attributes parent block hash", "parent block number", blockNumber-1, "parent block hash", payload_base_attributes.ParentBlockHash)
		attrs.HeadHash = common.HexToHash(payload_base_attributes.ParentBlockHash)
	} else {
		attrs.HeadHash = parentBlock.Hash()
		log.Info("Using provided head block", "head block hash", attrs.HeadHash)
	}

	prevRandao, err := b.beacon.Randao(attrs.Slot - 1)
	if err != nil {
		log.Error("could not get previous randao", "err", err)
		return res, fmt.Errorf("could not get previous randao: %w", err)
	}
	attrs.Random = *prevRandao

	attrs.Timestamp = hexutil.Uint64(b.genesisInfo.GenesisTime + (attrs.Slot)*bbTypes.SLOT_DURATION)

	if attrs.PayoutPoolAddress == (common.Address{}) {
		// If payout pool address is not provided, use the relay payout address
		attrs.PayoutPoolAddress = b.relay.GetPayoutAddress()
	} else {
		log.Info("Using provided payout pool address", "payout pool address", attrs.PayoutPoolAddress)
	}

	attrs.GasLimit = b.eth.GetBlockGasCeil()

	b.slotMu.Lock()
	slotAttrs, ok := b.slotAttrs[attrs.Slot]
	b.slotMu.Unlock()

	if !ok {
		log.Info("New slot", "slot being bid for", attrs.Slot, "current slot", currentSlot)

		b.slotMu.Lock()
		b.slotAttrs[attrs.Slot] = make([]builderTypes.BuilderPayloadAttributes, 0)
		b.slotAttrs[attrs.Slot] = append(b.slotAttrs[attrs.Slot], *attrs)
		b.slotMu.Unlock()

	} else {
		b.slotMu.Lock()
		// Check if we have already received a block bid for this slot with the same attributes
		for _, slotAttr := range slotAttrs {
			slotAttrJson, _ := json.Marshal(slotAttr)
			attrsJson, _ := json.Marshal(*attrs)
			if string(slotAttrJson) == string(attrsJson) {
				log.Info("Received duplicate block attributes", "slot", attrs.Slot, "timestamp", attrs.Timestamp, "head hash", attrs.HeadHash, "fee recipient", attrs.SuggestedFeeRecipient, "bid amount", attrs.BidAmount)
				b.slotMu.Unlock()
				return res, errors.New("duplicate block attributes for slot " + strconv.Itoa(int(attrs.Slot)))
			}
		}
		b.slotAttrs[attrs.Slot] = append(b.slotAttrs[attrs.Slot], *attrs)
		b.slotMu.Unlock()
	}

	go b.DataCleanUp()

	// We are bidding for a slot bid before, but with different attributes in request
	// cancel the previous context and create a new one, as the user wishes to change action immediately
	b.slotMu.Lock()
	prevContextCancel := b.slotCtxCancel
	b.slotMu.Unlock()

	if prevContextCancel != nil {
		prevContextCancel() // Cancel the previous context
		log.Info("Cancelled previous running slot context as new block bid received", "slot", attrs.Slot)
	}

	// Update time for bids since new block bid attributes have been received
	timeForBids := time.Until(time.Unix(int64(slotTimeCutOff), 0))
	slotCtx, slotCtxCancel := context.WithTimeout(context.Background(), timeForBids)
	b.slotMu.Lock()
	b.slotCtx = slotCtx
	b.slotCtxCancel = slotCtxCancel
	b.slotMu.Unlock()

	log.Info("Preparing block for submission", "slot", attrs.Slot, "timestamp", attrs.Timestamp, "head hash", attrs.HeadHash, "fee recipient", attrs.SuggestedFeeRecipient, "bid amount", attrs.BidAmount, "gas limit", attrs.GasLimit, "payout pool address", attrs.PayoutPoolAddress, "time for bids", timeForBids)
	log.Info("Time for bids", "time for bids", timeForBids)

	// If new context and attributes for submission for this slot, start the slot builder and submitter
	var (
		bidComplete   chan struct{ bidAmount *big.Int }
		submissionErr error
		buildingErr   error
	)
	b.slotSubmissionsLock.Lock()
	bidComplete, ok = b.slotBidCompleteChan[attrs.Slot]
	if !ok {
		// If not ok then means all 3 channels are not initialised
		// as we initialise them all at the same time
		log.Info("Initialising slot channels", "slot", attrs.Slot)
		channel := make(chan blockProperties)
		bidComplete = make(chan struct{ bidAmount *big.Int }, 2)
		bountyComplete := make(chan struct{ bidAmount *big.Int })
		b.slotSubmissionsChan[attrs.Slot] = channel
		b.slotBidCompleteChan[attrs.Slot] = bidComplete
		b.slotBountyCompleteChan[attrs.Slot] = bountyComplete

		deadline := time.Unix(int64(slotBountyTimeCutOff), 0)
		timeForBuilding := time.Until(deadline)
		buildingCtx, _ := context.WithTimeout(context.Background(), timeForBuilding)
		go b.slotSubmitter(startTime, deadline, attrs.Slot, blockNumber, proposerPubkey, channel, bidComplete, bountyComplete, &submissionErr)
		go b.prepareBlock(buildingCtx, slotBountyTimeCutOff, attrs.Slot, proposerPubkey, channel, &buildingErr)
		log.Info("Started slot builder and submitter", "slot", attrs.Slot, "time for building", timeForBuilding)
	}
	b.slotSubmissionsLock.Unlock()

	successfulSubmission := false

processing:
	for {
		select {
		case <-slotCtx.Done():
			log.Info("Slot context done/cancelled. Ending slot processing", "slot", attrs.Slot)
			break processing
		case submission := <-bidComplete:
			if (submission.bidAmount == nil) || (submission.bidAmount != nil && submission.bidAmount.Sign() < 1) {
				log.Info("Submitted empty bid.", "slot", attrs.Slot)
				break processing
			} else {
				successfulSubmission = true
				log.Info("Slot submission complete for given bid attributes", "slot", attrs.Slot, "bidAmount", attrs.BidAmount)
				break processing
			}
		default:

			if slotCtx.Err() != nil {
				log.Info("Slot context done/cancelled. Ending slot processing", "slot", attrs.Slot)
				break processing
			}
		}
	}

	b.slotSubmissionsLock.Lock()
	defer b.slotSubmissionsLock.Unlock()

	if submissionErr != nil && !successfulSubmission {
		log.Error("error during submission", "slot", attrs.Slot, "err", submissionErr)
		return res, fmt.Errorf("error during submission. slot: %d, err: %s", attrs.Slot, submissionErr)
	}

	if buildingErr != nil && !successfulSubmission {
		log.Error("error during block building", "slot", attrs.Slot, "err", buildingErr)
		return res, fmt.Errorf("error during block building. slot: %d, err: %s", attrs.Slot, buildingErr)
	}

	if len(b.slotBidAmounts[attrs.Slot]) == 0 {
		return res, fmt.Errorf("could not submit any block bid for slot %d", attrs.Slot)
	}

	res = b.slotSubmissions[attrs.Slot]

	return res, nil

}
