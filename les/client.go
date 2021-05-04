// Copyright 2016 The go-gdtu Authors
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

// Package les implements the Light Gdtu Subprotocol.
package les

import (
	"fmt"
	"time"

	"github.com/c88032111/go-gdtu/accounts"
	"github.com/c88032111/go-gdtu/common"
	"github.com/c88032111/go-gdtu/common/hexutil"
	"github.com/c88032111/go-gdtu/common/mclock"
	"github.com/c88032111/go-gdtu/consensus"
	"github.com/c88032111/go-gdtu/core"
	"github.com/c88032111/go-gdtu/core/bloombits"
	"github.com/c88032111/go-gdtu/core/rawdb"
	"github.com/c88032111/go-gdtu/core/types"
	"github.com/c88032111/go-gdtu/event"
	"github.com/c88032111/go-gdtu/gdtu/downloader"
	"github.com/c88032111/go-gdtu/gdtu/filters"
	"github.com/c88032111/go-gdtu/gdtu/gasprice"
	"github.com/c88032111/go-gdtu/gdtu/gdtuconfig"
	"github.com/c88032111/go-gdtu/internal/gdtuapi"
	"github.com/c88032111/go-gdtu/les/vflux"
	vfc "github.com/c88032111/go-gdtu/les/vflux/client"
	"github.com/c88032111/go-gdtu/light"
	"github.com/c88032111/go-gdtu/log"
	"github.com/c88032111/go-gdtu/node"
	"github.com/c88032111/go-gdtu/p2p"
	"github.com/c88032111/go-gdtu/p2p/enode"
	"github.com/c88032111/go-gdtu/p2p/enr"
	"github.com/c88032111/go-gdtu/params"
	"github.com/c88032111/go-gdtu/rlp"
	"github.com/c88032111/go-gdtu/rpc"
)

type LightGdtu struct {
	lesCommons

	peers              *serverPeerSet
	reqDist            *requestDistributor
	retriever          *retrieveManager
	odr                *LesOdr
	relay              *lesTxRelay
	handler            *clientHandler
	txPool             *light.TxPool
	blockchain         *light.LightChain
	serverPool         *vfc.ServerPool
	serverPoolIterator enode.Iterator
	pruner             *pruner

	bloomRequests chan chan *bloombits.Retrieval // Channel receiving bloom data retrieval requests
	bloomIndexer  *core.ChainIndexer             // Bloom indexer operating during block imports

	ApiBackend     *LesApiBackend
	eventMux       *event.TypeMux
	engine         consensus.Engine
	accountManager *accounts.Manager
	netRPCService  *gdtuapi.PublicNetAPI

	p2pServer *p2p.Server
	p2pConfig *p2p.Config
}

