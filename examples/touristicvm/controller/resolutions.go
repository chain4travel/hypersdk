// Copyright (C) 2023, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package controller

import (
	"context"

	"github.com/ava-labs/avalanchego/ids"
	"github.com/ava-labs/avalanchego/trace"
	"github.com/ava-labs/avalanchego/utils/logging"
	"github.com/ava-labs/hypersdk/chain"
	"github.com/ava-labs/hypersdk/codec"
	"github.com/ava-labs/hypersdk/examples/touristicvm/genesis"
	"github.com/ava-labs/hypersdk/examples/touristicvm/storage"
)

func (c *Controller) Genesis() *genesis.Genesis {
	return c.genesis
}

func (c *Controller) Logger() logging.Logger {
	return c.inner.Logger()
}

func (c *Controller) Tracer() trace.Tracer {
	return c.inner.Tracer()
}

func (c *Controller) GetTransaction(
	ctx context.Context,
	txID ids.ID,
) (bool, int64, bool, chain.Dimensions, uint64, error) {
	return storage.GetTransaction(ctx, c.metaDB, txID)
}
func (c *Controller) GetAssetFromState(
	ctx context.Context,
	asset ids.ID,
) (bool, []byte, uint8, []byte, uint64, uint64, codec.Address, bool, error) {
	return storage.GetAssetFromState(ctx, c.inner.ReadState, asset)
}
func (c *Controller) GetBalanceFromState(
	ctx context.Context,
	addr codec.Address,
	asset ids.ID,
) (uint64, error) {
	return storage.GetBalanceFromState(ctx, c.inner.ReadState, addr, asset)
}

func (c *Controller) GetNFTFromState(ctx context.Context, id ids.ID) (bool, []byte, codec.Address, []byte, bool, error) {
	return storage.GetNFTFromState(ctx, c.inner.ReadState, id)
}
func (c *Controller) GetLoanFromState(
	ctx context.Context,
	asset ids.ID,
	destination ids.ID,
) (uint64, error) {
	return storage.GetLoanFromState(ctx, c.inner.ReadState, asset, destination)
}
