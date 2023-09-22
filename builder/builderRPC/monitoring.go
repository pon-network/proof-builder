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

			deletePast := time.Now().Add(-24 * time.Hour)

			log.Info("Cleaning old rpc transactions", "timeCutoff", deletePast)

			txHashes, err := builderRPC.getTxsOlderThanTimestamp(deletePast.Unix())
			if err != nil {
				log.Error("Error getting txs older than 24 hours", "err", err)
				continue
			}

			if len(txHashes) == 0 {
				continue
			}

			removedHashes, err := builderRPC.mustCancelPrivateTransactions(txHashes)
			if err != nil {
				log.Error("Error cancelling private transactions", "err", err)
				// Error would only throw im txpool not available
				continue
			}

			log.Info("Cancelled old rpc transactions",
				"cancelled", len(removedHashes),
				"total", len(txHashes),
			)

			// Even if there are any old transactions the could not be cancelled, we still delete them from the db
			// as we cannot track and manage them anymore
			// e.g blob transactions finalized, legacy transactions synced in pending, etc...
			err = builderRPC.deleteTxsOlderThanTimestamp(deletePast.Unix())
			if err != nil {
				log.Error("Error deleting txs older than 24 hours", "err", err)
				continue
			}

		}
	}

}
