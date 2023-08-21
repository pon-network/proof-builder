package database

import (
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/log"
)

func (s *DatabaseService) InsertBlockBid(blockBid BuilderBlockBidEntry) error {
	tx := s.DB.MustBegin()

	blockBidEntryLoader := blockBid.ToBuilderBlockBidEntryLoarder()

	_, err := tx.NamedExec(`
		INSERT INTO blockbid 
		(inserted_at, slot, builder_pubkey, proposer_pubkey, fee_recipient, builder_wallet_address, signature, gas_used, gas_limit, mev, payout_pool_tx, payout_pool_gas_fee, rpbs,
			payout_pool_address, priority_transactions_count, transactions_count, block_hash, parent_hash, block_number, relay_response, value)
		VALUES (:inserted_at, :slot, :builder_pubkey, :proposer_pubkey, :fee_recipient, :builder_wallet_address, :signature, :gas_used, :gas_limit, :mev, :payout_pool_tx, :payout_pool_gas_fee, :rpbs,
			:payout_pool_address, :priority_transactions_count, :transactions_count, :block_hash, :parent_hash, :block_number, :relay_response, :value)
	`, blockBidEntryLoader)

	if err != nil {
		log.Error("Error saving block bid to database", "err", err)
		return err
	}

	err = tx.Commit()
	if err != nil {
		log.Error("Error saving block bid to database", "err", err)
		return err
	}

	log.Info("Block bid saved to database", "id", blockBid.ID, "slot", blockBid.Slot, "builder", blockBid.BuilderPubkey, "proposer", blockBid.ProposerPubkey, "feeRecipient", blockBid.FeeRecipient, "gasUsed", blockBid.GasUsed, "gasLimit", blockBid.GasLimit, "mev", blockBid.MEV, "payoutPoolTx", blockBid.PayoutPoolTx, "priorityTransactionsCount", blockBid.PriorityTransactionsCount, "transactionsCount", blockBid.TransactionsCount, "blockHash", blockBid.BlockHash, "value", blockBid.Value)

	return nil
}

func (s *DatabaseService) GetBlockBids() ([]BuilderBlockBidEntry, error) {
	loadedBids := []BuilderBlockBidEntryLoader{}
	err := s.DB.Select(&loadedBids, `SELECT * FROM blockbid ORDER BY inserted_at DESC`)
	if err != nil {
		return nil, err
	}

	bids := make([]BuilderBlockBidEntry, len(loadedBids))

	for i, loadedBid := range loadedBids {
		bidEntry, err := loadedBid.ToBuilderBlockBidEntry()
		if err != nil {
			return nil, err
		}
		bids[i] = bidEntry
	}

	return bids, nil
}

func (s *DatabaseService) GetBlockBidsPaginated(page int, pageSize int) ([]BuilderBlockBidEntry, error) {
	loadedBids := []BuilderBlockBidEntryLoader{}
	err := s.DB.Select(&loadedBids, `SELECT * FROM blockbid ORDER BY inserted_at DESC LIMIT $1 OFFSET $2`, pageSize, page*pageSize)
	if err != nil {
		return nil, err
	}

	bids := make([]BuilderBlockBidEntry, len(loadedBids))

	for i, loadedBid := range loadedBids {
		bidEntry, err := loadedBid.ToBuilderBlockBidEntry()
		if err != nil {
			return nil, err
		}
		bids[i] = bidEntry
	}

	return bids, nil
}

func (s *DatabaseService) CountTotalBlockBids() (uint64, error) {
	var count uint64
	err := s.DB.Get(&count, `SELECT COUNT(*) FROM blockbid`)
	if err != nil {
		return count, err
	}
	return count, err
}

func (s *DatabaseService) ComputeAverageBidAmountGroupByMonth(from time.Time, to time.Time, dataLabels []string) ([]*big.Float, error) {

	amounts := make(map[string]*big.Float)

	rows, err := s.DB.Query(`
		SELECT COALESCE(AVG(value), 0) as avg_value, strftime('%Y-%m', inserted_at) as month
		FROM blockbid
		WHERE inserted_at BETWEEN $1 AND $2
		GROUP BY month
		ORDER BY month DESC
	`, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var avgValue big.Float
		var avgValueFloatString string
		var month string
		if err := rows.Scan(&avgValueFloatString, &month); err != nil {
			return nil, err
		}
		avgValue.SetString(avgValueFloatString)
		amounts[month] = &avgValue
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	var amountValues []*big.Float
	for i := 0; i < len(dataLabels); i++ {
		month := from.AddDate(0, i, 0).Format("2006-01")
		if value, ok := amounts[month]; ok {
			amountValues = append(amountValues, value)
		} else {
			amountValues = append(amountValues, big.NewFloat(0))
		}
	}

	return amountValues, nil
}

func (s *DatabaseService) ComputeAverageMEVGroupByMonth(from time.Time, to time.Time, dataLabels []string) ([]*big.Float, error) {

	amounts := make(map[string]*big.Float)

	rows, err := s.DB.Query(`
		SELECT COALESCE(AVG(mev), 0) as avg_mev, strftime('%Y-%m', inserted_at) as month
		FROM blockbid
		WHERE inserted_at BETWEEN $1 AND $2
		GROUP BY month
		ORDER BY month DESC
	`, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var avgMEV big.Float
		var avgMEVFloatString string
		var month string
		if err := rows.Scan(&avgMEVFloatString, &month); err != nil {
			return nil, err
		}
		avgMEV.SetString(avgMEVFloatString)
		amounts[month] = &avgMEV
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	var AvgMEVs []*big.Float
	for i := 0; i < len(dataLabels); i++ {
		month := from.AddDate(0, i, 0).Format("2006-01")
		if value, ok := amounts[month]; ok {
			AvgMEVs = append(AvgMEVs, value)
		} else {
			AvgMEVs = append(AvgMEVs, big.NewFloat(0))
		}
	}

	return AvgMEVs, nil
}

// Count total bids sent in time range group by month
func (s *DatabaseService) CountTotalBidsGroupByMonth(from time.Time, to time.Time, dataLabels []string) ([]uint64, error) {

	count := make(map[string]uint64)

	rows, err := s.DB.Query(`
		SELECT COUNT(*) as bids, strftime('%Y-%m', inserted_at) as month
		FROM blockbid
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
