package builder

import (
	"crypto/ecdsa"

	"github.com/ethereum/go-ethereum/common"
)

type Config struct {
	Enabled                 bool              `toml:",omitempty"`
	BuilderWalletPrivateKey *ecdsa.PrivateKey `toml:",omitempty"`
	BuilderWalletAddress    common.Address    `toml:",omitempty"`
	BuilderSecretKey        string            `toml:",omitempty"`
	ListenAddr              string            `toml:",omitempty"`
	GenesisForkVersion      string            `toml:",omitempty"`
	GenesisTime             uint64            `toml:",omitempty"`
	GenesisValidatorsRoot   string            `toml:",omitempty"`
	BeaconEndpoints         []string          `toml:",omitempty"`
	RelayEndpoint           string            `toml:",omitempty"` // In the format PayOutPoolAddress@RelayEndpoint:RelayPort
	ExposedAccessPoint      string            `toml:",omitempty"`
	MetricsEnabled          bool              `toml:",omitempty"`
	DatabaseDir             string            `toml:",omitempty"` // Directory to store the sqlite database
	DatabaseReset           bool              `toml:",omitempty"` // Reset the database on startup
	RPBS                    string            `toml:",omitempty"` // RPBS Service
}

// DefaultConfig is the default config for the builder.
var DefaultConfig = Config{
	Enabled:                 false,
	BuilderWalletPrivateKey: &ecdsa.PrivateKey{},
	BuilderWalletAddress:    common.Address{},
	BuilderSecretKey:        "",
	ListenAddr:              ":10000",
	GenesisForkVersion:      "0x00001020",
	GenesisTime:             1616508000, // In seconds
	GenesisValidatorsRoot:   "0x043db0d9a83813551ee2f33450d23797757d430911a9320530",
	BeaconEndpoints:         []string{"http://localhost:5052"},
	RelayEndpoint:           "", // In the format PayOutPoolAddress@RelayEndpoint:RelayPort
	ExposedAccessPoint:      ":10000",
	MetricsEnabled:          false,
	DatabaseDir:             "",
	DatabaseReset:           false,
	RPBS:                    "",
}
