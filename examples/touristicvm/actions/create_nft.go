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
	zutils "github.com/ava-labs/hypersdk/utils"

	"github.com/ava-labs/hypersdk/state"
)

var _ chain.Action = (*CreateNFT)(nil)

type CreateNFT struct {
	Metadata []byte        `json:"metadata"`
	Owner    codec.Address `json:"owner"`
	URL      []byte        `json:"url"`
}

func (*CreateNFT) GetTypeID() uint8 {
	return createNFTID
}

func (*CreateNFT) StateKeys(_ chain.Auth, txID ids.ID) []string {
	return []string{
		string(storage.NFTKey(txID)),
	}
}

func (*CreateNFT) StateKeysMaxChunks() []uint16 {
	return []uint16{storage.NFTChunks}
}

func (*CreateNFT) OutputsWarpMessage() bool {
	return false
}

func (c *CreateNFT) Execute(
	ctx context.Context,
	_ chain.Rules,
	mu state.Mutable,
	_ int64,
	rauth chain.Auth,
	txID ids.ID,
	_ bool,
) (bool, uint64, []byte, *warp.UnsignedMessage, error) {

	if len(c.Metadata) == 0 {
		return false, CreateNFTComputeUnits, OutputMetadataEmpty, nil, nil
	}

	if err := storage.SetNFT(ctx, mu, txID, c.Metadata, c.Owner, string(c.URL)); err != nil {

		return false, CreateNFTComputeUnits, zutils.ErrBytes(err), nil, nil
	}

	return true, CreateNFTComputeUnits, nil, nil, nil
}

func (*CreateNFT) MaxComputeUnits(chain.Rules) uint64 {
	return CreateNFTComputeUnits
}

func (c *CreateNFT) Size() int {
	// TODO: add small bytes (smaller int prefix)
	return codec.BytesLen(c.Metadata) + codec.AddressLen + codec.BytesLen(c.URL)
}

func (c *CreateNFT) Marshal(p *codec.Packer) {
	p.PackBytes(c.Metadata)
	p.PackAddress(c.Owner)
	p.PackBytes(c.URL)
}

func UnmarshalCreateNFT(p *codec.Packer, _ *warp.Message) (chain.Action, error) {
	var create CreateNFT
	p.UnpackBytes(MaxMetadataSize, true, &create.Metadata)
	p.UnpackAddress(&create.Owner)
	p.UnpackBytes(MaxNFTURLSize, true, &create.URL)
	return &create, p.Err()
}

func (*CreateNFT) ValidRange(chain.Rules) (int64, int64) {
	// Returning -1, -1 means that the action is always valid.
	return -1, -1
}
