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
	"crypto/ecdsa"
	"fmt"
	"reflect"
	"time"

	"github.com/c88032111/go-gdtu/core/types"
	"github.com/c88032111/go-gdtu/crypto"
	"github.com/c88032111/go-gdtu/gdtu/protocols/gdtu"
	"github.com/c88032111/go-gdtu/internal/utesting"
	"github.com/c88032111/go-gdtu/p2p"
	"github.com/c88032111/go-gdtu/p2p/rlpx"
	"github.com/c88032111/go-gdtu/rlp"
)

type Message interface {
	Code() int
}

type Error struct {
	err error
}

func (e *Error) Unwrap() error  { return e.err }
func (e *Error) Error() string  { return e.err.Error() }
func (e *Error) Code() int      { return -1 }
func (e *Error) String() string { return e.Error() }

func errorf(format string, args ...interface{}) *Error {
	return &Error{fmt.Errorf(format, args...)}
}

// Hello is the RLP structure of the protocol handshake.
type Hello struct {
	Version    uint64
	Name       string
	Caps       []p2p.Cap
	ListenPort uint64
	ID         []byte // secp256k1 public key

	// Ignore additional fields (for forward compatibility).
	Rest []rlp.RawValue `rlp:"tail"`
}

func (h Hello) Code() int { return 0x00 }

// Disconnect is the RLP structure for a disconnect message.
type Disconnect struct {
	Reason p2p.DiscReason
}

func (d Disconnect) Code() int { return 0x01 }

type Ping struct{}

func (p Ping) Code() int { return 0x02 }

type Pgdtu struct{}

func (p Pgdtu) Code() int { return 0x03 }

// Status is the network packet for the status message for gdtu/64 and later.
type Status gdtu.StatusPacket

func (s Status) Code() int { return 16 }

// NewBlockHashes is the network packet for the block announcements.
type NewBlockHashes gdtu.NewBlockHashesPacket

func (nbh NewBlockHashes) Code() int { return 17 }

type Transactions gdtu.TransactionsPacket

func (t Transactions) Code() int { return 18 }

// GetBlockHeaders represents a block header query.
type GetBlockHeaders gdtu.GetBlockHeadersPacket

func (g GetBlockHeaders) Code() int { return 19 }

type BlockHeaders gdtu.BlockHeadersPacket

func (bh BlockHeaders) Code() int { return 20 }

// GetBlockBodies represents a GetBlockBodies request
type GetBlockBodies gdtu.GetBlockBodiesPacket

func (gbb GetBlockBodies) Code() int { return 21 }

// BlockBodies is the network packet for block content distribution.
type BlockBodies gdtu.BlockBodiesPacket

func (bb BlockBodies) Code() int { return 22 }

// NewBlock is the network packet for the block propagation message.
type NewBlock gdtu.NewBlockPacket

func (nb NewBlock) Code() int { return 23 }

// NewPooledTransactionHashes is the network packet for the tx hash propagation message.
type NewPooledTransactionHashes gdtu.NewPooledTransactionHashesPacket

func (nb NewPooledTransactionHashes) Code() int { return 24 }

// Conn represents an individual connection with a peer
type Conn struct {
	*rlpx.Conn
	ourKey              *ecdsa.PrivateKey
	gdtuProtocolVersion uint
	caps                []p2p.Cap
}

func (c *Conn) Read() Message {
	code, rawData, _, err := c.Conn.Read()
	if err != nil {
		return errorf("could not read from connection: %v", err)
	}

	var msg Message
	switch int(code) {
	case (Hello{}).Code():
		msg = new(Hello)
	case (Ping{}).Code():
		msg = new(Ping)
	case (Pgdtu{}).Code():
		msg = new(Pgdtu)
	case (Disconnect{}).Code():
		msg = new(Disconnect)
	case (Status{}).Code():
		msg = new(Status)
	case (GetBlockHeaders{}).Code():
		msg = new(GetBlockHeaders)
	case (BlockHeaders{}).Code():
		msg = new(BlockHeaders)
	case (GetBlockBodies{}).Code():
		msg = new(GetBlockBodies)
	case (BlockBodies{}).Code():
		msg = new(BlockBodies)
	case (NewBlock{}).Code():
		msg = new(NewBlock)
	case (NewBlockHashes{}).Code():
		msg = new(NewBlockHashes)
	case (Transactions{}).Code():
		msg = new(Transactions)
	case (NewPooledTransactionHashes{}).Code():
		msg = new(NewPooledTransactionHashes)
	default:
		return errorf("invalid message code: %d", code)
	}
	// if message is devp2p, decode here
	if err := rlp.DecodeBytes(rawData, msg); err != nil {
		return errorf("could not rlp decode message: %v", err)
	}
	return msg
}

// ReadAndServe serves GetBlockHeaders requests while waiting
// on another message from the node.
func (c *Conn) ReadAndServe(chain *Chain, timeout time.Duration) Message {
	start := time.Now()
	for time.Since(start) < timeout {
		timeout := time.Now().Add(10 * time.Second)
		c.SetReadDeadline(timeout)
		switch msg := c.Read().(type) {
		case *Ping:
			c.Write(&Pgdtu{})
		case *GetBlockHeaders:
			req := *msg
			headers, err := chain.GetHeaders(req)
			if err != nil {
				return errorf("could not get headers for inbound header request: %v", err)
			}

			if err := c.Write(headers); err != nil {
				return errorf("could not write to connection: %v", err)
			}
		default:
			return msg
		}
	}
	return errorf("no message received within %v", timeout)
}

