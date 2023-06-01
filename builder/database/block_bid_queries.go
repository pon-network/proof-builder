package database

import (
	"time"
	"github.com/ethereum/go-ethereum/log"
)

func (s *DatabaseService) InsertBlockBid(blockBid BuilderBlockBidEntry) error {
	tx := s.DB.MustBegin()
	_, err := tx.NamedExec(`INSERT INTO blockbid 
		(inserted_at, slot, builder_pubkey, proposer_pubkey, fee_recipient, builder_wallet_address, signature, gas_used, gas_limit, mev, payout_pool_tx, payout_pool_gas_fee, rpbs,
			payout_pool_address, priority_transactions_count, transactions_count, block_hash, parent_hash, block_number, value) 
		VALUES (:inserted_at, :slot, :builder_pubkey, :proposer_pubkey, :fee_recipient, :builder_wallet_address, :signature, :gas_used, :gas_limit, :mev, :payout_pool_tx, :payout_pool_gas_fee, :rpbs,
			:payout_pool_address, :priority_transactions_count, :transactions_count, :block_hash, :parent_hash, :block_number, :value)`, blockBid)
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

func (s *DatabaseService) GetBlockBid(id uint64) (BuilderBlockBidEntry, error) {
	bid := BuilderBlockBidEntry{}
	err := s.DB.Get(&bid, `SELECT * FROM blockbid WHERE id = $1`, id)
	if err != nil {
		return bid, err
	}
	return bid, err
}

func (s *DatabaseService) GetBlockBids() ([]BuilderBlockBidEntry, error) {
	bids := []BuilderBlockBidEntry{}
	err := s.DB.Select(&bids, `SELECT * FROM blockbid ORDER BY inserted_at DESC`)
	if err != nil {
		return bids, err
	}
	return bids, err
}

func (s *DatabaseService) GetBlockBidsPaginated(page int, pageSize int) ([]BuilderBlockBidEntry, error) {
	bids := []BuilderBlockBidEntry{}
	err := s.DB.Select(&bids, `SELECT * FROM blockbid ORDER BY inserted_at DESC LIMIT $1 OFFSET $2`, pageSize, page*pageSize)
	if err != nil {
		return bids, err
	}
	return bids, err
}

func (s *DatabaseService) GetBlockBidsBySlot(slot uint64) ([]BuilderBlockBidEntry, error) {
	bids := []BuilderBlockBidEntry{}
	err := s.DB.Select(&bids, `SELECT * FROM blockbid WHERE slot = $1`, slot)
	if err != nil {
		return bids, err
	}
	return bids, err
}

func (s *DatabaseService) GetBlockBidsInTimeRange(from, to uint64) ([]BuilderBlockBidEntry, error) {
	bids := []BuilderBlockBidEntry{}
	err := s.DB.Select(&bids, `SELECT * FROM blockbid WHERE inserted_at BETWEEN $1 AND $2`, from, to)
	if err != nil {
		return bids, err
	}
	return bids, err
}

func (s *DatabaseService) CountBlockBidsInTimeRange(from, to uint64) (uint64, error) {
	var count uint64
	err := s.DB.Get(&count, `SELECT COUNT(*) FROM blockbid WHERE inserted_at BETWEEN $1 AND $2`, from, to)
	if err != nil {
		return count, err
	}
	return count, err
}

func (s *DatabaseService) GetBlockBidsToFeeRecipient(feeRecipient string) ([]BuilderBlockBidEntry, error) {
	bids := []BuilderBlockBidEntry{}
	err := s.DB.Select(&bids, `SELECT * FROM blockbid WHERE fee_recipient = $1`, feeRecipient)
	if err != nil {
		return bids, err
	}
	return bids, err
}

func (s *DatabaseService) GetBlockBidsToPayoutPool(payoutPoolAddress string) ([]BuilderBlockBidEntry, error) {
	bids := []BuilderBlockBidEntry{}
	err := s.DB.Select(&bids, `SELECT * FROM blockbid WHERE payout_pool_address = $1`, payoutPoolAddress)
	if err != nil {
		return bids, err
	}
	return bids, err
}

func (s *DatabaseService) GetBlockBidsByBuilderWallet(builderWalletAddress string) ([]BuilderBlockBidEntry, error) {
	bids := []BuilderBlockBidEntry{}
	err := s.DB.Select(&bids, `SELECT * FROM blockbid WHERE builder_wallet_address = $1`, builderWalletAddress)
	if err != nil {
		return bids, err
	}
	return bids, err
}

func (s *DatabaseService) CountTotalBlockBids() (uint64, error) {
	var count uint64
	err := s.DB.Get(&count, `SELECT COUNT(*) FROM blockbid`)
	if err != nil {
		return count, err
	}
	return count, err
}

func (s *DatabaseService) CountTotalBlockBidsToFeeRecipient(feeRecipient string) (uint64, error) {
	var count uint64
	err := s.DB.Get(&count, `SELECT COUNT(*) FROM blockbid WHERE fee_recipient = $1`, feeRecipient)
	if err != nil {
		return count, err
	}
	return count, err
}

func (s *DatabaseService) CountTotalBlockBidsToPayoutPool(payoutPoolAddress string) (uint64, error) {
	var count uint64
	err := s.DB.Get(&count, `SELECT COUNT(*) FROM blockbid WHERE payout_pool_address = $1`, payoutPoolAddress)
	if err != nil {
		return count, err
	}
	return count, err
}

func (s *DatabaseService) CountTotalBlockBidsByBuilderWallet(builderWalletAddress string) (uint64, error) {
	var count uint64
	err := s.DB.Get(&count, `SELECT COUNT(*) FROM blockbid WHERE builder_wallet_address = $1`, builderWalletAddress)
	if err != nil {
		return count, err
	}
	return count, err
}

func (s *DatabaseService) ComputeTotalBlockBidsBidAmount() (uint64, error) {
	var amount uint64
	err := s.DB.Get(&amount, `SELECT SUM(value) FROM blockbid`)
	if err != nil {
		return amount, err
	}
	return amount, err
}

func (s *DatabaseService) ComputeTotalBlockBidsBidAmountInTimeRange(from, to uint64) (uint64, error) {
	var amount uint64
	err := s.DB.Get(&amount, `SELECT SUM(value) FROM blockbid WHERE inserted_at BETWEEN $1 AND $2`, from, to)
	if err != nil {
		return amount, err
	}
	return amount, err
}

func (s *DatabaseService) ComputeTotalBlockBidsBidAmountToFeeRecipient(feeRecipient string) (uint64, error) {
	var amount uint64
	err := s.DB.Get(&amount, `SELECT SUM(value) FROM blockbid WHERE fee_recipient = $1`, feeRecipient)
	if err != nil {
		return amount, err
	}
	return amount, err
}

func (s *DatabaseService) ComputeTotalBlockBidsBidAmountToFeeRecipientInTimeRange(feeRecipient string, from, to uint64) (uint64, error) {
	var amount uint64
	err := s.DB.Get(&amount, `SELECT SUM(value) FROM blockbid WHERE fee_recipient = $1 AND inserted_at BETWEEN $2 AND $3`, feeRecipient, from, to)
	if err != nil {
		return amount, err
	}
	return amount, err
}

func (s *DatabaseService) ComputeTotalBlockBidsBidAmountToPayoutPool(payoutPoolAddress string) (uint64, error) {
	var amount uint64
	err := s.DB.Get(&amount, `SELECT SUM(value) FROM blockbid WHERE payout_pool_address = $1`, payoutPoolAddress)
	if err != nil {
		return amount, err
	}
	return amount, err
}

func (s *DatabaseService) ComputeTotalBlockBidsBidAmountToPayoutPoolInTimeRange(payoutPoolAddress string, from, to uint64) (uint64, error) {
	var amount uint64
	err := s.DB.Get(&amount, `SELECT SUM(value) FROM blockbid WHERE payout_pool_address = $1 AND inserted_at BETWEEN $2 AND $3`, payoutPoolAddress, from, to)
	if err != nil {
		return amount, err
	}
	return amount, err
}

func (s *DatabaseService) ComputeTotalBlockBidsBidAmountByBuilderWallet(builderWalletAddress string) (uint64, error) {
	var amount uint64
	err := s.DB.Get(&amount, `SELECT SUM(value) FROM blockbid WHERE builder_wallet_address = $1`, builderWalletAddress)
	if err != nil {
		return amount, err
	}
	return amount, err
}

func (s *DatabaseService) ComputeTotalBlockBidsBidAmountByBuilderWalletInTimeRange(builderWalletAddress string, from, to uint64) (uint64, error) {
	var amount uint64
	err := s.DB.Get(&amount, `SELECT SUM(value) FROM blockbid WHERE builder_wallet_address = $1 AND inserted_at BETWEEN $2 AND $3`, builderWalletAddress, from, to)
	if err != nil {
		return amount, err
	}
	return amount, err
}

func (s *DatabaseService) ComputeTotalBlockBidsMEV() (uint64, error) {
	var amount uint64
	err := s.DB.Get(&amount, `SELECT SUM(mev) FROM blockbid`)
	if err != nil {
		return amount, err
	}
	return amount, err
}

func (s *DatabaseService) ComputeTotalBlockBidsMEVInTimeRange(from, to uint64) (uint64, error) {
	var amount uint64
	err := s.DB.Get(&amount, `SELECT SUM(mev) FROM blockbid WHERE inserted_at BETWEEN $1 AND $2`, from, to)
	if err != nil {
		return amount, err
	}
	return amount, err
}

func (s *DatabaseService) ComputeTotalBlockBidsMEVToFeeRecipient(feeRecipient string) (uint64, error) {
	var amount uint64
	err := s.DB.Get(&amount, `SELECT SUM(mev) FROM blockbid WHERE fee_recipient = $1`, feeRecipient)
	if err != nil {
		return amount, err
	}
	return amount, err
}

func (s *DatabaseService) ComputeTotalBlockBidsMEVToFeeRecipientInTimeRange(feeRecipient string, from, to uint64) (uint64, error) {
	var amount uint64
	err := s.DB.Get(&amount, `SELECT SUM(mev) FROM blockbid WHERE fee_recipient = $1 AND inserted_at BETWEEN $2 AND $3`, feeRecipient, from, to)
	if err != nil {
		return amount, err
	}
	return amount, err
}

func (s *DatabaseService) ComputeTotalBlockBidsMEVToPayoutPool(payoutPoolAddress string) (uint64, error) {
	var amount uint64
	err := s.DB.Get(&amount, `SELECT SUM(mev) FROM blockbid WHERE payout_pool_address = $1`, payoutPoolAddress)
	if err != nil {
		return amount, err
	}
	return amount, err
}

func (s *DatabaseService) ComputeTotalBlockBidsMEVToPayoutPoolInTimeRange(payoutPoolAddress string, from, to uint64) (uint64, error) {
	var amount uint64
	err := s.DB.Get(&amount, `SELECT SUM(mev) FROM blockbid WHERE payout_pool_address = $1 AND inserted_at BETWEEN $2 AND $3`, payoutPoolAddress, from, to)
	if err != nil {
		return amount, err
	}
	return amount, err
}

func (s *DatabaseService) ComputeTotalBlockBidsMEVByBuilderWallet(builderWalletAddress string) (uint64, error) {
	var amount uint64
	err := s.DB.Get(&amount, `SELECT SUM(mev) FROM blockbid WHERE builder_wallet_address = $1`, builderWalletAddress)
	if err != nil {
		return amount, err
	}
	return amount, err
}

func (s *DatabaseService) ComputeTotalBlockBidsMEVByBuilderWalletInTimeRange(builderWalletAddress string, from, to uint64) (uint64, error) {
	var amount uint64
	err := s.DB.Get(&amount, `SELECT SUM(mev) FROM blockbid WHERE builder_wallet_address = $1 AND inserted_at BETWEEN $2 AND $3`, builderWalletAddress, from, to)
	if err != nil {
		return amount, err
	}
	return amount, err
}

func (s *DatabaseService) ComputeAverageBidAmountInTimeRange(from, to uint64) (uint64, error) {
	var amount uint64
	err := s.DB.Get(&amount, `SELECT AVG(value) FROM blockbid WHERE inserted_at BETWEEN $1 AND $2`, from, to)
	if err != nil {
		return amount, err
	}
	return amount, err
}

func (s *DatabaseService) ComputeAverageMEVInTimeRange(from, to uint64) (uint64, error) {
	var amount uint64
	err := s.DB.Get(&amount, `SELECT AVG(mev) FROM blockbid WHERE inserted_at BETWEEN $1 AND $2`, from, to)
	if err != nil {
		return amount, err
	}
	return amount, err
}

func (s *DatabaseService) ComputeAverageBidAmountGroupByMonth(from time.Time, to time.Time, dataLabels []string) ([]float64, error) {

	amounts := make(map[string]float64)

	rows, err := s.DB.Query(`
		SELECT AVG(value) as avg_value, strftime('%Y-%m', inserted_at) as month
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

func (s *DatabaseService) ComputeAverageMEVGroupByMonth(from time.Time, to time.Time, dataLabels []string) ([]float64, error) {

	amounts := make(map[string]float64)

	rows, err := s.DB.Query(`
		SELECT AVG(mev) as avg_mev, strftime('%Y-%m', inserted_at) as month
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
		var avgMEV float64
		var month string
		if err := rows.Scan(&avgMEV, &month); err != nil {
			return nil, err
		}
		amounts[month] = avgMEV
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	var AvgMEVs []float64
	for i := 0; i < len(dataLabels); i++ {
		month := from.AddDate(0, i, 0).Format("2006-01")
		if value, ok := amounts[month]; ok {
			AvgMEVs = append(AvgMEVs, value)
		} else {
			AvgMEVs = append(AvgMEVs, 0)
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
