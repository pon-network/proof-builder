package bundles

import (
	"context"
	"crypto/ecdsa"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/eth"
	bundleTypes "github.com/bsn-eng/pon-golang-types/bundles"
	beaconTypes "github.com/bsn-eng/pon-golang-types/beaconclient"
	beacon "github.com/ethereum/go-ethereum/builder/beacon"

	"os"
	"path/filepath"

	"github.com/ethereum/go-ethereum/builder/database/vars"
	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
)

var (
	maxRetryCount = 3
	minBundleGas = uint64(42000)
)

type BundleService struct {
	eth                     *eth.Ethereum
	db                      *sqlx.DB
	builderWalletPrivateKey *ecdsa.PrivateKey
	beacon 				*beacon.MultiBeaconClient

	genesisInfo *beaconTypes.GenesisData

	Mu                    sync.Mutex
	NextBundleTimeStamp   time.Time
	NextBundleBlockNumber uint64
	NextBundlesMap        map[string]*bundleTypes.BuilderBundle // map of bundle unique id to bundle

	MaxLifetime uint64
	MaxFuture   uint64
}

func NewBundleService(dirPath string, reset bool, maxLifetime uint64, maxFuture uint64, eth *eth.Ethereum, builderWalletPrivateKey *ecdsa.PrivateKey, beaconClient *beacon.MultiBeaconClient) (*BundleService, error) {

	db_file := filepath.Join(dirPath, vars.DBName)

	// Check if the database path exists and if the file exist if not create it
	if _, err := os.Stat(dirPath); os.IsNotExist(err) {
		err = os.MkdirAll(dirPath, 0755)
		if err != nil {
			return nil, err
		}
	}

	db, err := sqlx.Connect("sqlite3", db_file)
	if err != nil {
		return nil, err
	}

	if reset {
		db.MustExec(ForceDropSchema)
	} else {
		// Drop tables if they are not of right configuration
		db.MustExec(DropSchema)
	}

	// Migrate the schema
	db.MustExec(CreateSchema)

	dbService := &BundleService{
		db:                      db,
		eth:                     eth,
		builderWalletPrivateKey: builderWalletPrivateKey,
		beacon: 				beaconClient,

		NextBundleTimeStamp:   time.Now(),
		NextBundleBlockNumber: 0,
		NextBundlesMap:        make(map[string]*bundleTypes.BuilderBundle),

		MaxLifetime: maxLifetime,
		MaxFuture:   maxFuture,
	}
	return dbService, err
}

func (s *BundleService) SetGenesis(genesisInfo *beaconTypes.GenesisData) {
	s.genesisInfo = genesisInfo
}

func (s *BundleService) Close() {
	s.db.Close()
}

func (s *BundleService) MonitorBundles() {

	go s.ProcessBundleNextTimestamp(context.Background())
	go s.ProcessBundleNextBlockNumber(context.Background())
	go s.CleanSentBundles(context.Background())

}