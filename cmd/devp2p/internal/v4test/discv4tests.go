// Copyright 2020 The go-gdtu Authors
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

package v4test

import (
	"bytes"
	"crypto/rand"
	"fmt"
	"net"
	"reflect"
	"time"

	"github.com/c88032111/go-gdtu/crypto"
	"github.com/c88032111/go-gdtu/internal/utesting"
	"github.com/c88032111/go-gdtu/p2p/discover/v4wire"
)

const (
	expiration   = 20 * time.Second
	wrgdtuPacket = 66
	macSize      = 256 / 8
)

var (
	// Remote node under test
	Remote string
	// IP where the first tester is listening, port will be assigned
	Listen1 string = "127.0.0.1"
	// IP where the second tester is listening, port will be assigned
	// Before running the test, you may have to `sudo ifconfig lo0 add 127.0.0.2` (on MacOS at least)
	Listen2 string = "127.0.0.2"
)

type pingWithJunk struct {
	Version    uint
	From, To   v4wire.Endpoint
	Expiration uint64
	JunkData1  uint
	JunkData2  []byte
}

func (req *pingWithJunk) Name() string { return "PING/v4" }
func (req *pingWithJunk) Kind() byte   { return v4wire.PingPacket }

type pingWrgdtuType struct {
	Version    uint
	From, To   v4wire.Endpoint
	Expiration uint64
}

func (req *pingWrgdtuType) Name() string { return "WRGDTU/v4" }
func (req *pingWrgdtuType) Kind() byte   { return wrgdtuPacket }

func futureExpiration() uint64 {
	return uint64(time.Now().Add(expiration).Unix())
}

// This test just sends a PING packet and expects a response.
func BasicPing(t *utesting.T) {
	te := newTestEnv(Remote, Listen1, Listen2)
	defer te.close()

	pingHash := te.send(te.l1, &v4wire.Ping{
		Version:    4,
		From:       te.localEndpoint(te.l1),
		To:         te.remoteEndpoint(),
		Expiration: futureExpiration(),
	})

	reply, _, _ := te.read(te.l1)
	if err := te.checkPgdtu(reply, pingHash); err != nil {
		t.Fatal(err)
	}
}

// checkPgdtu verifies that reply is a valid PGDTU matching the given ping hash.
func (te *testenv) checkPgdtu(reply v4wire.Packet, pingHash []byte) error {
	if reply == nil || reply.Kind() != v4wire.PgdtuPacket {
		return fmt.Errorf("expected PGDTU reply, got %v", reply)
	}
	pgdtu := reply.(*v4wire.Pgdtu)
	if !bytes.Equal(pgdtu.ReplyTok, pingHash) {
		return fmt.Errorf("PGDTU reply token mismatch: got %x, want %x", pgdtu.ReplyTok, pingHash)
	}
	wantEndpoint := te.localEndpoint(te.l1)
	if !reflect.DeepEqual(pgdtu.To, wantEndpoint) {
		return fmt.Errorf("PGDTU 'to' endpoint mismatch: got %+v, want %+v", pgdtu.To, wantEndpoint)
	}
	if v4wire.Expired(pgdtu.Expiration) {
		return fmt.Errorf("PGDTU is expired (%v)", pgdtu.Expiration)
	}
	return nil
}

// This test sends a PING packet with wrgdtu 'to' field and expects a PGDTU response.
func PingWrgdtuTo(t *utesting.T) {
	te := newTestEnv(Remote, Listen1, Listen2)
	defer te.close()

	wrgdtuEndpoint := v4wire.Endpoint{IP: net.ParseIP("192.0.2.0")}
	pingHash := te.send(te.l1, &v4wire.Ping{
		Version:    4,
		From:       te.localEndpoint(te.l1),
		To:         wrgdtuEndpoint,
		Expiration: futureExpiration(),
	})

	reply, _, _ := te.read(te.l1)
	if err := te.checkPgdtu(reply, pingHash); err != nil {
		t.Fatal(err)
	}
}

