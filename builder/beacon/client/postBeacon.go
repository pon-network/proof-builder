package beaconClient

import (
	"context"
	capella "github.com/attestantio/go-eth2-client/spec/capella"
)

func (b *beaconClient) PublishBlock(ctx context.Context, block capella.SignedBeaconBlock) error {
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
