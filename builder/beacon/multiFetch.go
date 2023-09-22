package multibeaconClient

import (
	"errors"
	"time"

	beaconTypes "github.com/bsn-eng/pon-golang-types/beaconclient"
	beaconData "github.com/ethereum/go-ethereum/builder/beacon/data"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
)

func (b *MultiBeaconClient) GetWithdrawals(slot uint64) (withdrawals *beaconTypes.Withdrawals, err error) {
	/*
		Get expected withdrawals for given slot.
		If any client fails, try the next one.
		Clients are attempted by best performance first.
		Performance is also updated in defer function (triggers background update).
	*/
	defer b.postBeaconCall()
	for _, client := range b.Clients {
		if withdrawals, err = client.Node.GetWithdrawals(slot); err != nil {
			log.Warn("failed to get withdrawals", "err", err, "endpoint", client.Node.BaseEndpoint())
			b.clientUpdate.Lock()
			client.LastResponseStatus = 500
			client.LastUsedTime = time.Now()
			b.clientUpdate.Unlock()
			continue
		}
		b.clientUpdate.Lock()
		client.LastResponseStatus = 200
		client.LastUsedTime = time.Now()
		b.clientUpdate.Unlock()

		return withdrawals, nil
	}

	return nil, err
}

func (b *MultiBeaconClient) GetSlotProposerMap(epoch uint64) (beaconData.SlotProposerMap, error) {
	/*
		Get proposer duties for a given epoch. This is used to create a map of slot to proposer.
		If any client fails, try the next one.
		Clients are attempted by best performance first.
		Performance is also updated in defer function (triggers background update).
	*/
	defer b.postBeaconCall()
	for _, client := range b.Clients {

		duties, err := client.Node.GetSlotProposerMap(epoch)
		if err != nil {
			log.Error("beacon client service: failed to get proposer duties", "err", err, "endpoint", client.Node.BaseEndpoint())
			b.clientUpdate.Lock()
			client.LastResponseStatus = 500
			client.LastUsedTime = time.Now()
			b.clientUpdate.Unlock()
			continue
		}

		b.clientUpdate.Lock()
		client.LastResponseStatus = 200
		client.LastUsedTime = time.Now()
		b.clientUpdate.Unlock()

		return duties, nil
	}

	return nil, errors.New("all beacon nodes failed")
}

func (b *MultiBeaconClient) Genesis() (genesisData *beaconTypes.GenesisData, err error) {
	/*
		Get chain genesis data.
		If any client fails, try the next one.
		Clients are attempted by best performance first.
		Performance is also updated in defer function (triggers background update).
	*/
	defer b.postBeaconCall()
	for _, client := range b.Clients {
		if genesisData, err = client.Node.Genesis(); err != nil {
			log.Warn("failed to get genesis", "err", err, "endpoint", client.Node.BaseEndpoint())
			b.clientUpdate.Lock()
			client.LastResponseStatus = 500
			client.LastUsedTime = time.Now()
			b.clientUpdate.Unlock()
			continue
		}

		b.clientUpdate.Lock()
		client.LastResponseStatus = 200
		client.LastUsedTime = time.Now()
		b.clientUpdate.Unlock()

		return genesisData, nil
	}

	return nil, err
}

func (b *MultiBeaconClient) Randao(slot uint64) (randao *common.Hash, err error) {
	/*
		Get randao of slot. Attempts to retrieve from known randao map first.
		If not found, attempts to retrieve from client.
		If any client fails, try the next one.
		Clients are attempted by best performance first.
		Performance is also updated in defer function (triggers background update).
	*/
	b.BeaconData.Mu.Lock()
	knownRandao, found := b.BeaconData.RandaoMap[slot]
	b.BeaconData.Mu.Unlock()
	if found {
		return &knownRandao, nil
	}

	defer b.postBeaconCall()
	for _, client := range b.Clients {
		if randao, err = client.Node.Randao(slot); err != nil {
			// log.Warn("failed to get randao", "err", err, "endpoint", client.Node.BaseEndpoint())
			b.clientUpdate.Lock()
			client.LastResponseStatus = 500
			client.LastUsedTime = time.Now()
			b.clientUpdate.Unlock()
			continue
		}

		b.clientUpdate.Lock()
		client.LastResponseStatus = 200
		client.LastUsedTime = time.Now()
		b.clientUpdate.Unlock()

		return randao, err
	}

	return nil, err
}

func (b *MultiBeaconClient) GetBlockHeader(slot uint64) (blockHeader *beaconTypes.BlockHeaderData, err error) {
	/*
		Get block header of slot.
		If any client fails, try the next one.
		Clients are attempted by best performance first.
		Performance is also updated in defer function (triggers background update).
	*/
	defer b.postBeaconCall()
	for _, client := range b.Clients {
		if blockHeader, err = client.Node.GetBlockHeader(slot); err != nil {
			log.Warn("failed to get block header", "err", err, "endpoint", client.Node.BaseEndpoint())
			b.clientUpdate.Lock()
			client.LastResponseStatus = 500
			client.LastUsedTime = time.Now()
			b.clientUpdate.Unlock()
			continue
		}

		b.clientUpdate.Lock()
		client.LastResponseStatus = 200
		client.LastUsedTime = time.Now()
		b.clientUpdate.Unlock()

		return blockHeader, nil
	}

	return nil, err
}

func (b *MultiBeaconClient) GetForkVersion(slot uint64, head bool) (forkName string, forkVersion string, err error) {
	/*
		Get fork version of chain.
		If any client fails, try the next one.
		Clients are attempted by best performance first.
		Performance is also updated in defer function (triggers background update).
	*/
	defer b.postBeaconCall()
	for _, client := range b.Clients {

		if forkName, forkVersion, err = client.Node.GetForkVersion(slot, head); err != nil {
			log.Warn("failed to get fork version", "err", err, "endpoint", client.Node.BaseEndpoint())
			b.clientUpdate.Lock()
			client.LastResponseStatus = 500
			client.LastUsedTime = time.Now()
			b.clientUpdate.Unlock()
			continue
		}

		b.clientUpdate.Lock()
		client.LastResponseStatus = 200
		client.LastUsedTime = time.Now()
		b.clientUpdate.Unlock()

		return forkName, forkVersion, nil
	}

	return "", "", err
}
