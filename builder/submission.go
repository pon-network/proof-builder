package builder

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	_ "os"
	"time"

	"github.com/attestantio/go-eth2-client/spec/bellatrix"
	capella "github.com/attestantio/go-eth2-client/spec/capella"
	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/holiman/uint256"

	"github.com/ethereum/go-ethereum/beacon/engine"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/log"

	builderTypes "github.com/bsn-eng/pon-golang-types/builder"
	commonTypes "github.com/bsn-eng/pon-golang-types/common"
	bbTypes "github.com/ethereum/go-ethereum/builder/types"
	"github.com/ethereum/go-ethereum/crypto"

	rpbsTypes "github.com/bsn-eng/pon-golang-types/rpbs"
	"github.com/ethereum/go-ethereum/builder/database"
)

func (b *Builder) slotSubmitter(startTime time.Time, deadline time.Time, slot uint64, blockNumber uint64, proposerPubkey commonTypes.PublicKey, channel chan blockProperties, bidComplete chan struct{ bidAmount *big.Int }, bountyComplete chan struct{ bidAmount *big.Int }, submissionErr *error) {

	var headEventCh = make(chan core.ChainHeadEvent, 64)
	subscription := b.eth.Backend().APIBackend.SubscribeChainHeadEvent(headEventCh)

	endTimer := time.NewTimer(time.Until(deadline))
	bountyTimer := time.NewTimer(time.Until(deadline.Add(-time.Second)))
	biddingTimer := time.NewTimer(time.Until(deadline.Add(-time.Second).Add(-b.submissionEndWindow)))

	var pendingBlock blockProperties
	var pendingBountyBlock blockProperties

	log.Info("Starting slot submitter", "slot", slot, "blockNumber", blockNumber, "timeUntilDeadline", time.Until(deadline))

submitter:
	for {
		select {
		case blockProps := <-channel:

			if blockProps.block == nil || blockProps.blockExecutionPayloadEnvlope == nil || blockProps.blockExecutionPayloadEnvlope.ExecutionPayload == nil {
				log.Error("Received nil block or block execution payload envelope")
				continue submitter
			}

			if !blockProps.bountyBid {
				// Check slot submissions if there are already 2 submissions for this slot
				b.slotSubmissionsLock.Lock()
				if len(b.slotBidAmounts[slot]) == 2 {
					b.slotSubmissionsLock.Unlock()
					continue submitter
				}
				b.slotSubmissionsLock.Unlock()

				log.Info("Received block properties", "slot", slot, "blockHash", blockProps.blockExecutionPayloadEnvlope.ExecutionPayload.BlockHash)
			} else {
				// Block is a bounty block

				// Check if we have already submitted a bounty block for this slot
				b.slotSubmissionsLock.Lock()
				if b.slotBountyAmount[slot] != nil && b.slotBountyAmount[slot].Sign() > 0 {
					b.slotSubmissionsLock.Unlock()
					continue submitter
				}
				b.slotSubmissionsLock.Unlock()

				log.Info("Received bounty block properties", "slot", slot, "blockHash", blockProps.blockExecutionPayloadEnvlope.ExecutionPayload.BlockHash)
			}

			log.Info("Attempting to submit block", "slot", slot, "blockHash", blockProps.blockExecutionPayloadEnvlope.ExecutionPayload.BlockHash, "timeUntilDeadline", time.Until(deadline))

			// Attempt to submit block
			b.slotSubmissionsLock.Lock()
			// check if at least 1 successful submission has been made for this slot
			if len(b.slotSubmissions[slot]) == 1 {

				// If there is already a submission for this slot
				// and the latest bid amount is the same as what was submitted
				// then we can ignore this block
				for _, bid := range b.slotSubmissions[slot] {
					if bid.BlockBid.Message.Value == blockProps.attrs.BidAmount {
						b.slotSubmissionsLock.Unlock()
						log.Debug("Already submitted a block for this slot with the same bid value", "slot", slot)
						continue submitter
					}
				}

				if !time.Now().After(deadline.Add(-time.Second).Add(-b.submissionEndWindow)) && !blockProps.bountyBid {
					b.slotSubmissionsLock.Unlock()
					log.Debug("Submitted just 1 block for this slot, but not in final submission window. Waiting for final submission window to submit this block", "slot", slot, "blockHash", blockProps.blockExecutionPayloadEnvlope.ExecutionPayload.BlockHash)
					pendingBlock = blockProps
					continue submitter
				} else if !time.Now().After(deadline.Add(-time.Second)) && blockProps.bountyBid {
					b.slotSubmissionsLock.Unlock()
					log.Debug("Not in bounty submission window. Waiting for bounty submission window to submit this bounty block", "slot", slot, "blockHash", blockProps.blockExecutionPayloadEnvlope.ExecutionPayload.BlockHash)
					pendingBountyBlock = blockProps
					continue submitter
				}
			}
			b.slotSubmissionsLock.Unlock()

			// Else if no successful submission has been made for this slot, or within a submission window, submit this block immediately
			log.Info("submitBestBlock", "slot", slot, "blockHash", blockProps.blockExecutionPayloadEnvlope.ExecutionPayload.BlockHash, "bountyBid", blockProps.bountyBid)

			timeTillDeadline := time.Until(deadline)
			log.Info("block builder submission time till deadline", "timeTillDeadline", timeTillDeadline)

			finalizedBid, err := b.submitBlockBid(blockProps, proposerPubkey)

			b.slotSubmissionsLock.Lock()
			if err != nil {
				log.Error("could not submit block", "err", err)
				*submissionErr = err
				b.slotSubmissionsLock.Unlock()
				continue submitter
			} else {
				log.Info("Submitted block", "slot", slot, "blockHash", blockProps.blockExecutionPayloadEnvlope.ExecutionPayload.BlockHash)
				finalizedBid.BidRequestTime = startTime
				finalizedBid.BlockBuiltTime = blockProps.builtTime
				b.slotSubmissions[slot] = append(b.slotSubmissions[slot], finalizedBid)
				if blockProps.bountyBid {
					b.slotBountyAmount[slot] = finalizedBid.BlockBid.Message.Value
					bountyComplete <- struct{ bidAmount *big.Int }{finalizedBid.BlockBid.Message.Value}
				} else {
					b.slotBidAmounts[slot] = append(b.slotBidAmounts[slot], finalizedBid.BlockBid.Message.Value)
					bidComplete <- struct{ bidAmount *big.Int }{finalizedBid.BlockBid.Message.Value}
				}
			}
			b.slotSubmissionsLock.Unlock()

		case <-biddingTimer.C:
			// Timer triggered, submit pending block if any

			// Check slot submissions if there are already 2 submissions for this slot
			b.slotSubmissionsLock.Lock()
			if len(b.slotBidAmounts[slot]) == 2 {
				b.slotSubmissionsLock.Unlock()
				continue submitter
			} else if len(b.slotBidAmounts[slot]) == 1 && pendingBlock.block != nil {
				// If there is already a submission for this slot
				// and the latest bid amount is the same as what was submitted
				// then we can ignore this block
				for _, bid := range b.slotSubmissions[slot] {
					if bid.BlockBid.Message.Value == pendingBlock.attrs.BidAmount {
						b.slotSubmissionsLock.Unlock()
						log.Debug("Already submitted a block for this slot with the same bid value", "slot", slot)
						continue submitter
					}
				}
			}
			b.slotSubmissionsLock.Unlock()

			// Check if there is a pending block
			if pendingBlock.block != nil {
				finalizedBid, err := b.submitBlockBid(pendingBlock, proposerPubkey)
				b.slotSubmissionsLock.Lock()
				if err != nil {
					log.Error("could not submit block", "err", err)
					*submissionErr = err
					b.slotSubmissionsLock.Unlock()
					continue submitter
				} else {
					log.Info("Submitted block", "slot", slot, "blockHash", pendingBlock.blockExecutionPayloadEnvlope.ExecutionPayload.BlockHash)
					finalizedBid.BidRequestTime = startTime
					finalizedBid.BlockBuiltTime = pendingBlock.builtTime
					b.slotSubmissions[slot] = append(b.slotSubmissions[slot], finalizedBid)
					b.slotBidAmounts[slot] = append(b.slotBidAmounts[slot], finalizedBid.BlockBid.Message.Value)
					bidComplete <- struct{ bidAmount *big.Int }{finalizedBid.BlockBid.Message.Value}
					pendingBlock = blockProperties{}
				}
				b.slotSubmissionsLock.Unlock()
			}

		case <-bountyTimer.C:
			// Timer triggered, submit pending bounty block if any

			// Check slot submissions if there has already been a successful bounty
			// submission for this slot
			b.slotSubmissionsLock.Lock()
			if b.slotBountyAmount[slot] != nil && b.slotBountyAmount[slot].Sign() > 0 {
				b.slotSubmissionsLock.Unlock()
				continue submitter
			}
			b.slotSubmissionsLock.Unlock()

			// Check if there is a pending bounty block
			if pendingBountyBlock.block != nil {
				finalizedBid, err := b.submitBlockBid(pendingBountyBlock, proposerPubkey)
				b.slotSubmissionsLock.Lock()
				if err != nil {
					log.Error("could not submit block", "err", err)
					*submissionErr = err
					b.slotSubmissionsLock.Unlock()
					continue submitter
				} else {
					log.Info("Submitted block", "slot", slot, "blockHash", pendingBlock.blockExecutionPayloadEnvlope.ExecutionPayload.BlockHash)
					finalizedBid.BidRequestTime = startTime
					finalizedBid.BlockBuiltTime = pendingBlock.builtTime
					b.slotSubmissions[slot] = append(b.slotSubmissions[slot], finalizedBid)
					b.slotBountyAmount[slot] = finalizedBid.BlockBid.Message.Value
					bountyComplete <- struct{ bidAmount *big.Int }{finalizedBid.BlockBid.Message.Value}
					pendingBountyBlock = blockProperties{}
				}
				b.slotSubmissionsLock.Unlock()
			}

		case <-endTimer.C:
			// Timer triggered, stop slot submitter
			log.Info("Slot submitter deadline reached", "slot", slot)
			break submitter

		case event := <-headEventCh:
			// If end timer fails, this is as a fail safe to stop the slot submitter from hanging
			// Check if head is 2 slots ahead now
			if event.Block.NumberU64() >= blockNumber+2 {
				log.Info("Head is 2 slots ahead, stopping slot submitter", "slot", slot)
				break submitter
			}
		}
	}

	subscription.Unsubscribe()

	bountyTimer.Stop()
	biddingTimer.Stop()
	endTimer.Stop()

	// Delete the channels from the mapping, do not close the
	// channels as they would be garbage collected eventually
	// and do not want to have a fatal error while another
	// routine (bounty) is finishing up with the channel results
	b.slotSubmissionsLock.Lock()
	delete(b.slotSubmissionsChan, slot)
	delete(b.slotBountyCompleteChan, slot)
	delete(b.slotBidCompleteChan, slot)
	b.slotSubmissionsLock.Unlock()

	log.Info("Stopped slot submitter", "slot", slot)

}

