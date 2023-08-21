package database

import (
	"database/sql"
	"errors"
	"math/big"
	"time"
)

func NewNullInt64(i int64) sql.NullInt64 {
	return sql.NullInt64{
		Int64: i,
		Valid: true,
	}
}

func NewNullString(s string) sql.NullString {
	return sql.NullString{
		String: s,
		Valid:  true,
	}
}

func NewNullTime(t time.Time) sql.NullTime {
	return sql.NullTime{
		Time:  t,
		Valid: true,
	}
}

type BuilderBlockBidEntry struct {
	ID         uint64    `db:"id"`
	InsertedAt time.Time `db:"inserted_at"`

	// BidPayload data
	Signature string `db:"signature"`

	Slot big.Int `db:"slot"`

	BuilderPubkey        string `db:"builder_pubkey"`
	ProposerPubkey       string `db:"proposer_pubkey"`
	FeeRecipient         string `db:"fee_recipient"`
	BuilderWalletAddress string `db:"builder_wallet_address"`

	GasUsed  big.Int `db:"gas_used"`
	GasLimit big.Int `db:"gas_limit"`

	MEV big.Int `db:"mev"`

	PayoutPoolTx      string  `db:"payout_pool_tx"`
	PayoutPoolAddress string  `db:"payout_pool_address"`
	PayoutPoolGasFee  big.Int `db:"payout_pool_gas_fee"`
	RPBS              string  `db:"rpbs"`

	PriorityTransactionsCount uint64 `db:"priority_transactions_count"`

	TransactionsCount uint64 `db:"transactions_count"`

	BlockHash   string  `db:"block_hash"`
	ParentHash  string  `db:"parent_hash"`
	BlockNumber big.Int `db:"block_number"`

	RelayResponse string `db:"relay_response"`

	Value big.Int `db:"value"`
}

func (b *BuilderBlockBidEntry) ToBuilderBlockBidEntryLoarder() BuilderBlockBidEntryLoader {

	return BuilderBlockBidEntryLoader{
		ID:                        b.ID,
		InsertedAt:                b.InsertedAt,
		Slot:                      b.Slot.String(),
		Signature:                 b.Signature,
		BuilderPubkey:             b.BuilderPubkey,
		ProposerPubkey:            b.ProposerPubkey,
		FeeRecipient:              b.FeeRecipient,
		BuilderWalletAddress:      b.BuilderWalletAddress,
		GasUsed:                   b.GasUsed.String(),
		GasLimit:                  b.GasLimit.String(),
		MEV:                       b.MEV.String(),
		PayoutPoolTx:              b.PayoutPoolTx,
		PayoutPoolAddress:         b.PayoutPoolAddress,
		PayoutPoolGasFee:          b.PayoutPoolGasFee.String(),
		RPBS:                      b.RPBS,
		PriorityTransactionsCount: b.PriorityTransactionsCount,
		TransactionsCount:         b.TransactionsCount,
		BlockHash:                 b.BlockHash,
		ParentHash:                b.ParentHash,
		BlockNumber:               b.BlockNumber.String(),
		RelayResponse:             b.RelayResponse,
		Value:                     b.Value.String(),
	}

}

type SignedBlindedBeaconBlockEntry struct {
	ID         uint64    `db:"id"`
	InsertedAt time.Time `db:"inserted_at"`

	BidId uint64 `db:"bid_id"`

	SignedBlindedBeaconBlock string `db:"signed_blinded_beacon_block"`

	Signature string `db:"signature"`
}

type SignedBeaconBlockSubmissionEntry struct {
	ID         uint64    `db:"id"`
	InsertedAt time.Time `db:"inserted_at"`

	BidId uint64 `db:"bid_id"`

	SignedBeaconBlock string `db:"signed_beacon_block"`

	Signature string `db:"signature"`

	SubmittedToChain bool           `db:"submitted_to_chain"`
	SubmissionError  sql.NullString `db:"submission_error"`
}

