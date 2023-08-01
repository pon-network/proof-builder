package builderRPC

import (
	"context"

	"github.com/ethereum/go-ethereum/builder/bundles"
	"github.com/ethereum/go-ethereum/builder/ethService"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/ethereum/go-ethereum/builder/database/vars"

	"os"
	"path/filepath"

	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
)

type BuilderRPCService struct {
	eth           *ethService.EthService
	bundleService *bundles.BundleService
	db            *sqlx.DB
}

func NewBuilderRPCService(dirPath string, node *node.Node, ethService *ethService.EthService, bundleService *bundles.BundleService) (*BuilderRPCService, error) {

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

	// Drop tables if they are not of right configuration
	db.MustExec(DropSchema)

	// Migrate the schema
	db.MustExec(CreateSchema)

	rpcService := &BuilderRPCService{
		eth:           ethService,
		bundleService: bundleService,
		db:            db,
	}

	node.RegisterAPIs(
		[]rpc.API{
			{
				Namespace:     "mev",
				Version:       "1.0",
				Service:       rpcService,
				Public:        true,
				Authenticated: false,
			},
		},
	)

	log.Info("Builder RPC Service registered", "namespace", "mev", "version", "1.0")

	return rpcService, nil
}

func (builderRPC *BuilderRPCService) Close() {
	builderRPC.db.Close()
}

func (builderRPC *BuilderRPCService) MonitorRPCtransactions() {

	go builderRPC.cleanOldTransactions(context.Background())
	
}
