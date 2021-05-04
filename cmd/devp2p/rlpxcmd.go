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

package main

import (
	"fmt"
	"net"

	"github.com/c88032111/go-gdtu/cmd/devp2p/internal/gdtutest"
	"github.com/c88032111/go-gdtu/crypto"
	"github.com/c88032111/go-gdtu/p2p"
	"github.com/c88032111/go-gdtu/p2p/rlpx"
	"github.com/c88032111/go-gdtu/rlp"
	"gopkg.in/urfave/cli.v1"
)

var (
	rlpxCommand = cli.Command{
		Name:  "rlpx",
		Usage: "RLPx Commands",
		Subcommands: []cli.Command{
			rlpxPingCommand,
			rlpxGdtuTestCommand,
		},
	}
	rlpxPingCommand = cli.Command{
		Name:   "ping",
		Usage:  "ping <node>",
		Action: rlpxPing,
	}
	rlpxGdtuTestCommand = cli.Command{
		Name:      "gdtu-test",
		Usage:     "Runs tests against a node",
		ArgsUsage: "<node> <chain.rlp> <genesis.json>",
		Action:    rlpxGdtuTest,
		Flags: []cli.Flag{
			testPatternFlag,
			testTAPFlag,
		},
	}
)

func rlpxPing(ctx *cli.Context) error {
	n := getNodeArg(ctx)
	fd, err := net.Dial("tcp", fmt.Sprintf("%v:%d", n.IP(), n.TCP()))
	if err != nil {
		return err
	}
	conn := rlpx.NewConn(fd, n.Pubkey())
	ourKey, _ := crypto.GenerateKey()
	_, err = conn.Handshake(ourKey)
	if err != nil {
		return err
	}
	code, data, _, err := conn.Read()
	if err != nil {
		return err
	}
	switch code {
	case 0:
		var h gdtutest.Hello
		if err := rlp.DecodeBytes(data, &h); err != nil {
			return fmt.Errorf("invalid handshake: %v", err)
		}
		fmt.Printf("%+v\n", h)
	case 1:
		var msg []p2p.DiscReason
		if rlp.DecodeBytes(data, &msg); len(msg) == 0 {
			return fmt.Errorf("invalid disconnect message")
		}
		return fmt.Errorf("received disconnect message: %v", msg[0])
	default:
		return fmt.Errorf("invalid message code %d, expected handshake (code zero)", code)
	}
	return nil
}

// rlpxGdtuTest runs the gdtu protocol test suite.
func rlpxGdtuTest(ctx *cli.Context) error {
	if ctx.NArg() < 3 {
		exit("missing path to chain.rlp as command-line argument")
	}
	suite, err := gdtutest.NewSuite(getNodeArg(ctx), ctx.Args()[1], ctx.Args()[2])
	if err != nil {
		exit(err)
	}
	return runTests(ctx, suite.GdtuTests())
}
