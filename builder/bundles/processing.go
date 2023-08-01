package bundles

import (
	"fmt"
	"time"

	"sort"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/miner"

	bundleTypes "github.com/bsn-eng/pon-golang-types/bundles"

	"github.com/google/uuid"
)

func (s *BundleService) AddBundle(
	txs []*types.Transaction,
	blockNumber uint64,
	minTimestamp uint64,
	maxTimestamp uint64,
	revertingTxHashes []*common.Hash,
) (*bundleTypes.BuilderBundle, error) {

	currentBlock := s.eth.BlockChain().CurrentBlock()
	if currentBlock == nil {
		return nil, fmt.Errorf("could not get current block")
	}

	if (blockNumber < currentBlock.Number.Uint64() && blockNumber != 0) || (maxTimestamp < uint64(time.Now().Unix()) && maxTimestamp != 0) {
		return nil, fmt.Errorf("bundle is already expired. Provided block number: %d, current block number: %d. Provided max timestamp: %d, current timestamp: %d", blockNumber, currentBlock.Number.Uint64(), maxTimestamp, uint64(time.Now().Unix()))
	} else if blockNumber != 0 && (minTimestamp != 0 || maxTimestamp != 0) {
		return nil, fmt.Errorf("cannot set both block number and time range")
	} else if blockNumber == 0 && minTimestamp == 0 && maxTimestamp == 0 {
		return nil, fmt.Errorf("must set either block number or time range")
	} else if blockNumber == 0 && minTimestamp == 0 && maxTimestamp != 0 {
		return nil, fmt.Errorf("must set min timestamp if max timestamp is set")
	} else if blockNumber == 0 && minTimestamp != 0 && maxTimestamp == 0 {
		return nil, fmt.Errorf("must set max timestamp if min timestamp is set")
	} else if blockNumber == 0 && minTimestamp > maxTimestamp {
		return nil, fmt.Errorf("min timestamp must be less than max timestamp")
	} else if blockNumber == 0 && maxTimestamp - minTimestamp > uint64(s.MaxLifetime) {
		return nil, fmt.Errorf("time range cannot be greater than %d seconds", s.MaxLifetime)
	} else if blockNumber == 0 && minTimestamp > uint64(time.Now().Unix()) && minTimestamp - uint64(time.Now().Unix()) > uint64(s.MaxFuture) {
		return nil, fmt.Errorf("min timestamp cannot be greater than %d seconds in the future", s.MaxFuture)
	}

	bundle, err := s.PrepareBundle(uuid.New().String(), txs, blockNumber, minTimestamp, maxTimestamp, revertingTxHashes)

	// Check if bundle meets minimum gas requirements
	if bundle.BundleTotalGas < minBundleGas {
		return nil, fmt.Errorf("bundle does not meet minimum gas requirements")
	}

	// If the specific block number has been set with no time range, then check the total gas for all bundles in that block
	// If the total gas is greater than the max block gas, then return an error that the bundle cannot be added as it will exceed the max block gas

	totalBlockBundlesGas := uint64(0)
	totalBlockBundlesGas, err = s.GetTotalPendingBundleGasForBlockNumber(bundle.BlockNumber)
	if err != nil {
		log.Debug("Error getting total bundle gas for block number", "err", err)
	}

	if totalBlockBundlesGas+bundle.BundleTotalGas > (miner.DefaultConfig.GasCeil + miner.PaymentTxGas) && bundle.BlockNumber != 0 && bundle.MinTimestamp == 0 && bundle.MaxTimestamp == 0 {
		return nil, fmt.Errorf("bundle cannot be added as it will exceed the max block gas")
	}

	// check if the block number is before or after the next block number
	// if it's before, we can process it immediately
	// if it is after, add the bundle just to the database and process it later

	s.Mu.Lock()
	nextBlockNumber := s.NextBundleBlockNumber
	nextTimestamp := s.NextBundleTimeStamp
	s.Mu.Unlock()

	if (blockNumber < nextBlockNumber && blockNumber != 0) {

		s.Mu.Lock()
		s.NextBundlesMap[bundle.ID] = bundle
		s.Mu.Unlock()

	} else if ((minTimestamp < uint64(time.Now().Unix())) && (maxTimestamp > uint64(time.Now().Unix()))) || (minTimestamp < uint64(nextTimestamp.Unix()) && maxTimestamp > uint64(nextTimestamp.Unix())) {

		s.Mu.Lock()
		s.NextBundlesMap[bundle.ID] = bundle
		s.Mu.Unlock()
	}
	// Add the bundle to storage

	log.Info("Adding bundle to storage", "bundle", bundle)
	bundleEntry, err := bundleTypes.BuilderBundleToEntry(bundle)
	if err != nil {
		return nil, err
	}
	err = s.InsertBundle(bundleEntry)
	if err != nil {
		return nil, err
	}

	return bundle, nil

}

