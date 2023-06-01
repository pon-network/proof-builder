package builder

import (
	"context"
	"crypto/ecdsa"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	_ "os"
	"strconv"
	"sync"
	"time"

	capellaApi "github.com/attestantio/go-eth2-client/api/v1/capella"
	"github.com/attestantio/go-eth2-client/spec/bellatrix"
	capella "github.com/attestantio/go-eth2-client/spec/capella"
	"github.com/attestantio/go-eth2-client/spec/phase0"

	"github.com/ethereum/go-ethereum/common"

	"github.com/ethereum/go-ethereum/beacon/engine"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/miner"

	beaconTypes "github.com/bsn-eng/pon-golang-types/beaconclient"
	"github.com/ethereum/go-ethereum/builder/bls"
	builderTypes "github.com/ethereum/go-ethereum/builder/types"
	"github.com/ethereum/go-ethereum/crypto"

	beacon "github.com/ethereum/go-ethereum/builder/beacon"
	"github.com/ethereum/go-ethereum/builder/database"
	RPBS "github.com/ethereum/go-ethereum/builder/rpbsService"
)

var executionPayloadCache = make(map[uint64]engine.ExecutableData)

type Builder struct {
	relay                IRelay
	eth                  IEthService
	beacon               *beacon.MultiBeaconClient
	rpbs                 *RPBS.RPBSService
	db                   *database.DatabaseService
	builderSecretKey     *bls.SecretKey
	builderPublicKey     builderTypes.PublicKey
	builderSigningDomain builderTypes.Domain

	builderWalletPrivateKey *ecdsa.PrivateKey
	builderWalletAddress    common.Address

	AccessPoint string

	MetricsEnabled bool

	slotMu        sync.Mutex
	slot          uint64
	slotAttrs     []builderTypes.BuilderPayloadAttributes
	slotCtx       context.Context
	slotCtxCancel context.CancelFunc
}

func NewBuilder(sk *bls.SecretKey, beaconClient *beacon.MultiBeaconClient, relay IRelay, builderSigningDomain builderTypes.Domain, eth IEthService, rpbs *RPBS.RPBSService, database *database.DatabaseService, config *Config) (*Builder, error) {
	pkBytes := bls.PublicKeyFromSecretKey(sk).Compress()
	pk := builderTypes.PublicKey{}
	err := pk.FromSlice(pkBytes)
	if err != nil {
		return nil, err
	}

	slotCtx, slotCtxCancel := context.WithCancel(context.Background())
	return &Builder{
		relay:                relay,
		eth:                  eth,
		beacon:               beaconClient,
		rpbs:                 rpbs,
		db:                   database,
		builderSecretKey:     sk,
		builderPublicKey:     pk,
		builderSigningDomain: builderSigningDomain,

		builderWalletPrivateKey: config.BuilderWalletPrivateKey,
		builderWalletAddress:    config.BuilderWalletAddress,

		AccessPoint:    config.ExposedAccessPoint,
		MetricsEnabled: config.MetricsEnabled,

		slot:          0,
		slotCtx:       slotCtx,
		slotCtxCancel: slotCtxCancel,
	}, nil
}

func (b *Builder) OnPrivateTransactions(transactionBytes *builderTypes.PrivateTransactionsPayload) ([]*types.Transaction, error) {
	txPool := b.eth.GetTxPool()
	if txPool == nil {
		return nil, fmt.Errorf("tx pool is not available")
	}

	transactions, err := engine.DecodeTransactions(transactionBytes.Transactions)
	if err != nil {
		return nil, fmt.Errorf("could not decode transactions: %w", err)
	}

	for _, tx := range transactions {
		err = txPool.AddLocal(tx)
		if err != nil {
			log.Error("could not add private tx", "err", err)
			return nil, fmt.Errorf("could not add private tx: %w", err)
		}
	}

	return transactions, nil

}