// This test sends a PING packet with wrgdtu 'from' field and expects a PGDTU response.
func PingWrgdtuFrom(t *utesting.T) {
	te := newTestEnv(Remote, Listen1, Listen2)
	defer te.close()

	wrgdtuEndpoint := v4wire.Endpoint{IP: net.ParseIP("192.0.2.0")}
	pingHash := te.send(te.l1, &v4wire.Ping{
		Version:    4,
		From:       wrgdtuEndpoint,
		To:         te.remoteEndpoint(),
		Expiration: futureExpiration(),
	})

	reply, _, _ := te.read(te.l1)
	if err := te.checkPgdtu(reply, pingHash); err != nil {
		t.Fatal(err)
	}
}

// This test sends a PING packet with additional data at the end and expects a PGDTU
// response. The remote node should respond because EIP-8 mandates ignoring additional
// trailing data.
func PingExtraData(t *utesting.T) {
	te := newTestEnv(Remote, Listen1, Listen2)
	defer te.close()

	pingHash := te.send(te.l1, &pingWithJunk{
		Version:    4,
		From:       te.localEndpoint(te.l1),
		To:         te.remoteEndpoint(),
		Expiration: futureExpiration(),
		JunkData1:  42,
		JunkData2:  []byte{9, 8, 7, 6, 5, 4, 3, 2, 1},
	})

	reply, _, _ := te.read(te.l1)
	if err := te.checkPgdtu(reply, pingHash); err != nil {
		t.Fatal(err)
	}
}

// This test sends a PING packet with additional data and wrgdtu 'from' field
// and expects a PGDTU response.
func PingExtraDataWrgdtuFrom(t *utesting.T) {
	te := newTestEnv(Remote, Listen1, Listen2)
	defer te.close()

	wrgdtuEndpoint := v4wire.Endpoint{IP: net.ParseIP("192.0.2.0")}
	req := pingWithJunk{
		Version:    4,
		From:       wrgdtuEndpoint,
		To:         te.remoteEndpoint(),
		Expiration: futureExpiration(),
		JunkData1:  42,
		JunkData2:  []byte{9, 8, 7, 6, 5, 4, 3, 2, 1},
	}
	pingHash := te.send(te.l1, &req)
	reply, _, _ := te.read(te.l1)
	if err := te.checkPgdtu(reply, pingHash); err != nil {
		t.Fatal(err)
	}
}

// This test sends a PING packet with an expiration in the past.
// The remote node should not respond.
func PingPastExpiration(t *utesting.T) {
	te := newTestEnv(Remote, Listen1, Listen2)
	defer te.close()

	te.send(te.l1, &v4wire.Ping{
		Version:    4,
		From:       te.localEndpoint(te.l1),
		To:         te.remoteEndpoint(),
		Expiration: -futureExpiration(),
	})

	reply, _, _ := te.read(te.l1)
	if reply != nil {
		t.Fatal("Expected no reply, got", reply)
	}
}

// This test sends an invalid packet. The remote node should not respond.
func WrgdtuPacketType(t *utesting.T) {
	te := newTestEnv(Remote, Listen1, Listen2)
	defer te.close()

	te.send(te.l1, &pingWrgdtuType{
		Version:    4,
		From:       te.localEndpoint(te.l1),
		To:         te.remoteEndpoint(),
		Expiration: futureExpiration(),
	})

	reply, _, _ := te.read(te.l1)
	if reply != nil {
		t.Fatal("Expected no reply, got", reply)
	}
}

// This test verifies that the default behaviour of ignoring 'from' fields is unaffected by
// the bonding process. After bonding, it pings the target with a different from endpoint.
func BondThenPingWithWrgdtuFrom(t *utesting.T) {
	te := newTestEnv(Remote, Listen1, Listen2)
	defer te.close()
	bond(t, te)

	wrgdtuEndpoint := v4wire.Endpoint{IP: net.ParseIP("192.0.2.0")}
	pingHash := te.send(te.l1, &v4wire.Ping{
		Version:    4,
		From:       wrgdtuEndpoint,
		To:         te.remoteEndpoint(),
		Expiration: futureExpiration(),
	})

	reply, _, _ := te.read(te.l1)
	if err := te.checkPgdtu(reply, pingHash); err != nil {
		t.Fatal(err)
	}
}

