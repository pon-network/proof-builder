package builderRPC

import (
	"context"
	"time"

	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/log"
)

func (builderRPC *BuilderRPCService) cleanOldTransactions(ctx context.Context) {

	var headEventCh = make(chan core.ChainHeadEvent, 1)

	subscription := builderRPC.eth.GetBlockChain().SubscribeChainHeadEvent(headEventCh)

	for {
		select {
		case <-ctx.Done():
			subscription.Unsubscribe()
			return
		// case err := <-subscription.Err():
		case <-headEventCh:

			currentTime := time.Now().Add(-24 * time.Hour)

			log.Info("Cleaning old rpc transactions", "timeCutoff", currentTime)

			txHashes, err := builderRPC.getTxsOlderThanTimestamp(currentTime.Unix())
			if err != nil {
				log.Error("Error getting txs older than 24 hours", "err", err)
				continue
			}

			if len(txHashes) == 0 {
				continue
			}

			err = builderRPC.cancelPrivateTransactions(txHashes)
			if err != nil {
				log.Error("Error cancelling private transactions", "err", err)
				continue
			}

			err = builderRPC.deleteTxsOlderThanTimestamp(currentTime.Unix())
			if err != nil {
				log.Error("Error deleting txs older than 24 hours", "err", err)
				continue
			}

		}
	}

}
