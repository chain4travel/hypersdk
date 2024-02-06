// Copyright (C) 2023, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package actions

import (
	"context"
	"errors"
	"github.com/ava-labs/avalanchego/ids"
	"github.com/ava-labs/avalanchego/vms/platformvm/warp"
	"github.com/ava-labs/hypersdk/chain"
	"github.com/ava-labs/hypersdk/codec"
	"github.com/ava-labs/hypersdk/consts"
	"github.com/ava-labs/hypersdk/examples/touristicvm/storage"
	"github.com/ava-labs/hypersdk/state"
	"github.com/ava-labs/hypersdk/utils"
)

var _ chain.Action = (*TransferNFT)(nil)
var OutputNFTNotFound = errors.New("NFT not found")
var OutputNotOwner = errors.New("not owner of NFT")

type TransferNFT struct {
	// To is the recipient of the [Value].
	To codec.Address `json:"to"`

	// NFT to transfer to [To].
	NFT ids.ID `json:"asset"`

	// Optional message to accompany transaction.
	Memo []byte `json:"memo"`
}

func (*TransferNFT) GetTypeID() uint8 {
	return transferNFTID
}

func (m *TransferNFT) StateKeys(codec.Address, ids.ID) []string {
	return []string{
		string(storage.NFTKey(m.NFT)),
		string(storage.BalanceKey(m.To, m.NFT)),
	}
}

func (*TransferNFT) StateKeysMaxChunks() []uint16 {
	return []uint16{storage.NFTChunks, storage.BalanceChunks}
}

func (*TransferNFT) OutputsWarpMessage() bool {
	return false
}

func (t *TransferNFT) Execute(
	ctx context.Context,
	_ chain.Rules,
	mu state.Mutable,
	_ int64,
	actor codec.Address,
	_ ids.ID,
	_ bool,
) (bool, uint64, []byte, *warp.UnsignedMessage, error) {
	if len(t.Memo) > MaxMemoSize {
		return false, CreateAssetComputeUnits, OutputMemoTooLarge, nil, nil
	}
	exists, metadata, owner, url, _, err := storage.GetNFT(ctx, mu, t.NFT)
	if err != nil {
		return false, TransferNFTComputeUnits, utils.ErrBytes(err), nil, nil
	}
	if !exists {
		return false, TransferNFTComputeUnits, utils.ErrBytes(OutputNFTNotFound), nil, nil
	}
	if actor != owner {
		return false, TransferNFTComputeUnits, utils.ErrBytes(OutputNotOwner), nil, nil
	}

	if err := storage.SetNFT(ctx, mu, t.NFT, metadata, t.To, string(url)); err != nil {
		return false, TransferNFTComputeUnits, utils.ErrBytes(err), nil, nil
	}
	return true, TransferNFTComputeUnits, nil, nil, nil
}

func (*TransferNFT) MaxComputeUnits(chain.Rules) uint64 {
	return TransferNFTComputeUnits
}

func (t *TransferNFT) Size() int {
	return codec.AddressLen + consts.IDLen + codec.BytesLen(t.Memo)
}

func (t *TransferNFT) Marshal(p *codec.Packer) {
	p.PackAddress(t.To)
	p.PackID(t.NFT)
	p.PackBytes(t.Memo)
}

func UnmarshalTransferNFT(p *codec.Packer, _ *warp.Message) (chain.Action, error) {
	var transfer TransferNFT
	p.UnpackAddress(&transfer.To)
	p.UnpackID(false, &transfer.NFT) // empty ID is the native asset
	p.UnpackBytes(MaxMemoSize, false, &transfer.Memo)
	return &transfer, p.Err()
}

func (*TransferNFT) ValidRange(chain.Rules) (int64, int64) {
	// Returning -1, -1 means that the action is always valid.
	return -1, -1
}
