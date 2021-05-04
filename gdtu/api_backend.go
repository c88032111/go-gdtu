// Copyright 2015 The go-gdtu Authors
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

package gdtu

import (
	"context"
	"errors"
	"math/big"

	"github.com/c88032111/go-gdtu/accounts"
	"github.com/c88032111/go-gdtu/common"
	"github.com/c88032111/go-gdtu/consensus"
	"github.com/c88032111/go-gdtu/core"
	"github.com/c88032111/go-gdtu/core/bloombits"
	"github.com/c88032111/go-gdtu/core/rawdb"
	"github.com/c88032111/go-gdtu/core/state"
	"github.com/c88032111/go-gdtu/core/types"
	"github.com/c88032111/go-gdtu/core/vm"
	"github.com/c88032111/go-gdtu/event"
	"github.com/c88032111/go-gdtu/gdtu/downloader"
	"github.com/c88032111/go-gdtu/gdtu/gasprice"
	"github.com/c88032111/go-gdtu/gdtudb"
	"github.com/c88032111/go-gdtu/miner"
	"github.com/c88032111/go-gdtu/params"
	"github.com/c88032111/go-gdtu/rpc"
)

// GdtuAPIBackend implements gdtuapi.Backend for full nodes
type GdtuAPIBackend struct {
	extRPCEnabled       bool
	allowUnprotectedTxs bool
	gdtu                *Gdtu
	gpo                 *gasprice.Oracle
}

// ChainConfig returns the active chain configuration.
func (b *GdtuAPIBackend) ChainConfig() *params.ChainConfig {
	return b.gdtu.blockchain.Config()
}

func (b *GdtuAPIBackend) CurrentBlock() *types.Block {
	return b.gdtu.blockchain.CurrentBlock()
}

func (b *GdtuAPIBackend) SetHead(number uint64) {
	b.gdtu.handler.downloader.Cancel()
	b.gdtu.blockchain.SetHead(number)
}

func (b *GdtuAPIBackend) HeaderByNumber(ctx context.Context, number rpc.BlockNumber) (*types.Header, error) {
	// Pending block is only known by the miner
	if number == rpc.PendingBlockNumber {
		block := b.gdtu.miner.PendingBlock()
		return block.Header(), nil
	}
	// Otherwise resolve and return the block
	if number == rpc.LatestBlockNumber {
		return b.gdtu.blockchain.CurrentBlock().Header(), nil
	}
	return b.gdtu.blockchain.GetHeaderByNumber(uint64(number)), nil
}

func (b *GdtuAPIBackend) HeaderByNumberOrHash(ctx context.Context, blockNrOrHash rpc.BlockNumberOrHash) (*types.Header, error) {
	if blockNr, ok := blockNrOrHash.Number(); ok {
		return b.HeaderByNumber(ctx, blockNr)
	}
	if hash, ok := blockNrOrHash.Hash(); ok {
		header := b.gdtu.blockchain.GetHeaderByHash(hash)
		if header == nil {
			return nil, errors.New("header for hash not found")
		}
		if blockNrOrHash.RequireCanonical && b.gdtu.blockchain.GetCanonicalHash(header.Number.Uint64()) != hash {
			return nil, errors.New("hash is not currently canonical")
		}
		return header, nil
	}
	return nil, errors.New("invalid arguments; neither block nor hash specified")
}

func (b *GdtuAPIBackend) HeaderByHash(ctx context.Context, hash common.Hash) (*types.Header, error) {
	return b.gdtu.blockchain.GetHeaderByHash(hash), nil
}

func (b *GdtuAPIBackend) BlockByNumber(ctx context.Context, number rpc.BlockNumber) (*types.Block, error) {
	// Pending block is only known by the miner
	if number == rpc.PendingBlockNumber {
		block := b.gdtu.miner.PendingBlock()
		return block, nil
	}
	// Otherwise resolve and return the block
	if number == rpc.LatestBlockNumber {
		return b.gdtu.blockchain.CurrentBlock(), nil
	}
	return b.gdtu.blockchain.GetBlockByNumber(uint64(number)), nil
}

func (b *GdtuAPIBackend) BlockByHash(ctx context.Context, hash common.Hash) (*types.Block, error) {
	return b.gdtu.blockchain.GetBlockByHash(hash), nil
}

func (b *GdtuAPIBackend) BlockByNumberOrHash(ctx context.Context, blockNrOrHash rpc.BlockNumberOrHash) (*types.Block, error) {
	if blockNr, ok := blockNrOrHash.Number(); ok {
		return b.BlockByNumber(ctx, blockNr)
	}
	if hash, ok := blockNrOrHash.Hash(); ok {
		header := b.gdtu.blockchain.GetHeaderByHash(hash)
		if header == nil {
			return nil, errors.New("header for hash not found")
		}
		if blockNrOrHash.RequireCanonical && b.gdtu.blockchain.GetCanonicalHash(header.Number.Uint64()) != hash {
			return nil, errors.New("hash is not currently canonical")
		}
		block := b.gdtu.blockchain.GetBlock(hash, header.Number.Uint64())
		if block == nil {
			return nil, errors.New("header found, but block body is missing")
		}
		return block, nil
	}
	return nil, errors.New("invalid arguments; neither block nor hash specified")
}