func (c *Conn) Write(msg Message) error {
	// check if message is gdtu protocol message
	var (
		payload []byte
		err     error
	)
	payload, err = rlp.EncodeToBytes(msg)
	if err != nil {
		return err
	}
	_, err = c.Conn.Write(uint64(msg.Code()), payload)
	return err
}

// handshake checks to make sure a `HELLO` is received.
func (c *Conn) handshake(t *utesting.T) Message {
	defer c.SetDeadline(time.Time{})
	c.SetDeadline(time.Now().Add(10 * time.Second))

	// write hello to client
	pub0 := crypto.FromECDSAPub(&c.ourKey.PublicKey)[1:]
	ourHandshake := &Hello{
		Version: 5,
		Caps:    c.caps,
		ID:      pub0,
	}
	if err := c.Write(ourHandshake); err != nil {
		t.Fatalf("could not write to connection: %v", err)
	}
	// read hello from client
	switch msg := c.Read().(type) {
	case *Hello:
		// set snappy if version is at least 5
		if msg.Version >= 5 {
			c.SetSnappy(true)
		}
		c.negotiateGdtuProtocol(msg.Caps)
		if c.gdtuProtocolVersion == 0 {
			t.Fatalf("unexpected gdtu protocol version")
		}
		return msg
	default:
		t.Fatalf("bad handshake: %#v", msg)
		return nil
	}
}

// negotiateGdtuProtocol sets the Conn's gdtu protocol version
// to highest advertised capability from peer
func (c *Conn) negotiateGdtuProtocol(caps []p2p.Cap) {
	var highestGdtuVersion uint
	for _, capability := range caps {
		if capability.Name != "gdtu" {
			continue
		}
		if capability.Version > highestGdtuVersion && capability.Version <= 65 {
			highestGdtuVersion = capability.Version
		}
	}
	c.gdtuProtocolVersion = highestGdtuVersion
}

// statusExchange performs a `Status` message exchange with the given
// node.
func (c *Conn) statusExchange(t *utesting.T, chain *Chain, status *Status) Message {
	defer c.SetDeadline(time.Time{})
	c.SetDeadline(time.Now().Add(20 * time.Second))

	// read status message from client
	var message Message
loop:
	for {
		switch msg := c.Read().(type) {
		case *Status:
			if have, want := msg.Head, chain.blocks[chain.Len()-1].Hash(); have != want {
				t.Fatalf("wrgdtu head block in status, want:  gd%x (block %d) have gd%x",
					want, chain.blocks[chain.Len()-1].NumberU64(), have)
			}
			if have, want := msg.TD.Cmp(chain.TD(chain.Len())), 0; have != want {
				t.Fatalf("wrgdtu TD in status: have %v want %v", have, want)
			}
			if have, want := msg.ForkID, chain.ForkID(); !reflect.DeepEqual(have, want) {
				t.Fatalf("wrgdtu fork ID in status: have %v, want %v", have, want)
			}
			message = msg
			break loop
		case *Disconnect:
			t.Fatalf("disconnect received: %v", msg.Reason)
		case *Ping:
			c.Write(&Pgdtu{}) // TODO (renaynay): in the future, this should be an error
			// (PINGs should not be a response upon fresh connection)
		default:
			t.Fatalf("bad status message: %s", pretty.Sdump(msg))
		}
	}
	// make sure gdtu protocol version is set for negotiation
	if c.gdtuProtocolVersion == 0 {
		t.Fatalf("gdtu protocol version must be set in Conn")
	}
	if status == nil {
		// write status message to client
		status = &Status{
			ProtocolVersion: uint32(c.gdtuProtocolVersion),
			NetworkID:       chain.chainConfig.ChainID.Uint64(),
			TD:              chain.TD(chain.Len()),
			Head:            chain.blocks[chain.Len()-1].Hash(),
			Genesis:         chain.blocks[0].Hash(),
			ForkID:          chain.ForkID(),
		}
	}

	if err := c.Write(status); err != nil {
		t.Fatalf("could not write to connection: %v", err)
	}

	return message
}

// waitForBlock waits for confirmation from the client that it has
// imported the given block.
func (c *Conn) waitForBlock(block *types.Block) error {
	defer c.SetReadDeadline(time.Time{})

	timeout := time.Now().Add(20 * time.Second)
	c.SetReadDeadline(timeout)
	for {
		req := &GetBlockHeaders{Origin: gdtu.HashOrNumber{Hash: block.Hash()}, Amount: 1}
		if err := c.Write(req); err != nil {
			return err
		}
		switch msg := c.Read().(type) {
		case *BlockHeaders:
			if len(*msg) > 0 {
				return nil
			}
			time.Sleep(100 * time.Millisecond)
		default:
			return fmt.Errorf("invalid message: %s", pretty.Sdump(msg))
		}
	}
}
