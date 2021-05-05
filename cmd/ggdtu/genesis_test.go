// Copyright 2016 The go-gdtu Authors
// This file is part of go-gdtu.
//
// go-gdtu is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// go-gdtu is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// algdtu with go-gdtu. If not, see <http://www.gnu.org/licenses/>.

package main

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
)

var customGenesisTests = []struct {
	genesis string
	query   string
	result  string
}{
	// Genesis file with an empty chain configuration (ensure missing fields work)
	{
		genesis: `{
			"alloc"      : {},
			"coinbase"   : "gd0000000000000000000000000000000000000000",
			"difficulty" : "gd20000",
			"extraData"  : "",
			"gasLimit"   : "gd2fefd8",
			"nonce"      : "gd0000000000001338",
			"mixhash"    : "gd0000000000000000000000000000000000000000000000000000000000000000",
			"parentHash" : "gd0000000000000000000000000000000000000000000000000000000000000000",
			"timestamp"  : "gd00",
			"config"     : {}
		}`,
		query:  "gdtu.getBlock(0).nonce",
		result: "gd0000000000001338",
	},
	// Genesis file with specific chain configurations
	{
		genesis: `{
			"alloc"      : {},
			"coinbase"   : "gd0000000000000000000000000000000000000000",
			"difficulty" : "gd20000",
			"extraData"  : "",
			"gasLimit"   : "gd2fefd8",
			"nonce"      : "gd0000000000001339",
			"mixhash"    : "gd0000000000000000000000000000000000000000000000000000000000000000",
			"parentHash" : "gd0000000000000000000000000000000000000000000000000000000000000000",
			"timestamp"  : "gd00",
			"config"     : {
				"homesteadBlock" : 42,
				"daoForkBlock"   : 141,
				"daoForkSupport" : true
			}
		}`,
		query:  "gdtu.getBlock(0).nonce",
		result: "gd0000000000001339",
	},
}

// Tests that initializing Ggdtu with a custom genesis block and chain definitions
// work properly.
func TestCustomGenesis(t *testing.T) {
	for i, tt := range customGenesisTests {
		// Create a temporary data directory to use and inspect later
		datadir := tmpdir(t)
		defer os.RemoveAll(datadir)

		// Initialize the data directory with the custom genesis block
		json := filepath.Join(datadir, "genesis.json")
		if err := ioutil.WriteFile(json, []byte(tt.genesis), 0600); err != nil {
			t.Fatalf("test %d: failed to write genesis file: %v", i, err)
		}
		runGgdtu(t, "--datadir", datadir, "init", json).WaitExit()

		// Query the custom genesis block
		ggdtu := runGgdtu(t, "--networkid", "1337", "--syncmode=full",
			"--datadir", datadir, "--maxpeers", "0", "--port", "0",
			"--nodiscover", "--nat", "none", "--ipcdisable",
			"--exec", tt.query, "console")
		ggdtu.ExpectRegexp(tt.result)
		ggdtu.ExpectExit()
	}
}
