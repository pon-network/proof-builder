package beaconClient

import (
	"context"

	capella "github.com/attestantio/go-eth2-client/spec/capella"

	beaconTypes "github.com/bsn-eng/pon-golang-types/beaconclient"
	beaconData "github.com/ethereum/go-ethereum/builder/beacon/data"
	"github.com/ethereum/go-ethereum/common"
)

type BeaconClientInstance interface {
	BaseEndpoint() string

	// get methods
	GetSlotProposerMap(uint64) (beaconData.SlotProposerMap, error)
	SyncStatus() (*beaconTypes.SyncStatusData, error)
	Genesis() (*beaconTypes.GenesisData, error)
	GetWithdrawals(uint64) (*beaconTypes.Withdrawals, error)
	Randao(uint64) (*common.Hash, error)
	GetBlockHeader(slot uint64) (*beaconTypes.BlockHeaderData, error)

	// post methods
	PublishBlock(context.Context, capella.SignedBeaconBlock) error

	// subscription methods
	SubscribeToHeadEvents(context.Context, chan beaconTypes.HeadEventData)
	SubscribeToPayloadAttributesEvents(context.Context, chan beaconTypes.PayloadAttributesEventData)
}
