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
	"errors"
	"fmt"
	"math/big"
	"sync/atomic"
	"time"

	"github.com/c88032111/go-gdtu/common"
	"github.com/c88032111/go-gdtu/core"
	"github.com/c88032111/go-gdtu/core/types"
	"github.com/c88032111/go-gdtu/gdtu/protocols/gdtu"
	"github.com/c88032111/go-gdtu/log"
	"github.com/c88032111/go-gdtu/p2p/enode"
	"github.com/c88032111/go-gdtu/trie"
)

// gdtuHandler implements the gdtu.Backend interface to handle the various network
// packets that are sent as replies or broadcasts.
type gdtuHandler handler

func (h *gdtuHandler) Chain() *core.BlockChain     { return h.chain }
func (h *gdtuHandler) StateBloom() *trie.SyncBloom { return h.stateBloom }
func (h *gdtuHandler) TxPool() gdtu.TxPool         { return h.txpool }

// RunPeer is invoked when a peer joins on the `gdtu` protocol.
func (h *gdtuHandler) RunPeer(peer *gdtu.Peer, hand gdtu.Handler) error {
	return (*handler)(h).runGdtuPeer(peer, hand)
}

// PeerInfo retrieves all known `gdtu` information about a peer.
func (h *gdtuHandler) PeerInfo(id enode.ID) interface{} {
	if p := h.peers.peer(id.String()); p != nil {
		return p.info()
	}
	return nil
}

// AcceptTxs retrieves whether transaction processing is enabled on the node
// or if inbound transactions should simply be dropped.
func (h *gdtuHandler) AcceptTxs() bool {
	return atomic.LoadUint32(&h.acceptTxs) == 1
}

// Handle is invoked from a peer's message handler when it receives a new remote
// message that the handler couldn't consume and serve itself.
func (h *gdtuHandler) Handle(peer *gdtu.Peer, packet gdtu.Packet) error {
	// Consume any broadcasts and announces, forwarding the rest to the downloader
	switch packet := packet.(type) {
	case *gdtu.BlockHeadersPacket:
		return h.handleHeaders(peer, *packet)

	case *gdtu.BlockBodiesPacket:
		txset, uncleset := packet.Unpack()
		return h.handleBodies(peer, txset, uncleset)

	case *gdtu.NodeDataPacket:
		if err := h.downloader.DeliverNodeData(peer.ID(), *packet); err != nil {
			log.Debug("Failed to deliver node state data", "err", err)
		}
		return nil

	case *gdtu.ReceiptsPacket:
		if err := h.downloader.DeliverReceipts(peer.ID(), *packet); err != nil {
			log.Debug("Failed to deliver receipts", "err", err)
		}
		return nil

	case *gdtu.NewBlockHashesPacket:
		hashes, numbers := packet.Unpack()
		return h.handleBlockAnnounces(peer, hashes, numbers)

	case *gdtu.NewBlockPacket:
		return h.handleBlockBroadcast(peer, packet.Block, packet.TD)

	case *gdtu.NewPooledTransactionHashesPacket:
		return h.txFetcher.Notify(peer.ID(), *packet)

	case *gdtu.TransactionsPacket:
		return h.txFetcher.Enqueue(peer.ID(), *packet, false)

	case *gdtu.PooledTransactionsPacket:
		return h.txFetcher.Enqueue(peer.ID(), *packet, true)

	default:
		return fmt.Errorf("unexpected gdtu packet type: %T", packet)
	}
}

