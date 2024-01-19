// Copyright (C) 2023, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package cmd

import (
	"context"
	"fmt"
	"reflect"

	"github.com/ava-labs/avalanchego/ids"
	"github.com/ava-labs/avalanchego/vms/platformvm/warp"
	"github.com/ava-labs/hypersdk/chain"
	"github.com/ava-labs/hypersdk/cli"
	"github.com/ava-labs/hypersdk/codec"
	"github.com/ava-labs/hypersdk/examples/touristicvm/actions"
	"github.com/ava-labs/hypersdk/examples/touristicvm/consts"
	trpc "github.com/ava-labs/hypersdk/examples/touristicvm/rpc"
	"github.com/ava-labs/hypersdk/rpc"
	"github.com/ava-labs/hypersdk/utils"
)

// sendAndWait may not be used concurrently
func sendAndWait(
	ctx context.Context, warpMsg *warp.Message, action chain.Action, cli *rpc.JSONRPCClient,
	ws *rpc.WebSocketClient, tCli *trpc.JSONRPCClient, factory chain.AuthFactory, printStatus bool,
) (bool, ids.ID, error) { //nolint:unparam
	parser, err := tCli.Parser(ctx)
	if err != nil {
		return false, ids.Empty, err
	}
	_, tx, _, err := cli.GenerateTransaction(ctx, parser, warpMsg, action, factory)
	if err != nil {
		return false, ids.Empty, err
	}
	if err := ws.RegisterTx(tx); err != nil {
		return false, ids.Empty, err
	}
	var result *chain.Result
	for {
		txID, txErr, txResult, err := ws.ListenTx(ctx)
		if err != nil {
			return false, ids.Empty, err
		}
		if txErr != nil {
			return false, ids.Empty, txErr
		}
		if txID == tx.ID() {
			result = txResult
			break
		}
		utils.Outf("{{yellow}}skipping unexpected transaction:{{/}} %s\n", tx.ID())
	}
	if printStatus {
		handler.Root().PrintStatus(tx.ID(), result.Success)
	}
	return result.Success, tx.ID(), nil
}

