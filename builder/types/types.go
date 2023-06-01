package builderTypes

import (
	"fmt"

	RPBS "github.com/ethereum/go-ethereum/builder/rpbsService"
	capella "github.com/attestantio/go-eth2-client/spec/capella"
	gethCommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
)

var ErrLength = fmt.Errorf("incorrect byte length")

type Signature [96]byte

func (s Signature) MarshalText() ([]byte, error) {
	return hexutil.Bytes(s[:]).MarshalText()
}

func (s *Signature) UnmarshalJSON(input []byte) error {
	b := hexutil.Bytes(s[:])
	err := b.UnmarshalJSON(input)
	if err != nil {
		return err
	}
	return s.FromSlice(b)
}

func (s *Signature) UnmarshalText(input []byte) error {
	b := hexutil.Bytes(s[:])
	err := b.UnmarshalText(input)
	if err != nil {
		return err
	}
	return s.FromSlice(b)
}

func (s Signature) String() string {
	return hexutil.Bytes(s[:]).String()
}

func (s *Signature) FromSlice(x []byte) error {
	if len(x) != 96 {
		return ErrLength
	}
	copy(s[:], x)
	return nil
}

type EcdsaSignature [65]byte

func (s EcdsaSignature) MarshalText() ([]byte, error) {
	return hexutil.Bytes(s[:]).MarshalText()
}

func (s *EcdsaSignature) UnmarshalJSON(input []byte) error {
	b := hexutil.Bytes(s[:])
	err := b.UnmarshalJSON(input)
	if err != nil {
		return err
	}
	return s.FromSlice(b)
}

func (s *EcdsaSignature) UnmarshalText(input []byte) error {
	b := hexutil.Bytes(s[:])
	err := b.UnmarshalText(input)
	if err != nil {
		return err
	}
	return s.FromSlice(b)
}

func (s EcdsaSignature) String() string {
	return hexutil.Bytes(s[:]).String()
}

func (s *EcdsaSignature) FromSlice(x []byte) error {
	if len(x) != 65 {
		return ErrLength
	}
	copy(s[:], x)
	return nil
}

type BuilderBlockBid struct {
	Signature              Signature               `json:"signature" ssz-size:"96"`
	Message                *BidPayload               `json:"message"`
	EcdsaSignature		   EcdsaSignature          `json:"ecdsa_signature"`
}

type BuilderPayloadAttributes struct {
	Timestamp             hexutil.Uint64     `json:"timestamp"`
	Random                gethCommon.Hash    `json:"prevRandao"`
	SuggestedFeeRecipient gethCommon.Address            `json:"suggestedFeeRecipient"`
	Slot                  uint64             `json:"slot,string"`
	HeadHash              gethCommon.Hash    `json:"headHash"`
	BidAmount             uint64 			 `json:"bidAmount,string"`
	GasLimit 			uint64             `json:"gasLimit,string"`
	Transactions          [][]byte 		     `json:"transactions"`
	Withdrawals types.Withdrawals `json:"withdrawals"`
	NoMempoolTxs		  bool               `json:"noMempoolTxs,string"`
	PayoutPoolAddress    gethCommon.Address            `json:"payoutPoolAddress"`
}

type PrivateTransactionsPayload struct {
	Transactions          [][]byte 		     `json:"transactions"`
}

type (
	Hash [32]byte
	Root = Hash
)

func (h Hash) MarshalText() ([]byte, error) {
	return hexutil.Bytes(h[:]).MarshalText()
}

func (h *Hash) UnmarshalJSON(input []byte) error {
	b := hexutil.Bytes(h[:])
	if err := b.UnmarshalJSON(input); err != nil {
		return err
	}
	return h.FromSlice(b)
}

func (h *Hash) UnmarshalText(input []byte) error {
	b := hexutil.Bytes(h[:])
	if err := b.UnmarshalText(input); err != nil {
		return err
	}
	return h.FromSlice(b)
}

func (h *Hash) FromSlice(x []byte) error {
	if len(x) != 32 {
		return ErrLength
	}
	copy(h[:], x)
	return nil
}

func (h Hash) String() string {
	return hexutil.Bytes(h[:]).String()
}

type Address [20]byte

func (a Address) MarshalText() ([]byte, error) {
	return hexutil.Bytes(a[:]).MarshalText()
}

func (a *Address) UnmarshalJSON(input []byte) error {
	b := hexutil.Bytes(a[:])
	if err := b.UnmarshalJSON(input); err != nil {
		return err
	}
	return a.FromSlice(b)
}

func (a *Address) UnmarshalText(input []byte) error {
	b := hexutil.Bytes(a[:])
	if err := b.UnmarshalText(input); err != nil {
		return err
	}
	return a.FromSlice(b)
}

func (a Address) String() string {
	return hexutil.Bytes(a[:]).String()
}

func (a *Address) FromSlice(x []byte) error {
	if len(x) != 20 {
		return ErrLength
	}
	copy(a[:], x)
	return nil
}



type PublicKey [48]byte

func (p PublicKey) MarshalText() ([]byte, error) {
	return hexutil.Bytes(p[:]).MarshalText()
}

func (p *PublicKey) UnmarshalJSON(input []byte) error {
	b := hexutil.Bytes(p[:])
	if err := b.UnmarshalJSON(input); err != nil {
		return err
	}
	return p.FromSlice(b)
}

func (p *PublicKey) UnmarshalText(input []byte) error {
	b := hexutil.Bytes(p[:])
	if err := b.UnmarshalText(input); err != nil {
		return err
	}
	return p.FromSlice(b)
}

func (p PublicKey) String() string {
	return hexutil.Bytes(p[:]).String()
}

func (p *PublicKey) FromSlice(x []byte) error {
	if len(x) != 48 {
		return ErrLength
	}
	copy(p[:], x)
	return nil
}

func HexToPubkey(s string) (ret PublicKey, err error) {
	err = ret.UnmarshalText([]byte(s))
	return ret, err
}

type BidPayload struct {
	Slot                 uint64    `json:"slot,string"`
	ParentHash           Hash      `json:"parent_hash" ssz-size:"32"`
	BlockHash            Hash      `json:"block_hash" ssz-size:"32"`
	BuilderPubkey        PublicKey `json:"builder_pubkey" ssz-size:"48"`
	ProposerPubkey       PublicKey `json:"proposer_pubkey" ssz-size:"48"`
	ProposerFeeRecipient Address   `json:"proposer_fee_recipient" ssz-size:"20"`
	GasLimit             uint64    `json:"gas_limit,string"`
	GasUsed              uint64    `json:"gas_used,string"`
	Value uint64 `json:"value"`
	
	ExecutionPayloadHeader *capella.ExecutionPayloadHeader 		   `json:"execution_payload_header"`
	Endpoint               string                  `json:"endpoint"`
	BuilderWalletAddress   Address                 `json:"builder_wallet_address"`
	PayoutPoolTransaction  []byte                  `json:"payout_pool_transaction"`
	RPBS 				   *RPBS.RpbsSignature 	   `json:"rpbs",omitempty`
	RPBSPubkey 			   string 			   `json:"rpbs_pubkey",omitempty`

}