func (s *BundleService) CancelBundle(bundleID string) error {

	s.Mu.Lock()
	nextBundle, ok := s.NextBundlesMap[bundleID]
	s.Mu.Unlock()

	if ok {
		if nextBundle.Adding {
			return fmt.Errorf("bundle is currently being added to a builder block, cannot cancel")
		}

		s.Mu.Lock()
		delete(s.NextBundlesMap, bundleID)
		s.Mu.Unlock()
	}

	// Get the bundle from storage
	existingBundleEntry, err := s.GetBundle(bundleID)
	if err != nil {
		return fmt.Errorf("could not retrieve bundle from storage: %v", err)
	}

	if existingBundleEntry.ID == "" {
		return fmt.Errorf("bundle does not exist")
	}

	if existingBundleEntry.Added {
		return fmt.Errorf("bundle has already been added to a builder block, cannot update")
	}

	err = s.RemoveBundleById(bundleID)
	if err != nil {
		return err
	}

	return nil

}

func (s *BundleService) ChangeBundle(
	bundleID string,
	txs []*types.Transaction,
	blockNumber uint64,
	minTimestamp uint64,
	maxTimestamp uint64,
	revertingTxHashes []*common.Hash,
) (*bundleTypes.BuilderBundle, error) {

	s.Mu.Lock()
	nextBundle, ok := s.NextBundlesMap[bundleID]
	s.Mu.Unlock()

	if ok {
		if nextBundle.Adding {
			return nil, fmt.Errorf("bundle is currently being added to a builder block, cannot update")
		}

		s.Mu.Lock()
		delete(s.NextBundlesMap, bundleID)
		s.Mu.Unlock()
	}

	// Get the bundle from storage
	existingBundleEntry, err := s.GetBundle(bundleID)
	if err != nil {
		return nil, fmt.Errorf("could not retrieve bundle from storage: %v", err)
	}

	if existingBundleEntry.ID == "" {
		return nil, fmt.Errorf("bundle does not exist")
	}

	if existingBundleEntry.Added {
		return nil, fmt.Errorf("bundle has already been added to a builder block, cannot update")
	}

	currentBlock := s.eth.BlockChain().CurrentBlock()
	if currentBlock == nil {
		return nil, fmt.Errorf("could not get current block")
	}

	if (blockNumber < currentBlock.Number.Uint64() && blockNumber != 0) || (maxTimestamp < uint64(time.Now().Unix()) && maxTimestamp != 0) {
		return nil, fmt.Errorf("cannot update bundle, new bundle is already expired")
	} else if blockNumber != 0 && (minTimestamp != 0 || maxTimestamp != 0) {
		return nil, fmt.Errorf("cannot set both block number and time range")
	} else if blockNumber == 0 && minTimestamp == 0 && maxTimestamp == 0 {
		return nil, fmt.Errorf("must set either block number or time range")
	} else if blockNumber == 0 && minTimestamp == 0 && maxTimestamp != 0 {
		return nil, fmt.Errorf("must set min timestamp if max timestamp is set")
	} else if blockNumber == 0 && minTimestamp != 0 && maxTimestamp == 0 {
		return nil, fmt.Errorf("must set max timestamp if min timestamp is set")
	} else if blockNumber == 0 && minTimestamp > maxTimestamp {
		return nil, fmt.Errorf("min timestamp must be less than max timestamp")
	} else if blockNumber == 0 && maxTimestamp - minTimestamp > uint64(s.MaxLifetime) {
		return nil, fmt.Errorf("time range cannot be greater than %d seconds", s.MaxLifetime)
	} else if blockNumber == 0 && minTimestamp > uint64(time.Now().Unix()) && minTimestamp - uint64(time.Now().Unix()) > uint64(s.MaxFuture) {
		return nil, fmt.Errorf("min timestamp cannot be greater than %d seconds in the future", s.MaxFuture)
	}

	bundle, err := s.PrepareBundle(bundleID, txs, blockNumber, minTimestamp, maxTimestamp, revertingTxHashes)

	// Compare the bundle hash to the existing bundle hash
	if bundle.BundleHash == existingBundleEntry.BundleHash {
		return nil, fmt.Errorf("bundle is the same as the existing bundle, cannot update")
	}

	// Check if bundle meets minimum gas requirements
	if bundle.BundleTotalGas < minBundleGas {
		return nil, fmt.Errorf("bundle does not meet minimum gas requirements")
	}

	// If the specific block number has been set with no time range, then check the total gas for all bundles in that block
	// If the total gas is greater than the max block gas, then return an error that the bundle cannot be added as it will exceed the max block gas

	totalBlockBundlesGas := uint64(0)
	totalBlockBundlesGas, err = s.GetTotalPendingBundleGasForBlockNumber(bundle.BlockNumber)
	if err != nil {
		log.Debug("Error getting total bundle gas for block number", "err", err)
	}

	if totalBlockBundlesGas+bundle.BundleTotalGas > (miner.DefaultConfig.GasCeil + miner.PaymentTxGas) && bundle.BlockNumber != 0 && bundle.MinTimestamp == 0 && bundle.MaxTimestamp == 0 {
		return nil, fmt.Errorf("bundle cannot be added as it will exceed the max block gas")
	}

	// check if the block number is before or after the next block number
	// if it's before, we can process it immediately
	// if it is after, add the bundle just to the database and process it later

	s.Mu.Lock()
	nextBlockNumber := s.NextBundleBlockNumber
	nextTimestamp := s.NextBundleTimeStamp
	s.Mu.Unlock()

	if (blockNumber < nextBlockNumber && blockNumber != 0) {

		s.Mu.Lock()
		s.NextBundlesMap[bundle.ID] = bundle
		s.Mu.Unlock()

	} else if ((minTimestamp < uint64(time.Now().Unix())) && (maxTimestamp > uint64(time.Now().Unix()))) || (minTimestamp < uint64(nextTimestamp.Unix()) && maxTimestamp > uint64(nextTimestamp.Unix())) {

		s.Mu.Lock()
		s.NextBundlesMap[bundle.ID] = bundle
		s.Mu.Unlock()
	}

	// Add the bundle to storage
	log.Info("Updating bundle to storage", "bundle", bundle)
	bundleEntry, err := bundleTypes.BuilderBundleToEntry(bundle)
	if err != nil {
		return nil, err
	}
	err = s.UpdateBundle(bundleEntry)
	if err != nil {
		return nil, err
	}

	return bundle, nil

}

