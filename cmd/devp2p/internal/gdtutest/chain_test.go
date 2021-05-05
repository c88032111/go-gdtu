// Copyright 2020 The go-gdtu Authors
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

package gdtutest

import (
	"path/filepath"
	"strconv"
	"testing"

	"github.com/c88032111/go-gdtu/gdtu/protocols/gdtu"
	"github.com/c88032111/go-gdtu/p2p"
	"github.com/stretchr/testify/assert"
)

// TestGdtuProtocolNegotiation tests whether the test suite
// can negotiate the highest gdtu protocol in a status message exchange
func TestGdtuProtocolNegotiation(t *testing.T) {
	var tests = []struct {
		conn     *Conn
		caps     []p2p.Cap
		expected uint32
	}{
		{
			conn: &Conn{},
			caps: []p2p.Cap{
				{Name: "gdtu", Version: 63},
				{Name: "gdtu", Version: 64},
				{Name: "gdtu", Version: 65},
			},
			expected: uint32(65),
		},
		{
			conn: &Conn{},
			caps: []p2p.Cap{
				{Name: "gdtu", Version: 0},
				{Name: "gdtu", Version: 89},
				{Name: "gdtu", Version: 65},
			},
			expected: uint32(65),
		},
		{
			conn: &Conn{},
			caps: []p2p.Cap{
				{Name: "gdtu", Version: 63},
				{Name: "gdtu", Version: 64},
				{Name: "wrgdtuProto", Version: 65},
			},
			expected: uint32(64),
		},
	}

	for i, tt := range tests {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			tt.conn.negotiateGdtuProtocol(tt.caps)
			assert.Equal(t, tt.expected, uint32(tt.conn.gdtuProtocolVersion))
		})
	}
}

// TestChain_GetHeaders tests whether the test suite can correctly
// respond to a GetBlockHeaders request from a node.
func TestChain_GetHeaders(t *testing.T) {
	chainFile, err := filepath.Abs("./testdata/chain.rlp")
	if err != nil {
		t.Fatal(err)
	}
	genesisFile, err := filepath.Abs("./testdata/genesis.json")
	if err != nil {
		t.Fatal(err)
	}

	chain, err := loadChain(chainFile, genesisFile)
	if err != nil {
		t.Fatal(err)
	}

	var tests = []struct {
		req      GetBlockHeaders
		expected BlockHeaders
	}{
		{
			req: GetBlockHeaders{
				Origin: gdtu.HashOrNumber{
					Number: uint64(2),
				},
				Amount:  uint64(5),
				Skip:    1,
				Reverse: false,
			},
			expected: BlockHeaders{
				chain.blocks[2].Header(),
				chain.blocks[4].Header(),
				chain.blocks[6].Header(),
				chain.blocks[8].Header(),
				chain.blocks[10].Header(),
			},
		},
		{
			req: GetBlockHeaders{
				Origin: gdtu.HashOrNumber{
					Number: uint64(chain.Len() - 1),
				},
				Amount:  uint64(3),
				Skip:    0,
				Reverse: true,
			},
			expected: BlockHeaders{
				chain.blocks[chain.Len()-1].Header(),
				chain.blocks[chain.Len()-2].Header(),
				chain.blocks[chain.Len()-3].Header(),
			},
		},
		{
			req: GetBlockHeaders{
				Origin: gdtu.HashOrNumber{
					Hash: chain.Head().Hash(),
				},
				Amount:  uint64(1),
				Skip:    0,
				Reverse: false,
			},
			expected: BlockHeaders{
				chain.Head().Header(),
			},
		},
	}

	for i, tt := range tests {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			headers, err := chain.GetHeaders(tt.req)
			if err != nil {
				t.Fatal(err)
			}
			assert.Equal(t, headers, tt.expected)
		})
	}
}
