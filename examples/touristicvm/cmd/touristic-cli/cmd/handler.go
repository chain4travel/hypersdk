// Copyright (C) 2023, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package cmd

import (
	"context"
	"fmt"
	hconsts "github.com/ava-labs/hypersdk/consts"

	"github.com/ava-labs/avalanchego/ids"
	"github.com/ava-labs/avalanchego/utils/crypto/secp256k1"
	"github.com/ava-labs/hypersdk/chain"
	"github.com/ava-labs/hypersdk/cli"
	"github.com/ava-labs/hypersdk/codec"
	"github.com/ava-labs/hypersdk/crypto/ed25519"
	"github.com/ava-labs/hypersdk/crypto/secp256r1"
	"github.com/ava-labs/hypersdk/pubsub"
	"github.com/ava-labs/hypersdk/rpc"
	"github.com/ava-labs/hypersdk/utils"
	"github.com/chain4travel/hypersdk/examples/touristicvm/auth"
	"github.com/chain4travel/hypersdk/examples/touristicvm/consts"
	brpc "github.com/chain4travel/hypersdk/examples/touristicvm/rpc"
)

var _ cli.Controller = (*Controller)(nil)

type Handler struct {
	h *cli.Handler
}

func NewHandler(h *cli.Handler) *Handler {
	return &Handler{h}
}

func (h *Handler) Root() *cli.Handler {
	return h.h
}
func (*Handler) GetAssetInfo(
	ctx context.Context,
	cli *brpc.JSONRPCClient,
	addr codec.Address,
	assetID ids.ID,
	checkBalance bool,
) ([]byte, uint8, uint64, ids.ID, error) {
	var sourceChainID ids.ID
	exists, symbol, decimals, metadata, supply, _, _, warp, err := cli.Asset(ctx, assetID, false)
	if err != nil {
		return nil, 0, 0, ids.Empty, err
	}
	if assetID != ids.Empty {
		if !exists {
			utils.Outf("{{red}}%s does not exist{{/}}\n", assetID)
			utils.Outf("{{red}}exiting...{{/}}\n")
			return nil, 0, 0, ids.Empty, nil
		}
		if warp {
			sourceChainID = ids.ID(metadata[hconsts.IDLen:])
			sourceAssetID := ids.ID(metadata[:hconsts.IDLen])
			utils.Outf(
				"{{yellow}}sourceChainID:{{/}} %s {{yellow}}sourceAssetID:{{/}} %s {{yellow}}supply:{{/}} %d\n",
				sourceChainID,
				sourceAssetID,
				supply,
			)
		} else {
			utils.Outf(
				"{{yellow}}symbol:{{/}} %s {{yellow}}decimals:{{/}} %d {{yellow}}metadata:{{/}} %s {{yellow}}supply:{{/}} %d {{yellow}}warp:{{/}} %t\n",
				symbol,
				decimals,
				metadata,
				supply,
				warp,
			)
		}
	}
	if !checkBalance {
		return symbol, decimals, 0, sourceChainID, nil
	}
	saddr, err := codec.AddressBech32(consts.HRP, addr)
	if err != nil {
		return nil, 0, 0, ids.Empty, err
	}
	balance, err := cli.Balance(ctx, saddr)
	if err != nil {
		return nil, 0, 0, ids.Empty, err
	}
	if balance == 0 {
		utils.Outf("{{red}}balance:{{/}} 0 %s\n", assetID)
		utils.Outf("{{red}}please send funds to %s{{/}}\n", saddr)
		utils.Outf("{{red}}exiting...{{/}}\n")
	} else {
		utils.Outf(
			"{{yellow}}balance:{{/}} %s %s\n",
			utils.FormatBalance(balance, decimals),
			symbol,
		)
	}
	return symbol, decimals, balance, sourceChainID, nil
}

func (h *Handler) DefaultActor() (
	ids.ID, *cli.PrivateKey, chain.AuthFactory,
	*rpc.JSONRPCClient, *rpc.WebSocketClient, *brpc.JSONRPCClient, error,
) {
	addr, priv, err := h.h.GetDefaultKey(true)
	if err != nil {
		return ids.Empty, nil, nil, nil, nil, nil, err
	}
	var factory chain.AuthFactory
	switch addr[0] {
	case consts.ED25519ID:
		factory = auth.NewED25519Factory(ed25519.PrivateKey(priv))
	case consts.SECP256R1ID:
		factory = auth.NewSECP256R1Factory(secp256r1.PrivateKey(priv))
	case consts.SECP256K1ID:
		pk, err := secp256k1.ToPrivateKey(priv)
		if err != nil {
			fmt.Errorf("invalid private key %w\n", err)
		}
		factory = auth.NewSECP256K1Factory(*pk)
	default:
		return ids.Empty, nil, nil, nil, nil, nil, ErrInvalidAddress
	}
	chainID, uris, err := h.h.GetDefaultChain(true)
	if err != nil {
		return ids.Empty, nil, nil, nil, nil, nil, err
	}
	jcli := rpc.NewJSONRPCClient(uris[0])
	networkID, _, _, err := jcli.Network(context.TODO())
	if err != nil {
		return ids.Empty, nil, nil, nil, nil, nil, err
	}
	ws, err := rpc.NewWebSocketClient(uris[0], rpc.DefaultHandshakeTimeout, pubsub.MaxPendingMessages, pubsub.MaxReadMessageSize)
	if err != nil {
		return ids.Empty, nil, nil, nil, nil, nil, err
	}
	// For [defaultActor], we always send requests to the first returned URI.
	return chainID, &cli.PrivateKey{
			Address: addr,
			Bytes:   priv,
		}, factory, jcli,
		ws,
		brpc.NewJSONRPCClient(
			uris[0],
			networkID,
			chainID,
		), nil
}

func (*Handler) GetBalance(
	ctx context.Context,
	cli *brpc.JSONRPCClient,
	addr codec.Address,
) (uint64, error) {
	saddr, err := codec.AddressBech32(consts.HRP, addr)
	if err != nil {
		return 0, err
	}
	balance, err := cli.Balance(ctx, saddr)
	if err != nil {
		return 0, err
	}
	if balance == 0 {
		utils.Outf("{{red}}balance:{{/}} 0 %s\n", consts.Symbol)
		utils.Outf("{{red}}please send funds to %s{{/}}\n", saddr)
		utils.Outf("{{red}}exiting...{{/}}\n")
		return 0, nil
	}
	utils.Outf(
		"{{yellow}}balance:{{/}} %s %s\n",
		utils.FormatBalance(balance, consts.Decimals),
		consts.Symbol,
	)
	return balance, nil
}

type Controller struct {
	databasePath string
}

func NewController(databasePath string) *Controller {
	return &Controller{databasePath}
}

func (c *Controller) DatabasePath() string {
	return c.databasePath
}

func (*Controller) Symbol() string {
	return consts.Symbol
}

func (*Controller) Decimals() uint8 {
	return consts.Decimals
}

func (*Controller) Address(addr codec.Address) string {
	return codec.MustAddressBech32(consts.HRP, addr)
}

func (*Controller) ParseAddress(addr string) (codec.Address, error) {
	return codec.ParseAddressBech32(consts.HRP, addr)
}
