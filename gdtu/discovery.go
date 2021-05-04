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

package gdtu

import (
	"github.com/c88032111/go-gdtu/core"
	"github.com/c88032111/go-gdtu/core/forkid"
	"github.com/c88032111/go-gdtu/p2p/dnsdisc"
	"github.com/c88032111/go-gdtu/p2p/enode"
	"github.com/c88032111/go-gdtu/rlp"
)

// gdtuEntry is the "gdtu" ENR entry which advertises gdtu protocol
// on the discovery network.
type gdtuEntry struct {
	ForkID forkid.ID // Fork identifier per EIP-2124

	// Ignore additional fields (for forward compatibility).
	Rest []rlp.RawValue `rlp:"tail"`
}

// ENRKey implements enr.Entry.
func (e gdtuEntry) ENRKey() string {
	return "gdtu"
}

// startGdtuEntryUpdate starts the ENR updater loop.
func (gdtu *Gdtu) startGdtuEntryUpdate(ln *enode.LocalNode) {
	var newHead = make(chan core.ChainHeadEvent, 10)
	sub := gdtu.blockchain.SubscribeChainHeadEvent(newHead)

	go func() {
		defer sub.Unsubscribe()
		for {
			select {
			case <-newHead:
				ln.Set(gdtu.currentGdtuEntry())
			case <-sub.Err():
				// Would be nice to sync with gdtu.Stop, but there is no
				// good way to do that.
				return
			}
		}
	}()
}

func (gdtu *Gdtu) currentGdtuEntry() *gdtuEntry {
	return &gdtuEntry{ForkID: forkid.NewID(gdtu.blockchain.Config(), gdtu.blockchain.Genesis().Hash(),
		gdtu.blockchain.CurrentHeader().Number.Uint64())}
}

// setupDiscovery creates the node discovery source for the `gdtu` and `snap`
// protocols.
func setupDiscovery(urls []string) (enode.Iterator, error) {
	if len(urls) == 0 {
		return nil, nil
	}
	client := dnsdisc.NewClient(dnsdisc.Config{})
	return client.NewIterator(urls...)
}