// New creates an instance of the light client.
func New(stack *node.Node, config *gdtuconfig.Config) (*LightGdtu, error) {
	chainDb, err := stack.OpenDatabase("lightchaindata", config.DatabaseCache, config.DatabaseHandles, "gdtu/db/chaindata/")
	if err != nil {
		return nil, err
	}
	lesDb, err := stack.OpenDatabase("les.client", 0, 0, "gdtu/db/lesclient/")
	if err != nil {
		return nil, err
	}
	chainConfig, genesisHash, genesisErr := core.SetupGenesisBlockWithOverride(chainDb, config.Genesis, config.OverrideBerlin)
	if _, isCompat := genesisErr.(*params.ConfigCompatError); genesisErr != nil && !isCompat {
		return nil, genesisErr
	}
	log.Info("Initialised chain configuration", "config", chainConfig)

	peers := newServerPeerSet()
	lgdtu := &LightGdtu{
		lesCommons: lesCommons{
			genesis:     genesisHash,
			config:      config,
			chainConfig: chainConfig,
			iConfig:     light.DefaultClientIndexerConfig,
			chainDb:     chainDb,
			lesDb:       lesDb,
			closeCh:     make(chan struct{}),
		},
		peers:          peers,
		eventMux:       stack.EventMux(),
		reqDist:        newRequestDistributor(peers, &mclock.System{}),
		accountManager: stack.AccountManager(),
		engine:         gdtuconfig.CreateConsensusEngine(stack, chainConfig, &config.Gdtuash, nil, false, chainDb),
		bloomRequests:  make(chan chan *bloombits.Retrieval),
		bloomIndexer:   core.NewBloomIndexer(chainDb, params.BloomBitsBlocksClient, params.HelperTrieConfirmations),
		p2pServer:      stack.Server(),
		p2pConfig:      &stack.Config().P2P,
	}

	var prenegQuery vfc.QueryFunc
	if lgdtu.p2pServer.DiscV5 != nil {
		prenegQuery = lgdtu.prenegQuery
	}
	lgdtu.serverPool, lgdtu.serverPoolIterator = vfc.NewServerPool(lesDb, []byte("serverpool:"), time.Second, prenegQuery, &mclock.System{}, config.UltraLightServers, requestList)
	lgdtu.serverPool.AddMetrics(suggestedTimeoutGauge, totalValueGauge, serverSelectableGauge, serverConnectedGauge, sessionValueMeter, serverDialedMeter)

	lgdtu.retriever = newRetrieveManager(peers, lgdtu.reqDist, lgdtu.serverPool.GetTimeout)
	lgdtu.relay = newLesTxRelay(peers, lgdtu.retriever)

	lgdtu.odr = NewLesOdr(chainDb, light.DefaultClientIndexerConfig, lgdtu.peers, lgdtu.retriever)
	lgdtu.chtIndexer = light.NewChtIndexer(chainDb, lgdtu.odr, params.CHTFrequency, params.HelperTrieConfirmations, config.LightNoPrune)
	lgdtu.bloomTrieIndexer = light.NewBloomTrieIndexer(chainDb, lgdtu.odr, params.BloomBitsBlocksClient, params.BloomTrieFrequency, config.LightNoPrune)
	lgdtu.odr.SetIndexers(lgdtu.chtIndexer, lgdtu.bloomTrieIndexer, lgdtu.bloomIndexer)

	checkpoint := config.Checkpoint
	if checkpoint == nil {
		checkpoint = params.TrustedCheckpoints[genesisHash]
	}
	// Note: NewLightChain adds the trusted checkpoint so it needs an ODR with
	// indexers already set but not started yet
	if lgdtu.blockchain, err = light.NewLightChain(lgdtu.odr, lgdtu.chainConfig, lgdtu.engine, checkpoint); err != nil {
		return nil, err
	}
	lgdtu.chainReader = lgdtu.blockchain
	lgdtu.txPool = light.NewTxPool(lgdtu.chainConfig, lgdtu.blockchain, lgdtu.relay)

	// Set up checkpoint oracle.
	lgdtu.oracle = lgdtu.setupOracle(stack, genesisHash, config)

	// Note: AddChildIndexer starts the update process for the child
	lgdtu.bloomIndexer.AddChildIndexer(lgdtu.bloomTrieIndexer)
	lgdtu.chtIndexer.Start(lgdtu.blockchain)
	lgdtu.bloomIndexer.Start(lgdtu.blockchain)

	// Start a light chain pruner to delete useless historical data.
	lgdtu.pruner = newPruner(chainDb, lgdtu.chtIndexer, lgdtu.bloomTrieIndexer)

	// Rewind the chain in case of an incompatible config upgrade.
	if compat, ok := genesisErr.(*params.ConfigCompatError); ok {
		log.Warn("Rewinding chain to upgrade configuration", "err", compat)
		lgdtu.blockchain.SetHead(compat.RewindTo)
		rawdb.WriteChainConfig(chainDb, genesisHash, chainConfig)
	}

	lgdtu.ApiBackend = &LesApiBackend{stack.Config().ExtRPCEnabled(), stack.Config().AllowUnprotectedTxs, lgdtu, nil}
	gpoParams := config.GPO
	if gpoParams.Default == nil {
		gpoParams.Default = config.Miner.GasPrice
	}
	lgdtu.ApiBackend.gpo = gasprice.NewOracle(lgdtu.ApiBackend, gpoParams)

	lgdtu.handler = newClientHandler(config.UltraLightServers, config.UltraLightFraction, checkpoint, lgdtu)
	if lgdtu.handler.ulc != nil {
		log.Warn("Ultra light client is enabled", "trustedNodes", len(lgdtu.handler.ulc.keys), "minTrustedFraction", lgdtu.handler.ulc.fraction)
		lgdtu.blockchain.DisableCheckFreq()
	}

	lgdtu.netRPCService = gdtuapi.NewPublicNetAPI(lgdtu.p2pServer, lgdtu.config.NetworkId)

	// Register the backend on the node
	stack.RegisterAPIs(lgdtu.APIs())
	stack.RegisterProtocols(lgdtu.Protocols())
	stack.RegisterLifecycle(lgdtu)

	// Check for unclean shutdown
	if uncleanShutdowns, discards, err := rawdb.PushUncleanShutdownMarker(chainDb); err != nil {
		log.Error("Could not update unclean-shutdown-marker list", "error", err)
	} else {
		if discards > 0 {
			log.Warn("Old unclean shutdowns found", "count", discards)
		}
		for _, tstamp := range uncleanShutdowns {
			t := time.Unix(int64(tstamp), 0)
			log.Warn("Unclean shutdown detected", "booted", t,
				"age", common.PrettyAge(t))
		}
	}
	return lgdtu, nil
}

