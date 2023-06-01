package database

import (
	"time"
)

func (s *DatabaseService) InsertBlindedBeaconBlock(signedBlindedBeaconBlock SignedBlindedBeaconBlockEntry, blockHash string) error {
	// Find the block bid for the given block hash
	var blockBid BuilderBlockBidEntry
	err := s.DB.Get(&blockBid, `SELECT * FROM blockbid WHERE block_hash = ?`, blockHash)
	if err != nil {
		return err
	}

	signedBlindedBeaconBlock.BidId = blockBid.ID

	// Add the signed blinded beacon block to the database with block bid id
	tx := s.DB.MustBegin()
	_, err = tx.NamedExec(`INSERT INTO blindedbeaconblock
		(bid_id, signed_blinded_beacon_block, signature, inserted_at)
		VALUES (:bid_id, :signed_blinded_beacon_block, :signature, :inserted_at)`, signedBlindedBeaconBlock)
	if err != nil {
		return err
	}
	err = tx.Commit()
	if err != nil {
		return err
	}
	return nil
}

func (s *DatabaseService) GetBlockBidsWonPaginated(page int, pageSize int) ([]BuilderBlockBidEntry, error) {
	var blockBids []BuilderBlockBidEntry
	err := s.DB.Select(&blockBids, `
		SELECT blockbid.*
		FROM blockbid
		JOIN blindedbeaconblock ON blockbid.id = blindedbeaconblock.bid_id
		ORDER BY blockbid.inserted_at DESC
		LIMIT $1 OFFSET $2
	`, pageSize, page*pageSize)
	if err != nil {
		return nil, err
	}
	return blockBids, nil
}


// Winning bids are block bids that have a corresponding signed blinded beacon block. Do so by joining the two tables and counting the number of rows.
func (s *DatabaseService) CountTotalBlockBidsWon() (uint64, error) {
	var count uint64
	err := s.DB.Get(&count, `SELECT COUNT(*) FROM blockbid JOIN blindedbeaconblock ON blockbid.id = blindedbeaconblock.bid_id`)
	if err != nil {
		return 0, err
	}
	return count, nil
}


func (s *DatabaseService) ComputeTotalBlockBidsWonBidAmount() (float64, error) {
	var count float64
	err := s.DB.Get(&count, `SELECT SUM(value) FROM blockbid JOIN blindedbeaconblock ON blockbid.id = blindedbeaconblock.bid_id`)
	if err != nil {
		return 0, err
	}
	return count, nil
}


func (s *DatabaseService) ComputeTotalBlockBidsWonMEV() (float64, error) {
	var count float64
	err := s.DB.Get(&count, `SELECT SUM(mev) FROM blockbid JOIN blindedbeaconblock ON blockbid.id = blindedbeaconblock.bid_id`)
	if err != nil {
		return 0, err
	}
	return count, nil
}

func (s *DatabaseService) ComputeTotalBlockBidsWonBidAmountGroupByMonth(from time.Time, to time.Time, dataLabels []string) ([]float64, error) {

	amounts := make(map[string]float64)

	rows, err := s.DB.Query(`
		SELECT SUM(value) as avg_value, strftime('%Y-%m', inserted_at) as month
		FROM blockbid
		JOIN blindedbeaconblock ON blockbid.id = blindedbeaconblock.bid_id
		WHERE inserted_at BETWEEN $1 AND $2
		GROUP BY month
		ORDER BY month DESC
	`, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var avgValue float64
		var month string
		if err := rows.Scan(&avgValue, &month); err != nil {
			return nil, err
		}
		amounts[month] = avgValue
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	var amountValues []float64
	for i := 0; i < len(dataLabels); i++ {
		month := from.AddDate(0, i, 0).Format("2006-01")
		if value, ok := amounts[month]; ok {
			amountValues = append(amountValues, value)
		} else {
			amountValues = append(amountValues, 0)
		}
	}

	return amountValues, nil
}

func (s *DatabaseService) ComputeTotalBlockBidsWonMEVGroupByMonth(from time.Time, to time.Time, dataLabels []string) ([]float64, error) {
	
	amounts := make(map[string]float64)

	rows, err := s.DB.Query(`
		SELECT SUM(mev) as avg_value, strftime('%Y-%m', inserted_at) as month
		FROM blockbid
		JOIN blindedbeaconblock ON blockbid.id = blindedbeaconblock.bid_id
		WHERE inserted_at BETWEEN $1 AND $2
		GROUP BY month
		ORDER BY month DESC
	`, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var totalMEV float64
		var month string
		if err := rows.Scan(&totalMEV, &month); err != nil {
			return nil, err
		}
		amounts[month] = totalMEV
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	var TotalMEVs []float64
	for i := 0; i < len(dataLabels); i++ {
		month := from.AddDate(0, i, 0).Format("2006-01")
		if value, ok := amounts[month]; ok {
			TotalMEVs = append(TotalMEVs, value)
		} else {
			TotalMEVs = append(TotalMEVs, 0)
		}
	}

	return TotalMEVs, nil
}

func (s *DatabaseService) CountTotalBidsWonGroupByMonth(from time.Time, to time.Time, dataLabels []string) ([]uint64, error) {
	
	count := make(map[string]uint64)

	rows, err := s.DB.Query(`
		SELECT COUNT(*) as bids, strftime('%Y-%m', inserted_at) as month
		FROM blockbid
		JOIN blindedbeaconblock ON blockbid.id = blindedbeaconblock.bid_id
		WHERE inserted_at BETWEEN $1 AND $2
		GROUP BY month
		ORDER BY month DESC
	`, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var bidCount uint64
		var month string
		if err := rows.Scan(&bidCount, &month); err != nil {
			return nil, err
		}
		count[month] = bidCount
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	var countValues []uint64
	for i := 0; i < len(dataLabels); i++ {
		month := from.AddDate(0, i, 0).Format("2006-01")
		if value, ok := count[month]; ok {
			countValues = append(countValues, value)
		} else {
			countValues = append(countValues, 0)
		}
	}

	return countValues, nil
}
