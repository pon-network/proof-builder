package bundles

import (
	"context"
	"time"

	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/log"

	bundleTypes "github.com/bsn-eng/pon-golang-types/bundles"

	_ "github.com/mattn/go-sqlite3"
)

func (s *BundleService) ProcessBundleNextTimestamp(ctx context.Context) {

	s.Mu.Lock()
	t := time.NewTimer(time.Until(s.NextBundleTimeStamp))
	s.Mu.Unlock()
	defer t.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			// Search through the next bundle entries and find the ones that are ready to be processed

			log.Info("Bundle monitoring: Checking for new bundles by nextBundleTimeStamp", "timestamp", time.Now())

			upcomingBundleEntries, err := s.GetPendingBundlesByTimestamp()
			if err != nil {
				log.Error("Error getting upcoming bundles", "err", err)
				continue
			}

			var upcomingBundles []*bundleTypes.BuilderBundle
			for _, entry := range upcomingBundleEntries {
				bundle, err := bundleTypes.BuilderBundleEntryToBundle(&entry)
				if err != nil {
					log.Error("Error converting bundle entry to bundle", "err", err)
					continue
				}
				upcomingBundles = append(upcomingBundles, bundle)
			}

			s.Mu.Lock()
			// check if the upcoming bundle is not in the next bundles map if it is not then add it
			for _, bundle := range upcomingBundles {
				if _, ok := s.NextBundlesMap[bundle.ID]; !ok {
					s.NextBundlesMap[bundle.ID] = bundle
				}
			}

			var processedMinTimestamp time.Time

			nextBundlesMapCopy := make(map[string]*bundleTypes.BuilderBundle)
			for _, b := range s.NextBundlesMap {
				nextBundlesMapCopy[b.ID] = b
			}

			// Attain the lock first to stop the block number processor from adding and processing new bundles to the map
			// as the timestamp processor may have ranges that are running valid for a given current block number in a bundle
			// as well
			for _, entry := range s.NextBundlesMap {

				if entry.MinTimestamp < uint64(time.Now().Unix()) && entry.MinTimestamp > 0 {

					if entry.Added || entry.FailedRetryCount > uint64(maxRetryCount) {
						// If the bundle has already been added or the bundle has failed to be added more than the max retry count
						// then remove the bundle from the map
						delete(nextBundlesMapCopy, entry.ID)
						continue
					}

					if entry.MaxTimestamp > 0 && (entry.MaxTimestamp < uint64(time.Now().Unix())) {
						// If there is a max timestamp and it is less than the current time then the bundle has expired
						// Do not process the bundle and remove it from the map
						err = s.UpdateBundleStatus(entry.BundleHash, false, "expired")
						if err != nil {
							log.Error("Error updating bundle status", "err", err)
						}
						delete(nextBundlesMapCopy, entry.ID)
						continue
					}

					if entry.MinTimestamp > uint64(processedMinTimestamp.Unix()) {
						processedMinTimestamp = time.Unix(int64(entry.MinTimestamp), 0)
					}

				}

				if entry.MaxTimestamp < uint64(time.Now().Unix()) && entry.MaxTimestamp > 0 {
					// remove the entry from the nextBundlesMapCopy
					err = s.UpdateBundleStatus(entry.BundleHash, false, "expired")
					if err != nil {
						log.Error("Error updating bundle status", "err", err)
					}
					delete(nextBundlesMapCopy, entry.ID)
				}
			}

			s.NextBundlesMap = nextBundlesMapCopy

			// get the next bundles based on min timestamp first and add them to the list
			nextBundleTimeStamp, err := s.GetNextBundleTimestamp(uint64(processedMinTimestamp.Unix()))
			if err != nil {
				log.Error("Error getting next bundle timestamp", "err", err)
				s.Mu.Unlock()
				continue
			}

			s.NextBundleTimeStamp = time.Unix(int64(nextBundleTimeStamp), 0)
			log.Info("Bundle monitoring: Next bundle timestamp", "timestamp", s.NextBundleTimeStamp)
			s.Mu.Unlock()
			t.Reset(time.Until(time.Unix(int64(nextBundleTimeStamp), 0)))
		}
	}
}