type BuilderBlockBidEntryLoader struct {
	ID         uint64    `db:"id"`
	InsertedAt time.Time `db:"inserted_at"`
	Slot       string    `db:"slot"` // Load as string

	// Other fields...
	Signature                 string `db:"signature"`
	BuilderPubkey             string `db:"builder_pubkey"`
	ProposerPubkey            string `db:"proposer_pubkey"`
	FeeRecipient              string `db:"fee_recipient"`
	BuilderWalletAddress      string `db:"builder_wallet_address"`
	GasUsed                   string `db:"gas_used"`  // Load as string
	GasLimit                  string `db:"gas_limit"` // Load as string
	MEV                       string `db:"mev"`       // Load as string
	PayoutPoolTx              string `db:"payout_pool_tx"`
	PayoutPoolAddress         string `db:"payout_pool_address"`
	PayoutPoolGasFee          string `db:"payout_pool_gas_fee"` // Load as string
	RPBS                      string `db:"rpbs"`
	PriorityTransactionsCount uint64 `db:"priority_transactions_count"`
	TransactionsCount         uint64 `db:"transactions_count"`
	BlockHash                 string `db:"block_hash"`
	ParentHash                string `db:"parent_hash"`
	BlockNumber               string `db:"block_number"` // Load as string
	RelayResponse             string `db:"relay_response"`
	Value                     string `db:"value"` // Load as string
}

func (b *BuilderBlockBidEntryLoader) ToBuilderBlockBidEntry() (BuilderBlockBidEntry, error) {

	bidEntry := new(BuilderBlockBidEntry)

	slotBigInt := new(big.Int)
	_, success := slotBigInt.SetString(b.Slot, 10)
	if !success {
		return *bidEntry, errors.New("Failed to convert slot to big.Int")
	}

	gasUsedBigInt := new(big.Int)
	_, success = gasUsedBigInt.SetString(b.GasUsed, 10)
	if !success {
		return *bidEntry, errors.New("Failed to convert gasUsed to big.Int")
	}

	gasLimitBigInt := new(big.Int)
	_, success = gasLimitBigInt.SetString(b.GasLimit, 10)
	if !success {
		return *bidEntry, errors.New("Failed to convert gasLimit to big.Int")
	}

	mevBigInt := new(big.Int)
	_, success = mevBigInt.SetString(b.MEV, 10)
	if !success {
		return *bidEntry, errors.New("Failed to convert mev to big.Int")
	}

	payoutPoolGasFeeBigInt := new(big.Int)
	_, success = payoutPoolGasFeeBigInt.SetString(b.PayoutPoolGasFee, 10)
	if !success {
		return *bidEntry, errors.New("Failed to convert payoutPoolGasFee to big.Int")
	}

	blockNumberBigInt := new(big.Int)
	_, success = blockNumberBigInt.SetString(b.BlockNumber, 10)
	if !success {
		return *bidEntry, errors.New("Failed to convert blockNumber to big.Int")
	}

	valueBigInt := new(big.Int)
	_, success = valueBigInt.SetString(b.Value, 10)
	if !success {
		return *bidEntry, errors.New("Failed to convert value to big.Int")
	}

	bidEntry.ID = b.ID
	bidEntry.InsertedAt = b.InsertedAt
	bidEntry.Signature = b.Signature
	bidEntry.Slot = *slotBigInt
	bidEntry.BuilderPubkey = b.BuilderPubkey
	bidEntry.ProposerPubkey = b.ProposerPubkey
	bidEntry.FeeRecipient = b.FeeRecipient
	bidEntry.BuilderWalletAddress = b.BuilderWalletAddress
	bidEntry.GasUsed = *gasUsedBigInt
	bidEntry.GasLimit = *gasLimitBigInt
	bidEntry.MEV = *mevBigInt
	bidEntry.PayoutPoolTx = b.PayoutPoolTx
	bidEntry.PayoutPoolAddress = b.PayoutPoolAddress
	bidEntry.PayoutPoolGasFee = *payoutPoolGasFeeBigInt
	bidEntry.RPBS = b.RPBS
	bidEntry.PriorityTransactionsCount = b.PriorityTransactionsCount
	bidEntry.TransactionsCount = b.TransactionsCount
	bidEntry.BlockHash = b.BlockHash
	bidEntry.ParentHash = b.ParentHash
	bidEntry.BlockNumber = *blockNumberBigInt
	bidEntry.RelayResponse = b.RelayResponse
	bidEntry.Value = *valueBigInt

	return *bidEntry, nil

}
