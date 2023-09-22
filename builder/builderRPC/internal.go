package builderRPC

import (
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
)

type txAdditionResponse struct {
	Hash string `json:"hash"`
	Error error `json:"error"`
}

func (builderRPC *BuilderRPCService) addPrivateTransactions(transactions []*types.Transaction) ([]txAdditionResponse, error) {
	txPool := builderRPC.eth.GetTxPool()
	if txPool == nil {
		return nil, fmt.Errorf("tx pool is not available")
	}

	var responseList []txAdditionResponse

	// Need to perform as batch as pool effectively locks during addition
	errs := txPool.Add(transactions, true, false)

	for i, tx := range transactions {
		if errs[i] == nil {
			responseList = append(responseList, txAdditionResponse{Hash: tx.Hash().String()})
		} else {
			responseList = append(responseList, txAdditionResponse{Hash: tx.Hash().String(), Error: errs[i]})
		}
	}

	for _, tx := range responseList {
		if tx.Error == nil {
			go builderRPC.insertRPCTransaction(tx.Hash)
		}
	}

	return responseList, nil

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
		removed := txPool.RemoveTx(tx.Hash())
		if removed {
			log.Error("could not remove private tx", "tx", tx.Hash().String())
			errorInRemoval = fmt.Errorf("could not remove private tx: %s", tx.Hash().String())
			break
		}
		removedTransactions = append(removedTransactions, tx)
	}

	if errorInRemoval != nil {
		txPool.Add(removedTransactions, true, false)
		return errorInRemoval
	}

	for _, tx := range removedTransactions {
		go builderRPC.deleteRPCtransaction(tx.Hash().String())
	}

	return nil
}


func (builderRPC *BuilderRPCService) mustCancelPrivateTransactions(hashes []common.Hash) ([]common.Hash, error) {
	txPool := builderRPC.eth.GetTxPool()
	if txPool == nil {
		return nil, fmt.Errorf("tx pool is not available")
	}

	var foundTransactions []*types.Transaction

	// First check is all transactions are in the pool
	for _, hash := range hashes {
		tx := txPool.Get(hash)
		if tx == nil {
			log.Debug("transaction is not in the pool", "tx", hash.String())
			continue
		}
		foundTransactions = append(foundTransactions, tx)
	}

	var removedTransactions []*types.Transaction
	var removedHashes []common.Hash

	for _, tx := range foundTransactions {
		removed := txPool.RemoveTx(tx.Hash())
		if removed {
			log.Debug("could not remove private tx", "tx", tx.Hash().String())
			continue
		}
		removedTransactions = append(removedTransactions, tx)
		removedHashes = append(removedHashes, tx.Hash())
	}

	for _, tx := range removedTransactions {
		go builderRPC.deleteRPCtransaction(tx.Hash().String())
	}

	return removedHashes, nil
}
