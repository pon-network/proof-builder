package builderRPC

import (
	"errors"

	"github.com/ethereum/go-ethereum/beacon/engine"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"

	bundleTypes "github.com/bsn-eng/pon-golang-types/bundles"
	rpcTypes "github.com/bsn-eng/pon-golang-types/rpc"
)

// To conform to geth rpc spec, we need to have methods that are public and can be called by the rpc server.
// These methods will then call the internal methods that are not exposed to the rpc server.


// Publicly exposed rpc methods
func (builderRPC *BuilderRPCService) SendPrivateTransaction(txs rpcTypes.JSONrpcPrivateTxs) ([]string, error) {
	// `mev_sendPrivateTransaction` is a public method that allows users to send private transactions to the tx pool.

	var txsBytes [][]byte
	for _, tx := range txs {
		txBytes, err := hexutil.Decode(tx.Tx)
		if err != nil {
			return nil, err
		}
		txsBytes = append(txsBytes, txBytes)
	}

	transactions, err := engine.DecodeTransactions(txsBytes)
	if err != nil {
		return nil, err
	}

	// Use central internal function for adding private transactions to the tx pool
	transactionHashes, err := builderRPC.addPrivateTransactions(transactions)
	if err != nil {
		return nil, err
	}

	return transactionHashes, nil
}


func (builderRPC *BuilderRPCService) SendPrivateRawTransaction(txs rpcTypes.JSONrpcPrivateRawTxs) ([]string, error) {
	// `mev_sendPrivateRawTransaction` is a public rpc method that allows users to send private raw transactions to the tx pool.

	var txsBytes [][]byte
	for _, tx := range txs {
		txBytes, err := hexutil.Decode(tx)
		if err != nil {
			return nil, err
		}
		txsBytes = append(txsBytes, txBytes)
	}

	transactions, err := engine.DecodeTransactions(txsBytes)
	if err != nil {
		return nil, err
	}

	// Use central internal function for adding private transactions to the tx pool
	transactionHashes, err := builderRPC.addPrivateTransactions(transactions)
	if err != nil {
		return nil, err
	}

	return transactionHashes, nil
}


func (builderRPC *BuilderRPCService) CancelPrivateTransaction(txs rpcTypes.JSONrpcPrivateTxHashes) ([]string, error) {
	// `mev_cancelPrivateTransaction` is a public rpc method that allows users to cancel private transactions from the tx pool.

	var txHashes []common.Hash
	var txHashesStrings []string
	for _, txObject := range txs {
		txHashes = append(txHashes, common.HexToHash(txObject.TxHash))
		txHashesStrings = append(txHashesStrings, txObject.TxHash)
	}

	err := builderRPC.cancelPrivateTransactions(txHashes)
	if err != nil {
		return nil, err
	}

	return txHashesStrings, nil

}

func (builderRPC *BuilderRPCService) SendBundle(bundleJSON rpcTypes.JSONrpcBundle) (*bundleTypes.BuilderBundleEntry, error) {
	// `mev_sendBundle` is a public rpc method that allows users to send bundles to the builder.

	if builderRPC.bundleService == nil {
		return nil, errors.New("bundle service not enabled")
	}

	// Ensure that the payload is valid
	if len(bundleJSON.Txs) == 0 {
		return nil, errors.New("bundle must contain at least one transaction")
	}
	if bundleJSON.BlockNumber == 0 && bundleJSON.MaxTimestamp == 0 {
		return nil, errors.New("bundle must contain either a block number or a max timestamp")
	}

	var txsBytes [][]byte
	for _, tx := range bundleJSON.Txs {
		txBytes, err := hexutil.Decode(tx)
		if err != nil {
			return nil, err
		}
		txsBytes = append(txsBytes, txBytes)
	}

	transactions, err := engine.DecodeTransactions(txsBytes)
	if err != nil {
		return nil, err
	}

	var revertingTxHashes []*common.Hash
	for _, txHashString := range bundleJSON.RevertingTxHashes {

		if len(txHashString) == 0 {
			continue
		}

		hash := common.HexToHash(txHashString)

		// ensure hash exists in the list of transactions
		found := false
		for _, tx := range transactions {
			if tx.Hash() == hash {
				found = true
				break
			}
		}

		if !found {
			return nil, errors.New("reverting tx hash not found in bundle")
		}

		revertingTxHashes = append(revertingTxHashes, &hash)
	}


	bundle, err := builderRPC.bundleService.AddBundle(
		transactions,
		bundleJSON.BlockNumber,
		bundleJSON.MinTimestamp,
		bundleJSON.MaxTimestamp,
		revertingTxHashes,
	)
	if err != nil {
		return nil, err
	}

	bundleEntry, err := bundleTypes.BuilderBundleToEntry(bundle)
	if err != nil {
		return nil, err
	}

	return bundleEntry, nil

}