func (b *Builder) submitBlockBid(block *types.Block, blockFees *big.Int, sealedAt time.Time, payoutPoolTx []byte, proposerPubkey builderTypes.PublicKey, attrs *builderTypes.BuilderPayloadAttributes) (builderTypes.BuilderBlockBid, error) {
	executionPayloadEnvlope := engine.BlockToExecutableData(block, big.NewInt(0))
	executableData := executionPayloadEnvlope.ExecutionPayload
	block, err := engine.ExecutableDataToBlock(*executableData)
	if err != nil {
		log.Error("could not format execution payload", "err", err)
		return builderTypes.BuilderBlockBid{}, err
	}

	executionPayloadHeader := block.Header()
	if err != nil {
		log.Error("could not format execution payload header", "err", err)
		return builderTypes.BuilderBlockBid{}, err
	}

	payoutpoolTx_jsonString, err := json.Marshal(payoutPoolTx)
	if err != nil {
		log.Error("could not convert payout pool tx to string", "err", err)
		return builderTypes.BuilderBlockBid{}, err
	}

	rpbsSig, err := b.rpbs.RPBSCommitToSignature(
		RPBS.RpbsCommitMessage{
			Slot:                 attrs.Slot,
			Amount:               attrs.BidAmount,
			BuilderWalletAddress: b.builderWalletAddress.String(),
			TxBytes:              string(payoutpoolTx_jsonString),
		},
	)

	if err != nil {
		log.Error("could not sign rpbs commit", "err", err)
		return builderTypes.BuilderBlockBid{}, err
	}

	rpbs_json, err := json.Marshal(rpbsSig)
	if err != nil {
		log.Error("could not convert rpbs blinded signature to string", "err", err)
		return builderTypes.BuilderBlockBid{}, err
	}

	rpbsPubkey, err := b.rpbs.PublicKey()
	if err != nil {
		log.Error("could not get rpbs pubkey", "err", err)
		return builderTypes.BuilderBlockBid{}, err
	}

	parentHash := phase0.Hash32{}
	copy(parentHash[:], executionPayloadHeader.ParentHash.Bytes()[:])

	feeRecipient := bellatrix.ExecutionAddress{}
	copy(feeRecipient[:], executionPayloadHeader.Coinbase.Bytes()[:])

	blockHash := phase0.Hash32{}
	copy(blockHash[:], block.Hash().Bytes()[:])

	transactionsRoot := phase0.Root{}
	copy(transactionsRoot[:], executionPayloadHeader.TxHash.Bytes()[:])

	withdrawalsRoot := phase0.Root{}
	copy(withdrawalsRoot[:], executionPayloadHeader.WithdrawalsHash.Bytes()[:])

	logsBloom := [256]byte{}
	copy(logsBloom[:], executionPayloadHeader.Bloom.Bytes()[:])

	baseFeePerGas := [32]byte{}
	copy(baseFeePerGas[:], executionPayloadHeader.BaseFee.Bytes()[:])

	prevRandao := [32]byte{}
	copy(prevRandao[:], executionPayloadHeader.MixDigest.Bytes()[:])

	receiptRoot := [32]byte{}
	copy(receiptRoot[:], executionPayloadHeader.ReceiptHash.Bytes()[:])

	stateRoot := [32]byte{}
	copy(stateRoot[:], executionPayloadHeader.Root.Bytes()[:])

	extraData := [32]byte{}
	copy(extraData[:], []byte("Blockswap PoN v1 ponrelay.com"))

	capellaExecutionPayloadHeader := capella.ExecutionPayloadHeader{
		ParentHash:       parentHash,
		FeeRecipient:     feeRecipient,
		StateRoot:        stateRoot,
		ReceiptsRoot:     receiptRoot,
		LogsBloom:        logsBloom,
		PrevRandao:       prevRandao,
		BlockNumber:      executionPayloadHeader.Number.Uint64(),
		GasLimit:         executionPayloadHeader.GasLimit,
		GasUsed:          executionPayloadHeader.GasUsed,
		Timestamp:        executionPayloadHeader.Time,
		ExtraData:        extraData[:],
		BaseFeePerGas:    baseFeePerGas,
		BlockHash:        blockHash,
		TransactionsRoot: transactionsRoot,
		WithdrawalsRoot:  withdrawalsRoot,
	}

	blockBidMsg := builderTypes.BidPayload{
		Slot:                 attrs.Slot,
		ParentHash:           builderTypes.Hash(executionPayloadHeader.ParentHash),
		BlockHash:            builderTypes.Hash(block.Hash()),
		BuilderPubkey:        b.builderPublicKey,
		ProposerPubkey:       proposerPubkey,
		ProposerFeeRecipient: builderTypes.Address(attrs.SuggestedFeeRecipient),
		GasLimit:             executableData.GasLimit,
		GasUsed:              executableData.GasUsed,
		Value:                attrs.BidAmount,

		BuilderWalletAddress:   builderTypes.Address(b.builderWalletAddress),
		PayoutPoolTransaction:  payoutPoolTx,
		ExecutionPayloadHeader: &capellaExecutionPayloadHeader,
		Endpoint:               b.AccessPoint + _PathSubmitBlindedBlock,
		RPBS:                   rpbsSig,
		RPBSPubkey:             rpbsPubkey,
	}

	signature, err := builderTypes.SignMessage(&blockBidMsg, b.builderSigningDomain, b.builderSecretKey)
	if err != nil {
		log.Error("could not sign builder bid", "err", err)
		return builderTypes.BuilderBlockBid{}, err
	}

	// ECDSA signature over the signature of the block bid message
	ecdsa_signature, err := crypto.Sign(crypto.Keccak256Hash(signature[:]).Bytes(), b.builderWalletPrivateKey)
	if err != nil {
		log.Error("Could not ECDSA sign block bid message sig", "err", err)
	}

	blockSubmitReq := builderTypes.BuilderBlockBid{
		Signature:      signature,
		Message:        &blockBidMsg,
		EcdsaSignature: builderTypes.EcdsaSignature(ecdsa_signature),
	}

	// Submit the block bid to the relay
	submissionResp, err := b.relay.SubmitBlockBid(&blockSubmitReq)
	if err != nil {
		log.Error("could not submit block", "err", err)
		return builderTypes.BuilderBlockBid{}, err
	}

	submissionResp_string, err := json.Marshal(submissionResp)
	if err != nil {
		log.Error("could not convert response from relay to string", "err", err)
		return builderTypes.BuilderBlockBid{}, err
	}

	log.Info("Submitted block", "slot", blockBidMsg.Slot, "value", blockBidMsg.Value, "parent", blockBidMsg.ParentHash, "response", submissionResp_string)
	executionPayloadCache[attrs.Slot] = *executableData

	log.Info("Executable data cached", "slot", attrs.Slot, "hash", executableData.BlockHash, "parentHash", executableData.ParentHash, "gasUsed", executableData.GasUsed, "transaction count", len(executableData.Transactions), "withdrawals count", len(executableData.Withdrawals))

	if b.MetricsEnabled {
		blockBidEntry := database.BuilderBlockBidEntry{
			InsertedAt:                time.Now(),
			Signature:                 signature.String(),
			Slot:                      attrs.Slot,
			BuilderPubkey:             b.builderPublicKey.String(),
			ProposerPubkey:            proposerPubkey.String(),
			FeeRecipient:              attrs.SuggestedFeeRecipient.String(),
			GasLimit:                  executableData.GasLimit,
			GasUsed:                   executableData.GasUsed,
			MEV:                       blockFees.Uint64(),
			PayoutPoolTx:              string(payoutpoolTx_jsonString),
			PayoutPoolAddress:         b.relay.GetPayoutAddress().String(),
			PayoutPoolGasFee:          miner.PaymentTxGas,
			RPBS:                      string(rpbs_json),
			PriorityTransactionsCount: uint64(len(attrs.Transactions)),
			TransactionsCount:         uint64(len(executableData.Transactions)),
			BlockHash:                 block.Hash().String(),
			ParentHash:                executionPayloadHeader.ParentHash.String(),
			BlockNumber:               executionPayloadHeader.Number.Uint64(),
			Value:                     big.NewInt(int64(attrs.BidAmount)).Uint64(),
		}
		go b.db.InsertBlockBid(blockBidEntry)
	}

	return blockSubmitReq, nil
}