// This test just sends FINDNODE. The remote node should not reply
// because the endpoint proof has not completed.
func FindnodeWithoutEndpointProof(t *utesting.T) {
	te := newTestEnv(Remote, Listen1, Listen2)
	defer te.close()

	req := v4wire.Findnode{Expiration: futureExpiration()}
	rand.Read(req.Target[:])
	te.send(te.l1, &req)

	reply, _, _ := te.read(te.l1)
	if reply != nil {
		t.Fatal("Expected no response, got", reply)
	}
}

// BasicFindnode sends a FINDNODE request after performing the endpoint
// proof. The remote node should respond.
func BasicFindnode(t *utesting.T) {
	te := newTestEnv(Remote, Listen1, Listen2)
	defer te.close()
	bond(t, te)

	findnode := v4wire.Findnode{Expiration: futureExpiration()}
	rand.Read(findnode.Target[:])
	te.send(te.l1, &findnode)

	reply, _, err := te.read(te.l1)
	if err != nil {
		t.Fatal("read find nodes", err)
	}
	if reply.Kind() != v4wire.NeighborsPacket {
		t.Fatal("Expected neighbors, got", reply.Name())
	}
}

// This test sends an unsolicited NEIGHBORS packet after the endpoint proof, then sends
// FINDNODE to read the remote table. The remote node should not return the node contained
// in the unsolicited NEIGHBORS packet.
func UnsolicitedNeighbors(t *utesting.T) {
	te := newTestEnv(Remote, Listen1, Listen2)
	defer te.close()
	bond(t, te)

	// Send unsolicited NEIGHBORS response.
	fakeKey, _ := crypto.GenerateKey()
	encFakeKey := v4wire.EncodePubkey(&fakeKey.PublicKey)
	neighbors := v4wire.Neighbors{
		Expiration: futureExpiration(),
		Nodes: []v4wire.Node{{
			ID:  encFakeKey,
			IP:  net.IP{1, 2, 3, 4},
			UDP: 30303,
			TCP: 30303,
		}},
	}
	te.send(te.l1, &neighbors)

	// Check if the remote node included the fake node.
	te.send(te.l1, &v4wire.Findnode{
		Expiration: futureExpiration(),
		Target:     encFakeKey,
	})

	reply, _, err := te.read(te.l1)
	if err != nil {
		t.Fatal("read find nodes", err)
	}
	if reply.Kind() != v4wire.NeighborsPacket {
		t.Fatal("Expected neighbors, got", reply.Name())
	}
	nodes := reply.(*v4wire.Neighbors).Nodes
	if contains(nodes, encFakeKey) {
		t.Fatal("neighbors response contains node from earlier unsolicited neighbors response")
	}
}

// This test sends FINDNODE with an expiration timestamp in the past.
// The remote node should not respond.
func FindnodePastExpiration(t *utesting.T) {
	te := newTestEnv(Remote, Listen1, Listen2)
	defer te.close()
	bond(t, te)

	findnode := v4wire.Findnode{Expiration: -futureExpiration()}
	rand.Read(findnode.Target[:])
	te.send(te.l1, &findnode)

	for {
		reply, _, _ := te.read(te.l1)
		if reply == nil {
			return
		} else if reply.Kind() == v4wire.NeighborsPacket {
			t.Fatal("Unexpected NEIGHBORS response for expired FINDNODE request")
		}
	}
}

// bond performs the endpoint proof with the remote node.
func bond(t *utesting.T, te *testenv) {
	te.send(te.l1, &v4wire.Ping{
		Version:    4,
		From:       te.localEndpoint(te.l1),
		To:         te.remoteEndpoint(),
		Expiration: futureExpiration(),
	})

	var gotPing, gotPgdtu bool
	for !gotPing || !gotPgdtu {
		req, hash, err := te.read(te.l1)
		if err != nil {
			t.Fatal(err)
		}
		switch req.(type) {
		case *v4wire.Ping:
			te.send(te.l1, &v4wire.Pgdtu{
				To:         te.remoteEndpoint(),
				ReplyTok:   hash,
				Expiration: futureExpiration(),
			})
			gotPing = true
		case *v4wire.Pgdtu:
			// TODO: maybe verify pgdtu data here
			gotPgdtu = true
		}
	}
}