func (builderRPC *BuilderRPCService) UpdateBundle(bundleJSON rpcTypes.JSONrpcBundle) (*bundleTypes.BuilderBundleEntry, error) {
	// `mev_updateBundle` is a public rpc method that allows users to update bundles in the builder.

	if builderRPC.bundleService == nil {
		return nil, errors.New("bundle service not enabled")
	}

	// Ensure that the payload is valid
	if len(bundleJSON.ID) == 0 {
		return nil, errors.New("bundle must contain an ID to update")
	}
	if len(bundleJSON.Txs) == 0 {
		return nil, errors.New("bundle must contain at least one transaction")
	}
	if bundleJSON.BlockNumber == 0 && bundleJSON.MaxTimestamp == 0 {
		return nil, errors.New("bundle must contain either a block number or a max timestamp")
	}

	var txsBytes [][]byte
	for _, tx := range bundleJSON.Txs {
		txBytes, err := hexutil.Decode(tx)
		if err != nil {
			return nil, err
		}
		txsBytes = append(txsBytes, txBytes)
	}

	transactions, err := engine.DecodeTransactions(txsBytes)
	if err != nil {
		return nil, err
	}

	var revertingTxHashes []*common.Hash
	for _, txHashString := range bundleJSON.RevertingTxHashes {
		hash := common.HexToHash(txHashString)
		revertingTxHashes = append(revertingTxHashes, &hash)
	}

	bundle, err := builderRPC.bundleService.ChangeBundle(
		bundleJSON.ID,
		transactions,
		bundleJSON.BlockNumber,
		bundleJSON.MinTimestamp,
		bundleJSON.MaxTimestamp,
		revertingTxHashes,
	)
	if err != nil {
		return nil, err
	}

	bundleEntry, err := bundleTypes.BuilderBundleToEntry(bundle)
	if err != nil {
		return nil, err
	}

	return bundleEntry, nil

}

func (builderRPC *BuilderRPCService) GetBundle(bundleJSON rpcTypes.JSONrpcBundle) (*bundleTypes.BuilderBundleEntry, error) {
	// `mev_getBundle` is a public rpc method that allows users to retrieve bundles from the builder.

	if builderRPC.bundleService == nil {
		return nil, errors.New("bundle service not enabled")
	}

	// Ensure that the payload is valid
	if len(bundleJSON.ID) == 0 {
		return nil, errors.New("bundle must contain an ID to retrieve")
	}

	bundle, err := builderRPC.bundleService.GetBundleByID(bundleJSON.ID)
	if err != nil {
		return nil, err
	}

	bundleEntry, err := bundleTypes.BuilderBundleToEntry(bundle)
	if err != nil {
		return nil, err
	}

	// For security and privacy, redact the transactions and reverting tx hashes from the response
	bundleEntry.Txs = ""
	bundleEntry.RevertingTxHashes = ""

	return bundleEntry, nil

}

func (builderRPC *BuilderRPCService) CancelBundle(bundleJSON rpcTypes.JSONrpcBundle) error {
	// `mev_cancelBundle` is a public rpc method that allows users to cancel bundles in the builder.

	if builderRPC.bundleService == nil {
		return errors.New("bundle service not enabled")
	}

	// Ensure that the payload is valid
	if len(bundleJSON.ID) == 0 {
		return errors.New("bundle must contain an ID to cancel")
	}

	err := builderRPC.bundleService.CancelBundle(bundleJSON.ID)
	if err != nil {
		return err
	}

	return nil

}

func (builderRPC *BuilderRPCService) BundleServiceStatus() (string, error) {
	// `mev_bundleServiceStatus` is a public rpc method that allows users to retrieve the status of the bundle service.

	if builderRPC.bundleService == nil {
		return "disabled", nil
	}

	return "enabled", nil

}
