package multibeaconClient

import (
	"context"
	"errors"
	"sort"
	"sync"
	"time"

	beaconTypes "github.com/bsn-eng/pon-golang-types/beaconclient"
	beaconClient "github.com/ethereum/go-ethereum/builder/beacon/client"
	beaconData "github.com/ethereum/go-ethereum/builder/beacon/data"

	"github.com/ethereum/go-ethereum/log"
)

type BeaconClient struct {
	Node               beaconClient.BeaconClientInstance
	NodeSpeed          time.Duration
	LastUsedTime       time.Time
	LastResponseStatus int
	SyncStatus         *beaconTypes.SyncStatusData
}

type MultiBeaconClient struct {
	Clients      []BeaconClient
	clientUpdate sync.Mutex
	BeaconData   *beaconData.BeaconData
}

func NewMultiBeaconClient(beaconUrls []string) (*MultiBeaconClient, error) {
	/*
		The multi beacon client is a wrapper around multiple beacon clients.
		It is used to fetch and post data to multiple clients for best availability.
		The multi beacon client will also keep track of the last used time and response
		status of each client, and will use the client with the best status.
		Data is kept centralized in the BeaconData struct, which is shared between all clients.
	*/
	clients := make([]BeaconClient, len(beaconUrls))
	for i, url := range beaconUrls {
		client, err := beaconClient.NewBeaconClient(url)
		if err != nil {
			log.Error("failed to create beacon client", "err", err)
			return nil, err
		}
		clients[i] = BeaconClient{Node: client}
	}

	return &MultiBeaconClient{Clients: clients, BeaconData: &beaconData.BeaconData{
		SlotProposerMap:          make(beaconData.SlotProposerMap),
		SlotPayloadAttributesMap: make(beaconData.SlotPayloadAttributesMap),
		RandaoMap:                make(beaconData.RandaoMap),
		CloseCh:                  make(chan struct{}),
		HeadSlotC:                make(chan beaconTypes.HeadEventData),
		PayloadAttributesC:       make(chan beaconTypes.PayloadAttributesEventData),
	}}, nil
}

func (b *MultiBeaconClient) Start() {
	/*
		This function starts the multi beacon client by waiting for at least one client to be synced,
		subscribing to head events and payload attributes events, and running them in separate goroutines.
		The `waitSynced()` function waits for at least one client to be synced by periodically calling the
		`SyncStatus()` function until a synced client is found. Once a synced client is found, the function
		subscribes to head events and payload attributes events using the `SubscribeToHeadEvents()` and
		`SubscribeToPayloadAttributesEvents()` functions, respectively. These events are run in separate
		goroutines to allow for concurrent processing.
	*/

	log.Info("Waiting for at least one client to be synced")
	b.waitSynced()
	log.Info("At least one client is synced, starting head and payload attributes subscriptions")

	go b.SubscribeToHeadEvents(context.Background(), b.BeaconData.HeadSlotC)
	go b.SubscribeToPayloadAttributesEvents(context.Background(), b.BeaconData.PayloadAttributesC)
}

func (b *MultiBeaconClient) Stop() {
	close(b.BeaconData.CloseCh)
}

func (b *MultiBeaconClient) waitSynced() {
	// wait for at least one client to be synced, call the sync status endpoint periodically till one is synced
	for {
		syncStatus, _ := b.SyncStatus()
		log.Info("sync status", "syncStatus", syncStatus)
		if syncStatus != nil && !syncStatus.IsSyncing {
			return
		}
		time.Sleep(5 * time.Second)
	}

}

func (b *MultiBeaconClient) UpdateRandaoMap(slot uint64) {
	/*
		Update the randao map with the randao for the given slot.
	*/

	// Get the randao for the slot
	randao, err := b.Randao(slot)
	if err != nil {
		// log.Warn("failed to get randao", "slot", slot, "err", err)
		return
	}

	// Update the randao map
	b.BeaconData.Mu.Lock()
	b.BeaconData.RandaoMap[slot] = *randao
	b.BeaconData.Mu.Unlock()

}

func (b *MultiBeaconClient) UpdateValidatorMap() {
	/*
		Update the proposer duties by getting the available proposers
		for the current epoch and the next epoch, in case some clients are
		behind, when chain is truly ahead, handles edge case of
		sync issues at edge of epoch.
	*/
	var wg sync.WaitGroup
	for _, client := range b.Clients {
		wg.Add(1)
		go func(client BeaconClient) {
			defer wg.Done()

			currentSlot := b.BeaconData.CurrentHead.Slot
			currentEpoch := currentSlot / 32

			b.updateValidatorMap(client, currentEpoch)

			// update the next epoch as well
			b.updateValidatorMap(client, currentEpoch+1)

		}(client)
	}
	wg.Wait()
}