// This test attempts to perform a traffic amplification attack against a
// 'victim' endpoint using FINDNODE. In this attack scenario, the attacker
// attempts to complete the endpoint proof non-interactively by sending a PGDTU
// with mismatching reply token from the 'victim' endpoint. The attack works if
// the remote node does not verify the PGDTU reply token field correctly. The
// attacker could then perform traffic amplification by sending many FINDNODE
// requests to the discovery node, which would reply to the 'victim' address.
func FindnodeAmplificationInvalidPgdtuHash(t *utesting.T) {
	te := newTestEnv(Remote, Listen1, Listen2)
	defer te.close()

	// Send PING to start endpoint verification.
	te.send(te.l1, &v4wire.Ping{
		Version:    4,
		From:       te.localEndpoint(te.l1),
		To:         te.remoteEndpoint(),
		Expiration: futureExpiration(),
	})

	var gotPing, gotPgdtu bool
	for !gotPing || !gotPgdtu {
		req, _, err := te.read(te.l1)
		if err != nil {
			t.Fatal(err)
		}
		switch req.(type) {
		case *v4wire.Ping:
			// Send PGDTU from this node ID, but with invalid ReplyTok.
			te.send(te.l1, &v4wire.Pgdtu{
				To:         te.remoteEndpoint(),
				ReplyTok:   make([]byte, macSize),
				Expiration: futureExpiration(),
			})
			gotPing = true
		case *v4wire.Pgdtu:
			gotPgdtu = true
		}
	}

	// Now send FINDNODE. The remote node should not respond because our
	// PGDTU did not reference the PING hash.
	findnode := v4wire.Findnode{Expiration: futureExpiration()}
	rand.Read(findnode.Target[:])
	te.send(te.l1, &findnode)

	// If we receive a NEIGHBORS response, the attack worked and the test fails.
	reply, _, _ := te.read(te.l1)
	if reply != nil && reply.Kind() == v4wire.NeighborsPacket {
		t.Error("Got neighbors")
	}
}

// This test attempts to perform a traffic amplification attack using FINDNODE.
// The attack works if the remote node does not verify the IP address of FINDNODE
// against the endpoint verification proof done by PING/PGDTU.
func FindnodeAmplificationWrgdtuIP(t *utesting.T) {
	te := newTestEnv(Remote, Listen1, Listen2)
	defer te.close()

	// Do the endpoint proof from the l1 IP.
	bond(t, te)

	// Now send FINDNODE from the same node ID, but different IP address.
	// The remote node should not respond.
	findnode := v4wire.Findnode{Expiration: futureExpiration()}
	rand.Read(findnode.Target[:])
	te.send(te.l2, &findnode)

	// If we receive a NEIGHBORS response, the attack worked and the test fails.
	reply, _, _ := te.read(te.l2)
	if reply != nil {
		t.Error("Got NEIGHORS response for FINDNODE from wrgdtu IP")
	}
}

var AllTests = []utesting.Test{
	{Name: "Ping/Basic", Fn: BasicPing},
	{Name: "Ping/WrgdtuTo", Fn: PingWrgdtuTo},
	{Name: "Ping/WrgdtuFrom", Fn: PingWrgdtuFrom},
	{Name: "Ping/ExtraData", Fn: PingExtraData},
	{Name: "Ping/ExtraDataWrgdtuFrom", Fn: PingExtraDataWrgdtuFrom},
	{Name: "Ping/PastExpiration", Fn: PingPastExpiration},
	{Name: "Ping/WrgdtuPacketType", Fn: WrgdtuPacketType},
	{Name: "Ping/BondThenPingWithWrgdtuFrom", Fn: BondThenPingWithWrgdtuFrom},
	{Name: "Findnode/WithoutEndpointProof", Fn: FindnodeWithoutEndpointProof},
	{Name: "Findnode/BasicFindnode", Fn: BasicFindnode},
	{Name: "Findnode/UnsolicitedNeighbors", Fn: UnsolicitedNeighbors},
	{Name: "Findnode/PastExpiration", Fn: FindnodePastExpiration},
	{Name: "Amplification/InvalidPgdtuHash", Fn: FindnodeAmplificationInvalidPgdtuHash},
	{Name: "Amplification/WrgdtuIP", Fn: FindnodeAmplificationWrgdtuIP},
}
