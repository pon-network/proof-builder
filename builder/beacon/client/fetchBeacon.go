package beaconClient

import (
	"fmt"

	beaconData "github.com/ethereum/go-ethereum/builder/beacon/data"

	beaconTypes "github.com/bsn-eng/pon-golang-types/beaconclient"
	"github.com/ethereum/go-ethereum/common"
)

func (b *beaconClient) GetSlotProposerMap(epoch uint64) (beaconData.SlotProposerMap, error) {
	// Get proposer duties for given epoch
	u := *b.beaconEndpoint
	u.Path = fmt.Sprintf("/eth/v1/validator/duties/proposer/%d", epoch)
	resp := new(beaconTypes.GetProposerDutiesResponse)

	err := b.fetchBeacon(&u, resp)
	if err != nil {
		return nil, err
	}

	proposerDuties := make(beaconData.SlotProposerMap)
	for _, duty := range resp.Data {
		proposerDuties[duty.Slot] = *duty
	}
	return proposerDuties, nil

}

func (b *beaconClient) SyncStatus() (*beaconTypes.SyncStatusData, error) {
	// Get sync status
	u := *b.beaconEndpoint
	u.Path = "/eth/v1/node/syncing"
	resp := new(beaconTypes.GetSyncStatusResponse)

	err := b.fetchBeacon(&u, resp)
	if err != nil {
		return nil, err
	}
	return resp.Data, nil
}

func (b *beaconClient) Genesis() (*beaconTypes.GenesisData, error) {
	// Get genesis data
	resp := new(beaconTypes.GetGenesisResponse)
	u := *b.beaconEndpoint

	u.Path = "/eth/v1/beacon/genesis"
	err := b.fetchBeacon(&u, &resp)
	if err != nil {
		return nil, err
	}
	return resp.Data, err
}

func (b *beaconClient) GetWithdrawals(slot uint64) (*beaconTypes.Withdrawals, error) {
	// Get withdrawals for given slot
	resp := new(beaconTypes.GetWithdrawalsResponse)
	u := *b.beaconEndpoint

	u.Path = fmt.Sprintf("/eth/v1/builder/states/%d/expected_withdrawals", slot)
	err := b.fetchBeacon(&u, &resp)
	if err != nil {
		return nil, err
	}
	return resp.Data, err
}

func (b *beaconClient) Randao(slot uint64) (*common.Hash, error) {
	// Get randao for given slot
	resp := new(beaconTypes.GetRandaoResponse)
	u := *b.beaconEndpoint
	u.Path = fmt.Sprintf("/eth/v1/beacon/states/%d/randao", slot)

	err := b.fetchBeacon(&u, &resp)
	if err != nil {
		return nil, err
	}

	data := resp.Data
	return &data.Randao, err
}

func (b *beaconClient) GetBlockHeader(slot uint64) (*beaconTypes.BlockHeaderData, error) {
	// Get block header for given slot
	resp := new(beaconTypes.GetBlockHeaderResponse)
	u := *b.beaconEndpoint
	u.Path = fmt.Sprintf("/eth/v1/beacon/headers/%d", slot)

	err := b.fetchBeacon(&u, &resp)
	if err != nil {
		return nil, err
	}

	return resp.Data, err
}
