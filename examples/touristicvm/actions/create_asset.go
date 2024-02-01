// Copyright (C) 2023, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package actions

import (
	"context"
	"github.com/ava-labs/hypersdk/examples/touristicvm/storage"

	"github.com/ava-labs/avalanchego/ids"
	"github.com/ava-labs/avalanchego/vms/platformvm/warp"
	"github.com/ava-labs/hypersdk/chain"
	"github.com/ava-labs/hypersdk/codec"
	"github.com/ava-labs/hypersdk/consts"
	"github.com/ava-labs/hypersdk/state"
	"github.com/ava-labs/hypersdk/utils"
)

var _ chain.Action = (*CreateAsset)(nil)

type CreateAsset struct {
	Symbol    []byte `json:"symbol"`
	Decimals  uint8  `json:"decimals"`
	Metadata  []byte `json:"metadata"`
	MaxSupply uint64 `json:"maxSupply"`
}

func (*CreateAsset) GetTypeID() uint8 {
	return createAssetID
}

func (*CreateAsset) StateKeys(_ chain.Auth, txID ids.ID) []string {
	return []string{
		string(storage.AssetKey(txID)),
	}
}

func (*CreateAsset) StateKeysMaxChunks() []uint16 {
	return []uint16{storage.AssetChunks}
}

func (*CreateAsset) OutputsWarpMessage() bool {
	return false
}

func (c *CreateAsset) Execute(
	ctx context.Context,
	_ chain.Rules,
	mu state.Mutable,
	_ int64,
	auth chain.Auth,
	txID ids.ID,
	_ bool,
) (bool, uint64, []byte, *warp.UnsignedMessage, error) {
	if len(c.Symbol) == 0 {
		return false, CreateAssetComputeUnits, OutputSymbolEmpty, nil, nil
	}
	if len(c.Symbol) > MaxSymbolSize {
		return false, CreateAssetComputeUnits, OutputSymbolTooLarge, nil, nil
	}
	if c.Decimals > MaxDecimals {
		return false, CreateAssetComputeUnits, OutputDecimalsTooLarge, nil, nil
	}
	if len(c.Metadata) == 0 {
		return false, CreateAssetComputeUnits, OutputMetadataEmpty, nil, nil
	}
	if len(c.Metadata) > MaxMetadataSize {
		return false, CreateAssetComputeUnits, OutputMetadataTooLarge, nil, nil
	}

	if c.MaxSupply > consts.MaxUint64 {
		return false, CreateAssetComputeUnits, MaxSupplyTooLarge, nil, nil
	}

	// It should only be possible to overwrite an existing asset if there is
	// a hash collision.
	if err := storage.SetAsset(ctx, mu, txID, c.Symbol, c.Decimals, c.Metadata, 0, c.MaxSupply, auth.Actor(), false); err != nil {
		return false, CreateAssetComputeUnits, utils.ErrBytes(err), nil, nil
	}
	return true, CreateAssetComputeUnits, nil, nil, nil
}

func (*CreateAsset) MaxComputeUnits(chain.Rules) uint64 {
	return CreateAssetComputeUnits
}

func (c *CreateAsset) Size() int {
	// TODO: add small bytes (smaller int prefix)
	return codec.BytesLen(c.Symbol) + consts.Uint8Len + codec.BytesLen(c.Metadata) + consts.Uint8Len
}

func (c *CreateAsset) Marshal(p *codec.Packer) {
	p.PackBytes(c.Symbol)
	p.PackByte(c.Decimals)
	p.PackBytes(c.Metadata)
	p.PackUint64(c.MaxSupply)
}

func UnmarshalCreateAsset(p *codec.Packer, _ *warp.Message) (chain.Action, error) {
	var create CreateAsset
	p.UnpackBytes(MaxSymbolSize, true, &create.Symbol)
	create.Decimals = p.UnpackByte()
	p.UnpackBytes(MaxMetadataSize, true, &create.Metadata)
	create.MaxSupply = p.UnpackUint64(true)
	return &create, p.Err()
}

func (*CreateAsset) ValidRange(chain.Rules) (int64, int64) {
	// Returning -1, -1 means that the action is always valid.
	return -1, -1
}
