// Copyright 2017 The go-gdtu Authors
// This file is part of the go-gdtu library.
//
// The go-gdtu library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-gdtu library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// algdtu with the go-gdtu library. If not, see <http://www.gnu.org/licenses/>.

// Package gdtuconfig contains the configuration of the GDTU and LES protocols.
package gdtuconfig

import (
	"math/big"
	"os"
	"os/user"
	"path/filepath"
	"runtime"
	"time"

	"github.com/c88032111/go-gdtu/common"
	"github.com/c88032111/go-gdtu/consensus"
	"github.com/c88032111/go-gdtu/consensus/clique"
	"github.com/c88032111/go-gdtu/consensus/gdtuash"
	"github.com/c88032111/go-gdtu/core"
	"github.com/c88032111/go-gdtu/gdtu/downloader"
	"github.com/c88032111/go-gdtu/gdtu/gasprice"
	"github.com/c88032111/go-gdtu/gdtudb"
	"github.com/c88032111/go-gdtu/log"
	"github.com/c88032111/go-gdtu/miner"
	"github.com/c88032111/go-gdtu/node"
	"github.com/c88032111/go-gdtu/params"
)

// FullNodeGPO contains default gasprice oracle settings for full node.
var FullNodeGPO = gasprice.Config{
	Blocks:     20,
	Percentile: 60,
	MaxPrice:   gasprice.DefaultMaxPrice,
}

// LightClientGPO contains default gasprice oracle settings for light client.
var LightClientGPO = gasprice.Config{
	Blocks:     2,
	Percentile: 60,
	MaxPrice:   gasprice.DefaultMaxPrice,
}

// Defaults contains default settings for use on the Gdtu main net.
var Defaults = Config{
	SyncMode: downloader.FastSync,
	Gdtuash: gdtuash.Config{
		CacheDir:         "gdtuash",
		CachesInMem:      2,
		CachesOnDisk:     3,
		CachesLockMmap:   false,
		DatasetsInMem:    1,
		DatasetsOnDisk:   2,
		DatasetsLockMmap: false,
	},
	NetworkId:               1,
	TxLookupLimit:           2350000,
	LightPeers:              100,
	UltraLightFraction:      75,
	DatabaseCache:           512,
	TrieCleanCache:          154,
	TrieCleanCacheJournal:   "triecache",
	TrieCleanCacheRejournal: 60 * time.Minute,
	TrieDirtyCache:          256,
	TrieTimeout:             60 * time.Minute,
	SnapshotCache:           102,
	Miner: miner.Config{
		GasFloor: 8000000,
		GasCeil:  8000000,
		GasPrice: big.NewInt(params.GWei),
		Recommit: 3 * time.Second,
	},
	TxPool:      core.DefaultTxPoolConfig,
	RPCGasCap:   25000000,
	GPO:         FullNodeGPO,
	RPCTxFeeCap: 1, // 1 gdtuer
}

func init() {
	home := os.Getenv("HOME")
	if home == "" {
		if user, err := user.Current(); err == nil {
			home = user.HomeDir
		}
	}
	if runtime.GOOS == "darwin" {
		Defaults.Gdtuash.DatasetDir = filepath.Join(home, "Library", "Gdtuash")
	} else if runtime.GOOS == "windows" {
		localappdata := os.Getenv("LOCALAPPDATA")
		if localappdata != "" {
			Defaults.Gdtuash.DatasetDir = filepath.Join(localappdata, "Gdtuash")
		} else {
			Defaults.Gdtuash.DatasetDir = filepath.Join(home, "AppData", "Local", "Gdtuash")
		}
	} else {
		Defaults.Gdtuash.DatasetDir = filepath.Join(home, ".gdtuash")
	}
}

//go:generate gencodec -type Config -formats toml -out gen_config.go

