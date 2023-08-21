package builder

import (
	"net/http"

	capellaApi "github.com/attestantio/go-eth2-client/api/v1/capella"
	capella "github.com/attestantio/go-eth2-client/spec/capella"
	"github.com/attestantio/go-eth2-client/spec/phase0"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/txpool"
	"github.com/ethereum/go-ethereum/eth"

	builderTypes "github.com/bsn-eng/pon-golang-types/builder"
	"github.com/ethereum/go-ethereum/miner"
)

type IRelay interface {
	SubmitBlockBid(msg *builderTypes.BuilderBlockBid, bounty bool) (interface{}, error)
	GetPayoutAddress() common.Address
	GetEndpoint() string
	CheckStatus() error
}

type IBuilder interface {
	handleBlockBid(w http.ResponseWriter, req *http.Request)
	handleBlockBountyBid(w http.ResponseWriter, req *http.Request)
	handleBlindedBlockSubmission(w http.ResponseWriter, req *http.Request)
	handleStatus(w http.ResponseWriter, req *http.Request)
	handleIndex(w http.ResponseWriter, req *http.Request)
	ProcessBuilderBid(attrs *builderTypes.BuilderPayloadAttributes) ([]builderTypes.BlockBidResponse, error)
	ProcessBuilderBountyBid(attrs *builderTypes.BuilderPayloadAttributes) ([]builderTypes.BlockBidResponse, error)
	SubmitBlindedBlock(capellaApi.BlindedBeaconBlock, phase0.BLSSignature) (capella.ExecutionPayload, error)
	Start() error
}

type IEthService interface {
	BuildBlock(attrs *builderTypes.BuilderPayloadAttributes, sealedBlockCallback miner.SealedBlockCallbackFn) error
	Synced() bool
	GetTxPool() *txpool.TxPool
	GetBlockChain() *core.BlockChain
	Backend() *eth.Ethereum
	GetPayoutPoolTxGas() uint64
	GetBlockGasCeil() uint64
}
