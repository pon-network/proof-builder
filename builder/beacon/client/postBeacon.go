package beaconClient

import (
	"context"
)

func (b *beaconClient) PublishBlock(ctx context.Context, block interface{}) error {
	// Publish a block to the beacon chain
	u := *b.beaconEndpoint
	u.Path = "/eth/v1/beacon/blocks"

	err := b.postBeacon(&u, &block)
	return err
}