// VfluxRequest sends a batch of requests to the given node through discv5 UDP TalkRequest and returns the responses
func (s *LightGdtu) VfluxRequest(n *enode.Node, reqs vflux.Requests) vflux.Replies {
	if s.p2pServer.DiscV5 == nil {
		return nil
	}
	reqsEnc, _ := rlp.EncodeToBytes(&reqs)
	repliesEnc, _ := s.p2pServer.DiscV5.TalkRequest(s.serverPool.DialNode(n), "vfx", reqsEnc)
	var replies vflux.Replies
	if len(repliesEnc) == 0 || rlp.DecodeBytes(repliesEnc, &replies) != nil {
		return nil
	}
	return replies
}

// vfxVersion returns the version number of the "les" service subdomain of the vflux UDP
// service, as advertised in the ENR record
func (s *LightGdtu) vfxVersion(n *enode.Node) uint {
	if n.Seq() == 0 {
		var err error
		if s.p2pServer.DiscV5 == nil {
			return 0
		}
		if n, err = s.p2pServer.DiscV5.RequestENR(n); n != nil && err == nil && n.Seq() != 0 {
			s.serverPool.Persist(n)
		} else {
			return 0
		}
	}

	var les []rlp.RawValue
	if err := n.Load(enr.WithEntry("les", &les)); err != nil || len(les) < 1 {
		return 0
	}
	var version uint
	rlp.DecodeBytes(les[0], &version) // Ignore additional fields (for forward compatibility).
	return version
}

// prenegQuery sends a capacity query to the given server node to determine whether
// a connection slot is immediately available
func (s *LightGdtu) prenegQuery(n *enode.Node) int {
	if s.vfxVersion(n) < 1 {
		// UDP query not supported, always try TCP connection
		return 1
	}

	var requests vflux.Requests
	requests.Add("les", vflux.CapacityQueryName, vflux.CapacityQueryReq{
		Bias:      180,
		AddTokens: []vflux.IntOrInf{{}},
	})
	replies := s.VfluxRequest(n, requests)
	var cqr vflux.CapacityQueryReply
	if replies.Get(0, &cqr) != nil || len(cqr) != 1 { // Note: Get returns an error if replies is nil
		return -1
	}
	if cqr[0] > 0 {
		return 1
	}
	return 0
}

type LightDummyAPI struct{}

// Gdturbase is the address that mining rewards will be send to
func (s *LightDummyAPI) Gdturbase() (common.Address, error) {
	return common.Address{}, fmt.Errorf("mining is not supported in light mode")
}

// Coinbase is the address that mining rewards will be send to (alias for Gdturbase)
func (s *LightDummyAPI) Coinbase() (common.Address, error) {
	return common.Address{}, fmt.Errorf("mining is not supported in light mode")
}

