package database

import (
	"fmt"
	bbTypes "github.com/ethereum/go-ethereum/builder/types"
	"math/big"
	"time"
)

func (s *DatabaseService) InsertBlindedBeaconBlock(signedBlindedBeaconBlock SignedBlindedBeaconBlockEntry, blockHash string) error {
	// Find the block bid for the given block hash
	var blockBid BuilderBlockBidEntryLoader
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
	var loadedBlockBids []BuilderBlockBidEntryLoader
	err := s.DB.Select(&loadedBlockBids, `
		SELECT blockbid.*
		FROM blockbid
		JOIN blindedbeaconblock ON blockbid.id = blindedbeaconblock.bid_id
		ORDER BY blockbid.inserted_at DESC
		LIMIT $1 OFFSET $2
	`, pageSize, page*pageSize)
	if err != nil {
		return nil, err
	}

	blockBids := make([]BuilderBlockBidEntry, len(loadedBlockBids))
	for i, loadedBlockBid := range loadedBlockBids {
		bidEntry, err := loadedBlockBid.ToBuilderBlockBidEntry()
		if err != nil {
			return nil, err
		}
		blockBids[i] = bidEntry
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

func (s *DatabaseService) ComputeTotalBlockBidsWonBidAmount() (*big.Int, error) {
	var total big.Int
	var totalString string
	err := s.DB.Get(&totalString, `SELECT COALESCE(TOTAL(value), 0) FROM blockbid JOIN blindedbeaconblock ON blockbid.id = blindedbeaconblock.bid_id`)
	if err != nil {
		return big.NewInt(0), err
	}
	totalString, err = bbTypes.ConvertScientificToDecimal(totalString)
	if err != nil {
		return big.NewInt(0), err
	}
	_, ok := total.SetString(totalString, 10)
	if !ok {
		return big.NewInt(0), fmt.Errorf("could not convert string to big.Int")
	}
	return &total, nil
}

func (s *DatabaseService) ComputeTotalBlockBidsWonMEV() (*big.Int, error) {
	var total big.Int
	var totalString string
	err := s.DB.Get(&totalString, `SELECT COALESCE(TOTAL(mev), 0) FROM blockbid JOIN blindedbeaconblock ON blockbid.id = blindedbeaconblock.bid_id`)
	if err != nil {
		return big.NewInt(0), err
	}

	totalString, err = bbTypes.ConvertScientificToDecimal(totalString)
	if err != nil {
		return big.NewInt(0), err
	}

	_, ok := total.SetString(totalString, 10)
	if !ok {
		return big.NewInt(0), fmt.Errorf("could not convert string to big.Int")
	}
	return &total, nil
}

func (s *DatabaseService) ComputeTotalBlockBidsWonBidAmountGroupByMonth(from time.Time, to time.Time, dataLabels []string) ([]*big.Int, error) {

	amounts := make(map[string]*big.Int)

	rows, err := s.DB.Query(`
		SELECT COALESCE(TOTAL(value), 0) as total_value, strftime('%Y-%m', bb.inserted_at) as month
		FROM blockbid bb
		JOIN blindedbeaconblock bbb ON bb.id = bbb.bid_id
		WHERE bb.inserted_at BETWEEN $1 AND $2
		GROUP BY month
		ORDER BY month DESC
	`, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var totalValue big.Int
		var totalValueString string
		var month string
		if err := rows.Scan(&totalValueString, &month); err != nil {
			return nil, err
		}
		totalValueString, err = bbTypes.ConvertScientificToDecimal(totalValueString)
		if err != nil {
			return nil, err
		}

		_, ok := totalValue.SetString(totalValueString, 10)
		if !ok {
			return nil, fmt.Errorf("could not convert string to big.Int")
		}
		amounts[month] = &totalValue
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	var amountValues []*big.Int
	for i := 0; i < len(dataLabels); i++ {
		month := from.AddDate(0, i, 0).Format("2006-01")
		if value, ok := amounts[month]; ok {
			amountValues = append(amountValues, value)
		} else {
			amountValues = append(amountValues, big.NewInt(0))
		}
	}

	return amountValues, nil
}

func (s *DatabaseService) ComputeTotalBlockBidsWonMEVGroupByMonth(from time.Time, to time.Time, dataLabels []string) ([]*big.Int, error) {

	amounts := make(map[string]*big.Int)

	rows, err := s.DB.Query(`
		SELECT COALESCE(TOTAL(mev), 0) as total_value, strftime('%Y-%m', bb.inserted_at) as month
		FROM blockbid bb
		JOIN blindedbeaconblock bbb ON bb.id = bbb.bid_id
		WHERE bb.inserted_at BETWEEN $1 AND $2
		GROUP BY month
		ORDER BY month DESC
	`, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var totalMEV big.Int
		var totalMEVString string
		var month string
		if err := rows.Scan(&totalMEVString, &month); err != nil {
			return nil, err
		}

		totalMEVString, err = bbTypes.ConvertScientificToDecimal(totalMEVString)
		if err != nil {
			return nil, err
		}

		_, ok := totalMEV.SetString(totalMEVString, 10)
		if !ok {
			return nil, fmt.Errorf("could not convert string to big.Int")
		}
		amounts[month] = &totalMEV
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	var TotalMEVs []*big.Int
	for i := 0; i < len(dataLabels); i++ {
		month := from.AddDate(0, i, 0).Format("2006-01")
		if value, ok := amounts[month]; ok {
			TotalMEVs = append(TotalMEVs, value)
		} else {
			TotalMEVs = append(TotalMEVs, big.NewInt(0))
		}
	}

	return TotalMEVs, nil
}

func (s *DatabaseService) CountTotalBidsWonGroupByMonth(from time.Time, to time.Time, dataLabels []string) ([]uint64, error) {

	count := make(map[string]uint64)

	rows, err := s.DB.Query(`
		SELECT COUNT(*) as bids, strftime('%Y-%m', bb.inserted_at) as month
		FROM blockbid bb
		JOIN blindedbeaconblock bbb ON bb.id = bbb.bid_id
		WHERE bb.inserted_at BETWEEN $1 AND $2
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