func (b *Builder) submitBlockBid(
	blockProperties blockProperties,
	proposerPubkey commonTypes.PublicKey,
) (builderTypes.BlockBidResponse, error) {
	executionPayloadEnvlope := blockProperties.blockExecutionPayloadEnvlope
	executableData := executionPayloadEnvlope.ExecutionPayload
	block := blockProperties.block
	payoutPoolTx := blockProperties.payoutPoolTx
	blockFees := executionPayloadEnvlope.BlockValue
	attrs := blockProperties.attrs

	// Verify the last tx is the payout pool tx by checking the bytes are the same
	last_tx := executableData.Transactions[len(executableData.Transactions)-1]
	if !bytes.Equal(last_tx, payoutPoolTx) {
		log.Error("last tx is not the payout pool tx")
		return builderTypes.BlockBidResponse{}, fmt.Errorf("last tx is not the payout pool tx")
	}

	// Transactions are all the transactions in the block except the payout pool tx (last tx)
	transactions := executableData.Transactions[:len(executableData.Transactions)-1]

	transactions_encodedList := make([]string, len(transactions))
	for i, tx := range transactions {
		transactions_encodedList[i] = hexutil.Encode(tx)
	}

	rpbsSig, err := b.rpbs.RpbsSignatureGeneration(
		rpbsTypes.RPBSCommitMessage{
			Slot:                 attrs.Slot,
			Amount:               attrs.BidAmount,
			BuilderWalletAddress: b.builderWalletAddress.String(),
			PayoutTxBytes:        hexutil.Encode(blockProperties.payoutPoolTx),
			TxBytes:              transactions_encodedList,
		},
	)

	if err != nil {
		log.Error("could not sign rpbs commit", "err", err)
		return builderTypes.BlockBidResponse{}, err
	}

	rpbs_json, err := json.Marshal(rpbsSig)
	if err != nil {
		log.Error("could not convert rpbs blinded signature to string", "err", err)
		return builderTypes.BlockBidResponse{}, err
	}

	rpbsPubkey, err := b.rpbs.PublicKey()
	if err != nil {
		log.Error("could not get rpbs pubkey", "err", err)
		return builderTypes.BlockBidResponse{}, err
	}

	executionPayloadHeader := block.Header()
	if err != nil {
		log.Error("could not format execution payload header", "err", err)
		return builderTypes.BlockBidResponse{}, err
	}

	parentHash := phase0.Hash32{}
	copy(parentHash[:], executionPayloadHeader.ParentHash.Bytes()[:])

	feeRecipient := bellatrix.ExecutionAddress{}
	copy(feeRecipient[:], executionPayloadHeader.Coinbase.Bytes()[:])

	blockHash := phase0.Hash32{}
	copy(blockHash[:], block.Hash().Bytes()[:])

	logsBloom := [256]byte{}
	copy(logsBloom[:], executionPayloadHeader.Bloom.Bytes()[:])

	baseFeePerGas := uint256.NewInt(0)
	i, overflow := uint256.FromBig(executionPayloadHeader.BaseFee)
	if !overflow {
		baseFeePerGas = i
	}

	prevRandao := [32]byte{}
	copy(prevRandao[:], executionPayloadHeader.MixDigest.Bytes()[:])

	receiptRoot := [32]byte{}
	copy(receiptRoot[:], executionPayloadHeader.ReceiptHash.Bytes()[:])

	stateRoot := [32]byte{}
	copy(stateRoot[:], executionPayloadHeader.Root.Bytes()[:])

	extraData := [32]byte{}
	copy(extraData[:], executionPayloadHeader.Extra[:])

	// Can safely default to 0 if not in right fork version
	// as wont affect blockhash copied from execution payload header
	blobGasUsed := uint64(0)
	if executionPayloadHeader.BlobGasUsed != nil {
		blobGasUsed = *executionPayloadHeader.BlobGasUsed
	}

	excessBlobGasUsed := uint64(0)
	if executionPayloadHeader.ExcessBlobGas != nil {
		excessBlobGasUsed = *executionPayloadHeader.ExcessBlobGas
	}

	var bellatrixTransactions []bellatrix.Transaction = make([]bellatrix.Transaction, 0)
	for _, tx := range executableData.Transactions {
		var bellatrixTx bellatrix.Transaction = make([]byte, len(tx))
		copy(bellatrixTx[:], tx[:])
		bellatrixTransactions = append(bellatrixTransactions, bellatrixTx)
	}

	var denebWithdrawals []*capella.Withdrawal = make([]*capella.Withdrawal, 0)
	for _, withdrawal := range executableData.Withdrawals {
		denebWithdrawals = append(denebWithdrawals, &capella.Withdrawal{
			Index:          capella.WithdrawalIndex(withdrawal.Index),
			ValidatorIndex: phase0.ValidatorIndex(withdrawal.Validator),
			Address:        bellatrix.ExecutionAddress(withdrawal.Address),
			Amount:         phase0.Gwei(withdrawal.Amount),
		})
	}

	// Assign all the fields to the base execution payload
	baseExecutionPayload := commonTypes.BaseExecutionPayload{
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
		Transactions: bellatrixTransactions,
		Withdrawals: denebWithdrawals,
		BlobGasUsed: 	blobGasUsed,
		ExcessBlobGas: 	excessBlobGasUsed,
	}

	b.beacon.BeaconData.Mu.Lock()
	currentForkVersion := b.beacon.BeaconData.CurrentForkVersion
	b.beacon.BeaconData.Mu.Unlock()

	// Convert the base execution payload into a versioned execution payload
	versionedExecutionPayload, err := commonTypes.ConstructExecutionPayload(
		currentForkVersion,
		baseExecutionPayload,
	)
	if err != nil {
		log.Error("could not convert base execution payload to versioned execution payload", "err", err)
		return builderTypes.BlockBidResponse{}, err
	}

	// Convert the versioned execution payload into a versioned execution payload header
	// This hashes the the transactions and withdrawals in the execution payload
	versionedExecutionPayloadHeader, err := versionedExecutionPayload.ToVersionedExecutionPayloadHeader()
	if err != nil {
		log.Error("could not convert execution payload to versioned execution payload header", "err", err)
		return builderTypes.BlockBidResponse{}, err
	}

	blockBidMsg := builderTypes.BidPayload{
		Slot:                 attrs.Slot,
		ParentHash:           commonTypes.Hash(executionPayloadHeader.ParentHash),
		BlockHash:            commonTypes.Hash(block.Hash()),
		BuilderPubkey:        b.builderPublicKey,
		ProposerPubkey:       proposerPubkey,
		ProposerFeeRecipient: commonTypes.Address(attrs.SuggestedFeeRecipient),
		GasLimit:             executableData.GasLimit,
		GasUsed:              executableData.GasUsed,
		Value:                attrs.BidAmount,

		BuilderWalletAddress:   commonTypes.Address(b.builderWalletAddress),
		PayoutPoolTransaction:  payoutPoolTx,
		ExecutionPayloadHeader: &versionedExecutionPayloadHeader,
		Endpoint:               b.AccessPoint + _PathSubmitBlindedBlock,
		RPBS:                   rpbsSig,
		RPBSPubkey:             rpbsPubkey,
	}

	signature, err := bbTypes.SignMessage(&blockBidMsg, b.builderSigningDomain, b.builderSecretKey)
	if err != nil {
		log.Error("could not sign builder bid", "err", err)
		return builderTypes.BlockBidResponse{}, err
	}

	blockBidMsgBytes, err := blockBidMsg.HashTreeRoot()
	if err != nil {
		log.Error("could not marshal block bid msg", "err", err)
		return builderTypes.BlockBidResponse{}, err
	}

	// ECDSA signature over the block bid message
	ecdsa_signature, err := crypto.Sign(blockBidMsgBytes[:], b.builderWalletPrivateKey)
	if err != nil {
		log.Error("Could not ECDSA sign block bid message sig", "err", err)
		return builderTypes.BlockBidResponse{}, err
	}

	blockSubmitReq := builderTypes.BuilderBlockBid{
		Signature:      signature,
		Message:        &blockBidMsg,
		EcdsaSignature: commonTypes.EcdsaSignature(ecdsa_signature),
	}

	var RetryCount int = 3
	var submissionResp interface{}
	var submissionErr error

	for i := 0; i < RetryCount; i++ {
		submissionResp, submissionErr = b.relay.SubmitBlockBid(&blockSubmitReq, blockProperties.bountyBid)
		if submissionErr == nil {
			break
		}
	}

	if submissionErr != nil {
		if b.BundlesEnabled {
			// Set all bundles in attrs.Bundles to adding=false in case any bundles were ignored
			go b.bundles.SetBundlesAddingFalse(attrs.Bundles)
		}
		log.Error("could not submit block", "relay_submission_err", submissionErr)
		return builderTypes.BlockBidResponse{}, fmt.Errorf("could not submit block bid to relay. relay submission error response: %v", submissionErr)
	}

	blockBidSubmitted := builderTypes.BlockBidResponse{
		RelayResponse:      submissionResp,
		BlockBid:           blockSubmitReq,
		BlockSubmittedTime: time.Now(),
	}

	if b.BundlesEnabled {

		// Not all bundles in attrs.Bundles will be added to the block bid sent, rather the tried bundles
		go b.bundles.ProcessSentBundles(blockProperties.triedBundles, blockSubmitReq.Message.BlockHash.String(), blockSubmitReq.Message.Slot)

		// Set all bundles in attrs.Bundles to adding=false in case any bundles were ignored
		go b.bundles.SetBundlesAddingFalse(attrs.Bundles)

		attrs.Bundles = blockProperties.triedBundles
	}

	submissionResp_string, err := json.Marshal(submissionResp)
	if err != nil {
		log.Error("could not convert response from relay to string", "err", err)
		return builderTypes.BlockBidResponse{}, err
	}

	log.Info("Block bid response received from relay", "response", string(submissionResp_string))

	log.Info("Submitted block", "slot", blockBidMsg.Slot, "value", blockBidMsg.Value, "parent", blockBidMsg.ParentHash)
	b.slotMu.Lock()
	_, ok := b.executionPayloadCache[attrs.Slot]
	if !ok {
		b.executionPayloadCache[attrs.Slot] = make(map[string]engine.ExecutableData)
	}
	b.executionPayloadCache[attrs.Slot][executableData.BlockHash.String()] = *executableData
	b.slotMu.Unlock()

	log.Info("Executable data cached", "slot", attrs.Slot, "hash", executableData.BlockHash, "parentHash", executableData.ParentHash, "gasUsed", executableData.GasUsed, "transaction count", len(executableData.Transactions), "withdrawals count", len(executableData.Withdrawals))

	if b.MetricsEnabled {
		blockBidEntry := database.BuilderBlockBidEntry{
			InsertedAt:                time.Now(),
			Signature:                 signature.String(),
			Slot:                      *big.NewInt(int64(attrs.Slot)),
			BuilderPubkey:             b.builderPublicKey.String(),
			ProposerPubkey:            proposerPubkey.String(),
			FeeRecipient:              attrs.SuggestedFeeRecipient.String(),
			GasLimit:                  *big.NewInt(int64(executableData.GasLimit)),
			GasUsed:                   *big.NewInt(int64(executableData.GasUsed)),
			MEV:                       *blockFees,
			PayoutPoolTx:              hexutil.Encode(payoutPoolTx),
			PayoutPoolAddress:         b.relay.GetPayoutAddress().String(),
			PayoutPoolGasFee:          *big.NewInt(int64(b.eth.GetPayoutPoolTxGas())),
			RPBS:                      string(rpbs_json),
			PriorityTransactionsCount: uint64(len(attrs.Transactions)),
			TransactionsCount:         uint64(len(executableData.Transactions)),
			BlockHash:                 block.Hash().String(),
			ParentHash:                executionPayloadHeader.ParentHash.String(),
			BlockNumber:               *executionPayloadHeader.Number,
			RelayResponse:             string(submissionResp_string),
			Value:                     *attrs.BidAmount,
		}
		go b.db.InsertBlockBid(blockBidEntry)
	}

	return blockBidSubmitted, nil
}

