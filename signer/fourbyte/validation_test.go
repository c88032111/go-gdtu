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

package fourbyte

import (
	"math/big"
	"testing"

	"github.com/c88032111/go-gdtu/common"
	"github.com/c88032111/go-gdtu/common/hexutil"
	"github.com/c88032111/go-gdtu/signer/core"
)

func mixAddr(a string) (*common.MixedcaseAddress, error) {
	return common.NewMixedcaseAddressFromString(a)
}
func toHexBig(h string) hexutil.Big {
	b := big.NewInt(0).SetBytes(common.FromHex(h))
	return hexutil.Big(*b)
}
func toHexUint(h string) hexutil.Uint64 {
	b := big.NewInt(0).SetBytes(common.FromHex(h))
	return hexutil.Uint64(b.Uint64())
}
func dummyTxArgs(t txtestcase) *core.SendTxArgs {
	to, _ := mixAddr(t.to)
	from, _ := mixAddr(t.from)
	n := toHexUint(t.n)
	gas := toHexUint(t.g)
	gasPrice := toHexBig(t.gp)
	value := toHexBig(t.value)
	var (
		data, input *hexutil.Bytes
	)
	if t.d != "" {
		a := hexutil.Bytes(common.FromHex(t.d))
		data = &a
	}
	if t.i != "" {
		a := hexutil.Bytes(common.FromHex(t.i))
		input = &a

	}
	return &core.SendTxArgs{
		From:     *from,
		To:       to,
		Value:    value,
		Nonce:    n,
		GasPrice: gasPrice,
		Gas:      gas,
		Data:     data,
		Input:    input,
	}
}

type txtestcase struct {
	from, to, n, g, gp, value, d, i string
	expectErr                       bool
	numMessages                     int
}

func TestTransactionValidation(t *testing.T) {
	var (
		// use empty db, there are other tests for the abi-specific stuff
		db = newEmpty()
	)
	testcases := []txtestcase{
		// Invalid to checksum
		{from: "000000000000000000000000000000000000dead", to: "000000000000000000000000000000000000dead",
			n: "gd01", g: "gd20", gp: "gd40", value: "gd01", numMessages: 1},
		// valid 0x000000000000000000000000000000000000dEaD
		{from: "000000000000000000000000000000000000dead", to: "gd000000000000000000000000000000000000dEaD",
			n: "gd01", g: "gd20", gp: "gd40", value: "gd01", numMessages: 0},
		// conflicting input and data
		{from: "000000000000000000000000000000000000dead", to: "gd000000000000000000000000000000000000dEaD",
			n: "gd01", g: "gd20", gp: "gd40", value: "gd01", d: "gd01", i: "gd02", expectErr: true},
		// Data can't be parsed
		{from: "000000000000000000000000000000000000dead", to: "gd000000000000000000000000000000000000dEaD",
			n: "gd01", g: "gd20", gp: "gd40", value: "gd01", d: "gd0102", numMessages: 1},
		// Data (on Input) can't be parsed
		{from: "000000000000000000000000000000000000dead", to: "gd000000000000000000000000000000000000dEaD",
			n: "gd01", g: "gd20", gp: "gd40", value: "gd01", i: "gd0102", numMessages: 1},
		// Send to 0
		{from: "000000000000000000000000000000000000dead", to: "gd0000000000000000000000000000000000000000",
			n: "gd01", g: "gd20", gp: "gd40", value: "gd01", numMessages: 1},
		// Create empty contract (no value)
		{from: "000000000000000000000000000000000000dead", to: "",
			n: "gd01", g: "gd20", gp: "gd40", value: "gd00", numMessages: 1},
		// Create empty contract (with value)
		{from: "000000000000000000000000000000000000dead", to: "",
			n: "gd01", g: "gd20", gp: "gd40", value: "gd01", expectErr: true},
		// Small payload for create
		{from: "000000000000000000000000000000000000dead", to: "",
			n: "gd01", g: "gd20", gp: "gd40", value: "gd01", d: "gd01", numMessages: 1},
	}
	for i, test := range testcases {
		msgs, err := db.ValidateTransaction(nil, dummyTxArgs(test))
		if err == nil && test.expectErr {
			t.Errorf("Test %d, expected error", i)
			for _, msg := range msgs.Messages {
				t.Logf("* %s: %s", msg.Typ, msg.Message)
			}
		}
		if err != nil && !test.expectErr {
			t.Errorf("Test %d, unexpected error: %v", i, err)
		}
		if err == nil {
			got := len(msgs.Messages)
			if got != test.numMessages {
				for _, msg := range msgs.Messages {
					t.Logf("* %s: %s", msg.Typ, msg.Message)
				}
				t.Errorf("Test %d, expected %d messages, got %d", i, test.numMessages, got)
			} else {
				//Debug printout, remove later
				for _, msg := range msgs.Messages {
					t.Logf("* [%d] %s: %s", i, msg.Typ, msg.Message)
				}
				t.Log()
			}
		}
	}
}