func (b *GdtuAPIBackend) StateAndHeaderByNumber(ctx context.Context, number rpc.BlockNumber) (*state.StateDB, *types.Header, error) {
	// Pending state is only known by the miner
	if number == rpc.PendingBlockNumber {
		block, state := b.gdtu.miner.Pending()
		return state, block.Header(), nil
	}
	// Otherwise resolve the block number and return its state
	header, err := b.HeaderByNumber(ctx, number)
	if err != nil {
		return nil, nil, err
	}
	if header == nil {
		return nil, nil, errors.New("header not found")
	}
	stateDb, err := b.gdtu.BlockChain().StateAt(header.Root)
	return stateDb, header, err
}

func (b *GdtuAPIBackend) StateAndHeaderByNumberOrHash(ctx context.Context, blockNrOrHash rpc.BlockNumberOrHash) (*state.StateDB, *types.Header, error) {
	if blockNr, ok := blockNrOrHash.Number(); ok {
		return b.StateAndHeaderByNumber(ctx, blockNr)
	}
	if hash, ok := blockNrOrHash.Hash(); ok {
		header, err := b.HeaderByHash(ctx, hash)
		if err != nil {
			return nil, nil, err
		}
		if header == nil {
			return nil, nil, errors.New("header for hash not found")
		}
		if blockNrOrHash.RequireCanonical && b.gdtu.blockchain.GetCanonicalHash(header.Number.Uint64()) != hash {
			return nil, nil, errors.New("hash is not currently canonical")
		}
		stateDb, err := b.gdtu.BlockChain().StateAt(header.Root)
		return stateDb, header, err
	}
	return nil, nil, errors.New("invalid arguments; neither block nor hash specified")
}

func (b *GdtuAPIBackend) GetReceipts(ctx context.Context, hash common.Hash) (types.Receipts, error) {
	return b.gdtu.blockchain.GetReceiptsByHash(hash), nil
}

func (b *GdtuAPIBackend) GetLogs(ctx context.Context, hash common.Hash) ([][]*types.Log, error) {
	receipts := b.gdtu.blockchain.GetReceiptsByHash(hash)
	if receipts == nil {
		return nil, nil
	}
	logs := make([][]*types.Log, len(receipts))
	for i, receipt := range receipts {
		logs[i] = receipt.Logs
	}
	return logs, nil
}

func (b *GdtuAPIBackend) GetTd(ctx context.Context, hash common.Hash) *big.Int {
	return b.gdtu.blockchain.GetTdByHash(hash)
}

func (b *GdtuAPIBackend) GetEVM(ctx context.Context, msg core.Message, state *state.StateDB, header *types.Header) (*vm.EVM, func() error, error) {
	vmError := func() error { return nil }

	txContext := core.NewEVMTxContext(msg)
	context := core.NewEVMBlockContext(header, b.gdtu.BlockChain(), nil)
	return vm.NewEVM(context, txContext, state, b.gdtu.blockchain.Config(), *b.gdtu.blockchain.GetVMConfig()), vmError, nil
}

func (b *GdtuAPIBackend) SubscribeRemovedLogsEvent(ch chan<- core.RemovedLogsEvent) event.Subscription {
	return b.gdtu.BlockChain().SubscribeRemovedLogsEvent(ch)
}

func (b *GdtuAPIBackend) SubscribePendingLogsEvent(ch chan<- []*types.Log) event.Subscription {
	return b.gdtu.miner.SubscribePendingLogs(ch)
}

func (b *GdtuAPIBackend) SubscribeChainEvent(ch chan<- core.ChainEvent) event.Subscription {
	return b.gdtu.BlockChain().SubscribeChainEvent(ch)
}

func (b *GdtuAPIBackend) SubscribeChainHeadEvent(ch chan<- core.ChainHeadEvent) event.Subscription {
	return b.gdtu.BlockChain().SubscribeChainHeadEvent(ch)
}

func (b *GdtuAPIBackend) SubscribeChainSideEvent(ch chan<- core.ChainSideEvent) event.Subscription {
	return b.gdtu.BlockChain().SubscribeChainSideEvent(ch)
}

func (b *GdtuAPIBackend) SubscribeLogsEvent(ch chan<- []*types.Log) event.Subscription {
	return b.gdtu.BlockChain().SubscribeLogsEvent(ch)
}