// handleHeaders is invoked from a peer's message handler when it transmits a batch
// of headers for the local node to process.
func (h *gdtuHandler) handleHeaders(peer *gdtu.Peer, headers []*types.Header) error {
	p := h.peers.peer(peer.ID())
	if p == nil {
		return errors.New("unregistered during callback")
	}
	// If no headers were received, but we're expencting a checkpoint header, consider it that
	if len(headers) == 0 && p.syncDrop != nil {
		// Stop the timer either way, decide later to drop or not
		p.syncDrop.Stop()
		p.syncDrop = nil

		// If we're doing a fast (or snap) sync, we must enforce the checkpoint block to avoid
		// eclipse attacks. Unsynced nodes are welcome to connect after we're done
		// joining the network
		if atomic.LoadUint32(&h.fastSync) == 1 {
			peer.Log().Warn("Dropping unsynced node during sync", "addr", peer.RemoteAddr(), "type", peer.Name())
			return errors.New("unsynced node cannot serve sync")
		}
	}
	// Filter out any explicitly requested headers, deliver the rest to the downloader
	filter := len(headers) == 1
	if filter {
		// If it's a potential sync progress check, validate the content and advertised chain weight
		if p.syncDrop != nil && headers[0].Number.Uint64() == h.checkpointNumber {
			// Disable the sync drop timer
			p.syncDrop.Stop()
			p.syncDrop = nil

			// Validate the header and either drop the peer or continue
			if headers[0].Hash() != h.checkpointHash {
				return errors.New("checkpoint hash mismatch")
			}
			return nil
		}
		// Otherwise if it's a whitelisted block, validate against the set
		if want, ok := h.whitelist[headers[0].Number.Uint64()]; ok {
			if hash := headers[0].Hash(); want != hash {
				peer.Log().Info("Whitelist mismatch, dropping peer", "number", headers[0].Number.Uint64(), "hash", hash, "want", want)
				return errors.New("whitelist block mismatch")
			}
			peer.Log().Debug("Whitelist block verified", "number", headers[0].Number.Uint64(), "hash", want)
		}
		// Irrelevant of the fork checks, send the header to the fetcher just in case
		headers = h.blockFetcher.FilterHeaders(peer.ID(), headers, time.Now())
	}
	if len(headers) > 0 || !filter {
		err := h.downloader.DeliverHeaders(peer.ID(), headers)
		if err != nil {
			log.Debug("Failed to deliver headers", "err", err)
		}
	}
	return nil
}

// handleBodies is invoked from a peer's message handler when it transmits a batch
// of block bodies for the local node to process.
func (h *gdtuHandler) handleBodies(peer *gdtu.Peer, txs [][]*types.Transaction, uncles [][]*types.Header) error {
	// Filter out any explicitly requested bodies, deliver the rest to the downloader
	filter := len(txs) > 0 || len(uncles) > 0
	if filter {
		txs, uncles = h.blockFetcher.FilterBodies(peer.ID(), txs, uncles, time.Now())
	}
	if len(txs) > 0 || len(uncles) > 0 || !filter {
		err := h.downloader.DeliverBodies(peer.ID(), txs, uncles)
		if err != nil {
			log.Debug("Failed to deliver bodies", "err", err)
		}
	}
	return nil
}

// handleBlockAnnounces is invoked from a peer's message handler when it transmits a
// batch of block announcements for the local node to process.
func (h *gdtuHandler) handleBlockAnnounces(peer *gdtu.Peer, hashes []common.Hash, numbers []uint64) error {
	// Schedule all the unknown hashes for retrieval
	var (
		unknownHashes  = make([]common.Hash, 0, len(hashes))
		unknownNumbers = make([]uint64, 0, len(numbers))
	)
	for i := 0; i < len(hashes); i++ {
		if !h.chain.HasBlock(hashes[i], numbers[i]) {
			unknownHashes = append(unknownHashes, hashes[i])
			unknownNumbers = append(unknownNumbers, numbers[i])
		}
	}
	for i := 0; i < len(unknownHashes); i++ {
		h.blockFetcher.Notify(peer.ID(), unknownHashes[i], unknownNumbers[i], time.Now(), peer.RequestOneHeader, peer.RequestBodies)
	}
	return nil
}

// handleBlockBroadcast is invoked from a peer's message handler when it transmits a
// block broadcast for the local node to process.
func (h *gdtuHandler) handleBlockBroadcast(peer *gdtu.Peer, block *types.Block, td *big.Int) error {
	// Schedule the block for import
	h.blockFetcher.Enqueue(peer.ID(), block)

	// Assuming the block is importable by the peer, but possibly not yet done so,
	// calculate the head hash and TD that the peer truly must have.
	var (
		trueHead = block.ParentHash()
		trueTD   = new(big.Int).Sub(td, block.Difficulty())
	)
	// Update the peer's total difficulty if better than the previous
	if _, td := peer.Head(); trueTD.Cmp(td) > 0 {
		peer.SetHead(trueHead, trueTD)
		h.chainSync.handlePeerEvent(peer)
	}
	return nil
}
