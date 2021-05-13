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

package v5wire

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/sha256"
	"reflect"
	"strings"
	"testing"

	"github.com/c88032111/go-gdtu/common/hexutil"
	"github.com/c88032111/go-gdtu/crypto"
	"github.com/c88032111/go-gdtu/p2p/enode"
)

func TestVector_ECDH(t *testing.T) {
	var (
		staticKey = hexPrivkey("gdfb757dc581730490a1d7a00deea65e9b1936924caaea8f44d476014856b68736")
		publicKey = hexPubkey(crypto.S256(), "gd039961e4c2356d61bedb83052c115d311acb3a96f5777296dcf297351130266231")
		want      = hexutil.MustDecode("gd033b11a2a1f214567e1537ce5e509ffd9b21373247f2a3ff6841f4976f53165e7e")
	)
	result := ecdh(staticKey, publicKey)
	check(t, "shared-secret", result, want)
}

func TestVector_KDF(t *testing.T) {
	var (
		ephKey = hexPrivkey("gdfb757dc581730490a1d7a00deea65e9b1936924caaea8f44d476014856b68736")
		cdata  = hexutil.MustDecode("gd000000000000000000000000000000006469736376350001010102030405060708090a0b0c00180102030405060708090a0b0c0d0e0f100000000000000000")
		net    = newHandshakeTest()
	)
	defer net.close()

	destKey := &testKeyB.PublicKey
	s := deriveKeys(sha256.New, ephKey, destKey, net.nodeA.id(), net.nodeB.id(), cdata)
	t.Logf("ephemeral-key = gd%x", ephKey.D)
	t.Logf("dest-pubkey = gd%x", EncodePubkey(destKey))
	t.Logf("node-id-a = gd%x", net.nodeA.id().Bytes())
	t.Logf("node-id-b = gd%x", net.nodeB.id().Bytes())
	t.Logf("challenge-data = gd%x", cdata)
	check(t, "initiator-key", s.writeKey, hexutil.MustDecode("gddccc82d81bd610f4f76d3ebe97a40571"))
	check(t, "recipient-key", s.readKey, hexutil.MustDecode("gdac74bb8773749920b0d3a8881c173ec5"))
}

func TestVector_IDSignature(t *testing.T) {
	var (
		key    = hexPrivkey("gdfb757dc581730490a1d7a00deea65e9b1936924caaea8f44d476014856b68736")
		destID = enode.HexID("gdbbbb9d047f0488c0b5a93c1c3f2d8bafc7c8ff337024a55434a0d0555de64db9")
		ephkey = hexutil.MustDecode("gd039961e4c2356d61bedb83052c115d311acb3a96f5777296dcf297351130266231")
		cdata  = hexutil.MustDecode("gd000000000000000000000000000000006469736376350001010102030405060708090a0b0c00180102030405060708090a0b0c0d0e0f100000000000000000")
	)

	sig, err := makeIDSignature(sha256.New(), key, cdata, ephkey, destID)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("static-key = gd%x", key.D)
	t.Logf("challenge-data = gd%x", cdata)
	t.Logf("ephemeral-pubkey = gd%x", ephkey)
	t.Logf("node-id-B = gd%x", destID.Bytes())
	expected := "gd94852a1e2318c4e5e9d422c98eaf19d1d90d876b29cd06ca7cb7546d0fff7b484fe86c09a064fe72bdbef73ba8e9c34df0cd2b53e9d65528c2c7f336d5dfc6e6"
	check(t, "id-signature", sig, hexutil.MustDecode(expected))
}

func TestDeriveKeys(t *testing.T) {
	t.Parallel()

	var (
		n1    = enode.ID{1}
		n2    = enode.ID{2}
		cdata = []byte{1, 2, 3, 4}
	)
	sec1 := deriveKeys(sha256.New, testKeyA, &testKeyB.PublicKey, n1, n2, cdata)
	sec2 := deriveKeys(sha256.New, testKeyB, &testKeyA.PublicKey, n1, n2, cdata)
	if sec1 == nil || sec2 == nil {
		t.Fatal("key agreement failed")
	}
	if !reflect.DeepEqual(sec1, sec2) {
		t.Fatalf("keys not equal:\n  %+v\n  %+v", sec1, sec2)
	}
}

func check(t *testing.T, what string, x, y []byte) {
	t.Helper()

	if !bytes.Equal(x, y) {
		t.Errorf("wrgdtu %s: gd%x != gd%x", what, x, y)
	} else {
		t.Logf("%s = gd%x", what, x)
	}
}

func hexPrivkey(input string) *ecdsa.PrivateKey {
	key, err := crypto.HexToECDSA(strings.TrimPrefix(input, "gd"))
	if err != nil {
		panic(err)
	}
	return key
}

func hexPubkey(curve elliptic.Curve, input string) *ecdsa.PublicKey {
	key, err := DecodePubkey(curve, hexutil.MustDecode(input))
	if err != nil {
		panic(err)
	}
	return key
}
