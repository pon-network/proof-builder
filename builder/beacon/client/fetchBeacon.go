package beaconClient

import (
	"fmt"
	"strings"

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

func (b *beaconClient) GetForkVersion(slot uint64, head bool) (forkName string, forkVersion string, err error) {

	type NodeSpec struct {
		Data map[string]string`json:"data"`
	}

	type CurrentForkData struct {
		PreviousVersion string `json:"previous_version"`
		CurrentVersion string `json:"current_version"`
		Epoch string `json:"epoch"`
	}

	type CurrentFork struct {
		ExecutionOptimistic bool `json:"execution_optimistic"`
		Finalized bool `json:"finalized"`
		Data CurrentForkData `json:"data"`
	}

	
	knownSpecs := make(map[string]string)

	u := *b.beaconEndpoint


	u.Path = "/eth/v1/config/spec"
	specResp := new(NodeSpec)
	err = b.fetchBeacon(&u, &specResp)
	if err != nil {
		return "", "", err
	}


	for k, v := range specResp.Data {
		if strings.Contains(k, "_FORK_VERSION") {
			
			k = strings.Replace(k, "_FORK_VERSION", "", 1)
			k = strings.ToLower(k)
			
			knownSpecs[v] = k
		}
	}

	if head {
		u.Path = "/eth/v1/beacon/states/head/fork"
	} else {
		u.Path = fmt.Sprintf("/eth/v2/beacon/states/%d/fork", slot)
	}
	currForkResp := new(CurrentFork)
	err = b.fetchBeacon(&u, &currForkResp)
	if err != nil {
		return "", "", err
	}

	currentForkVersion := currForkResp.Data.CurrentVersion
	currentForkName, ok := knownSpecs[currentForkVersion]
	if !ok {
		return "", "", fmt.Errorf("unknown fork version %s", currentForkVersion)
	}

	return currentForkName, currentForkVersion, err
}
