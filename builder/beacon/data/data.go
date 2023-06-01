package beaconData

import (
	"sync"

	beaconTypes "github.com/bsn-eng/pon-golang-types/beaconclient"
	"github.com/ethereum/go-ethereum/common"
)

type BeaconData struct {
	CurrentSlot  uint64
	CurrentEpoch uint64

	CurrentHead beaconTypes.HeadEventData

	Mu                       sync.Mutex
	RandaoMap                RandaoMap
	SlotProposerMap          SlotProposerMap
	SlotPayloadAttributesMap SlotPayloadAttributesMap

	CloseCh            chan struct{}
	HeadSlotC          chan beaconTypes.HeadEventData
	PayloadAttributesC chan beaconTypes.PayloadAttributesEventData
}

type RandaoMap map[uint64]common.Hash
type SlotProposerMap map[uint64]beaconTypes.ProposerDutyData
type SlotPayloadAttributesMap map[uint64]beaconTypes.PayloadAttributesEventData
