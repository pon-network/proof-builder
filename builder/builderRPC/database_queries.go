package builderRPC

import (
	"github.com/ethereum/go-ethereum/common"
)

func (builderRPC *BuilderRPCService) insertRPCTransaction(txHash string) error {

	tx := builderRPC.db.MustBegin()
	_, err := tx.NamedExec(`INSERT INTO rpctxs
		(tx_hash) VALUES (:tx_hash)`, map[string]interface{}{
			"tx_hash": txHash,
		})
	if err != nil {
		return err
	}
	err = tx.Commit()
	if err != nil {
		return err
	}
	return nil
}

func (builderRPC *BuilderRPCService) deleteRPCtransaction(txHash string) error {
	tx := builderRPC.db.MustBegin()
	_, err := tx.Exec("DELETE FROM rpctxs WHERE tx_hash = $1", txHash)
	if err != nil {
		return err
	}
	err = tx.Commit()
	if err != nil {
		return err
	}
	return nil
}

func (builderRPC *BuilderRPCService) getTxsOlderThanTimestamp(timestamp int64) ([]common.Hash, error) {
	var txs []string
	var hashes []common.Hash
	err := builderRPC.db.Select(&txs, "SELECT tx_hash FROM rpctxs WHERE inserted_at < $1", timestamp)
	if err != nil {
		return hashes, err
	}

	for _, tx := range txs {
		hashes = append(hashes, common.HexToHash(tx))
	}
	return hashes, nil
}

func (builderRPC *BuilderRPCService) deleteTxsOlderThanTimestamp(timestamp int64) error {
	tx := builderRPC.db.MustBegin()
	_, err := tx.Exec("DELETE FROM rpctxs WHERE inserted_at < $1", timestamp)
	if err != nil {
		return err
	}
	err = tx.Commit()
	if err != nil {
		return err
	}
	return nil
}