// Hashrate returns the POW hashrate
func (s *LightDummyAPI) Hashrate() hexutil.Uint {
	return 0
}

// Mining returns an indication if this node is currently mining.
func (s *LightDummyAPI) Mining() bool {
	return false
}

// APIs returns the collection of RPC services the gdtu package offers.
// NOTE, some of these services probably need to be moved to somewhere else.
func (s *LightGdtu) APIs() []rpc.API {
	apis := gdtuapi.GetAPIs(s.ApiBackend)
	apis = append(apis, s.engine.APIs(s.BlockChain().HeaderChain())...)
	return append(apis, []rpc.API{
		{
			Namespace: "gdtu",
			Version:   "1.0",
			Service:   &LightDummyAPI{},
			Public:    true,
		}, {
			Namespace: "gdtu",
			Version:   "1.0",
			Service:   downloader.NewPublicDownloaderAPI(s.handler.downloader, s.eventMux),
			Public:    true,
		}, {
			Namespace: "gdtu",
			Version:   "1.0",
			Service:   filters.NewPublicFilterAPI(s.ApiBackend, true, 5*time.Minute),
			Public:    true,
		}, {
			Namespace: "net",
			Version:   "1.0",
			Service:   s.netRPCService,
			Public:    true,
		}, {
			Namespace: "les",
			Version:   "1.0",
			Service:   NewPrivateLightAPI(&s.lesCommons),
			Public:    false,
		}, {
			Namespace: "vflux",
			Version:   "1.0",
			Service:   s.serverPool.API(),
			Public:    false,
		},
	}...)
}

func (s *LightGdtu) ResetWithGenesisBlock(gb *types.Block) {
	s.blockchain.ResetWithGenesisBlock(gb)
}

func (s *LightGdtu) BlockChain() *light.LightChain      { return s.blockchain }
func (s *LightGdtu) TxPool() *light.TxPool              { return s.txPool }
func (s *LightGdtu) Engine() consensus.Engine           { return s.engine }
func (s *LightGdtu) LesVersion() int                    { return int(ClientProtocolVersions[0]) }
func (s *LightGdtu) Downloader() *downloader.Downloader { return s.handler.downloader }
func (s *LightGdtu) EventMux() *event.TypeMux           { return s.eventMux }

// Protocols returns all the currently configured network protocols to start.
func (s *LightGdtu) Protocols() []p2p.Protocol {
	return s.makeProtocols(ClientProtocolVersions, s.handler.runPeer, func(id enode.ID) interface{} {
		if p := s.peers.peer(id.String()); p != nil {
			return p.Info()
		}
		return nil
	}, s.serverPoolIterator)
}

// Start implements node.Lifecycle, starting all internal goroutines needed by the
// light gdtu protocol implementation.
func (s *LightGdtu) Start() error {
	log.Warn("Light client mode is an experimental feature")

	discovery, err := s.setupDiscovery(s.p2pConfig)
	if err != nil {
		return err
	}
	s.serverPool.AddSource(discovery)
	s.serverPool.Start()
	// Start bloom request workers.
	s.wg.Add(bloomServiceThreads)
	s.startBloomHandlers(params.BloomBitsBlocksClient)
	s.handler.start()

	return nil
}

// Stop implements node.Lifecycle, terminating all internal goroutines used by the
// Gdtu protocol.
func (s *LightGdtu) Stop() error {
	close(s.closeCh)
	s.serverPool.Stop()
	s.peers.close()
	s.reqDist.close()
	s.odr.Stop()
	s.relay.Stop()
	s.bloomIndexer.Close()
	s.chtIndexer.Close()
	s.blockchain.Stop()
	s.handler.stop()
	s.txPool.Stop()
	s.engine.Close()
	s.pruner.close()
	s.eventMux.Stop()
	rawdb.PopUncleanShutdownMarker(s.chainDb)
	s.chainDb.Close()
	s.lesDb.Close()
	s.wg.Wait()
	log.Info("Light gdtu stopped")
	return nil
}
