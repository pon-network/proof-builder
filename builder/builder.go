package builder

import (
	"context"
	"crypto/ecdsa"
	"fmt"

	_ "os"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"

	"github.com/ethereum/go-ethereum/beacon/engine"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/log"

	beaconTypes "github.com/bsn-eng/pon-golang-types/beaconclient"
	builderTypes "github.com/bsn-eng/pon-golang-types/builder"
	commonTypes "github.com/bsn-eng/pon-golang-types/common"
	"github.com/ethereum/go-ethereum/builder/bls"
	"github.com/ethereum/go-ethereum/builder/bundles"
	bbTypes "github.com/ethereum/go-ethereum/builder/types"

	beacon "github.com/ethereum/go-ethereum/builder/beacon"
	"github.com/ethereum/go-ethereum/builder/builderRPC"
	"github.com/ethereum/go-ethereum/builder/database"
	RPBS "github.com/ethereum/go-ethereum/builder/rpbsService"
)

type Builder struct {
	relay                IRelay
	eth                  IEthService
	beacon               *beacon.MultiBeaconClient
	rpbs                 *RPBS.RPBSService
	rpc                  *builderRPC.BuilderRPCService
	bundles              *bundles.BundleService
	db                   *database.DatabaseService
	blockValidator       *core.BlockValidator
	builderSecretKey     *bls.SecretKey
	builderPublicKey     commonTypes.PublicKey
	builderSigningDomain bbTypes.Domain

	builderWalletPrivateKey *ecdsa.PrivateKey
	builderWalletAddress    common.Address

	genesisInfo *beaconTypes.GenesisData

	submissionEndWindow time.Duration
	buildInterval       time.Duration

	AccessPoint string

	MetricsEnabled bool
	BundlesEnabled bool

	slotMu    sync.Mutex
	slotAttrs map[uint64][]builderTypes.BuilderPayloadAttributes

	slotBountyMu    sync.Mutex
	slotBountyAttrs map[uint64][]builderTypes.BuilderPayloadAttributes

	slotCtx       context.Context
	slotCtxCancel context.CancelFunc

	slotSubmissionsLock    sync.Mutex
	slotSubmissionsChan    map[uint64]chan blockProperties
	slotBidCompleteChan    map[uint64]chan struct{ bidAmount uint64 }
	slotBountyCompleteChan map[uint64]chan struct{ bidAmount uint64 }
	slotSubmissions        map[uint64][]builderTypes.BlockBidResponse
	slotBidAmounts         map[uint64][]uint64 // Tracking of normal auction bid amounts for a slot
	slotBountyAmount       map[uint64]uint64   // Submitted bounty bid amount for a slot

	// Execution payload cache: map of slot -> map of block hash -> execution payload
	executionPayloadCache map[uint64]map[string]engine.ExecutableData
}

func NewBuilder(sk *bls.SecretKey, blockValidator *core.BlockValidator, beaconClient *beacon.MultiBeaconClient, relay IRelay, eth IEthService, rpbs *RPBS.RPBSService, rpc *builderRPC.BuilderRPCService, bundleService *bundles.BundleService, database *database.DatabaseService, config *Config) (*Builder, error) {
	pkBytes := bls.PublicKeyFromSecretKey(sk).Compress()
	pk := commonTypes.PublicKey{}
	err := pk.FromSlice(pkBytes)
	if err != nil {
		return nil, err
	}

	submissionEndWindow, err := time.ParseDuration(fmt.Sprintf("%ds", config.SubmissionEndWindow))
	if err != nil {
		return nil, fmt.Errorf("failed to parse submission delay: %v", err)
	}

	buildInterval, err := time.ParseDuration(fmt.Sprintf("%dms", config.EngineRateLimit))
	if err != nil {
		return nil, fmt.Errorf("failed to parse engine rate limit: %v", err)
	}

	slotCtx, slotCtxCancel := context.WithCancel(context.Background())

	return &Builder{
		relay:            relay,
		eth:              eth,
		beacon:           beaconClient,
		rpbs:             rpbs,
		rpc:              rpc,
		db:               database,
		bundles:          bundleService,
		blockValidator:   blockValidator,
		builderSecretKey: sk,
		builderPublicKey: pk,

		builderWalletPrivateKey: config.BuilderWalletPrivateKey,
		builderWalletAddress:    config.BuilderWalletAddress,

		submissionEndWindow: submissionEndWindow,
		buildInterval:       buildInterval,

		AccessPoint:    config.ExposedAccessPoint,
		MetricsEnabled: config.MetricsEnabled,
		BundlesEnabled: config.BundlesEnabled,

		slotCtx:       slotCtx,
		slotCtxCancel: slotCtxCancel,
		slotAttrs:     make(map[uint64][]builderTypes.BuilderPayloadAttributes),
		slotBountyAttrs: make(map[uint64][]builderTypes.BuilderPayloadAttributes),

		slotSubmissionsChan: make(map[uint64]chan blockProperties),
		slotBidCompleteChan: make(map[uint64]chan struct{ bidAmount uint64 }),
		slotBountyCompleteChan: make(map[uint64]chan struct{ bidAmount uint64 }),
		slotSubmissions:     make(map[uint64][]builderTypes.BlockBidResponse),
		slotBidAmounts:      make(map[uint64][]uint64),
		slotBountyAmount:    make(map[uint64]uint64),

		executionPayloadCache: make(map[uint64]map[string]engine.ExecutableData),
	}, nil
}

func (b *Builder) Start() error {

	// Start beacon client
	b.beacon.Start()

	// Set genesis info
	genesisInfo, err := b.beacon.Genesis()
	if err != nil {
		return fmt.Errorf("failed to get genesis info: %w", err)
	}
	log.Info("genesis info:",
		"genesisTime", genesisInfo.GenesisTime,
		"genesisForkVersion", genesisInfo.GenesisForkVersion,
		"genesisValidatorsRoot", genesisInfo.GenesisValidatorsRoot)
	b.genesisInfo = genesisInfo

	// Configure builder signing domanin
	genesisForkVersionBytes, err := hexutil.Decode(genesisInfo.GenesisForkVersion)
	if err != nil {
		return fmt.Errorf("invalid genesisForkVersion: %w", err)
	}
	var genesisForkVersion [4]byte
	copy(genesisForkVersion[:], genesisForkVersionBytes[:4])
	builderSigningDomain := bbTypes.ComputeDomain(bbTypes.DomainTypeAppBuilder, genesisForkVersion, commonTypes.Root{})
	b.builderSigningDomain = builderSigningDomain

	// If bundles are enabled then start bundle service
	if b.BundlesEnabled {
		b.bundles.SetGenesis(genesisInfo)
		b.bundles.MonitorBundles()
	}

	// Start RPC server
	b.rpc.MonitorRPCtransactions()

	return nil

}