func (b *MultiBeaconClient) updateValidatorMap(client BeaconClient, epoch uint64) {
	/*
		Update the validator map for the given epoch.
		This gets the proposer duties for the given epoch from the client
		and updates the slot proposer map with the proposer duties.
		(slot to proposer)
	*/
	validatorMap, err := client.Node.GetSlotProposerMap(epoch)
	if err != nil {
		log.Warn("failed to get validator map", "err", err, "endpoint", client.Node.BaseEndpoint())
		b.clientUpdate.Lock()
		client.LastResponseStatus = 500
		client.LastUsedTime = time.Now()
		b.clientUpdate.Unlock()
		return
	}

	b.clientUpdate.Lock()
	client.LastResponseStatus = 200
	client.LastUsedTime = time.Now()
	b.clientUpdate.Unlock()

	b.BeaconData.Mu.Lock()
	for k, v := range validatorMap {
		b.BeaconData.SlotProposerMap[k] = v
	}
	b.BeaconData.Mu.Unlock()

}

func (b *MultiBeaconClient) GetSlotProposer(requestedSlot uint64) (*beaconTypes.ProposerDutyData, error) {
	/*
		Get the proposer for the given slot.
		This checks the slot proposer map for the given slot and returns
		the proposer if found. If not found, it gets the proposer duties
		for the epoch of the given slot and updates the slot proposer map
		with the proposer duties for the epoch. Then it checks the slot
		proposer map again for the given slot and returns the proposer if
		found. If not found, it returns an error.
	*/
	b.BeaconData.Mu.Lock()
	proposer, found := b.BeaconData.SlotProposerMap[requestedSlot]
	b.BeaconData.Mu.Unlock()
	if !found {
		log.Warn("inconsistent proposer mapping", "requestSlot", requestedSlot)
		proposerMap, err := b.GetSlotProposerMap(requestedSlot / 32)
		if err != nil {
			return nil, err
		}

		b.BeaconData.Mu.Lock()
		for _, proposer := range proposerMap {
			b.BeaconData.SlotProposerMap[proposer.Slot] = proposer
		}

		proposer, found = b.BeaconData.SlotProposerMap[requestedSlot]
		b.BeaconData.Mu.Unlock()
		if !found {
			return nil, errors.New("failed to find proposer")
		}

	}

	return &proposer, nil
}

func (b *MultiBeaconClient) GetCurrentHead() (beaconTypes.HeadEventData, error) {
	/*
		Get the current head.
		This returns the current head from the beacon data.
		The beacon data is updated in background by head event.
		Holds true known head for all clients, for cases where some clients
		have heads behind the true head.
	*/
	b.BeaconData.Mu.Lock()
	data := b.BeaconData.CurrentHead
	defer b.BeaconData.Mu.Unlock()
	return data, nil
}

func (b *MultiBeaconClient) GetPayloadAttributesForSlot(requestedSlot uint64) (*beaconTypes.PayloadAttributesEventData, error) {
	/*
		Get the payload attributes for the given slot.
		This checks the slot payload attributes map for the given slot and
		returns the payload attributes if found. If not found, it gets the
		payload attributes for the epoch of the given slot and updates the
		slot payload attributes map with the payload attributes for the
		epoch. Then it checks the slot payload attributes map again for the
		given slot and returns the payload attributes if found. If not
		found, it returns an error.
	*/
	b.BeaconData.Mu.Lock()
	payloadAttributes, found := b.BeaconData.SlotPayloadAttributesMap[requestedSlot]
	b.BeaconData.Mu.Unlock()
	if !found {
		log.Warn("inconsistent payload attributes mapping", "requestSlot", requestedSlot)

		// Get the proposer for the slot
		proposer, err := b.GetSlotProposer(requestedSlot)
		if err != nil {
			return nil, err
		}

		// Get the parent block for the slot
		parentBlockHeader, err := b.GetBlockHeader(requestedSlot - 1)
		if err != nil {
			return nil, err
		}

		// Get the previous randao
		previousRandao, err := b.Randao(requestedSlot - 1)
		if err != nil {
			return nil, err
		}

		// Get withdrawals of this slot
		withdrawals, err := b.GetWithdrawals(requestedSlot)
		if err != nil {
			return nil, err
		}

		attrs := beaconTypes.PayloadAttributes{
			Withdrawals: withdrawals,
			PrevRandao:  previousRandao.String(),
			// Timestamp not needed
			// Cannot get suggested fee recipient and not needed
		}

		// Construct the payload attributes
		payloadAttributes := beaconTypes.PayloadAttributesEventData{
			ProposerIndex: proposer.Index,
			ProposalSlot:  requestedSlot,
			// Parent block number not needed
			ParentBlockRoot:   parentBlockHeader.Header.Message.StateRoot,
			ParentBlockHash:   parentBlockHeader.Root,
			PayloadAttributes: &attrs,
		}

		b.BeaconData.Mu.Lock()
		b.BeaconData.SlotPayloadAttributesMap[requestedSlot] = payloadAttributes
		b.BeaconData.Mu.Unlock()
		if !found {
			return nil, errors.New("failed to find payload attributes")
		}

	}

	return &payloadAttributes, nil
}

