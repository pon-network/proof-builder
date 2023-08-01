package ethService

import (
	"errors"
	"time"

	builderTypes "github.com/bsn-eng/pon-golang-types/builder"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/txpool"
	"github.com/ethereum/go-ethereum/eth"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/miner"
)

type EthService struct {
	eth *eth.Ethereum
}

func NewEthService(eth *eth.Ethereum) *EthService {
	return &EthService{eth: eth}
}

func (s *EthService) BuildBlock(attrs *builderTypes.BuilderPayloadAttributes, sealedBlockCallback miner.SealedBlockCallbackFn) error {
	// Send a request to generate a full block in the background.
	// The result can be obtained via the returned channel.

	resCh, err := s.eth.Miner().BuildBlockWithCallback(attrs.HeadHash, uint64(attrs.Timestamp), common.HexToAddress(attrs.SuggestedFeeRecipient.String()), attrs.Random, attrs.NoMempoolTxs, attrs.Transactions, attrs.Bundles, attrs.Withdrawals, attrs.PayoutPoolAddress, attrs.BidAmount, attrs.GasLimit, sealedBlockCallback)
	if err != nil {
		return err
	}
	timer := time.NewTimer(4 * time.Second)
	defer timer.Stop()

	select {
	case block := <-resCh:
		if block == nil {
			return errors.New("received nil block from sealing work")
		}
		return nil
	case <-timer.C:
		log.Error("timeout waiting for block", "parent hash", attrs.HeadHash, "slot", attrs.Slot)
		return errors.New("timeout waiting for block result")
	}
}

func (s *EthService) Synced() bool {
	return s.eth.Synced()
}

func (s *EthService) GetTxPool() *txpool.TxPool {
	return s.eth.TxPool()
}

func (s *EthService) GetBlockChain() *core.BlockChain {
	return s.eth.BlockChain()
}

func (s* EthService) Backend() *eth.Ethereum {
	return s.eth
}