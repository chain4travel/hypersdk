// Copyright (C) 2023, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package rpc

import (
	"net/http"

	"github.com/ava-labs/avalanchego/ids"

	"github.com/ava-labs/hypersdk/chain"
	"github.com/ava-labs/hypersdk/codec"
	"github.com/ava-labs/hypersdk/examples/touristicvm/consts"
	"github.com/ava-labs/hypersdk/examples/touristicvm/genesis"
)

type JSONRPCServer struct {
	c Controller
}

func NewJSONRPCServer(c Controller) *JSONRPCServer {
	return &JSONRPCServer{c}
}

type GenesisReply struct {
	Genesis *genesis.Genesis `json:"genesis"`
}

func (j *JSONRPCServer) Genesis(_ *http.Request, _ *struct{}, reply *GenesisReply) (err error) {
	reply.Genesis = j.c.Genesis()
	return nil
}

type TxArgs struct {
	TxID ids.ID `json:"txId"`
}

type TxReply struct {
	Timestamp int64            `json:"timestamp"`
	Success   bool             `json:"success"`
	Units     chain.Dimensions `json:"units"`
	Fee       uint64           `json:"fee"`
}

func (j *JSONRPCServer) Tx(req *http.Request, args *TxArgs, reply *TxReply) error {
	ctx, span := j.c.Tracer().Start(req.Context(), "Server.Tx")
	defer span.End()

	found, t, success, units, fee, err := j.c.GetTransaction(ctx, args.TxID)
	if err != nil {
		return err
	}
	if !found {
		return ErrTxNotFound
	}
	reply.Timestamp = t
	reply.Success = success
	reply.Units = units
	reply.Fee = fee
	return nil
}

type AssetArgs struct {
	Asset ids.ID `json:"asset"`
}

type AssetReply struct {
	Symbol    []byte `json:"symbol"`
	Decimals  uint8  `json:"decimals"`
	Metadata  []byte `json:"metadata"`
	Supply    uint64 `json:"supply"`
	MaxSupply uint64 `json:"maxSupply"`
	Owner     string `json:"owner"`
	Warp      bool   `json:"warp"`
}

func (j *JSONRPCServer) Asset(req *http.Request, args *AssetArgs, reply *AssetReply) error {
	ctx, span := j.c.Tracer().Start(req.Context(), "Server.NFT")
	defer span.End()

	exists, symbol, decimals, metadata, supply, maxSupply, owner, warp, err := j.c.GetAssetFromState(ctx, args.Asset)
	if err != nil {
		return err
	}
	if !exists {
		return ErrAssetNotFound
	}
	reply.Symbol = symbol
	reply.Decimals = decimals
	reply.Metadata = metadata
	reply.Supply = supply
	reply.MaxSupply = maxSupply
	reply.Owner = codec.MustAddressBech32(consts.HRP, owner)
	reply.Warp = warp
	return err
}

type BalanceArgs struct {
	Address string `json:"address"`
	Asset   ids.ID `json:"asset"`
}

type BalanceReply struct {
	Amount uint64 `json:"amount"`
}

func (j *JSONRPCServer) Balance(req *http.Request, args *BalanceArgs, reply *BalanceReply) error {
	ctx, span := j.c.Tracer().Start(req.Context(), "Server.Balance")
	defer span.End()

	addr, err := codec.ParseAddressBech32(consts.HRP, args.Address)
	if err != nil {
		return err
	}
	balance, err := j.c.GetBalanceFromState(ctx, addr, args.Asset)
	if err != nil {
		return err
	}
	reply.Amount = balance
	return err
}

type NFTArgs struct {
	Id ids.ID `json:"asset"`
}

type NFTReply struct {
	Symbol   []byte `json:"symbol"`
	Metadata []byte `json:"metadata"`
	Owner    string `json:"owner"`
	Url      []byte `json:"url"`
}

func (j *JSONRPCServer) NFT(req *http.Request, args *NFTArgs, reply *NFTReply) error {
	ctx, span := j.c.Tracer().Start(req.Context(), "Server.NFT")
	defer span.End()

	exists, metadata, owner, url, _, err := j.c.GetNFTFromState(ctx, args.Id)
	if err != nil {
		return err
	}
	if !exists {
		return ErrAssetNotFound
	}
	reply.Metadata = metadata
	reply.Owner = codec.MustAddressBech32(consts.HRP, owner)
	reply.Url = url
	return err
}
