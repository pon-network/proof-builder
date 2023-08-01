package builder

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	_ "os"
	"strconv"
	"time"

	"github.com/ethereum/go-ethereum/common"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/miner"

	builderTypes "github.com/bsn-eng/pon-golang-types/builder"
	commonTypes "github.com/bsn-eng/pon-golang-types/common"
	bbTypes "github.com/ethereum/go-ethereum/builder/types"
)

func (b *Builder) ProcessBuilderBountyBid(attrs *builderTypes.BuilderPayloadAttributes) ([]builderTypes.BlockBidResponse, error) {

	res := []builderTypes.BlockBidResponse{}
	startTime := time.Now()

	if attrs == nil {
		return res, fmt.Errorf("nil attributes")
	}

	if attrs.Slot == 0 {
		return res, fmt.Errorf("slot must be specified for bounty bid")
	}

	b.slotSubmissionsLock.Lock()
	bountySubmitted, ok := b.slotBountyAmount[attrs.Slot]
	if ok && (bountySubmitted>0) {
		b.slotSubmissionsLock.Unlock()
		return res, fmt.Errorf("bounty bid already submitted for slot %d", attrs.Slot)
	}

	// check if bid amount is 2 times greater at least than the previous bid
	if len(b.slotBidAmounts[attrs.Slot]) > 0 {
		lastBidAmount := b.slotBidAmounts[attrs.Slot][len(b.slotBidAmounts[attrs.Slot])-1]
		if attrs.BidAmount < lastBidAmount*2 {
			b.slotSubmissionsLock.Unlock()
			log.Info("bounty bid amount is not 2 times greater than the previous bid known", "slot", attrs.Slot, "bid amount", attrs.BidAmount, "last bid amount", lastBidAmount)
			return res, fmt.Errorf("bounty bid amount is not 2 times greater than the previous bid known, slot %d, bid amount %d, last bid amount %d", attrs.Slot, attrs.BidAmount, lastBidAmount)
		}
	}

	b.slotSubmissionsLock.Unlock()

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

	currentSlotStartTime := b.genesisInfo.GenesisTime + (currentSlot-1)*bbTypes.SLOT_DURATION
	slotBountyTimeCutOff := currentSlotStartTime + 3 // 3s into the current slot for the bounty bid

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
		if uint64(time.Now().Unix()) > slotBountyTimeCutOff {
			log.Error("slot past and not available to bounty bid on", "slot", attrs.Slot, "current slot", currentSlot, "next available slot for bid", currentSlot+1)
			return res, errors.New("slot " + strconv.Itoa(int(attrs.Slot)) + " is past and not available to bid on, current slot is " + strconv.Itoa(int(currentSlot)) + " and next available slot for bid is " + strconv.Itoa(int(currentSlot+1)))
		}
	} else {
		// Bidding on next slot, thus the bounty slot time cut off is 3s into the next slot
		slotBountyTimeCutOff = slotBountyTimeCutOff + bbTypes.SLOT_DURATION
	}

	knownAttrs := make([]builderTypes.BuilderPayloadAttributes, 0)

	b.slotMu.Lock()
	slotAttrs, ok := b.slotAttrs[attrs.Slot]
	if ok {
		knownAttrs = append(knownAttrs, slotAttrs...)
	}
	b.slotMu.Unlock()

	b.slotBountyMu.Lock()
	slotBountyAttrs, bountyAttrsOk := b.slotBountyAttrs[attrs.Slot]
	if bountyAttrsOk {
		knownAttrs = append(knownAttrs, slotBountyAttrs...)
	}
	b.slotBountyMu.Unlock()

	blockNumber := currentBlock.Number.Uint64() + (attrs.Slot - currentSlot)

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

	if len(knownAttrs) == 0 {

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

		parentBlock := b.eth.GetBlockChain().GetBlockByNumber(blockNumber - 1)
		if parentBlock == nil {
			log.Debug("parent block not found, using payload attributes parent block hash", "parent block number", blockNumber-1, "parent block hash", payload_base_attributes.ParentBlockHash)
			attrs.HeadHash = common.HexToHash(payload_base_attributes.ParentBlockHash)
		} else {
			attrs.HeadHash = parentBlock.Hash()
			log.Info("Using provided head block", "head block hash", attrs.HeadHash)
		}

		// If the attrs.Random is null address, use current block random
		if attrs.Random == (common.Hash{}) {
			prevRandao, err := b.beacon.Randao(attrs.Slot - 1)
			if err != nil {
				log.Error("could not get previous randao", "err", err)
				return res, fmt.Errorf("could not get previous randao: %w", err)
			}
			attrs.Random = *prevRandao
		}

		if attrs.Timestamp == 0 {
			attrs.Timestamp = hexutil.Uint64(b.genesisInfo.GenesisTime + (attrs.Slot)*bbTypes.SLOT_DURATION)
			log.Debug("timestamp not provided, creating timestamp", "timestamp", b.genesisInfo.GenesisTime+(attrs.Slot)*bbTypes.SLOT_DURATION, "slot", attrs.Slot)
		} else if attrs.Timestamp != hexutil.Uint64(b.genesisInfo.GenesisTime+attrs.Slot*bbTypes.SLOT_DURATION) {
			log.Error("timestamp not correct", "timestamp", attrs.Timestamp, "slot", attrs.Slot)
			return res, fmt.Errorf("Incorrect timestamp for slot. Expected %d, got %d", b.genesisInfo.GenesisTime+attrs.Slot*bbTypes.SLOT_DURATION, attrs.Timestamp)
		}

		// Attributes timestamp is checked and set, so time until bids can be calculated
		// correctly as the timestamp is the valid true time of the end of the slot

		attrs.PayoutPoolAddress = b.relay.GetPayoutAddress()

		if attrs.GasLimit <= 0 {
			attrs.GasLimit = miner.DefaultConfig.GasCeil
		}

	} else {
		b.slotMu.Lock()
		// We already have attributes for this slot,
		// so update the attributes with the latest attributes
		// if nothing new has been provided

		// Aims to use bounty bid attributes if they are available as last to append
		// to the list of known attributes
		latestAttrs := knownAttrs[len(slotAttrs)-1]
		attrs.Withdrawals = latestAttrs.Withdrawals
		attrs.PayoutPoolAddress = latestAttrs.PayoutPoolAddress

		if attrs.GasLimit == 0 {
			attrs.GasLimit = latestAttrs.GasLimit
		}
		if attrs.Timestamp == 0 {
			attrs.Timestamp = latestAttrs.Timestamp
		} else if attrs.Timestamp != hexutil.Uint64(b.genesisInfo.GenesisTime+attrs.Slot*bbTypes.SLOT_DURATION) {
			log.Error("timestamp not correct", "timestamp", attrs.Timestamp, "slot", attrs.Slot)
			return res, fmt.Errorf("Incorrect timestamp for slot. Expected %d, got %d", b.genesisInfo.GenesisTime+attrs.Slot*bbTypes.SLOT_DURATION, attrs.Timestamp)
		}
		if attrs.HeadHash == (common.Hash{}) {
			attrs.HeadHash = latestAttrs.HeadHash
		}
		if attrs.Random == (common.Hash{}) {
			attrs.Random = latestAttrs.Random
		}
		if attrs.SuggestedFeeRecipient == (common.Address{}) {
			attrs.SuggestedFeeRecipient = latestAttrs.SuggestedFeeRecipient
		}
		if attrs.Transactions == nil {
			attrs.Transactions = latestAttrs.Transactions
		}
		b.slotMu.Unlock()

	}

	if !bountyAttrsOk {
		log.Info("New slot bounty", "bounty submission for slot", attrs.Slot, "current slot", currentSlot)

		b.slotBountyMu.Lock()
		b.slotBountyAttrs[attrs.Slot] = make([]builderTypes.BuilderPayloadAttributes, 0)
		b.slotBountyAttrs[attrs.Slot] = append(b.slotBountyAttrs[attrs.Slot], *attrs)
		b.slotBountyMu.Unlock()

	} else {
		b.slotBountyMu.Lock()
		// Check if we have already received a block bid for this slot with the same attributes
		for _, slotBountyAttr := range slotBountyAttrs {
			slotBountyAttrJson, _ := json.Marshal(slotBountyAttr)
			attrsJson, _ := json.Marshal(*attrs)
			if string(slotBountyAttrJson) == string(attrsJson) {
				log.Info("Received duplicate bounty block attributes", "slot", attrs.Slot, "timestamp", attrs.Timestamp, "head hash", attrs.HeadHash, "fee recipient", attrs.SuggestedFeeRecipient, "bid amount", attrs.BidAmount)
				b.slotMu.Unlock()
				return res, errors.New("duplicate bounty block attributes for slot " + strconv.Itoa(int(attrs.Slot)))
			}
		}
		b.slotBountyAttrs[attrs.Slot] = append(b.slotBountyAttrs[attrs.Slot], *attrs)
		b.slotBountyMu.Unlock()
	}

	go b.DataCleanUp()

	timeForBounty := time.Until(time.Unix(int64(slotBountyTimeCutOff), 0))
	ctx, ctxCancel := context.WithTimeout(context.Background(), timeForBounty)

	defer ctxCancel()

	// If first time submitting for this slot, start the slot builder and submitter
	var (
		bountyComplete chan struct{ bidAmount uint64 }
		submissionErr  error
		buildingErr    error
	)
	b.slotSubmissionsLock.Lock()
	bountyComplete, ok = b.slotBountyCompleteChan[attrs.Slot]
	if !ok {
		// If not ok then means all 3 channels are not initialised
		// as we initialise them all at the same time

		channel := make(chan blockProperties)
		bidComplete := make(chan struct{ bidAmount uint64 })
		bountyComplete = make(chan struct{ bidAmount uint64 })
		b.slotSubmissionsChan[attrs.Slot] = channel
		b.slotBidCompleteChan[attrs.Slot] = bidComplete
		b.slotBountyCompleteChan[attrs.Slot] = bountyComplete

		deadline := time.Unix(int64(slotBountyTimeCutOff), 0)
		timeForBuilding := time.Until(deadline)
		buildingCtx, _ := context.WithTimeout(context.Background(), timeForBuilding)
		go b.slotSubmitter(startTime, deadline, attrs.Slot, blockNumber, proposerPubkey, channel, bidComplete, bountyComplete, &submissionErr)
		go b.prepareBlock(buildingCtx, slotBountyTimeCutOff, attrs.Slot, proposerPubkey, channel, &buildingErr)
	}
	b.slotSubmissionsLock.Unlock()

	successfulSubmission := false

