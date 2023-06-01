package builderTypes

import (
	ssz "github.com/ferranbt/fastssz"
)

// MarshalSSZ ssz marshals the BidPayload object
func (b *BidPayload) MarshalSSZ() ([]byte, error) {
	return ssz.MarshalSSZ(b)
}

// MarshalSSZTo ssz marshals the BidPayload object to a target array
func (b *BidPayload) MarshalSSZTo(buf []byte) (dst []byte, err error) {
	dst = buf

	// Field (0) 'Slot'
	dst = ssz.MarshalUint64(dst, b.Slot)

	// Field (1) 'ParentHash'
	dst = append(dst, b.ParentHash[:]...)

	// Field (2) 'BlockHash'
	dst = append(dst, b.BlockHash[:]...)

	// Field (3) 'BuilderPubkey'
	dst = append(dst, b.BuilderPubkey[:]...)

	// Field (4) 'ProposerPubkey'
	dst = append(dst, b.ProposerPubkey[:]...)

	// Field (5) 'ProposerFeeRecipient'
	dst = append(dst, b.ProposerFeeRecipient[:]...)

	// Field (6) 'GasLimit'
	dst = ssz.MarshalUint64(dst, b.GasLimit)

	// Field (7) 'GasUsed'
	dst = ssz.MarshalUint64(dst, b.GasUsed)

	// Field (8) 'Value'
	dst = ssz.MarshalUint64(dst, b.Value)

	return
}

// SizeSSZ returns the ssz encoded size in bytes for the BidPayload object
func (b *BidPayload) SizeSSZ() (size int) {
	size = 236
	return
}

// HashTreeRoot ssz hashes the BidPayload object
func (b *BidPayload) HashTreeRoot() ([32]byte, error) {
	return ssz.HashWithDefaultHasher(b)
}

// HashTreeRootWith ssz hashes the BidPayload object with a hasher
func (b *BidPayload) HashTreeRootWith(hh ssz.HashWalker) (err error) {
	indx := hh.Index()

	// Field (0) 'Slot'
	hh.PutUint64(b.Slot)

	// Field (1) 'ParentHash'
	hh.PutBytes(b.ParentHash[:])

	// Field (2) 'BlockHash'
	hh.PutBytes(b.BlockHash[:])

	// Field (3) 'BuilderPubkey'
	hh.PutBytes(b.BuilderPubkey[:])

	// Field (4) 'ProposerPubkey'
	hh.PutBytes(b.ProposerPubkey[:])

	// Field (5) 'ProposerFeeRecipient'
	hh.PutBytes(b.ProposerFeeRecipient[:])

	// Field (6) 'GasLimit'
	hh.PutUint64(b.GasLimit)

	// Field (7) 'GasUsed'
	hh.PutUint64(b.GasUsed)

	// Field (8) 'Value'
	hh.PutUint64(b.Value)

	hh.Merkleize(indx)
	return
}

// GetTree ssz hashes the BidPayload object
func (b *BidPayload) GetTree() (*ssz.Node, error) {
	return ssz.ProofTree(b)
}