func (b *GdtuAPIBackend) SendTx(ctx context.Context, signedTx *types.Transaction) error {
	return b.gdtu.txPool.AddLocal(signedTx)
}

func (b *GdtuAPIBackend) GetPoolTransactions() (types.Transactions, error) {
	pending, err := b.gdtu.txPool.Pending()
	if err != nil {
		return nil, err
	}
	var txs types.Transactions
	for _, batch := range pending {
		txs = append(txs, batch...)
	}
	return txs, nil
}

func (b *GdtuAPIBackend) GetPoolTransaction(hash common.Hash) *types.Transaction {
	return b.gdtu.txPool.Get(hash)
}

func (b *GdtuAPIBackend) GetTransaction(ctx context.Context, txHash common.Hash) (*types.Transaction, common.Hash, uint64, uint64, error) {
	tx, blockHash, blockNumber, index := rawdb.ReadTransaction(b.gdtu.ChainDb(), txHash)
	return tx, blockHash, blockNumber, index, nil
}

func (b *GdtuAPIBackend) GetPoolNonce(ctx context.Context, addr common.Address) (uint64, error) {
	return b.gdtu.txPool.Nonce(addr), nil
}

func (b *GdtuAPIBackend) Stats() (pending int, queued int) {
	return b.gdtu.txPool.Stats()
}

func (b *GdtuAPIBackend) TxPoolContent() (map[common.Address]types.Transactions, map[common.Address]types.Transactions) {
	return b.gdtu.TxPool().Content()
}

func (b *GdtuAPIBackend) TxPool() *core.TxPool {
	return b.gdtu.TxPool()
}

func (b *GdtuAPIBackend) SubscribeNewTxsEvent(ch chan<- core.NewTxsEvent) event.Subscription {
	return b.gdtu.TxPool().SubscribeNewTxsEvent(ch)
}

func (b *GdtuAPIBackend) Downloader() *downloader.Downloader {
	return b.gdtu.Downloader()
}

func (b *GdtuAPIBackend) SuggestPrice(ctx context.Context) (*big.Int, error) {
	return b.gpo.SuggestPrice(ctx)
}

func (b *GdtuAPIBackend) ChainDb() gdtudb.Database {
	return b.gdtu.ChainDb()
}

func (b *GdtuAPIBackend) EventMux() *event.TypeMux {
	return b.gdtu.EventMux()
}

func (b *GdtuAPIBackend) AccountManager() *accounts.Manager {
	return b.gdtu.AccountManager()
}

func (b *GdtuAPIBackend) ExtRPCEnabled() bool {
	return b.extRPCEnabled
}

func (b *GdtuAPIBackend) UnprotectedAllowed() bool {
	return b.allowUnprotectedTxs
}

func (b *GdtuAPIBackend) RPCGasCap() uint64 {
	return b.gdtu.config.RPCGasCap
}

func (b *GdtuAPIBackend) RPCTxFeeCap() float64 {
	return b.gdtu.config.RPCTxFeeCap
}

func (b *GdtuAPIBackend) BloomStatus() (uint64, uint64) {
	sections, _, _ := b.gdtu.bloomIndexer.Sections()
	return params.BloomBitsBlocks, sections
}

func (b *GdtuAPIBackend) ServiceFilter(ctx context.Context, session *bloombits.MatcherSession) {
	for i := 0; i < bloomFilterThreads; i++ {
		go session.Multiplex(bloomRetrievalBatch, bloomRetrievalWait, b.gdtu.bloomRequests)
	}
}

func (b *GdtuAPIBackend) Engine() consensus.Engine {
	return b.gdtu.engine
}

func (b *GdtuAPIBackend) CurrentHeader() *types.Header {
	return b.gdtu.blockchain.CurrentHeader()
}

func (b *GdtuAPIBackend) Miner() *miner.Miner {
	return b.gdtu.Miner()
}

func (b *GdtuAPIBackend) StartMining(threads int) error {
	return b.gdtu.StartMining(threads)
}

func (b *GdtuAPIBackend) StateAtBlock(ctx context.Context, block *types.Block, reexec uint64) (*state.StateDB, func(), error) {
	return b.gdtu.stateAtBlock(block, reexec)
}

func (b *GdtuAPIBackend) StatesInRange(ctx context.Context, fromBlock *types.Block, toBlock *types.Block, reexec uint64) ([]*state.StateDB, func(), error) {
	return b.gdtu.statesInRange(fromBlock, toBlock, reexec)
}

func (b *GdtuAPIBackend) StateAtTransaction(ctx context.Context, block *types.Block, txIndex int, reexec uint64) (core.Message, vm.BlockContext, *state.StateDB, func(), error) {
	return b.gdtu.stateAtTransaction(block, txIndex, reexec)
}
