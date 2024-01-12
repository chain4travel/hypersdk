// Copyright (C) 2023, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package rpc

import (
	"context"
	"strings"
	"sync"

	"github.com/ava-labs/avalanchego/ids"

	"github.com/ava-labs/hypersdk/chain"
	"github.com/ava-labs/hypersdk/examples/touristicvm/consts"
	"github.com/ava-labs/hypersdk/examples/touristicvm/genesis"
	_ "github.com/ava-labs/hypersdk/examples/touristicvm/registry" // ensure registry populated
	"github.com/ava-labs/hypersdk/examples/touristicvm/storage"
	"github.com/ava-labs/hypersdk/requester"
	"github.com/ava-labs/hypersdk/rpc"
	"github.com/ava-labs/hypersdk/utils"
)

type JSONRPCClient struct {
	requester *requester.EndpointRequester

	networkID  uint32
	chainID    ids.ID
	g          *genesis.Genesis
	assetsLock sync.Mutex
	assets     map[ids.ID]*AssetReply
	nftLock    sync.Mutex
	nfts       map[ids.ID]*NFTReply
}

// New creates a new client object.
func NewJSONRPCClient(uri string, networkID uint32, chainID ids.ID) *JSONRPCClient {
	uri = strings.TrimSuffix(uri, "/")
	uri += JSONRPCEndpoint
	req := requester.New(uri, consts.Name)
	return &JSONRPCClient{
		requester: req,
		networkID: networkID,
		chainID:   chainID,
		g:         nil,
		assets:    map[ids.ID]*AssetReply{},
	}
}

func (cli *JSONRPCClient) Genesis(ctx context.Context) (*genesis.Genesis, error) {
	if cli.g != nil {
		return cli.g, nil
	}

	resp := new(GenesisReply)
	err := cli.requester.SendRequest(
		ctx,
		"genesis",
		nil,
		resp,
	)
	if err != nil {
		return nil, err
	}
	cli.g = resp.Genesis
	return resp.Genesis, nil
}

func (cli *JSONRPCClient) Tx(ctx context.Context, id ids.ID) (bool, bool, int64, uint64, error) {
	resp := new(TxReply)
	err := cli.requester.SendRequest(
		ctx,
		"tx",
		&TxArgs{TxID: id},
		resp,
	)
	switch {
	// We use string parsing here because the JSON-RPC library we use may not
	// allows us to perform errors.Is.
	case err != nil && strings.Contains(err.Error(), ErrTxNotFound.Error()):
		return false, false, -1, 0, nil
	case err != nil:
		return false, false, -1, 0, err
	}
	return true, resp.Success, resp.Timestamp, resp.Fee, nil
}

func (cli *JSONRPCClient) Asset(
	ctx context.Context,
	asset ids.ID,
	useCache bool,
) (bool, []byte, uint8, []byte, uint64, uint64, string, bool, error) {
	cli.assetsLock.Lock()
	r, ok := cli.assets[asset]
	cli.assetsLock.Unlock()
	if ok && useCache {
		return true, r.Symbol, r.Decimals, r.Metadata, r.Supply, r.MaxSupply, r.Owner, r.Warp, nil
	}
	resp := new(AssetReply)
	err := cli.requester.SendRequest(
		ctx,
		"asset",
		&AssetArgs{
			Asset: asset,
		},
		resp,
	)
	switch {
	// We use string parsing here because the JSON-RPC library we use may not
	// allows us to perform errors.Is.
	case err != nil && strings.Contains(err.Error(), ErrAssetNotFound.Error()):
		return false, nil, 0, nil, 0, 0, "", false, nil
	case err != nil:
		return false, nil, 0, nil, 0, 0, "", false, err
	}
	cli.assetsLock.Lock()
	cli.assets[asset] = resp
	cli.assetsLock.Unlock()
	return true, resp.Symbol, resp.Decimals, resp.Metadata, resp.Supply, resp.MaxSupply, resp.Owner, resp.Warp, nil
}
func (cli *JSONRPCClient) Balance(ctx context.Context, addr string) (uint64, error) {
	resp := new(BalanceReply)
	err := cli.requester.SendRequest(
		ctx,
		"balance",
		&BalanceArgs{
			Address: addr,
		},
		resp,
	)
	return resp.Amount, err
}

func (cli *JSONRPCClient) NFT(
	ctx context.Context,
	nftID ids.ID,
	useCache bool,
) (bool, []byte, []byte, string, []byte, error) {
	cli.nftLock.Lock()
	r, ok := cli.nfts[nftID]
	cli.nftLock.Unlock()
	if ok && useCache {
		return true, r.Symbol, r.Metadata, r.Owner, r.Url, nil
	}
	resp := new(NFTReply)
	err := cli.requester.SendRequest(
		ctx,
		"nft",
		&NFTArgs{
			Id: nftID,
		},
		resp,
	)
	switch {
	// We use string parsing here because the JSON-RPC library we use may not
	// allows us to perform errors.Is.
	case err != nil && strings.Contains(err.Error(), ErrAssetNotFound.Error()):
		return false, nil, nil, "", nil, nil
	case err != nil:
		return false, nil, nil, "", nil, err
	}
	cli.nftLock.Lock()
	cli.nfts[nftID] = resp
	cli.nftLock.Unlock()
	return true, resp.Symbol, resp.Metadata, resp.Owner, resp.Url, nil
}

func (cli *JSONRPCClient) WaitForBalance(
	ctx context.Context,
	addr string,
	min uint64,
) error {
	return rpc.Wait(ctx, func(ctx context.Context) (bool, error) {
		balance, err := cli.Balance(ctx, addr)
		if err != nil {
			return false, err
		}
		shouldExit := balance >= min
		if !shouldExit {
			utils.Outf(
				"{{yellow}}waiting for %s balance: %s{{/}}\n",
				utils.FormatBalance(min, consts.Decimals),
				addr,
			)
		}
		return shouldExit, nil
	})
}

func (cli *JSONRPCClient) WaitForTransaction(ctx context.Context, txID ids.ID) (bool, uint64, error) {
	var success bool
	var fee uint64
	if err := rpc.Wait(ctx, func(ctx context.Context) (bool, error) {
		found, isuccess, _, ifee, err := cli.Tx(ctx, txID)
		if err != nil {
			return false, err
		}
		success = isuccess
		fee = ifee
		return found, nil
	}); err != nil {
		return false, 0, err
	}
	return success, fee, nil
}

var _ chain.Parser = (*Parser)(nil)

type Parser struct {
	networkID uint32
	chainID   ids.ID
	genesis   *genesis.Genesis
}

func (p *Parser) ChainID() ids.ID {
	return p.chainID
}

func (p *Parser) Rules(t int64) chain.Rules {
	return p.genesis.Rules(t, p.networkID, p.chainID)
}

func (*Parser) Registry() (chain.ActionRegistry, chain.AuthRegistry) {
	return consts.ActionRegistry, consts.AuthRegistry
}

func (*Parser) StateManager() chain.StateManager {
	return &storage.StateManager{}
}

func (cli *JSONRPCClient) Parser(ctx context.Context) (chain.Parser, error) {
	g, err := cli.Genesis(ctx)
	if err != nil {
		return nil, err
	}
	return &Parser{cli.networkID, cli.chainID, g}, nil
}
