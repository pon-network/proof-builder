package builder

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/ethereum/go-ethereum/common/hexutil"

	"github.com/ethereum/go-ethereum/eth"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/node"
	"github.com/gorilla/mux"

	"github.com/ethereum/go-ethereum/core"

	beacon "github.com/ethereum/go-ethereum/builder/beacon"
	"github.com/ethereum/go-ethereum/builder/bls"
	"github.com/ethereum/go-ethereum/builder/database"
	httplogger "github.com/ethereum/go-ethereum/builder/httplogger"
	RPBS "github.com/ethereum/go-ethereum/builder/rpbsService"

	"github.com/ethereum/go-ethereum/builder/ethService"

	"github.com/ethereum/go-ethereum/builder/builderRPC"
	"github.com/ethereum/go-ethereum/builder/bundles"
)

const (
	_PathCheckStatus          = "/eth/v1/builder/status"
	_PathSubmitBlockBid       = "/eth/v1/builder/submit_block_bid"
	_PathSubmitBlockBountyBid = "/eth/v1/builder/submit_block_bounty_bid"
	_PathSubmitBlindedBlock   = "/eth/v1/builder/blinded_blocks"

	_PathIndex = "/eth/v1/builder/"
)

type Service struct {
	srv     *http.Server
	builder IBuilder
}

func (s *Service) Start() error {
	if s.srv != nil {
		log.Info("PoN Builder Service started", "endpoint", s.srv.Addr)
		go s.srv.ListenAndServe()
	}

	err := s.builder.Start()
	if err != nil {
		return fmt.Errorf("PoN Builder Service failed to start: %w", err)
	}

	return nil
}

func (s *Service) Stop() error {
	if s.srv != nil {
		s.srv.Close()
	}
	return nil
}

func getRouter(builder IBuilder) http.Handler {
	router := mux.NewRouter()

	// Add routes
	router.HandleFunc(_PathIndex, builder.handleIndex).Methods(http.MethodGet)
	router.HandleFunc(_PathCheckStatus, builder.handleStatus).Methods(http.MethodGet)
	router.HandleFunc(_PathSubmitBlockBid, builder.handleBlockBid).Methods(http.MethodPost)
	router.HandleFunc(_PathSubmitBlockBountyBid, builder.handleBlockBountyBid).Methods(http.MethodPost)
	router.HandleFunc(_PathSubmitBlindedBlock, builder.handleBlindedBlockSubmission).Methods(http.MethodPost)

	loggedRouter := httplogger.LoggingMiddleware(router)
	return loggedRouter
}

func NewService(listenAddr string, builder IBuilder) *Service {

	return &Service{
		srv: &http.Server{
			Addr:    listenAddr,
			Handler: getRouter(builder),
		},
		builder: builder,
	}
}

func Register(stack *node.Node, backend *eth.Ethereum, cfg *Config) error {

	var relay IRelay
	var err error

	if cfg.RelayEndpoint != "" {
		relay, err = NewRelay(cfg.RelayEndpoint)
		if err != nil {
			return fmt.Errorf("Relay to create relay: %w", err)
		}
	} else {
		return fmt.Errorf("Relay not specified: %w", errors.New("no relay specified"))
	}

	ethereumService := ethService.NewEthService(backend)

	rpbsService := RPBS.NewRPBSService(cfg.RPBS)

	beaconClient, err := beacon.NewMultiBeaconClient(cfg.BeaconEndpoints)
	if err != nil {
		return fmt.Errorf("failed to create multi beacon client: %w", err)
	}

	envBuilderSkBytes, err := hexutil.Decode(cfg.BuilderSecretKey)
	if err != nil {
		return fmt.Errorf("incorrect builder API secret key provided: %w", err)
	}
	builderSk, err := bls.SecretKeyFromBytes(envBuilderSkBytes[:])
	if err != nil {
		return fmt.Errorf("incorrect builder API secret key provided: %w", err)
	}

	var db *database.DatabaseService = nil
	if cfg.MetricsEnabled {
		db, err = database.NewDatabaseService(cfg.DatabaseDir, cfg.DatabaseReset)
		if err != nil {
			return fmt.Errorf("failed to connect to database: %w", err)
		}
	}

	var builderBundleService *bundles.BundleService = nil
	if cfg.BundlesEnabled {
		builderBundleService, err = bundles.NewBundleService(cfg.DatabaseDir, cfg.BundlesReset, uint64(cfg.BundlesMaxLifetime), uint64(cfg.BundlesMaxFuture), backend, cfg.BuilderWalletPrivateKey, beaconClient)
		if err != nil {
			return fmt.Errorf("failed to create bundle service: %w", err)
		}
	}

	builderRPCService, err := builderRPC.NewBuilderRPCService(cfg.DatabaseDir, stack, ethereumService, builderBundleService)
	if err != nil {
		return fmt.Errorf("failed to create builder RPC service: %w", err)
	}

	blockValidator := core.NewBlockValidator(backend.BlockChain().Config(), backend.BlockChain(), backend.Engine())

	builderBackend, err := NewBuilder(builderSk, blockValidator, beaconClient, relay, ethereumService, rpbsService, builderRPCService, builderBundleService, db, cfg)
	if err != nil {
		return err
	}

	builderService := NewService(cfg.ListenAddr, builderBackend)

	log.Info("PoN Builder service registered", "listenAddr", cfg.ListenAddr)

	stack.RegisterLifecycle(builderService)

	return nil
}
