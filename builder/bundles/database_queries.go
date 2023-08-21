package bundles

import (
	"fmt"
	"math/big"
	"time"

	bbTypes "github.com/ethereum/go-ethereum/builder/types"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/jmoiron/sqlx"

	bundleTypes "github.com/bsn-eng/pon-golang-types/bundles"
)

func (s *BundleService) InsertBundle(blockBundle *bundleTypes.BuilderBundleEntry) error {

	tx := s.db.MustBegin()
	_, err := tx.NamedExec(`INSERT INTO blockbundles
		(id, bundle_hash, txs, block_number, min_timestamp, max_timestamp, reverting_tx_hashes, builder_pubkey, builder_signature, bundle_transaction_count, bundle_total_gas, added, error, error_message, failed_retry_count)
		VALUES
		(:id, :bundle_hash, :txs, :block_number, :min_timestamp, :max_timestamp, :reverting_tx_hashes, :builder_pubkey, :builder_signature, :bundle_transaction_count, :bundle_total_gas, :added, :error, :error_message, :failed_retry_count)`, blockBundle)
	if err != nil {
		return err
	}
	err = tx.Commit()
	if err != nil {
		return err
	}
	return nil
}

func (s *BundleService) GetBundleByHash(bundleHash string) (bundleTypes.BuilderBundleEntry, error) {
	var blockBundle bundleTypes.BuilderBundleEntry
	err := s.db.Get(&blockBundle, "SELECT * FROM blockbundles WHERE bundle_hash=$1", bundleHash)
	if err != nil {
		return blockBundle, err
	}
	return blockBundle, nil
}

func (s *BundleService) GetBundlesByHashes(bundlHashes []string) ([]bundleTypes.BuilderBundleEntry, error) {
	var blockBundles []bundleTypes.BuilderBundleEntry
	query, args, err := sqlx.In("SELECT * FROM blockbundles WHERE bundle_hash IN (?)", bundlHashes)
	if err != nil {
		return blockBundles, err
	}
	err = s.db.Select(&blockBundles, query, args...)
	if err != nil {
		return blockBundles, err
	}
	return blockBundles, nil
}

func (s *BundleService) GetBundlesByBlockNumber(blockNumber uint64) ([]bundleTypes.BuilderBundleEntry, error) {
	var blockBundles []bundleTypes.BuilderBundleEntry
	err := s.db.Select(&blockBundles, "SELECT * FROM blockbundles WHERE block_number=$1", blockNumber)
	if err != nil {
		return blockBundles, err
	}
	return blockBundles, nil
}

func (s *BundleService) GetTotalPendingBundleGasForBlockNumber(blockNumber uint64) (*big.Int, error) {
	var totalBundleGas big.Int
	var totalBundleGasString string
	err := s.db.Get(&totalBundleGasString, "SELECT COALESCE(TOTAL(bundle_total_gas), 0) FROM blockbundles WHERE block_number=$1 AND added=false AND failed_retry_count < $2", blockNumber, maxRetryCount)
	if err != nil {
		return big.NewInt(0), err
	}
	totalBundleGasString, err = bbTypes.ConvertScientificToDecimal(totalBundleGasString)
	if err != nil {
		return big.NewInt(0), err
	}
	_, ok := totalBundleGas.SetString(totalBundleGasString, 10)
	if !ok {
		return big.NewInt(0), fmt.Errorf("failed to convert string to big.Int")
	}
	return &totalBundleGas, nil
}

func (s *BundleService) GetPendingBundlesByBlockNumber() ([]bundleTypes.BuilderBundleEntry, error) {
	var blockBundles []bundleTypes.BuilderBundleEntry

	query := `
		SELECT *
		FROM blockbundles b1
		WHERE added = false AND block_number = (
			SELECT COALESCE(MIN(block_number), 0)
			FROM blockbundles b2
			WHERE b2.added = false
		)
		AND failed_retry_count < $1
		ORDER BY inserted_at ASC
	`

	err := s.db.Select(&blockBundles, query, maxRetryCount)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch upcoming bundles: %v", err)
	}

	return blockBundles, nil
}

func (s *BundleService) GetPendingBundlesByTimestamp() ([]bundleTypes.BuilderBundleEntry, error) {
	var blockBundles []bundleTypes.BuilderBundleEntry

	query := `
		SELECT *
		FROM blockbundles b1
		WHERE added = false AND min_timestamp = (
			SELECT COALESCE(MIN(min_timestamp), 1)
			FROM blockbundles b2
			WHERE b2.added = false
		)
		AND failed_retry_count < $1
		ORDER BY inserted_at ASC
	`

	err := s.db.Select(&blockBundles, query, maxRetryCount)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch upcoming bundles: %v", err)
	}

	return blockBundles, nil
}

func (s *BundleService) GetNextBundleBlockNumber(blockNumber uint64) (uint64, error) {
	var nextBlockNumber int64
	err := s.db.Get(&nextBlockNumber, "SELECT COALESCE(MIN(block_number), 0) FROM blockbundles WHERE block_number > $1 AND added=false AND failed_retry_count < $2", int64(blockNumber), maxRetryCount)
	if err != nil {
		return 0, err
	}
	return uint64(nextBlockNumber), nil
}

