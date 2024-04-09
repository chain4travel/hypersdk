// Copyright (C) 2023, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package rpc

import (
	"context"

	"github.com/ava-labs/avalanchego/ids"
	"github.com/ava-labs/avalanchego/trace"
	"github.com/ava-labs/hypersdk/chain"
	"github.com/ava-labs/hypersdk/codec"
	"github.com/chain4travel/hypersdk/examples/touristicvm/genesis"
)

type Controller interface {
	Genesis() *genesis.Genesis
	Tracer() trace.Tracer
	GetTransaction(context.Context, ids.ID) (bool, int64, bool, chain.Dimensions, uint64, error)
	GetAssetFromState(context.Context, ids.ID) (bool, []byte, uint8, []byte, uint64, uint64, codec.Address, bool, error)
	GetBalanceFromState(context.Context, codec.Address, ids.ID) (uint64, error)
	GetNFTFromState(context.Context, ids.ID) (bool, []byte, codec.Address, []byte, bool, error)
}