func (b *Builder) SubmitBlindedBlock(signedblindedBlock commonTypes.VersionedSignedBlindedBeaconBlock) (commonTypes.VersionedExecutionPayload, error) {

	b.beacon.BeaconData.Mu.Lock()
	currentForkVersion := b.beacon.BeaconData.CurrentForkVersion
	b.beacon.BeaconData.Mu.Unlock()

	// Check forkVersion of the signed blinded beacon block is the same as the current fork version
	// If not, the builder will not be able to publish the block to the beacon chain or know which version
	// of the execution payload to use
	signedBlindedBlock_version, err := signedblindedBlock.Version()
	if err != nil {
		log.Error("could not get version of signed blinded beacon block", "err", err)
		return commonTypes.VersionedExecutionPayload{}, err
	}

	if signedBlindedBlock_version != currentForkVersion {
		log.Error("signed blinded beacon block version does not match current fork version", "signedBlindedBlock_version", signedBlindedBlock_version, "currentForkVersion", currentForkVersion)
		return commonTypes.VersionedExecutionPayload{}, errors.New("signed blinded beacon block version does not match current fork version")
	}

	// Unpack the versioned signed blinded beacon block from its available version into a base signed blinded beacon block for access
	baseSignedBlindedBeaconBlock, err := signedblindedBlock.ToBaseSignedBlindedBeaconBlock()
	if err != nil {
		log.Error("could not convert signed blinded beacon block to base signed blinded beacon block", "err", err)
		return commonTypes.VersionedExecutionPayload{}, err
	}

	blindedBlock := baseSignedBlindedBeaconBlock.Message
	signature := baseSignedBlindedBeaconBlock.Signature

	slot := uint64(blindedBlock.Slot)

	log.Info("Blinded block submission request", "slot", blindedBlock.Slot, "payloadSignature", signature.String())

	var executableData engine.ExecutableData

	b.slotMu.Lock()
	executableDataSlot, exists := b.executionPayloadCache[slot]
	if exists {
		executableData, exists = executableDataSlot[blindedBlock.Body.ExecutionPayloadHeader.BlockHash.String()]
	}
	b.slotMu.Unlock()

	if !exists {
		return commonTypes.VersionedExecutionPayload{}, errors.New("execution payload not found")
	}

	log.Info("executionPayload found", "slot", slot, "hash", executableData.BlockHash.String())

	if b.MetricsEnabled {
		// The signed blinded beacon block has been received and is being processed
		blindedBeaconBlockEntry := database.SignedBlindedBeaconBlockEntry{
			InsertedAt:               time.Now(),
			Signature:                signature.String(),
			SignedBlindedBeaconBlock: signedblindedBlock.String(),
		}
		go b.db.InsertBlindedBeaconBlock(blindedBeaconBlockEntry, blindedBlock.Body.ExecutionPayloadHeader.BlockHash.String())
	}

	// If execution payload is found and block building for second submission is still in progress, cancel it
	if b.slotCtxCancel != nil {
		b.slotCtxCancel()
	}

	var withdrawals []*capella.Withdrawal = make([]*capella.Withdrawal, 0)
	for _, withdrawal := range executableData.Withdrawals {
		withdrawals = append(withdrawals, &capella.Withdrawal{
			Index:          capella.WithdrawalIndex(withdrawal.Index),
			ValidatorIndex: phase0.ValidatorIndex(withdrawal.Validator),
			Amount:         phase0.Gwei(withdrawal.Amount),
			Address:        bellatrix.ExecutionAddress(withdrawal.Address),
		})
	}

	var transactions []bellatrix.Transaction = make([]bellatrix.Transaction, 0)
	for _, tx := range executableData.Transactions {
		var bellatrixTx bellatrix.Transaction = make([]byte, len(tx))
		copy(bellatrixTx[:], tx[:])
		transactions = append(transactions, bellatrixTx)
	}

	// Convert the versioned signed blinded beacon block into a versioned signed beacon block
	// This would have the transactions and withdrawals feilds as nil as those cannot be derived from the blinded block
	versionedSignedBeaconBlock, err := signedblindedBlock.ToVersionedSignedBeaconBlock()
	if err != nil {
		log.Error("could not convert signed blinded beacon block to versioned signed beacon block", "err", err)
		return commonTypes.VersionedExecutionPayload{}, err
	}

	// Unpack the obtained versioned signed beacon block into a base signed beacon block for access
	baseSignedBeaconBlock, err := versionedSignedBeaconBlock.ToBaseSignedBeaconBlock()
	if err != nil {
		log.Error("could not convert versioned signed beacon block to base signed beacon block", "err", err)
		return commonTypes.VersionedExecutionPayload{}, err
	}

	// Update the transactions and withdrawals fields of the base signed beacon block
	baseSignedBeaconBlock.Message.Body.ExecutionPayload.Transactions = transactions
	baseSignedBeaconBlock.Message.Body.ExecutionPayload.Withdrawals = withdrawals

	// Reconstruct the versioned signed beacon block from the updated base signed beacon block
	versionedSignedBeaconBlock, err = commonTypes.ConstructSignedBeaconBlock(
		currentForkVersion,
		baseSignedBeaconBlock,
	)
	if err != nil {
		log.Error("could not convert base signed beacon block to versioned signed beacon block", "err", err)
		return commonTypes.VersionedExecutionPayload{}, err
	}

	// Create a versioned execution payload from the execution payload in the block
	versionedExecutionPayload, err := commonTypes.ConstructExecutionPayload(
		currentForkVersion,
		*baseSignedBeaconBlock.Message.Body.ExecutionPayload,
	)
	if err != nil {
		log.Error("could not convert base execution payload to versioned execution payload", "err", err)
		return commonTypes.VersionedExecutionPayload{}, err
	}

	go b.beacon.PublishBlock(context.Background(), versionedSignedBeaconBlock, b.MetricsEnabled, b.db, b.BundlesEnabled, b.bundles.ProcessBlockAdded)
	log.Info("Publishing block", "slot", slot, "block_hash", baseSignedBeaconBlock.Message.Body.ExecutionPayload.BlockHash.String())

	// Publishing function, internally would update the builder metrics with the signed beacon block being published (if metrics enabled)
	// Publishing function, internally would update the bundle service on any successfully added bundles (if bundles enabled)

	return versionedExecutionPayload, nil

}