func (s *BundleService) GetNextBundleTimestamp(timestamp uint64) (uint64, error) {
	var nextTimestamp int64
	err := s.db.Get(&nextTimestamp, "SELECT COALESCE(MIN(min_timestamp), 0) FROM blockbundles WHERE min_timestamp > $1 AND added=false AND failed_retry_count < $2", int64(timestamp), maxRetryCount)
	if err != nil {
		return 0, err
	}
	if nextTimestamp == 0 {
		// if we are 2s into the current slot, then no block bid can be submitted for the current slot anyway
		// so set next timestamp to the next slot
		currentSlot := (uint64(time.Now().Unix()) - s.genesisInfo.GenesisTime) / bbTypes.SLOT_DURATION
		currentSlotTime := hexutil.Uint64(s.genesisInfo.GenesisTime + currentSlot*bbTypes.SLOT_DURATION)
		currentSlotTimeCuttOff := currentSlotTime + 2
		if uint64(time.Now().Unix()) > uint64(currentSlotTimeCuttOff) {
			nextTimestamp = int64(uint64(currentSlotTime) + bbTypes.SLOT_DURATION)
		} else {
			// if we are not 2s into the current slot, then we can still submit a block bid for the current slot, check for new bundles the next second
			nextTimestamp = time.Now().Unix() + 1
		}
	}
	return uint64(nextTimestamp), nil
}

func (s *BundleService) UpdateBundleStatus(bundleHash string, added bool, errorMsg string) error {
	tx := s.db.MustBegin()
	_, err := tx.Exec("UPDATE blockbundles SET added=$1, error=$2, error_message=$3 WHERE bundle_hash=$4 AND added=false", added, (errorMsg != ""), errorMsg, bundleHash)
	if err != nil {
		return err
	}
	err = tx.Commit()
	if err != nil {
		return err
	}
	return nil
}

func (s *BundleService) RemoveBundleById(bundleID string) error {
	tx := s.db.MustBegin()
	_, err := tx.Exec("DELETE FROM blockbundles WHERE id=$1", bundleID)
	if err != nil {
		return err
	}
	err = tx.Commit()
	if err != nil {
		return err
	}
	return nil
}

func (s *BundleService) RemoveBundleByHash(bundleHash string) error {
	tx := s.db.MustBegin()
	_, err := tx.Exec("DELETE FROM blockbundles WHERE bundle_hash=$1", bundleHash)
	if err != nil {
		return err
	}
	err = tx.Commit()
	if err != nil {
		return err
	}
	return nil
}

func (s *BundleService) UpdateBundle(blockbundle *bundleTypes.BuilderBundleEntry) error {
	tx := s.db.MustBegin()
	_, err := tx.NamedExec(`UPDATE blockbundles
		SET
			bundle_hash=:bundle_hash,
			txs=:txs,
			block_number=:block_number,
			min_timestamp=:min_timestamp,
			max_timestamp=:max_timestamp,
			reverting_tx_hashes=:reverting_tx_hashes,
			builder_pubkey=:builder_pubkey,
			builder_signature=:builder_signature,
			bundle_transaction_count=:bundle_transaction_count,
			bundle_total_gas=:bundle_total_gas,
			added=:added,
			error=:error,
			inserted_at=:inserted_at,
			failed_retry_count=:failed_retry_count
		WHERE id=:id`, blockbundle)
	if err != nil {
		return err
	}
	err = tx.Commit()
	if err != nil {
		return err
	}
	return nil
}

func (s *BundleService) GetBundle(bundleID string) (bundleTypes.BuilderBundleEntry, error) {
	var blockBundle bundleTypes.BuilderBundleEntry
	err := s.db.Get(&blockBundle, "SELECT * FROM blockbundles WHERE id=$1", bundleID)
	if err != nil {
		return blockBundle, err
	}
	return blockBundle, nil
}

func (s *BundleService) AddSentBundle(bundleHash string, blockHash string, slot uint64) error {
	tx := s.db.MustBegin()
	_, err := tx.Exec("INSERT INTO sentbundles (bundle_hash, block_hash, slot) VALUES ($1, $2, $3)", bundleHash, blockHash, slot)
	if err != nil {
		return err
	}
	err = tx.Commit()
	if err != nil {
		return err
	}
	return nil
}

func (s *BundleService) GetSentBundleHashesForSlot(slot uint64) ([]string, error) {
	var bundleHashes []string
	err := s.db.Select(&bundleHashes, "SELECT bundle_hash FROM sentbundles WHERE slot=$1", slot)
	if err != nil {
		return nil, err
	}
	return bundleHashes, nil
}

func (s *BundleService) GetSentBundleHashesForBlock(blockHash string) ([]string, error) {
	var bundleHashes []string
	err := s.db.Select(&bundleHashes, "SELECT bundle_hash FROM sentbundles WHERE block_hash=$1", blockHash)
	if err != nil {
		return nil, err
	}
	return bundleHashes, nil
}

func (s *BundleService) RemoveSentBundlesBehindSlot(slot uint64) error {
	tx := s.db.MustBegin()
	_, err := tx.Exec("DELETE FROM sentbundles WHERE slot<$1", slot)
	if err != nil {
		return err
	}
	err = tx.Commit()
	if err != nil {
		return err
	}
	return nil
}
