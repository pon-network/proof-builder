package builder

import (
	"errors"

	"github.com/attestantio/go-eth2-client/spec/bellatrix"
	capella "github.com/attestantio/go-eth2-client/spec/capella"
	"github.com/attestantio/go-eth2-client/spec/phase0"
	utilbellatrix "github.com/attestantio/go-eth2-client/util/bellatrix"
	utilcapella "github.com/attestantio/go-eth2-client/util/capella"
	"github.com/ethereum/go-ethereum/common"
)

var (
	EmptyWithdrawalMerkleRoot = "0x792930bbd5baac43bcc798ee49aa8185ef76bb3b44ba62b91d86ae569e4bb535"
)

func (b *Builder) DataCleanUp() {
	b.slotMu.Lock()
	b.slotSubmissionsLock.Lock()
	defer b.slotSubmissionsLock.Unlock()
	defer b.slotMu.Unlock()

	b.beacon.BeaconData.Mu.Lock()
	currentSlot := b.beacon.BeaconData.CurrentSlot
	b.beacon.BeaconData.Mu.Unlock()

	// Check if execution payload cache has any old slots since we are in a new slot bid
	for slot := range b.executionPayloadCache {
		if int64(slot) < int64(currentSlot)-32 { // Delete any slots that are 32 slots behind the current slot
			delete(b.executionPayloadCache, slot)
		}
	}

	for slot := range b.slotSubmissions {
		if int64(slot) < int64(currentSlot)-32 { // Delete any slots that are 32 slots behind the current slot
			delete(b.slotSubmissions, slot)
		}
	}

	for slot := range b.slotSubmissionsChan {
		if int64(slot) < int64(currentSlot)-32 { // Delete any slots that are 32 slots behind the current slot
			delete(b.slotSubmissionsChan, slot)
			delete(b.slotBidCompleteChan, slot)
			delete(b.slotBountyCompleteChan, slot)
		}
	}

	for slot := range b.slotAttrs {
		if int64(slot) < int64(currentSlot)-32 { // Delete any slots that are 32 slots behind the current slot
			delete(b.slotAttrs, slot)
		}
	}

	for slot := range b.slotBountyAttrs {
		if int64(slot) < int64(currentSlot)-32 { // Delete any slots that are 32 slots behind the current slot
			delete(b.slotBountyAttrs, slot)
		}
	}

	for slot := range b.slotBidAmounts {
		if int64(slot) < int64(currentSlot)-32 { // Delete any slots that are 32 slots behind the current slot
			delete(b.slotBidAmounts, slot)
		}
	}

	for slot := range b.slotBountyAmount {
		if int64(slot) < int64(currentSlot)-32 { // Delete any slots that are 32 slots behind the current slot
			delete(b.slotBountyAmount, slot)
		}
	}

}

func ComputeWithdrawalsRoot(w []*capella.Withdrawal) (phase0.Root, error) {
	if w == nil || len(w) == 0 {
		emptyRoot := phase0.Root{}
		emptyHash := common.HexToHash(EmptyWithdrawalMerkleRoot)
		copy(emptyRoot[:], emptyHash.Bytes()[:])
		return emptyRoot, errors.New("no withdrawals supplied")
	}
	withdrawals := utilcapella.ExecutionPayloadWithdrawals{Withdrawals: w}
	return withdrawals.HashTreeRoot()
}

func ComputeTransactionsRoot(t []bellatrix.Transaction) (phase0.Root, error) {

	transactions := utilbellatrix.ExecutionPayloadTransactions{Transactions: t}
	return transactions.HashTreeRoot()
}