func handleTx(c *trpc.JSONRPCClient, tx *chain.Transaction, result *chain.Result) {
	summaryStr := string(result.Output)
	actor := tx.Auth.Actor()
	status := "❌"
	if result.Success {
		status = "✅"
		switch action := tx.Action.(type) { //nolint:gocritic
		case *actions.Transfer:
			summaryStr = fmt.Sprintf("%s %s -> %s", utils.FormatBalance(action.Value, consts.Decimals), consts.Symbol, codec.MustAddressBech32(consts.HRP, action.To))
		case *actions.CreateAsset:
			summaryStr = fmt.Sprintf("assetID: %s symbol: %s decimals: %d metadata: %s", tx.ID(), action.Symbol, action.Decimals, action.Metadata)
		case *actions.MintAsset:
			_, symbol, decimals, _, _, _, _, _, err := c.Asset(context.TODO(), action.Asset, true)
			if err != nil {
				utils.Outf("{{red}}could not fetch asset info:{{/}} %v", err)
				return
			}
			amountStr := utils.FormatBalance(action.Value, decimals)
			summaryStr = fmt.Sprintf("%s %s -> %s", amountStr, symbol, codec.MustAddressBech32(consts.HRP, action.To))
		case *actions.CreateNFT:
			summaryStr = fmt.Sprintf("assetID: %s owner: %s metadata: %s url: %s", tx.ID(), codec.MustAddressBech32(consts.HRP, action.Owner), action.Metadata, action.URL)
		case *actions.TransferNFT:
			exists, metadata, _, url, err := c.NFT(context.TODO(), action.NFT, true)
			if err != nil {
				utils.Outf("{{red}}could not fetch NFT info:{{/}} %v", err)
				return
			} else if !exists {
				utils.Outf("{{red}}could not find NFT:{{/}} %s", action.NFT)
				return
			}
			summaryStr = fmt.Sprintf("assetID: %s new owner: %s metadata: %s url: %s", tx.ID(), codec.MustAddressBech32(consts.HRP, action.To), metadata, url)
		case *actions.ImportAsset:
			wm := tx.WarpMessage
			signers, _ := wm.Signature.NumSigners()
			wt, _ := actions.UnmarshalWarpTransfer(wm.Payload)
			summaryStr = fmt.Sprintf("source: %s signers: %d | ", wm.SourceChainID, signers)
			if wt.Return {
				summaryStr += fmt.Sprintf("%s %s -> %s (return: %t)", utils.FormatBalance(wt.Value, wt.Decimals), wt.Symbol, codec.MustAddressBech32(consts.HRP, wt.To), wt.Return)
			} else {
				summaryStr += fmt.Sprintf("%s %s (new: %s, original: %s) -> %s (return: %t)", utils.FormatBalance(wt.Value, wt.Decimals), wt.Symbol, actions.ImportedAssetID(wt.Asset, wm.SourceChainID), wt.Asset, codec.MustAddressBech32(consts.HRP, wt.To), wt.Return)
			}
			if wt.Reward > 0 {
				summaryStr += fmt.Sprintf(" | reward: %s", utils.FormatBalance(wt.Reward, wt.Decimals))
			}
			if wt.SwapIn > 0 {
				_, outSymbol, outDecimals, _, _, _, _, _, err := c.Asset(context.TODO(), wt.AssetOut, true)
				if err != nil {
					utils.Outf("{{red}}could not fetch asset info:{{/}} %v", err)
					return
				}
				summaryStr += fmt.Sprintf(" | swap in: %s %s swap out: %s %s expiry: %d fill: %t", utils.FormatBalance(wt.SwapIn, wt.Decimals), wt.Symbol, utils.FormatBalance(wt.SwapOut, outDecimals), outSymbol, wt.SwapExpiry, action.Fill)
			}
		case *actions.ExportAsset:
			wt, _ := actions.UnmarshalWarpTransfer(result.WarpMessage.Payload)
			summaryStr = fmt.Sprintf("destination: %s | ", action.Destination)
			var outputAssetID ids.ID
			if !action.Return {
				outputAssetID = actions.ImportedAssetID(action.Asset, result.WarpMessage.SourceChainID)
				summaryStr += fmt.Sprintf("%s %s (%s) -> %s (return: %t)", utils.FormatBalance(action.Value, wt.Decimals), wt.Symbol, action.Asset, codec.MustAddressBech32(consts.HRP, action.To), action.Return)
			} else {
				outputAssetID = wt.Asset
				summaryStr += fmt.Sprintf("%s %s (current: %s, original: %s) -> %s (return: %t)", utils.FormatBalance(action.Value, wt.Decimals), wt.Symbol, action.Asset, wt.Asset, codec.MustAddressBech32(consts.HRP, action.To), action.Return)
			}
			if wt.Reward > 0 {
				summaryStr += fmt.Sprintf(" | reward: %s", utils.FormatBalance(wt.Reward, wt.Decimals))
			}
			if wt.SwapIn > 0 {
				_, outSymbol, outDecimals, _, _, _, _, _, err := c.Asset(context.TODO(), wt.AssetOut, true)
				if err != nil {
					utils.Outf("{{red}}could not fetch asset info:{{/}} %v", err)
					return
				}
				summaryStr += fmt.Sprintf(" | swap in: %s %s (%s) swap out: %s %s expiry: %d", utils.FormatBalance(wt.SwapIn, wt.Decimals), wt.Symbol, outputAssetID, utils.FormatBalance(wt.SwapOut, outDecimals), outSymbol, wt.SwapExpiry)
			}
		}
	}
	utils.Outf(
		"%s {{yellow}}%s{{/}} {{yellow}}actor:{{/}} %s {{yellow}}summary (%s):{{/}} [%s] {{yellow}}fee (max %.2f%%):{{/}} %s %s {{yellow}}consumed:{{/}} [%s]\n",
		status,
		tx.ID(),
		codec.MustAddressBech32(consts.HRP, actor),
		reflect.TypeOf(tx.Action),
		summaryStr,
		float64(result.Fee)/float64(tx.Base.MaxFee)*100,
		utils.FormatBalance(result.Fee, consts.Decimals),
		consts.Symbol,
		cli.ParseDimensions(result.Consumed),
	)
}
