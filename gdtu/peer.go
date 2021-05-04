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
	"math/big"
	"sync"
	"time"

	"github.com/c88032111/go-gdtu/gdtu/protocols/gdtu"
	"github.com/c88032111/go-gdtu/gdtu/protocols/snap"
)

// gdtuPeerInfo represents a short summary of the `gdtu` sub-protocol metadata known
// about a connected peer.
type gdtuPeerInfo struct {
	Version    uint     `json:"version"`    // Gdtu protocol version negotiated
	Difficulty *big.Int `json:"difficulty"` // Total difficulty of the peer's blockchain
	Head       string   `json:"head"`       // Hex hash of the peer's best owned block
}

// gdtuPeer is a wrapper around gdtu.Peer to maintain a few extra metadata.
type gdtuPeer struct {
	*gdtu.Peer
	snapExt *snapPeer // Satellite `snap` connection

	syncDrop *time.Timer   // Connection dropper if `gdtu` sync progress isn't validated in time
	snapWait chan struct{} // Notification channel for snap connections
	lock     sync.RWMutex  // Mutex protecting the internal fields
}

// info gathers and returns some `gdtu` protocol metadata known about a peer.
func (p *gdtuPeer) info() *gdtuPeerInfo {
	hash, td := p.Head()

	return &gdtuPeerInfo{
		Version:    p.Version(),
		Difficulty: td,
		Head:       hash.Hex(),
	}
}

// snapPeerInfo represents a short summary of the `snap` sub-protocol metadata known
// about a connected peer.
type snapPeerInfo struct {
	Version uint `json:"version"` // Snapshot protocol version negotiated
}

// snapPeer is a wrapper around snap.Peer to maintain a few extra metadata.
type snapPeer struct {
	*snap.Peer
}

// info gathers and returns some `snap` protocol metadata known about a peer.
func (p *snapPeer) info() *snapPeerInfo {
	return &snapPeerInfo{
		Version: p.Version(),
	}
}
