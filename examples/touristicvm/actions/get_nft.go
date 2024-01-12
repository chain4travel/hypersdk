package actions

import (
	"context"
	"fmt"
	"github.com/ava-labs/hypersdk/examples/touristicvm/storage"

	"github.com/ava-labs/avalanchego/ids"
	"github.com/ava-labs/avalanchego/vms/platformvm/warp"
	"github.com/ava-labs/hypersdk/chain"
	"github.com/ava-labs/hypersdk/codec"
	"github.com/ava-labs/hypersdk/consts"
	zutils "github.com/ava-labs/hypersdk/utils"

	"github.com/ava-labs/hypersdk/state"
)

var _ chain.Action = (*GetNFT)(nil)

type GetNFT struct {
	ID       []byte        `json:"id"`
	Metadata []byte        `json:"metadata"`
	Owner    codec.Address `json:"owner"`
	URL      []byte        `json:"url"`
}

func (*GetNFT) GetTypeID() uint8 {
	return getNFTID
}

func (*GetNFT) StateKeys(_ chain.Auth, txID ids.ID) []string {
	return []string{
		string(storage.NFTKey(txID)),
	}
}

func (*GetNFT) StateKeysMaxChunks() []uint16 {
	return []uint16{storage.NFTChunks}
}

func (*GetNFT) OutputsWarpMessage() bool {
	return false
}
func (c *GetNFT) Execute(
	ctx context.Context,
	_ chain.Rules,
	mu state.Mutable,
	_ int64,
	rauth chain.Auth,
	txID ids.ID,
	_ bool,
) (bool, uint64, []byte, *warp.UnsignedMessage, error) {

	if len(c.ID) == 0 {
		return false, CreateNFTComputeUnits, OutputSymbolEmpty, nil, nil
	}
	if len(c.Metadata) == 0 {
		return false, CreateNFTComputeUnits, OutputMetadataEmpty, nil, nil
	}

	if err := storage.SetNFT(ctx, mu, txID, c.Metadata, c.Owner, string(c.URL)); err != nil {

		return false, CreateNFTComputeUnits, zutils.ErrBytes(err), nil, nil
	}

	_, _, _, urlBytes, _, _ := storage.GetNFT(ctx, mu, ids.ID(c.ID))

	fmt.Printf("%s", urlBytes)

	return true, CreateNFTComputeUnits, urlBytes, nil, nil
}
func (*GetNFT) MaxComputeUnits(chain.Rules) uint64 {
	return CreateNFTComputeUnits
}

func (c *GetNFT) Size() int {
	// TODO: add small bytes (smaller int prefix)
	return codec.BytesLen(c.ID) + consts.Uint8Len + codec.BytesLen(c.Metadata) + consts.Uint8Len + codec.AddressLen + consts.Uint8Len + codec.BytesLen(c.URL)
}

func (c *GetNFT) Marshal(p *codec.Packer) {
	p.PackBytes(c.ID)
	p.PackBytes(c.URL)
	p.PackBytes(c.Metadata)
	p.PackAddress(c.Owner)
}

func UnmarshalGetNFT(p *codec.Packer, _ *warp.Message) (chain.Action, error) {
	var create GetNFT
	p.UnpackBytes(MaxNFTIDSize, true, &create.ID)
	p.UnpackBytes(MaxNFTURLSize, true, &create.URL)
	p.UnpackAddress(&create.Owner)
	p.UnpackBytes(MaxMetadataSize, true, &create.Metadata)
	return &create, p.Err()
}

func (*GetNFT) ValidRange(chain.Rules) (int64, int64) {
	// Returning -1, -1 means that the action is always valid.
	return -1, -1
}
