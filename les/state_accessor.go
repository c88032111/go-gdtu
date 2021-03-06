// Copyright 2021 The go-gdtu Authors
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
	"context"
	"errors"
	"fmt"

	"github.com/c88032111/go-gdtu/core"
	"github.com/c88032111/go-gdtu/core/state"
	"github.com/c88032111/go-gdtu/core/types"
	"github.com/c88032111/go-gdtu/core/vm"
	"github.com/c88032111/go-gdtu/light"
)

// stateAtBlock retrieves the state database associated with a certain block.
func (lgdtu *LightGdtu) stateAtBlock(ctx context.Context, block *types.Block, reexec uint64) (*state.StateDB, func(), error) {
	return light.NewState(ctx, block.Header(), lgdtu.odr), func() {}, nil
}

// statesInRange retrieves a batch of state databases associated with the specific
// block ranges.
func (lgdtu *LightGdtu) statesInRange(ctx context.Context, fromBlock *types.Block, toBlock *types.Block, reexec uint64) ([]*state.StateDB, func(), error) {
	var states []*state.StateDB
	for number := fromBlock.NumberU64(); number <= toBlock.NumberU64(); number++ {
		header, err := lgdtu.blockchain.GetHeaderByNumberOdr(ctx, number)
		if err != nil {
			return nil, nil, err
		}
		states = append(states, light.NewState(ctx, header, lgdtu.odr))
	}
	return states, nil, nil
}

// stateAtTransaction returns the execution environment of a certain transaction.
func (lgdtu *LightGdtu) stateAtTransaction(ctx context.Context, block *types.Block, txIndex int, reexec uint64) (core.Message, vm.BlockContext, *state.StateDB, func(), error) {
	// Short circuit if it's genesis block.
	if block.NumberU64() == 0 {
		return nil, vm.BlockContext{}, nil, nil, errors.New("no transaction in genesis")
	}
	// Create the parent state database
	parent, err := lgdtu.blockchain.GetBlock(ctx, block.ParentHash(), block.NumberU64()-1)
	if err != nil {
		return nil, vm.BlockContext{}, nil, nil, err
	}
	statedb, _, err := lgdtu.stateAtBlock(ctx, parent, reexec)
	if err != nil {
		return nil, vm.BlockContext{}, nil, nil, err
	}
	if txIndex == 0 && len(block.Transactions()) == 0 {
		return nil, vm.BlockContext{}, statedb, func() {}, nil
	}
	// Recompute transactions up to the target index.
	signer := types.MakeSigner(lgdtu.blockchain.Config(), block.Number())
	for idx, tx := range block.Transactions() {
		// Assemble the transaction call message and return if the requested offset
		msg, _ := tx.AsMessage(signer)
		txContext := core.NewEVMTxContext(msg)
		context := core.NewEVMBlockContext(block.Header(), lgdtu.blockchain, nil)
		statedb.Prepare(tx.Hash(), block.Hash(), idx)
		if idx == txIndex {
			return msg, context, statedb, func() {}, nil
		}
		// Not yet the searched for transaction, execute on top of the current state
		vmenv := vm.NewEVM(context, txContext, statedb, lgdtu.blockchain.Config(), vm.Config{})
		if _, err := core.ApplyMessage(vmenv, msg, new(core.GasPool).AddGas(tx.Gas())); err != nil {
			return nil, vm.BlockContext{}, nil, nil, fmt.Errorf("transaction gd%x failed: %v", tx.Hash(), err)
		}
		// Ensure any modifications are committed to the state
		// Only delete empty objects if EIP158/161 (a.k.a Spurious Dragon) is in effect
		statedb.Finalise(vmenv.ChainConfig().IsEIP158(block.Number()))
	}
	return nil, vm.BlockContext{}, nil, nil, fmt.Errorf("transaction index %d out of range for block gd%x", txIndex, block.Hash())
}