func (b *MultiBeaconClient) SyncStatus() (*beaconTypes.SyncStatusData, error) {
	/*
		Get the sync status of the best performing client.
		All clients are checked for sync status and the best performing
		client is used to get the sync status.
	*/
	var foundSyncedNode bool

	var wg sync.WaitGroup
	for i, instance := range b.Clients {
		wg.Add(1)
		go func(client BeaconClient, index int) {
			defer wg.Done()

			startTime := time.Now()
			syncStatus, err := client.Node.SyncStatus()
			endTime := time.Now()
			if err != nil {
				log.Error("failed to get sync status", "err", err, "endpoint", client.Node.BaseEndpoint())
				b.clientUpdate.Lock()
				client.LastResponseStatus = 500
				client.LastUsedTime = time.Now()
				client.NodeSpeed = endTime.Sub(startTime)
				b.Clients[index] = client
				b.clientUpdate.Unlock()
				return
			}

			b.clientUpdate.Lock()
			client.LastResponseStatus = 200
			client.LastUsedTime = time.Now()
			client.SyncStatus = syncStatus
			client.NodeSpeed = endTime.Sub(startTime)
			b.Clients[index] = client
			b.clientUpdate.Unlock()

			if !syncStatus.IsSyncing && !foundSyncedNode {
				foundSyncedNode = true
			}

		}(instance, i)
	}

	wg.Wait()

	// Sync status, after `waitSynced`, is a function that is called asynchronously on every head event
	// This is to ensure that the sync status is always up to date
	// Since this is the only function that is always called asynchronously, we can use it as the main performance updater
	// This is the point that node speed is updated, as it is the only functino that has a uniform payload/response across all nodes
	b.updateClientPerformance()

	if b.Clients[0].SyncStatus == nil {
		return nil, errors.New("all beacon nodes failed")
	}

	return b.Clients[0].SyncStatus, nil
}

func (b *MultiBeaconClient) postBeaconCall() {
	/*
		The `postBeaconCall()` function is used to update the performance of each beacon client in
		the `MultiBeaconClient` struct. It is called after every beacon call to ensure that the
		performance of each client is up to date. The function acquires a lock on the `clientUpdate`
		mutex to ensure that the clients are not updated concurrently.
	*/
	go b.updateClientPerformance()
}

func (b *MultiBeaconClient) ReturnAllNodeURLs() []string {
	// Return all node URLs
	var urls []string
	for _, instance := range b.Clients {
		urls = append(urls, instance.Node.BaseEndpoint())
	}

	return urls
}

func (b *MultiBeaconClient) updateClientPerformance() {
	/*
		The `updateClientPerformance()` function is used to update the performance of each beacon client in
		the `MultiBeaconClient` struct. It sorts the clients based on their sync status, last response
		status, node speed, and last used time. The function acquires a lock on the `clientUpdate` mutex to
		ensure that the clients are not updated concurrently. The sorted clients are then stored back in the
		`MultiBeaconClient` struct.
	*/
	clients := b.Clients

	b.clientUpdate.Lock()
	/*
		The clients are sorted based on the following criteria:
		1. Sync status: Syncing clients are prioritized over non-syncing clients
		2. Head slot: Clients with a higher head slot are prioritized over clients with a lower head slot
		3. Last response status: Clients with a 200 response status are prioritized over clients with a 500 response status
		4. Node speed: Clients with a faster response time are prioritized over clients with a slower response time
		5. Last used time: Clients that have been used more recently are prioritized over clients that have not been used recently
	*/
	sort.SliceStable(clients, func(i, j int) bool {
		if clients[i].SyncStatus.IsSyncing == clients[j].SyncStatus.IsSyncing {
			if clients[i].SyncStatus.HeadSlot == clients[j].SyncStatus.HeadSlot {
				if clients[i].LastResponseStatus == clients[j].LastResponseStatus {
					if clients[i].NodeSpeed == clients[j].NodeSpeed {
						return clients[i].LastUsedTime.After(clients[j].LastUsedTime)
					}
					return clients[i].NodeSpeed < clients[j].NodeSpeed
				}
				return clients[i].LastResponseStatus < clients[j].LastResponseStatus
			}
			return clients[i].SyncStatus.HeadSlot > clients[j].SyncStatus.HeadSlot
		}
		return clients[i].SyncStatus.IsSyncing
	})
	b.clientUpdate.Unlock()

}