// Config contains configuration options for of the GDTU and LES protocols.
type Config struct {
	// The genesis block, which is inserted if the database is empty.
	// If nil, the Gdtu main net block is used.
	Genesis *core.Genesis `toml:",omitempty"`

	// Protocol options
	NetworkId uint64 // Network ID to use for selecting peers to connect to
	SyncMode  downloader.SyncMode

	// This can be set to list of enrtree:// URLs which will be queried for
	// for nodes to connect to.
	GdtuDiscoveryURLs []string
	SnapDiscoveryURLs []string

	NoPruning  bool // Whgdtuer to disable pruning and flush everything to disk
	NoPrefetch bool // Whgdtuer to disable prefetching and only load state on demand

	TxLookupLimit uint64 `toml:",omitempty"` // The maximum number of blocks from head whose tx indices are reserved.

	// Whitelist of required block number -> hash values to accept
	Whitelist map[uint64]common.Hash `toml:"-"`

	// Light client options
	LightServ          int  `toml:",omitempty"` // Maximum percentage of time allowed for serving LES requests
	LightIngress       int  `toml:",omitempty"` // Incoming bandwidth limit for light servers
	LightEgress        int  `toml:",omitempty"` // Outgoing bandwidth limit for light servers
	LightPeers         int  `toml:",omitempty"` // Maximum number of LES client peers
	LightNoPrune       bool `toml:",omitempty"` // Whgdtuer to disable light chain pruning
	LightNoSyncServe   bool `toml:",omitempty"` // Whgdtuer to serve light clients before syncing
	SyncFromCheckpoint bool `toml:",omitempty"` // Whgdtuer to sync the header chain from the configured checkpoint

	// Ultra Light client options
	UltraLightServers      []string `toml:",omitempty"` // List of trusted ultra light servers
	UltraLightFraction     int      `toml:",omitempty"` // Percentage of trusted servers to accept an announcement
	UltraLightOnlyAnnounce bool     `toml:",omitempty"` // Whgdtuer to only announce headers, or also serve them

	// Database options
	SkipBcVersionCheck bool `toml:"-"`
	DatabaseHandles    int  `toml:"-"`
	DatabaseCache      int
	DatabaseFreezer    string

	TrieCleanCache          int
	TrieCleanCacheJournal   string        `toml:",omitempty"` // Disk journal directory for trie cache to survive node restarts
	TrieCleanCacheRejournal time.Duration `toml:",omitempty"` // Time interval to regenerate the journal for clean cache
	TrieDirtyCache          int
	TrieTimeout             time.Duration
	SnapshotCache           int
	Preimages               bool

	// Mining options
	Miner miner.Config

	// Gdtuash options
	Gdtuash gdtuash.Config

	// Transaction pool options
	TxPool core.TxPoolConfig

	// Gas Price Oracle options
	GPO gasprice.Config

	// Enables tracking of SHA3 preimages in the VM
	EnablePreimageRecording bool

	// Miscellaneous options
	DocRoot string `toml:"-"`

	// Type of the EWASM interpreter ("" for default)
	EWASMInterpreter string

	// Type of the EVM interpreter ("" for default)
	EVMInterpreter string

	// RPCGasCap is the global gas cap for gdtu-call variants.
	RPCGasCap uint64 `toml:",omitempty"`

	// RPCTxFeeCap is the global transaction fee(price * gaslimit) cap for
	// send-transction variants. The unit is gdtuer.
	RPCTxFeeCap float64 `toml:",omitempty"`

	// Checkpoint is a hardcoded checkpoint which can be nil.
	Checkpoint *params.TrustedCheckpoint `toml:",omitempty"`

	// CheckpointOracle is the configuration for checkpoint oracle.
	CheckpointOracle *params.CheckpointOracleConfig `toml:",omitempty"`

	// Berlin block override (TODO: remove after the fork)
	OverrideBerlin *big.Int `toml:",omitempty"`
}

// CreateConsensusEngine creates a consensus engine for the given chain configuration.
func CreateConsensusEngine(stack *node.Node, chainConfig *params.ChainConfig, config *gdtuash.Config, notify []string, noverify bool, db gdtudb.Database) consensus.Engine {
	// If proof-of-authority is requested, set it up
	if chainConfig.Clique != nil {
		return clique.New(chainConfig.Clique, db)
	}
	// Otherwise assume proof-of-work
	switch config.PowMode {
	case gdtuash.ModeFake:
		log.Warn("Gdtuash used in fake mode")
		return gdtuash.NewFaker()
	case gdtuash.ModeTest:
		log.Warn("Gdtuash used in test mode")
		return gdtuash.NewTester(nil, noverify)
	case gdtuash.ModeShared:
		log.Warn("Gdtuash used in shared mode")
		return gdtuash.NewShared()
	default:
		engine := gdtuash.New(gdtuash.Config{
			CacheDir:         stack.ResolvePath(config.CacheDir),
			CachesInMem:      config.CachesInMem,
			CachesOnDisk:     config.CachesOnDisk,
			CachesLockMmap:   config.CachesLockMmap,
			DatasetDir:       config.DatasetDir,
			DatasetsInMem:    config.DatasetsInMem,
			DatasetsOnDisk:   config.DatasetsOnDisk,
			DatasetsLockMmap: config.DatasetsLockMmap,
		}, notify, noverify)
		engine.SetThreads(-1) // Disable CPU mining
		return engine
	}
}
