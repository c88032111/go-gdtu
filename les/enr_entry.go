// Copyright 2019 The go-gdtu Authors
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

package les

import (
	"github.com/c88032111/go-gdtu/core/forkid"
	"github.com/c88032111/go-gdtu/p2p"
	"github.com/c88032111/go-gdtu/p2p/dnsdisc"
	"github.com/c88032111/go-gdtu/p2p/enode"
	"github.com/c88032111/go-gdtu/rlp"
)

// lesEntry is the "les" ENR entry. This is set for LES servers only.
type lesEntry struct {
	// Ignore additional fields (for forward compatibility).
	VfxVersion uint
	Rest       []rlp.RawValue `rlp:"tail"`
}

func (lesEntry) ENRKey() string { return "les" }

// gdtuEntry is the "gdtu" ENR entry. This is redeclared here to avoid depending on package gdtu.
type gdtuEntry struct {
	ForkID forkid.ID
	_      []rlp.RawValue `rlp:"tail"`
}

func (gdtuEntry) ENRKey() string { return "gdtu" }

// setupDiscovery creates the node discovery source for the gdtu protocol.
func (gdtu *LightGdtu) setupDiscovery(cfg *p2p.Config) (enode.Iterator, error) {
	it := enode.NewFairMix(0)

	// Enable DNS discovery.
	if len(gdtu.config.GdtuDiscoveryURLs) != 0 {
		client := dnsdisc.NewClient(dnsdisc.Config{})
		dns, err := client.NewIterator(gdtu.config.GdtuDiscoveryURLs...)
		if err != nil {
			return nil, err
		}
		it.AddSource(dns)
	}

	// Enable DHT.
	if cfg.DiscoveryV5 && gdtu.p2pServer.DiscV5 != nil {
		it.AddSource(gdtu.p2pServer.DiscV5.RandomNodes())
	}

	forkFilter := forkid.NewFilter(gdtu.blockchain)
	iterator := enode.Filter(it, func(n *enode.Node) bool { return nodeIsServer(forkFilter, n) })
	return iterator, nil
}

// nodeIsServer checks whether n is an LES server node.
func nodeIsServer(forkFilter forkid.Filter, n *enode.Node) bool {
	var les lesEntry
	var gdtu gdtuEntry
	return n.Load(&les) == nil && n.Load(&gdtu) == nil && forkFilter(gdtu.ForkID) == nil
}
