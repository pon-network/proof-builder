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
	BeaconEndpoints         []string          `toml:",omitempty"`
	RelayEndpoint           string            `toml:",omitempty"` // In the format PayOutPoolAddress@RelayEndpoint:RelayPort
	ExposedAccessPoint      string            `toml:",omitempty"`
	MetricsEnabled          bool              `toml:",omitempty"`
	DatabaseDir             string            `toml:",omitempty"` // Directory to store the sqlite databases
	DatabaseReset           bool              `toml:",omitempty"` // Reset the database on startup
	BundlesEnabled          bool              `toml:",omitempty"` // Enable bundles
	BundlesReset            bool              `toml:",omitempty"` // Reset the bundles on startup
	BundlesMaxLifetime      int               `toml:",omitempty"` // Max lifetime of a bundle in seconds
	BundlesMaxFuture        int               `toml:",omitempty"` // Max future time of a bundle in seconds
	RPBS                    string            `toml:",omitempty"` // RPBS Service
	SubmissionEndWindow         int               `toml:",omitempty"` // Delay in seconds before submitting finalized bid
	EngineRateLimit         int               `toml:",omitempty"` // Rate limit for builder engine in ms
}

// DefaultConfig is the default config for the builder.
var DefaultConfig = Config{
	Enabled:                 false,
	BuilderWalletPrivateKey: &ecdsa.PrivateKey{},
	BuilderWalletAddress:    common.Address{},
	BuilderSecretKey:        "",
	ListenAddr:              ":10000",
	BeaconEndpoints:         []string{"http://localhost:5052"},
	RelayEndpoint:           "", // In the format PayOutPoolAddress@RelayEndpoint:RelayPort
	ExposedAccessPoint:      ":10000",
	MetricsEnabled:          false,
	DatabaseDir:             "",
	DatabaseReset:           false,
	BundlesEnabled:          true,
	BundlesReset:            false,
	BundlesMaxLifetime:      3600 * 720, // 30 days (in seconds)
	BundlesMaxFuture:        3600 * 720, // 30 days (in seconds)
	RPBS:                    "",
	SubmissionEndWindow:         2,
	EngineRateLimit:         500,
}
