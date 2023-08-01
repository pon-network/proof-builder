package database

func (s *DatabaseService) InsertBeaconBlock(signedBeaconBlock SignedBeaconBlockSubmissionEntry, blockHash string) error {
	// Find the block bid for the given block hash
	var blockBid BuilderBlockBidEntry
	err := s.DB.Get(&blockBid, `SELECT * FROM blockbid WHERE block_hash = ?`, blockHash)
	if err != nil {
		return err
	}

	signedBeaconBlock.BidId = blockBid.ID

	// Add the signed beacon block to the database with block bid id
	tx := s.DB.MustBegin()
	_, err = tx.NamedExec(`INSERT INTO beaconblock
		(bid_id, signed_beacon_block, signature, submitted_to_chain, submission_error, inserted_at)
		VALUES (:bid_id, :signed_beacon_block, :signature, :submitted_to_chain, :submission_error, :inserted_at)`, signedBeaconBlock)
	if err != nil {
		return err
	}
	err = tx.Commit()
	if err != nil {
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
