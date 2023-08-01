package builderRPC

import (
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/core/types"
)

func (builderRPC *BuilderRPCService) addPrivateTransactions(transactions []*types.Transaction) ([]string, error) {
	txPool := builderRPC.eth.GetTxPool()
	if txPool == nil {
		return nil, fmt.Errorf("tx pool is not available")
	}

	var addedTransactions []*types.Transaction
	var hashes []string

	var errInAddition error

	for _, tx := range transactions {
		
		err := txPool.AddLocal(tx)
		if err != nil {
			log.Error("could not add private tx", "err", err)
			errInAddition = err
			break
		}

		addedTransactions = append(addedTransactions, tx)
		hashes = append(hashes, tx.Hash().String())
	}

	if errInAddition != nil {
		for _, tx := range addedTransactions {
			txPool.RemoveTx(tx.Hash(), true)
		}
		return nil, errInAddition
	}

	for _, tx := range addedTransactions {
		go builderRPC.insertRPCTransaction(tx.Hash().String())
	}

	return hashes, nil

}

func (builderRPC *BuilderRPCService) cancelPrivateTransactions(hashes []common.Hash) error {
	txPool := builderRPC.eth.GetTxPool()
	if txPool == nil {
		return fmt.Errorf("tx pool is not available")
	}

	var foundTransactions []*types.Transaction

	// First check is all transactions are in the pool
	for _, hash := range hashes {
		tx := txPool.Get(hash)
		if tx == nil {
			return fmt.Errorf("transaction %s is not in the pool", hash.String())
		}
		foundTransactions = append(foundTransactions, tx)
	}

	var errorInRemoval error
	var removedTransactions []*types.Transaction

	for _, tx := range foundTransactions {
		removed := txPool.RemoveTx(tx.Hash(), true)
		if removed == 0 {
			log.Error("could not remove private tx", "tx", tx.Hash().String())
			errorInRemoval = fmt.Errorf("could not remove private tx: %s", tx.Hash().String())
			break
		}
		removedTransactions = append(removedTransactions, tx)
	}

	if errorInRemoval != nil {
		for _, tx := range removedTransactions {
			txPool.AddLocal(tx)
		}
		return errorInRemoval
	}

	for _, tx := range removedTransactions {
		go builderRPC.deleteRPCtransaction(tx.Hash().String())
	}

	return nil
}