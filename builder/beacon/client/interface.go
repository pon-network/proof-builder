package beaconClient

import (
	"context"

	commonTypes "github.com/bsn-eng/pon-golang-types/common"
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
	GetForkVersion(slot uint64, head bool) (forkName string, forkVersion string, err error)

	// post methods
	PublishBlock(context.Context, commonTypes.VersionedSignedBeaconBlock) error

	// subscription methods
	SubscribeToHeadEvents(context.Context, chan beaconTypes.HeadEventData)
	SubscribeToPayloadAttributesEvents(context.Context, chan beaconTypes.PayloadAttributesEvent)
}