func (b *Builder) ProcessBuilderBid(attrs *builderTypes.BuilderPayloadAttributes) (builderTypes.BuilderBlockBid, error) {

	res := builderTypes.BuilderBlockBid{}

	if attrs == nil {
		return res, nil
	}

	if !b.eth.Synced() {
		return res, errors.New("backend not Synced")
	}

	currentSlot := b.beacon.BeaconData.CurrentSlot

	// If attrs.Slot is not the next slot, return
	if attrs.Slot <= currentSlot && attrs.Slot != 0 {
		log.Error("slot past and not available to bid on", "slot", attrs.Slot, "current slot", currentSlot, "next available slot for bid", currentSlot+1)
		return res, errors.New("slot " + strconv.Itoa(int(attrs.Slot)) + " is past and not available to bid on, current slot is " + strconv.Itoa(int(currentSlot)) + " and next available slot for bid is " + strconv.Itoa(int(currentSlot+1)))
	}

	if attrs.Slot > currentSlot+1 {
		log.Error("slot too far in the future", "slot", attrs.Slot, "current slot", currentSlot)
		return res, errors.New("slot " + strconv.Itoa(int(attrs.Slot)) + " is too far in the future, current slot is " + strconv.Itoa(int(currentSlot)) + " and next available slot for bid is " + strconv.Itoa(int(currentSlot+1)))
	}

	vd, err := b.beacon.GetSlotProposer(attrs.Slot)
	if err != nil {
		log.Info("could not get validator while submitting block", "err", err, "slot", attrs.Slot)
		return res, err
	}

	payload_base_attributes, err := b.beacon.GetPayloadAttributesForSlot(attrs.Slot)
	if err != nil {
		log.Info("could not get payload attributes while submitting block", "err", err, "slot", attrs.Slot)
		return res, err
	}

	withdrawal_list := types.Withdrawals{}
	for _, w := range *payload_base_attributes.PayloadAttributes.Withdrawals {
		withdrawal_list = append(withdrawal_list, &types.Withdrawal{
			Index:     uint64(w.Index),
			Validator: uint64(w.ValidatorIndex),
			Address:   common.HexToAddress(w.Address),
			Amount:    uint64(w.Amount),
		})
	}

	attrs.Withdrawals = withdrawal_list

	proposerPubkey, err := builderTypes.HexToPubkey(string(vd.PubkeyHex))
	if err != nil {
		log.Error("could not parse pubkey", "err", err, "pubkey", vd.PubkeyHex)
		return res, err
	}

	currentBlock := b.eth.CurrentBlock()

	parentBlock := b.eth.GetBlockByHash(attrs.HeadHash)
	if parentBlock == nil {
		log.Warn("Block hash not found in blocktree", "head block hash provided", attrs.HeadHash)
		log.Info("Using head block instead", "head block hash", currentBlock.Hash())
		attrs.HeadHash = currentBlock.Hash()
	} else {
		log.Info("Using provided head block", "head block hash", attrs.HeadHash)
	}

	// If the attrs.Random is null address, use current block random
	if attrs.Random == (common.Hash{}) {
		prevRandao, err := b.beacon.Randao(attrs.Slot - 1)
		if err != nil {
			log.Error("could not get previous randao", "err", err)
			return res, err
		}
		attrs.Random = *prevRandao
	}

	if attrs.Timestamp == 0 {
		attrs.Timestamp = hexutil.Uint64(builderTypes.GENESIS_TIME + (attrs.Slot)*builderTypes.SLOT_DURATION)
		log.Info("timestamp not provided, creating timestamp", "timestamp", builderTypes.GENESIS_TIME+(attrs.Slot)*builderTypes.SLOT_DURATION, "slot", attrs.Slot)
	} else if attrs.Timestamp != hexutil.Uint64(builderTypes.GENESIS_TIME+attrs.Slot*builderTypes.SLOT_DURATION) {
		log.Error("timestamp not correct", "timestamp", attrs.Timestamp, "slot", attrs.Slot)
		return res, errors.New("timestamp " + strconv.Itoa(int(attrs.Timestamp)) + " not correct, slot is " + strconv.Itoa(int(attrs.Slot)))
	}

	b.slotMu.Lock()

	// Check if we are in a new slot
	if b.slot != attrs.Slot {
		if b.slotCtxCancel != nil {
			b.slotCtxCancel()
		}

		log.Info("New slot", "slot", attrs.Slot, "current slot", currentSlot, "next available slot for bid", currentSlot+1)

		// Attributes contains the block timestamp. Check this against the current time to know how long we have to put a bid.

		timeForBids := time.Until(time.Unix(int64(attrs.Timestamp+2), 0))

		slotCtx, slotCtxCancel := context.WithTimeout(context.Background(), timeForBids)
		b.slot = attrs.Slot
		b.slotAttrs = nil
		b.slotCtx = slotCtx
		b.slotCtxCancel = slotCtxCancel

		// Check if execution payload cache has any old slots
		for slot := range executionPayloadCache {
			if slot < currentSlot-32 { // Delete any slots that are 32 slots behind the current slot
				delete(executionPayloadCache, slot)
			}
		}
	}

	log.Info("Received block attributes", "slot", attrs.Slot, "timestamp", attrs.Timestamp, "head hash", attrs.HeadHash, "fee recipient", attrs.SuggestedFeeRecipient, "bid amount", attrs.BidAmount)

	// Check if we have already received a block bid for this slot with exactly the same attributes
	for _, slotAttr := range b.slotAttrs {
		slotAttrJson, _ := json.Marshal(slotAttr)
		attrsJson, _ := json.Marshal(*attrs)
		if string(slotAttrJson) == string(attrsJson) {
			log.Info("Received duplicate block attributes", "slot", attrs.Slot, "timestamp", attrs.Timestamp, "head hash", attrs.HeadHash, "fee recipient", attrs.SuggestedFeeRecipient, "bid amount", attrs.BidAmount)
			b.slotMu.Unlock()
			return res, errors.New("duplicate block attributes for slot " + strconv.Itoa(int(attrs.Slot)))
		}
	}

	// User is requesting to send a new block bid for this slot, cancel the previous context and create a new one
	if b.slotCtxCancel != nil {
		b.slotCtxCancel()
	}

	timeForBids := time.Until(time.Unix(int64(attrs.Timestamp+2), 0))

	slotCtx, slotCtxCancel := context.WithTimeout(context.Background(), timeForBids)
	b.slotCtx = slotCtx
	b.slotCtxCancel = slotCtxCancel

	b.slotAttrs = append(b.slotAttrs, *attrs)
	// Can be used later to keep track of the different bids for the same slot by the same user
	b.slotMu.Unlock()
	attrs.PayoutPoolAddress = b.relay.GetPayoutAddress()

	if attrs.GasLimit <= 0 {
		attrs.GasLimit = 0
	}

	result, err := b.prepareBlock(b.slotCtx, proposerPubkey, attrs)
	if err != nil {
		log.Error("could not run building job", "err", err)
		return builderTypes.BuilderBlockBid{}, err
	}

	return result, nil
}

