package database

import "github.com/ethereum/go-ethereum/log"

func (s *DatabaseService) InsertBeaconBlock(signedBeaconBlock SignedBeaconBlockSubmissionEntry, blockHash string) error {
	// Find the block bid for the given block hash
	var blockBid BuilderBlockBidEntryLoader
	err := s.DB.Get(&blockBid, `SELECT * FROM blockbid WHERE block_hash = ?`, blockHash)
	if err != nil {
		log.Debug("Inserting beacon block Error getting block bid", "err", err)
		return err
	}

	signedBeaconBlock.BidId = blockBid.ID

	// Check if the block has already been submitted to the chain
	var existingBlock SignedBeaconBlockSubmissionEntry
	err = s.DB.Get(&existingBlock, `SELECT * FROM beaconblock WHERE bid_id = ?`, blockBid.ID)
	if err == nil {
		if existingBlock.SubmittedToChain {
			log.Debug("Beacon block already submitted to chain", "bid_id", blockBid.ID)
			return nil
		}

		// The block has already been inserted into the database, so just update the fields
		_, err = s.DB.NamedExec(`UPDATE beaconblock SET
			signed_beacon_block = :signed_beacon_block,
			signature = :signature,
			submitted_to_chain = :submitted_to_chain,
			submission_error = :submission_error,
			inserted_at = :inserted_at
			WHERE bid_id = :bid_id`, signedBeaconBlock)
		if err != nil {
			log.Debug("Inserting beacon block Error updating beacon block", "err", err)
			return err
		}

	}

	// Add the signed beacon block to the database with block bid id
	tx := s.DB.MustBegin()
	_, err = tx.NamedExec(`INSERT INTO beaconblock
		(bid_id, signed_beacon_block, signature, submitted_to_chain, submission_error, inserted_at)
		VALUES (:bid_id, :signed_beacon_block, :signature, :submitted_to_chain, :submission_error, :inserted_at)`, signedBeaconBlock)
	if err != nil {
		log.Debug("Inserting beacon block Error inserting beacon block", "err", err)
		return err
	}
	err = tx.Commit()
	if err != nil {
		log.Debug("Inserting beacon block Error committing transaction", "err", err)
		return err
	}
	return nil
}

// Bids successfully submitted to the chain are beacon blocks that have been submitted to the chain. Do so by joining the two tables bid and beacon block and counting the number of rows where submitted_to_chain is true.
func (s *DatabaseService) CountTotalBlockBidsSubmittedToChain() (uint64, error) {
	var count uint64
	err := s.DB.Get(&count, `SELECT COUNT(*) FROM blockbid JOIN beaconblock ON blockbid.id = beaconblock.bid_id WHERE beaconblock.submitted_to_chain = 1`)
	if err != nil {
		return 0, err
	}
	return count, nil
}