func (s *BundleService) GetBundleByID(bundleID string) (*bundleTypes.BuilderBundle, error) {

	bundleEntry, err := s.GetBundle(bundleID)
	if err != nil {
		return nil, err
	}

	bundle, err := bundleTypes.BuilderBundleEntryToBundle(&bundleEntry)
	if err != nil {
		return nil, err
	}

	return bundle, nil

}

func (s *BundleService) PrepareBundle(
	bundleID string,
	txs []*types.Transaction,
	blockNumber uint64,
	minTimestamp uint64,
	maxTimestamp uint64,
	revertingTxHashes []*common.Hash,
) (*bundleTypes.BuilderBundle, error) {

	var bundle bundleTypes.BuilderBundle

	bundle.Txs = txs

	bundle.ID = bundleID

	// calculate total gas
	var totalGas uint64
	for _, tx := range txs {
		totalGas += tx.Gas()
	}
	bundle.BundleTotalGas = totalGas

	bundle.BundleTransactionCount = uint64(len(txs))

	bundle.BlockNumber = blockNumber
	bundle.MinTimestamp = minTimestamp
	bundle.MaxTimestamp = maxTimestamp

	bundle.RevertingTxHashes = revertingTxHashes

	bundle.BundleDateTime = time.Now()

	bundleHash, err := bundle.HashTreeRoot()
	if err != nil {
		return nil, err
	}
	bundle.BundleHash = hexutil.Encode(bundleHash[:])
	bundle.BuilderPubkey = common.Bytes2Hex(crypto.FromECDSAPub(&s.builderWalletPrivateKey.PublicKey))

	builderSignature, err := crypto.Sign(bundleHash[:], s.builderWalletPrivateKey)
	if err != nil {
		return nil, err
	}
	bundle.BuilderSignature = common.Bytes2Hex(builderSignature)

	return &bundle, nil

}

