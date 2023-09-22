package beaconClient

import (
	"context"

	commonTypes "github.com/bsn-eng/pon-golang-types/common"
)

func (b *beaconClient) PublishBlock(ctx context.Context, block commonTypes.VersionedSignedBeaconBlock) error {
	// Publish a block to the beacon chain
	u := *b.beaconEndpoint
	u.Path = "/eth/v1/beacon/blocks"

	block_json, err := block.MarshalJSON()
	if err != nil {
		return err
	}

	err = b.postBeacon(&u, block_json)
	return err
}