type blockQueueEntry struct {
	block               *types.Block
	blockExecutableData *engine.ExecutableData
	payoutPoolTx        []byte
	sealedAt            time.Time
	blockValue          *big.Int
}

func (b *Builder) prepareBlock(slotCtx context.Context, proposerPubkey builderTypes.PublicKey, attrs *builderTypes.BuilderPayloadAttributes) (builderTypes.BuilderBlockBid, error) {
	// Attributes contains the block timestamp. Check this against the current time to know how long we have to build the block.
	// We need to build the block before the timestamp, otherwise the block will be invalid.
	log.Info("block builder run building job", "slot", attrs.Slot, "timestamp", attrs.Timestamp, "head hash", attrs.HeadHash, "fee recipient", attrs.SuggestedFeeRecipient, "bid amount", attrs.BidAmount)

	timeForJob := time.Until(time.Unix(int64(attrs.Timestamp+2), 0))
	log.Info("block builder time for new job", "timeForJob", timeForJob)

	ctx, cancel := context.WithTimeout(slotCtx, timeForJob)
	defer cancel()

	// Submission queue for the given payload attributes
	// multiple jobs can run for different attributes fot the given slot
	// 1. When new block is ready we check if its gasUsed is higher than gasUsed of last best block
	//    if it is we set queueBest* to values of the new block and notify queueSignal channel.
	// 2. Submission goroutine waits for queueSignal and submits queueBest* if its more valuable than
	//    existing queueLastBestEntry blockValue (gasUsed).
	//    Submission goroutine is globally rate limited to have fixed rate of submissions for all jobs.

	var (
		queueSignal = make(chan struct{}, 1)

		queueLastSubmittedBlockBid builderTypes.BuilderBlockBid
		queueError                 error

		submission1 bool = false
		submission2 bool = false

		queueMu        sync.Mutex
		queueBestEntry blockQueueEntry
	)

	log.Debug("prepareBlock", "slot", attrs.Slot, "parent", attrs.HeadHash)

	submitBestBlock := func() {

		contextDeadline, ok := ctx.Deadline()
		if !ok {
			log.Error("could not get context deadline")
			return
		}

		if submission1 == false || (submission2 == false && time.Until(contextDeadline) < 2*time.Second) {

			log.Info("submitBestBlock", "slot", attrs.Slot, "parent", attrs.HeadHash)

			timeTillDeadline := time.Until(contextDeadline)
			log.Info("block builder submission time till deadline", "timeTillDeadline", timeTillDeadline)

			finalizedBid, err := b.submitBlockBid(queueBestEntry.block, queueBestEntry.blockValue, queueBestEntry.sealedAt, queueBestEntry.payoutPoolTx, proposerPubkey, attrs)
			if err != nil {
				log.Error("could not run sealed block hook", "err", err)
				queueMu.Lock()
				if queueLastSubmittedBlockBid.Message == (builderTypes.BuilderBlockBid{}.Message) {
					queueError = err
				}
				queueMu.Unlock()
				return
			} else {
				queueMu.Lock()
				queueLastSubmittedBlockBid = finalizedBid
				queueError = nil
				if submission1 == false {
					submission1 = true
				} else if submission2 == false {
					submission2 = true
				}
				queueMu.Unlock()
				return
			}

		}

	}

	// Empties queue, submits the best block for current job and waits for new submissions
	go processQueueForSubmission(ctx, queueSignal, submitBestBlock)

	// Populates queue with submissions that increase block value
	sealedBlockCallback := func(block *types.Block, fees *big.Int, payoutPoolTx []byte, err error) {

		if ctx.Err() != nil {
			return
		}

		if err != nil {
			log.Error("block builder error", "err", err)
			queueMu.Lock()
			if queueLastSubmittedBlockBid.Message == (builderTypes.BuilderBlockBid{}.Message) {
				queueError = err
			}
			queueMu.Unlock()
			return
		}

		if payoutPoolTx == nil {
			log.Error("could not create payout pool transaction")
			queueMu.Lock()
			if queueLastSubmittedBlockBid.Message == (builderTypes.BuilderBlockBid{}.Message) {
				queueError = errors.New("could not create payout pool transaction")
			}
			queueMu.Unlock()
			return
		}

		sealedAt := time.Now()
		log.Info("block builder received new block", "sealedAt", sealedAt, "blockValue", fees, "existingValue", queueBestEntry.blockValue)

		if queueBestEntry.blockValue == nil {
			executionPayloadEnvelope := engine.BlockToExecutableData(block, fees)

			queueMu.Lock()
			queueBestEntry = blockQueueEntry{
				block:               block,
				blockExecutableData: executionPayloadEnvelope.ExecutionPayload,
				sealedAt:            sealedAt,
				blockValue:          executionPayloadEnvelope.BlockValue,
				payoutPoolTx:        payoutPoolTx,
			}
			queueMu.Unlock()

			log.Info("block builder first block", "sealedAt", sealedAt, "blockValue", fees)

			select {
			case queueSignal <- struct{}{}:
			default:
			}

		} else if fees.Cmp(queueBestEntry.blockValue) >= 0 {
			// Allow greater than or equal to, so that we can submit the same block again if geth creates a new block with the same gasUsed
			// This can happen if the block builder is restarted, and the same block is built again
			// This is a workaround for a bug in geth, where sometimes builds a block with 0 gasUsed, or builds a block with different
			// logs and extraData than the previous but with the same gasUsed
			executionPayloadEnvelope := engine.BlockToExecutableData(block, fees)

			queueMu.Lock()
			queueBestEntry = blockQueueEntry{
				block:               block,
				blockExecutableData: executionPayloadEnvelope.ExecutionPayload,
				sealedAt:            sealedAt,
				blockValue:          executionPayloadEnvelope.BlockValue,
				payoutPoolTx:        payoutPoolTx,
			}
			queueMu.Unlock()

			log.Info("block builder new best block", "sealedAt", sealedAt, "blockValue", fees)

			select {
			case queueSignal <- struct{}{}:
			default:
			}

		} else if submission1 == false {
			// If not submitted a block yet or encountered an error in submission
			// trigger queueSignal to submit the best block again

			log.Info("block builder resubmitting first block", "sealedAt", queueBestEntry.sealedAt, "blockValue", queueBestEntry.blockValue)
			select {
			case queueSignal <- struct{}{}:
			default:
			}
		} else {
			log.Info("new block does not increase block value, skipping", "sealedAt", sealedAt, "blockValue", fees, "existingValue", queueBestEntry.blockValue)
		}
		return

	}

	// resubmits block builder requests to workers to build the best block with the highest value (gasUsed)
	// From the block builder
	complete := runRetryLoop(ctx, 500*time.Millisecond, func() {
		log.Info("Building Block with Geth", "slot", attrs.Slot, "parent", attrs.HeadHash)

		if attrs.NoMempoolTxs == true && queueBestEntry.block != nil && (submission1 == true) {
			//  If mempool transactions are disabled, we only want to build the block once
			//  and then submit it. This is because the block builder will essentially build
			//  the same block every time, so we don't want to waste time building the same
			log.Info("Mempool transactions disabled, skipping futher block building attempts")
			cancel()
			return
		}

		err := b.eth.BuildBlock(attrs, sealedBlockCallback)
		if err != nil {
			log.Warn("Failed to build block", "err", err)
			queueMu.Lock()
			if submission1 == false {
				queueError = err
			}
			queueMu.Unlock()
		}
	})

	if complete {
		deadline, ok := ctx.Deadline()
		if ok {
			log.Info("BuildBlock retry loop completed", "slot", attrs.Slot, "parent", attrs.HeadHash, "timeUntilDeadline", time.Until(deadline))
		} else {
			log.Info("BuildBlock retry loop completed", "slot", attrs.Slot, "parent", attrs.HeadHash)
		}
		if submission1 == false {
			queueMu.Lock()
			if queueError == nil {
				// Then it means the context was cancelled or deadline reached before a block was submitted
				queueError = fmt.Errorf("Failed to submit any block on time: context cancelled/ slot submission deadline reached")
			}
			queueMu.Unlock()
		}

		return queueLastSubmittedBlockBid, queueError
	}

	return builderTypes.BuilderBlockBid{}, fmt.Errorf("context cancelled")
}

