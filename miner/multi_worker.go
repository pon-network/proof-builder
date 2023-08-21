package miner

import (
	"errors"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"

	bundleTypes "github.com/bsn-eng/pon-golang-types/bundles"
)

type multiWorker struct {
	workers       []*worker
	regularWorker *worker
}

func (w *multiWorker) stop() {
	for _, worker := range w.workers {
		worker.stop()
	}
}

func (w *multiWorker) start() {
	for _, worker := range w.workers {
		worker.start()
	}
}

func (w *multiWorker) close() {
	for _, worker := range w.workers {
		worker.close()
	}
}

func (w *multiWorker) isRunning() bool {
	for _, worker := range w.workers {
		if worker.isRunning() {
			return true
		}
	}
	return false
}

// pendingBlockAndReceipts returns pending block and corresponding receipts from the `regularWorker`
func (w *multiWorker) pendingBlockAndReceipts() (*types.Block, types.Receipts) {
	// return a snapshot to avoid contention on currentMu mutex
	return w.regularWorker.pendingBlockAndReceipts()
}

func (w *multiWorker) setGasCeil(ceil uint64) {
	for _, worker := range w.workers {
		worker.setGasCeil(ceil)
	}
}

func (w *multiWorker) setExtra(extra []byte) {
	for _, worker := range w.workers {
		worker.setExtra(extra)
	}
}

func (w *multiWorker) setRecommitInterval(interval time.Duration) {
	for _, worker := range w.workers {
		worker.setRecommitInterval(interval)
	}
}

func (w *multiWorker) setEtherbase(addr common.Address) {
	for _, worker := range w.workers {
		worker.setEtherbase(addr)
	}
}

func (w *multiWorker) enablePreseal() {
	for _, worker := range w.workers {
		worker.enablePreseal()
	}
}

func (w *multiWorker) disablePreseal() {
	for _, worker := range w.workers {
		worker.disablePreseal()
	}
}

type resPair struct {
	res          *types.Block
	fees         *big.Int
	bidAmount    *big.Int
	payoutTx     []byte
	triedBundles []bundleTypes.BuilderBundle
	err          error
}

func (w *multiWorker) BuildBlockWithCallback(parent common.Hash, timestamp uint64, coinbase common.Address, random common.Hash, noTxs bool, transactions [][]byte, bundles []bundleTypes.BuilderBundle, withdrawals types.Withdrawals, payoutPoolAddress common.Address, bidAmount *big.Int, gasLimit uint64, sealedBlockCallback SealedBlockCallbackFn) (chan *resPair, error) {
	allResPairs := []resPair{}

	for _, worker := range w.workers {
		res, fees, bidAmount, payoutTx, triedBundles, err := worker.getSealingBlock(
			parent,
			timestamp,
			coinbase,
			random,
			withdrawals,
			noTxs,
			transactions,
			bundles,
			payoutPoolAddress,
			bidAmount,
			gasLimit)
		allResPairs = append(allResPairs, resPair{res, fees, bidAmount, payoutTx, triedBundles, err})
	}

	if len(allResPairs) == 0 {
		return nil, errors.New("no worker could start async block construction")
	}

	resCh := make(chan *resPair)

	go func(resCh chan *resPair) {
		var chResPair *resPair = nil
		for _, chPair := range allResPairs {

			// call the callback function for each worker regardless of the result as it may be used for logging
			// since the build is async
			go sealedBlockCallback(chPair.res, chPair.fees, chPair.bidAmount, chPair.payoutTx, chPair.triedBundles, chPair.err)

			if chPair.err != nil {
				log.Error("could not generate block", "err", chPair.err)
				continue
			}
			if chResPair == nil || (chPair.res != nil && chPair.fees.Cmp(chResPair.fees) >= 0) {
				chResPair = &chPair
			}
		}
		resCh <- chResPair
	}(resCh)

	return resCh, nil
}

func newMultiWorker(config *Config, chainConfig *params.ChainConfig, engine consensus.Engine, eth Backend, mux *event.TypeMux, isLocalBlock func(header *types.Header) bool, init bool) *multiWorker {

	regularWorker := newWorker(config, chainConfig, engine, eth, mux, isLocalBlock, init)

	log.Info("creating new regular worker")
	return &multiWorker{
		regularWorker: regularWorker,
		workers:       []*worker{regularWorker},
	}
}