processing:
	for {
		select {
		case <-ctx.Done():
			log.Info("Slot context done/cancelled for bounty bid. Ending slot bounty processing", "slot", attrs.Slot)
			break processing
		case submission := <-bountyComplete:
			if submission.bidAmount == 0 {
				log.Info("Submitted empty bid.", "slot", attrs.Slot)
				break processing
			}
			if submission.bidAmount == attrs.BidAmount {
				successfulSubmission = true
				log.Info("Slot submission complete for given bounty attributes", "slot", attrs.Slot, "bidAmount", attrs.BidAmount)
				break processing
			}

			// Else the submission was not for these attributes and rather for another set of attributes

		default:

		}
	}

	b.slotSubmissionsLock.Lock()
	defer b.slotSubmissionsLock.Unlock()

	if submissionErr != nil && !successfulSubmission {
		log.Error("error during bounty submission", "slot", attrs.Slot, "err", submissionErr)
		return res, fmt.Errorf("error during bounty submission. slot: %d, err: %s", attrs.Slot, submissionErr)
	}

	if buildingErr != nil && !successfulSubmission {
		log.Error("error during block building", "slot", attrs.Slot, "err", buildingErr)
		return res, fmt.Errorf("error during bounty block building. slot: %d, err: %s", attrs.Slot, buildingErr)
	}

	if (b.slotBountyAmount[attrs.Slot]==0) {
		return res, fmt.Errorf("could not submit any bounty block bid for slot %d", attrs.Slot)
	}

	res = b.slotSubmissions[attrs.Slot]

	return res, nil

}
