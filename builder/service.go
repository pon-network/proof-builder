package builder

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/ethereum/go-ethereum/common/hexutil"

	// "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/eth"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/gorilla/mux"

	beacon "github.com/ethereum/go-ethereum/builder/beacon"
	"github.com/ethereum/go-ethereum/builder/bls"
	"github.com/ethereum/go-ethereum/builder/database"
	httplogger "github.com/ethereum/go-ethereum/builder/httplogger"
	RPBS "github.com/ethereum/go-ethereum/builder/rpbsService"
	builderTypes "github.com/ethereum/go-ethereum/builder/types"
)

const (
	_PathCheckStatus         = "/eth/v1/builder/status"
	_PathSubmitBlockBid      = "/eth/v1/builder/submit_block_bid"
	_PathSubmitBlindedBlock  = "/eth/v1/builder/blinded_blocks"
	_PathPrivateTransactions = "/eth/v1/builder/private_transactions"

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
	router.HandleFunc(_PathSubmitBlindedBlock, builder.handleBlindedBlockSubmission).Methods(http.MethodPost)
	router.HandleFunc(_PathPrivateTransactions, builder.handlePrivateTransactions).Methods(http.MethodPost)

	loggedRouter := httplogger.LoggingMiddleware(router)
	return loggedRouter
}

func NewService(listenAddr string, builder IBuilder) *Service {

	return &Service{
		srv: &http.Server{
			Addr:    listenAddr,
			Handler: getRouter(builder),
			/*
			   ReadTimeout:
			   ReadHeaderTimeout:
			   WriteTimeout:
			   IdleTimeout:
			*/
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
			return fmt.Errorf("failed to create remote relay: %w", err)
		}
	} else {
		return fmt.Errorf("PoN Relay not specified: %w", errors.New("no relay specified"))
	}

	ethereumService := NewEthService(backend)

	rpbsService := RPBS.NewRPBSService(cfg.RPBS)

	beaconClient, err := beacon.NewMultiBeaconClient(cfg.BeaconEndpoints)
	if err != nil {
		return fmt.Errorf("failed to create multi beacon client: %w", err)
	}
	go beaconClient.Start()

	genesisInfo, err := beaconClient.Genesis()
	if err != nil {
		return fmt.Errorf("failed to get genesis info: %w", err)
	}
	log.Info("genesis info:",
		"genesisTime", genesisInfo.GenesisTime,
		"genesisForkVersion", genesisInfo.GenesisForkVersion,
		"genesisValidatorsRoot", genesisInfo.GenesisValidatorsRoot)

	cfg.GenesisForkVersion = genesisInfo.GenesisForkVersion
	builderTypes.GENESIS_TIME = genesisInfo.GenesisTime

	envBuilderSkBytes, err := hexutil.Decode(cfg.BuilderSecretKey)
	if err != nil {
		return fmt.Errorf("incorrect builder API secret key provided: %w", err)
	}
	builderSk, err := bls.SecretKeyFromBytes(envBuilderSkBytes[:])
	if err != nil {
		return fmt.Errorf("incorrect builder API secret key provided: %w", err)
	}
	genesisForkVersionBytes, err := hexutil.Decode(cfg.GenesisForkVersion)
	if err != nil {
		return fmt.Errorf("invalid genesisForkVersion: %w", err)
	}
	var genesisForkVersion [4]byte
	copy(genesisForkVersion[:], genesisForkVersionBytes[:4])
	builderSigningDomain := builderTypes.ComputeDomain(builderTypes.DomainTypeAppBuilder, genesisForkVersion, builderTypes.Root{})

	var db *database.DatabaseService = nil
	if cfg.MetricsEnabled {
		db, err = database.NewDatabaseService(cfg.DatabaseDir, cfg.DatabaseReset)
		if err != nil {
			return fmt.Errorf("failed to connect to database: %w", err)
		}
	}

	builderBackend, err := NewBuilder(builderSk, beaconClient, relay, builderSigningDomain, ethereumService, rpbsService, db, cfg)
	if err != nil {
		return err
	}

	builderService := NewService(cfg.ListenAddr, builderBackend)

	log.Info("PoN Builder service registered", "listenAddr", cfg.ListenAddr)

	stack.RegisterAPIs([]rpc.API{
		{
			Namespace:     "pon-builder",
			Version:       "1.0",
			Service:       builderService,
			Public:        true,
			Authenticated: true,
		},
	})

	stack.RegisterLifecycle(builderService)

	return nil
}