func (s *BundleService) GetReadyBundles(blockNumber uint64, currentTime time.Time) ([]bundleTypes.BuilderBundle, error) {

	var bundles []bundleTypes.BuilderBundle

	s.Mu.Lock()
	defer s.Mu.Unlock()

	for _, bundle := range s.NextBundlesMap {
		// Get bundles that are the current block number or that have a timerange that includes the current time
		if bundle.BlockNumber == blockNumber || ((bundle.MinTimestamp < uint64(currentTime.Unix())) && (bundle.MaxTimestamp > uint64(currentTime.Unix()))) {
			if !bundle.Adding && !bundle.Added {
				bundles = append(bundles, *bundle)
			}
		}
	}

	// Sort the bundles by block number first, then by soonest min timestamp first
	sort.Slice(bundles, func(i, j int) bool {
		if bundles[i].BlockNumber == bundles[j].BlockNumber {
			if bundles[i].BundleTotalGas == bundles[j].BundleTotalGas {
				return bundles[i].BundleDateTime.Unix() < bundles[j].BundleDateTime.Unix()
			}
			return bundles[i].BundleTotalGas > bundles[j].BundleTotalGas
		}
		return bundles[i].BlockNumber < bundles[j].BlockNumber
	})

	return bundles, nil

}

func (s *BundleService) SetBundlesAddingTrue(bundles []bundleTypes.BuilderBundle) ([]bundleTypes.BuilderBundle, error) {

	s.Mu.Lock()
	defer s.Mu.Unlock()

	for _, bundle := range bundles {
		bundle.Adding = true
		// get bundle from map and update
		nextBundle, ok := s.NextBundlesMap[bundle.ID]
		if ok {
			nextBundle.Adding = true
			if bundle.FailedRetryCount > 0 {
				nextBundle.FailedRetryCount = bundle.FailedRetryCount
			}
			if !nextBundle.Error {
				nextBundle.Error = bundle.Error
			}
			if bundle.ErrorMessage != "" {
				nextBundle.ErrorMessage = bundle.ErrorMessage
			}
			s.NextBundlesMap[bundle.ID] = nextBundle
		}
	}

	return bundles, nil

}

func (s *BundleService) SetBundlesAddingFalse(bundles []bundleTypes.BuilderBundle) ([]bundleTypes.BuilderBundle, error) {

	s.Mu.Lock()
	defer s.Mu.Unlock()

	for _, bundle := range bundles {
		bundle.Adding = false
		// get bundle from map and update
		nextBundle, ok := s.NextBundlesMap[bundle.ID]
		if ok {
			nextBundle.Adding = false
			if bundle.FailedRetryCount > 0 {
				nextBundle.FailedRetryCount = bundle.FailedRetryCount
			}
			if !nextBundle.Error {
				nextBundle.Error = bundle.Error
			}
			if bundle.ErrorMessage != "" {
				nextBundle.ErrorMessage = bundle.ErrorMessage
			}
			s.NextBundlesMap[bundle.ID] = nextBundle
		}
	}

	return bundles, nil

}

