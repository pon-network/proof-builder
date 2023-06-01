package database

import (
	"database/sql"
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

	Slot uint64 `db:"slot"`

	BuilderPubkey        string `db:"builder_pubkey"`
	ProposerPubkey       string `db:"proposer_pubkey"`
	FeeRecipient         string `db:"fee_recipient"`
	BuilderWalletAddress string `db:"builder_wallet_address"`

	GasUsed  uint64 `db:"gas_used"`
	GasLimit uint64 `db:"gas_limit"`

	MEV uint64 `db:"mev"`

	PayoutPoolTx      string `db:"payout_pool_tx"`
	PayoutPoolAddress string `db:"payout_pool_address"`
	PayoutPoolGasFee  uint64 `db:"payout_pool_gas_fee"`
	RPBS              string `db:"rpbs"`

	PriorityTransactionsCount uint64 `db:"priority_transactions_count"`

	TransactionsCount uint64 `db:"transactions_count"`

	BlockHash   string `db:"block_hash"`
	ParentHash  string `db:"parent_hash"`
	BlockNumber uint64 `db:"block_number"`

	Value uint64 `db:"value"`
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