func (b *Builder) SubmitBlindedBlock(blindedBlock capellaApi.BlindedBeaconBlock, signature phase0.BLSSignature) (capella.ExecutionPayload, error) {

	var executionPayload capella.ExecutionPayload
	var beaconBlock beaconTypes.SignedBeaconBlock

	slot := uint64(blindedBlock.Slot)

	executableData, exists := executionPayloadCache[slot]
	if !exists {
		return capella.ExecutionPayload{}, errors.New("execution payload not found")
	}

	log.Info("executionPayload found", "slot", slot, "hash", executableData.BlockHash.String())

	// Block hash is unique to the block, so we can use it to verify that the block has not been modified
	if blindedBlock.Body.ExecutionPayloadHeader.BlockHash.String() != executableData.BlockHash.String() {
		return capella.ExecutionPayload{}, errors.New("Block hash has been changed from the expected value, block has been modified")
	}

	if b.MetricsEnabled {
		blindedBeaconBlockEntry := database.SignedBlindedBeaconBlockEntry{
			InsertedAt:               time.Now(),
			Signature:                signature.String(),
			SignedBlindedBeaconBlock: blindedBlock.String(),
		}
		go b.db.InsertBlindedBeaconBlock(blindedBeaconBlockEntry, blindedBlock.Body.ExecutionPayloadHeader.BlockHash.String())
	}

	withdrawals := []*capella.Withdrawal{}
	for _, withdrawal := range executableData.Withdrawals {
		withdrawalAddress := bellatrix.ExecutionAddress{}
		copy(withdrawalAddress[:], withdrawal.Address.Bytes()[:])

		withdrawals = append(withdrawals, &capella.Withdrawal{
			Index:          capella.WithdrawalIndex(withdrawal.Index),
			ValidatorIndex: phase0.ValidatorIndex(withdrawal.Validator),
			Amount:         phase0.Gwei(withdrawal.Amount),
			Address:        withdrawalAddress,
		})
	}

	transactions := []bellatrix.Transaction{}
	for _, transaction := range executableData.Transactions {
		transactions = append(transactions, transaction)
	}

	executionPayload = capella.ExecutionPayload{
		ParentHash:    blindedBlock.Body.ExecutionPayloadHeader.ParentHash,
		FeeRecipient:  blindedBlock.Body.ExecutionPayloadHeader.FeeRecipient,
		StateRoot:     blindedBlock.Body.ExecutionPayloadHeader.StateRoot,
		ReceiptsRoot:  blindedBlock.Body.ExecutionPayloadHeader.ReceiptsRoot,
		LogsBloom:     blindedBlock.Body.ExecutionPayloadHeader.LogsBloom,
		PrevRandao:    blindedBlock.Body.ExecutionPayloadHeader.PrevRandao,
		BlockNumber:   blindedBlock.Body.ExecutionPayloadHeader.BlockNumber,
		GasLimit:      blindedBlock.Body.ExecutionPayloadHeader.GasLimit,
		GasUsed:       blindedBlock.Body.ExecutionPayloadHeader.GasUsed,
		Timestamp:     blindedBlock.Body.ExecutionPayloadHeader.Timestamp,
		ExtraData:     blindedBlock.Body.ExecutionPayloadHeader.ExtraData,
		BaseFeePerGas: blindedBlock.Body.ExecutionPayloadHeader.BaseFeePerGas,
		BlockHash:     blindedBlock.Body.ExecutionPayloadHeader.BlockHash,
		Withdrawals:   withdrawals,
		Transactions:  transactions,
	}

	beaconBlock = beaconTypes.SignedBeaconBlock{
		Message: &capella.BeaconBlock{
			Slot:          blindedBlock.Slot,
			ProposerIndex: blindedBlock.ProposerIndex,
			ParentRoot:    blindedBlock.ParentRoot,
			StateRoot:     blindedBlock.StateRoot,
			Body: &capella.BeaconBlockBody{
				RANDAOReveal:          blindedBlock.Body.RANDAOReveal,
				ETH1Data:              blindedBlock.Body.ETH1Data,
				Graffiti:              blindedBlock.Body.Graffiti,
				ProposerSlashings:     blindedBlock.Body.ProposerSlashings,
				AttesterSlashings:     blindedBlock.Body.AttesterSlashings,
				Attestations:          blindedBlock.Body.Attestations,
				Deposits:              blindedBlock.Body.Deposits,
				VoluntaryExits:        blindedBlock.Body.VoluntaryExits,
				SyncAggregate:         blindedBlock.Body.SyncAggregate,
				BLSToExecutionChanges: blindedBlock.Body.BLSToExecutionChanges,
				ExecutionPayload:      &executionPayload,
			},
		},
		Signature: signature,
	}

	go b.beacon.PublishBlock(context.Background(), beaconBlock, b.MetricsEnabled, b.db)

	return executionPayload, nil

}
