package builder

import (
	"net/http"

	capellaApi "github.com/attestantio/go-eth2-client/api/v1/capella"
	capella "github.com/attestantio/go-eth2-client/spec/capella"
	"github.com/attestantio/go-eth2-client/spec/phase0"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/txpool"
	"github.com/ethereum/go-ethereum/core/types"

	builderTypes "github.com/bsn-eng/pon-golang-types/builder"
	"github.com/ethereum/go-ethereum/miner"
)

type IRelay interface {
	SubmitBlockBid(msg *builderTypes.BuilderBlockBid) (interface{}, error)
	GetPayoutAddress() common.Address
	GetEndpoint() string
	CheckStatus() error
}

type IBuilder interface {
	handleBlockBid(w http.ResponseWriter, req *http.Request)
	handleBlindedBlockSubmission(w http.ResponseWriter, req *http.Request)
	handlePrivateTransactions(w http.ResponseWriter, req *http.Request)
	handleStatus(w http.ResponseWriter, req *http.Request)
	handleIndex(w http.ResponseWriter, req *http.Request)
	ProcessBuilderBid(attrs *builderTypes.BuilderPayloadAttributes) (builderTypes.BuilderBlockBid, error)
	OnPrivateTransactions(transactionBytes *builderTypes.PrivateTransactionsPayload) ([]*types.Transaction, error)
	SubmitBlindedBlock(capellaApi.BlindedBeaconBlock, phase0.BLSSignature) (capella.ExecutionPayload, error)
}

type IEthService interface {
	BuildBlock(attrs *builderTypes.BuilderPayloadAttributes, sealedBlockCallback miner.SealedBlockCallbackFn) error
	GetBlockByHash(hash common.Hash) *types.Block
	Synced() bool
	CurrentBlock() *types.Header
	GetTxPool() *txpool.TxPool
}