func (s *BundleService) ProcessBundleNextBlockNumber(ctx context.Context) {

	var headEventCh = make(chan core.ChainHeadEvent, 1)

	subscription := s.eth.BlockChain().SubscribeChainHeadEvent(headEventCh)

	for {
		select {
		case <-ctx.Done():
			subscription.Unsubscribe()
			return
		// case err := <-subscription.Err():
		case event := <-headEventCh:

			currentBlock := event.Block

			s.Mu.Lock()
			if currentBlock.Number().Uint64()+1 < s.NextBundleBlockNumber {
				s.Mu.Unlock()
				// No need to process any bundles as we are further than 2 blocks away from the next bundle block number
				continue
			}
			s.Mu.Unlock()

			log.Info("Bundle monitoring: Checking for new bundles by nextBundleBlockNumber", "blockNumber", currentBlock.Number().Uint64())

			// Check for the next block number that has bundles to it and retrieve the bundles
			upcomingBundleEntries, err := s.GetPendingBundlesByBlockNumber()
			if err != nil {
				log.Error("Error getting upcoming bundles", "err", err)
				continue
			}

			var upcomingBundles []*bundleTypes.BuilderBundle
			for _, entry := range upcomingBundleEntries {
				bundle, err := bundleTypes.BuilderBundleEntryToBundle(&entry)
				if err != nil {
					log.Error("Error converting bundle entry to bundle", "err", err)
					continue
				}
				upcomingBundles = append(upcomingBundles, bundle)
			}

			s.Mu.Lock()
			// check if the upcoming bundle is not in the next bundles map if it is not then add it
			for _, bundle := range upcomingBundles {
				if _, ok := s.NextBundlesMap[bundle.ID]; !ok {
					s.NextBundlesMap[bundle.ID] = bundle
				}
			}

			nextBundlesMapCopy := make(map[string]*bundleTypes.BuilderBundle)
			for _, b := range s.NextBundlesMap {
				nextBundlesMapCopy[b.ID] = b
			}

			for _, entry := range s.NextBundlesMap {

				if entry.BlockNumber-1 == currentBlock.Number().Uint64() {

					if entry.Added || entry.FailedRetryCount > uint64(maxRetryCount) {
						// If the bundle has already been added or the bundle has failed to be added more than the max retry count
						// then remove the bundle from the map
						delete(nextBundlesMapCopy, entry.ID)
						continue
					}

					// If in the current block but not past a specific timestamp then do not process the bundle
					if entry.MinTimestamp > uint64(time.Now().Unix()) && entry.MinTimestamp > 0 {
						continue
					}

					// check if the bundle has expired
					if entry.MaxTimestamp > 0 && (entry.MaxTimestamp < uint64(time.Now().Unix())) {
						// If there is a max timestamp and it is less than the current time then the bundle has expired even though the block number is current
						// Do not process the bundle and remove it from the map
						err = s.UpdateBundleStatus(entry.BundleHash, false, "expired")
						if err != nil {
							log.Error("Error updating bundle status", "err", err)
						}
						delete(nextBundlesMapCopy, entry.ID)
						continue
					}

				}

				if entry.BlockNumber < currentBlock.Number().Uint64() && entry.MaxTimestamp == 0 && entry.MinTimestamp == 0 {
					// entry is block number specific and has no timestamp range so remove if block number is less than current block number
					// remove the entry from the nextBundlesMapCopy
					err = s.UpdateBundleStatus(entry.BundleHash, false, "expired")
					if err != nil {
						log.Error("Error updating bundle status", "err", err)
					}
					delete(nextBundlesMapCopy, entry.ID)
				}

			}

			// Finished processing all the bundles in the current block
			nextBlockNumber := currentBlock.Number().Uint64() + 1

			if nextBlockNumber > s.NextBundleBlockNumber {
				// Incase storage has no bundles for the next block number then set the next bundle block number to the next block number
				// just to allow the next block number processor to process the next block number if any bundles are added
				s.NextBundleBlockNumber = nextBlockNumber
			}

			// Attempt at updating the next bundle block number with the any true upcoming bundle block number
			// If there are no upcoming bundles then the next bundle block number will be the current block number + 1
			upcomingBundleBlockNumber, err := s.GetNextBundleBlockNumber(currentBlock.Number().Uint64())
			if err != nil {
				log.Error("Error getting upcoming bundle block number", "err", err)
			}
			if upcomingBundleBlockNumber > s.NextBundleBlockNumber {
				s.NextBundleBlockNumber = upcomingBundleBlockNumber
			}
			log.Info("Bundle monitoring: Next bundle block number", "blockNumber", s.NextBundleBlockNumber)

			s.Mu.Unlock()

		}
	}
}

func (s *BundleService) CleanSentBundles(ctx context.Context) {
	var headEventCh = make(chan core.ChainHeadEvent, 1)

	subscription := s.eth.BlockChain().SubscribeChainHeadEvent(headEventCh)

	for {

		select {
		case <-ctx.Done():
			subscription.Unsubscribe()
			return
		// case err := <-subscription.Err():
		case <-headEventCh:

			s.beacon.BeaconData.Mu.Lock()
			currentSlot := s.beacon.BeaconData.CurrentSlot
			s.beacon.BeaconData.Mu.Unlock()

			// Remove sent bundles that are behind the current slot since
			// they will not be added to the chain and the current slot has already passed
			// So no need to keep this sent history
			s.RemoveSentBundlesBehindSlot(currentSlot)

		}
	}
}