func (s *BundleService) SetBundlesAdded(bundles []bundleTypes.BuilderBundle) error {

	// Set the bundles as added and remove them from the map, then update the bundles in storage
	s.Mu.Lock()
	defer s.Mu.Unlock()

	for _, bundle := range bundles {
		bundle.Added = true
		bundle.Adding = false
		// get bundle from map and update
		nextBundle, ok := s.NextBundlesMap[bundle.ID]
		if ok {
			delete(s.NextBundlesMap, nextBundle.ID)
		}
	}

	// Update the bundles in storage
	for _, bundle := range bundles {
		bundleEntry, err := bundleTypes.BuilderBundleToEntry(&bundle)
		if err != nil {
			return err
		}
		err = s.UpdateBundle(bundleEntry)
		if err != nil {
			return err
		}
	}

	return nil

}

func (s *BundleService) RetrieveBundlesByHashes(bundleHashes []string) ([]bundleTypes.BuilderBundle, error) {

	var bundles []bundleTypes.BuilderBundle

	bundleEntries, err := s.GetBundlesByHashes(bundleHashes)
	if err != nil {
		return nil, err
	}

	for _, bundleEntry := range bundleEntries {
		bundle, err := bundleTypes.BuilderBundleEntryToBundle(&bundleEntry)
		if err != nil {
			return nil, err
		}
		bundles = append(bundles, *bundle)
	}

	return bundles, nil

}

func (s *BundleService) BundleSent(bundleHash string, blockHash string, slot uint64) error {

	err := s.AddSentBundle(bundleHash, blockHash, slot)
	if err != nil {
		return err
	}

	return nil
}

func (s *BundleService) ProcessSentBundles(bundles []bundleTypes.BuilderBundle, blockHash string, slot uint64) error {

	s.Mu.Lock()
	defer s.Mu.Unlock()

	for _, bundle := range bundles {
		bundle.Adding = false
		if bundle.Included {
			// bundle was successfully added to the block bid
			err := s.BundleSent(bundle.BundleHash, blockHash, slot)
			if err != nil {
				log.Error("could not update bundles sent", "bundleID", bundle.ID, "error", err)
				// return err
			}
		} else {
			bundle.FailedRetryCount += 1
			// bundle was not successfully added to the block bid
		}
		bundle.Included = false
		// get bundle from map and update
		nextBundle, ok := s.NextBundlesMap[bundle.ID]
		if ok {
			nextBundle.Adding = false
			if bundle.FailedRetryCount > 0 {
				nextBundle.FailedRetryCount = bundle.FailedRetryCount
			}
			if !nextBundle.Error {
				nextBundle.Error = bundle.Error
			}
			if bundle.ErrorMessage != "" {
				nextBundle.ErrorMessage = bundle.ErrorMessage
			}
			s.NextBundlesMap[bundle.ID] = nextBundle
		}

		bundleEntry, err := bundleTypes.BuilderBundleToEntry(&bundle)
		if err != nil {
			// return err
		}
		err = s.UpdateBundle(bundleEntry)
		if err != nil {
			// return err
		}
	}

	return nil
}

func (s *BundleService) ProcessBlockAdded(blockHash string) {

	// Get the bundles that were added to the block
	bundleHashes, err := s.GetSentBundleHashesForBlock(blockHash)
	if err != nil {
		log.Debug("Failed to get sent bundle hashes for block", "err", err)
	}

	if len(bundleHashes) == 0 {
		return
	}

	// Get these bundles and set them as added
	bundles, err := s.RetrieveBundlesByHashes(bundleHashes)
	if err != nil {
		log.Debug("Failed to get bundles by hashes", "err", err)
	}

	s.SetBundlesAdded(bundles)

}
