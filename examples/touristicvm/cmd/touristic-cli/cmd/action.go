// Copyright (C) 2023, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package cmd

import (
	"context"
	"errors"
	"github.com/ava-labs/avalanchego/ids"
	"github.com/ava-labs/avalanchego/utils/set"
	"github.com/ava-labs/avalanchego/vms/platformvm/warp"
	"github.com/ava-labs/hypersdk/chain"
	"github.com/ava-labs/hypersdk/codec"
	"github.com/ava-labs/hypersdk/consts"
	"github.com/ava-labs/hypersdk/examples/touristicvm/actions"
	tconsts "github.com/ava-labs/hypersdk/examples/touristicvm/consts"
	trpc "github.com/ava-labs/hypersdk/examples/touristicvm/rpc"
	"github.com/ava-labs/hypersdk/pubsub"
	"github.com/ava-labs/hypersdk/rpc"
	hutils "github.com/ava-labs/hypersdk/utils"
	"github.com/manifoldco/promptui"
	"github.com/spf13/cobra"
	"strconv"
	"time"
)

var actionCmd = &cobra.Command{
	Use: "action",
	RunE: func(*cobra.Command, []string) error {
		return ErrMissingSubcommand
	},
}

var transferCmd = &cobra.Command{
	Use: "transfer",
	RunE: func(*cobra.Command, []string) error {
		ctx := context.Background()
		_, priv, factory, cli, sCli, tCli, err := handler.DefaultActor()
		if err != nil {
			return err
		}

		// Get balance info
		balance, err := handler.GetBalance(ctx, tCli, priv.Address)
		if balance == 0 || err != nil {
			return err
		}

		// Select recipient
		recipient, err := handler.Root().PromptAddress("recipient")
		if err != nil {
			return err
		}

		// Select amount
		amount, err := handler.Root().PromptAmount("amount", tconsts.Decimals, balance, nil)
		if err != nil {
			return err
		}

		// Confirm action
		cont, err := handler.Root().PromptContinue()
		if !cont || err != nil {
			return err
		}

		// Generate transaction
		_, _, err = sendAndWait(ctx, nil, &actions.Transfer{
			To:    recipient,
			Value: amount,
		}, cli, sCli, tCli, factory, true)
		return err
	},
}
var createAssetCmd = &cobra.Command{
	Use: "create-asset",
	RunE: func(*cobra.Command, []string) error {
		ctx := context.Background()
		_, _, factory, cli, scli, tcli, err := handler.DefaultActor()
		if err != nil {
			return err
		}

		// Add symbol to token
		symbol, err := handler.Root().PromptString("symbol", 1, actions.MaxSymbolSize)
		if err != nil {
			return err
		}

		// Add decimal to token
		decimals, err := handler.Root().PromptInt("decimals", actions.MaxDecimals)
		if err != nil {
			return err
		}

		// Add metadata to token
		metadata, err := handler.Root().PromptString("metadata", 1, actions.MaxMetadataSize)
		if err != nil {
			return err
		}

		promptMaxSupply := promptui.Prompt{Label: "max supply (empty or '0' for 'default/MaxUint64')",
			Validate: func(input string) error {
				var err error
				if input != "" {
					_, err = strconv.ParseUint(input, 10, 64)
				}
				return err
			},
		}
		maxSupplyText, err := promptMaxSupply.Run()
		if err != nil {
			return err
		}
		// Confirm action
		cont, err := handler.Root().PromptContinue()
		if !cont || err != nil {
			return err
		}

		var maxSupply uint64

		if maxSupplyText != "" {
			maxSupply, err = strconv.ParseUint(maxSupplyText, 10, 64)
			if err != nil {
				return err
			}
		} else {
			maxSupply = consts.MaxUint64
		}

		// Generate transaction
		_, _, err = sendAndWait(ctx, nil, &actions.CreateAsset{
			Symbol:    []byte(symbol),
			Decimals:  uint8(decimals), // already constrain above to prevent overflow
			Metadata:  []byte(metadata),
			MaxSupply: maxSupply,
		}, cli, scli, tcli, factory, true)
		return err
	},
}
var mintAssetCmd = &cobra.Command{
	Use: "mint-asset",
	RunE: func(*cobra.Command, []string) error {
		ctx := context.Background()
		_, priv, factory, cli, scli, tcli, err := handler.DefaultActor()
		if err != nil {
			return err
		}

		// Select token to mint
		assetID, err := handler.Root().PromptAsset("assetID", false)
		if err != nil {
			return err
		}
		exists, symbol, decimals, metadata, supply, maxSupply, owner, warp, err := tcli.Asset(ctx, assetID, false)
		if err != nil {
			return err
		}
		if !exists {
			hutils.Outf("{{red}}%s does not exist{{/}}\n", assetID)
			hutils.Outf("{{red}}exiting...{{/}}\n")
			return nil
		}
		if warp {
			hutils.Outf("{{red}}cannot mint a warped asset{{/}}\n", assetID)
			hutils.Outf("{{red}}exiting...{{/}}\n")
			return nil
		}
		if owner != codec.MustAddressBech32(tconsts.HRP, priv.Address) {
			hutils.Outf("{{red}}%s is the owner of %s, you are not{{/}}\n", owner, assetID)
			hutils.Outf("{{red}}exiting...{{/}}\n")
			return nil
		}
		hutils.Outf(
			"{{yellow}}symbol:{{/}} %s {{yellow}}decimals:{{/}} %s {{yellow}}metadata:{{/}} %s {{yellow}}supply:{{/}} %d {{yellow}}max-supply:{{/}} %d\n",
			string(symbol),
			decimals,
			string(metadata),
			supply,
			maxSupply,
		)

		// Select recipient
		recipient, err := handler.Root().PromptAddress("recipient")
		if err != nil {
			return err
		}

		// Select amount
		amount, err := handler.Root().PromptAmount("amount", decimals, consts.MaxUint64-supply, nil)
		if err != nil {
			return err
		}

		// Confirm action
		cont, err := handler.Root().PromptContinue()
		if !cont || err != nil {
			return err
		}

		// Generate transaction
		_, _, err = sendAndWait(ctx, nil, &actions.MintAsset{
			Asset: assetID,
			To:    recipient,
			Value: amount,
		}, cli, scli, tcli, factory, true)
		return err
	},
}
var createNFTCmd = &cobra.Command{
	Use: "create-nft",
	RunE: func(*cobra.Command, []string) error {
		ctx := context.Background()
		_, _, factory, cli, scli, tcli, err := handler.DefaultActor()
		if err != nil {
			return err
		}

		//// Add symbol to token
		//ID, err := handler.Root().PromptString("ID", 1, 256)
		//if err != nil {
		//	return err
		//}

		// Add decimal to token
		URL, err := handler.Root().PromptString("NFT Url", 1, 256)
		if err != nil {
			return err
		}

		// Add metadata to token
		metadata, err := handler.Root().PromptString("metadata", 1, actions.MaxMetadataSize)
		if err != nil {
			return err
		}

		owner, err := handler.Root().PromptAddress("recipient")
		if err != nil {
			return err
		}
		// Confirm action
		cont, err := handler.Root().PromptContinue()
		if !cont || err != nil {
			return err
		}

		nft := &actions.CreateNFT{
			Metadata: []byte(metadata),
			Owner:    owner,
			URL:      []byte(URL),
		}

		// Generate transaction
		_, _, err = sendAndWait(ctx, nil, nft, cli, scli, tcli, factory, true)
		return err
	},
}
var getNFTCmd = &cobra.Command{
	Use: "get-nft",
	RunE: func(*cobra.Command, []string) error {

		ctx := context.Background()
		_, priv, _, _, _, tCli, err := handler.DefaultActor()
		if err != nil {
			return err
		}

		nftID, err := handler.Root().PromptAsset("nftID", false)
		if err != nil {
			return err
		}
		exists, metadata, owner, url, err := tCli.NFT(ctx, nftID, false)
		if err != nil {
			return err
		}
		if !exists {
			hutils.Outf("{{red}}%s does not exist{{/}}\n", nftID)
			hutils.Outf("{{red}}exiting...{{/}}\n")
			return nil
		}
		if owner != codec.MustAddressBech32(tconsts.HRP, priv.Address) {
			hutils.Outf("{{red}}%s is the owner of %s, you are not{{/}}\n", owner, nftID)
			hutils.Outf("{{red}}exiting...{{/}}\n")
			return nil
		}
		hutils.Outf(
			"{{yellow}}owner:{{/}} %s {{yellow}}url:{{/}} %s {{yellow}}metadata:{{/}} %s\n",
			owner,
			string(url),
			string(metadata),
		)
		return err
	},
}
var transferNFTCmd = &cobra.Command{
	Use: "transfer-nft",
	RunE: func(*cobra.Command, []string) error {
		ctx := context.Background()
		_, priv, factory, cli, sCli, tCli, err := handler.DefaultActor()
		if err != nil {
			return err
		}

		nftID, err := handler.Root().PromptAsset("nftID", false)
		if err != nil {
			return err
		}
		exists, _, owner, _, err := tCli.NFT(ctx, nftID, false)
		if err != nil {
			return err
		}
		if !exists {
			hutils.Outf("{{red}}%s does not exist{{/}}\n", nftID)
			hutils.Outf("{{red}}exiting...{{/}}\n")
			return nil
		}
		if owner != codec.MustAddressBech32(tconsts.HRP, priv.Address) {
			hutils.Outf("{{red}}%s is the owner of %s, you are not{{/}}\n", owner, nftID)
			hutils.Outf("{{red}}exiting...{{/}}\n")
			return nil
		}
		// Select recipient
		recipient, err := handler.Root().PromptAddress("recipient")
		if err != nil {
			return err
		}

		// Confirm action
		cont, err := handler.Root().PromptContinue()
		if !cont || err != nil {
			return err
		}

		// Generate transaction
		_, _, err = sendAndWait(ctx, nil, &actions.TransferNFT{
			NFT: nftID,
			To:  recipient,
		}, cli, sCli, tCli, factory, true)
		return err
	},
}
var burnNFTCmd = &cobra.Command{
	Use: "burn-nft",
	RunE: func(*cobra.Command, []string) error {
		ctx := context.Background()
		_, priv, factory, cli, sCli, tCli, err := handler.DefaultActor()
		if err != nil {
			return err
		}

		nftID, err := handler.Root().PromptAsset("nftID", false)
		if err != nil {
			return err
		}
		exists, _, owner, _, err := tCli.NFT(ctx, nftID, false)
		if err != nil {
			return err
		}
		if !exists {
			hutils.Outf("{{red}}%s does not exist{{/}}\n", nftID)
			hutils.Outf("{{red}}exiting...{{/}}\n")
			return nil
		}
		if owner != codec.MustAddressBech32(tconsts.HRP, priv.Address) {
			hutils.Outf("{{red}}%s is the owner of %s, you are not{{/}}\n", owner, nftID)
			hutils.Outf("{{red}}exiting...{{/}}\n")
			return nil
		}
		// Confirm action
		cont, err := handler.Root().PromptContinue()
		if !cont || err != nil {
			return err
		}

		// Generate transaction
		_, _, err = sendAndWait(ctx, nil, &actions.TransferNFT{
			NFT: nftID,
			To:  codec.BlackholeAddress,
		}, cli, sCli, tCli, factory, true)
		return err
	},
}
var importAssetCmd = &cobra.Command{
	Use: "import-asset",
	RunE: func(*cobra.Command, []string) error {
		ctx := context.Background()
		currentChainID, _, factory, dcli, dscli, dtcli, err := handler.DefaultActor()
		if err != nil {
			return err
		}

		// Select source
		_, uris, err := handler.Root().PromptChain("sourceChainID", set.Of(currentChainID))
		if err != nil {
			return err
		}
		scli := rpc.NewJSONRPCClient(uris[0])

		// Perform import
		return performImport(ctx, scli, dcli, dscli, dtcli, ids.Empty, factory)
	},
}

func performImport(
	ctx context.Context,
	scli *rpc.JSONRPCClient,
	dcli *rpc.JSONRPCClient,
	dscli *rpc.WebSocketClient,
	dtcli *trpc.JSONRPCClient,
	exportTxID ids.ID,
	factory chain.AuthFactory,
) error {
	// Select TxID (if not provided)
	var err error
	if exportTxID == ids.Empty {
		exportTxID, err = handler.Root().PromptID("export txID")
		if err != nil {
			return err
		}
	}

	// Generate warp signature (as long as >= 80% stake)
	var (
		msg                     *warp.Message
		subnetWeight, sigWeight uint64
	)
	for ctx.Err() == nil {
		msg, subnetWeight, sigWeight, err = scli.GenerateAggregateWarpSignature(ctx, exportTxID)
		if sigWeight >= (subnetWeight*4)/5 && err == nil {
			break
		}
		if err == nil {
			hutils.Outf(
				"{{yellow}}waiting for signature weight:{{/}} %d {{yellow}}observed:{{/}} %d\n",
				subnetWeight,
				sigWeight,
			)
		} else {
			hutils.Outf("{{red}}encountered error:{{/}} %v\n", err)
		}
		cont, err := handler.Root().PromptBool("try again")
		if err != nil {
			return err
		}
		if !cont {
			hutils.Outf("{{red}}exiting...{{/}}\n")
			return nil
		}
	}
	if ctx.Err() != nil {
		return ctx.Err()
	}
	wt, err := actions.UnmarshalWarpTransfer(msg.UnsignedMessage.Payload)
	if err != nil {
		return err
	}
	outputAssetID := wt.Asset
	if !wt.Return {
		outputAssetID = actions.ImportedAssetID(wt.Asset, msg.SourceChainID)
	}
	hutils.Outf(
		"%s {{yellow}}to:{{/}} %s {{yellow}}source assetID:{{/}} %s {{yellow}}source symbol:{{/}} %s {{yellow}}output assetID:{{/}} %s {{yellow}}value:{{/}} %s {{yellow}}reward:{{/}} %s {{yellow}}return:{{/}} %t\n",
		hutils.ToID(
			msg.UnsignedMessage.Payload,
		),
		codec.MustAddressBech32(tconsts.HRP, wt.To),
		wt.Asset,
		wt.Symbol,
		outputAssetID,
		hutils.FormatBalance(wt.Value, wt.Decimals),
		hutils.FormatBalance(wt.Reward, wt.Decimals),
		wt.Return,
	)
	if wt.SwapIn > 0 {
		_, outSymbol, outDecimals, _, _, _, _, _, err := dtcli.Asset(ctx, wt.AssetOut, false)
		if err != nil {
			return err
		}
		hutils.Outf(
			"{{yellow}}asset in:{{/}} %s {{yellow}}swap in:{{/}} %s {{yellow}}asset out:{{/}} %s {{yellow}}symbol out:{{/}} %s {{yellow}}swap out:{{/}} %s {{yellow}}swap expiry:{{/}} %d\n",
			outputAssetID,
			hutils.FormatBalance(wt.SwapIn, wt.Decimals),
			wt.AssetOut,
			outSymbol,
			hutils.FormatBalance(wt.SwapOut, outDecimals),
			wt.SwapExpiry,
		)
	}
	hutils.Outf(
		"{{yellow}}signature weight:{{/}} %d {{yellow}}total weight:{{/}} %d\n",
		sigWeight,
		subnetWeight,
	)

	// Select fill
	var fill bool
	if wt.SwapIn > 0 {
		fill, err = handler.Root().PromptBool("fill")
		if err != nil {
			return err
		}
	}
	if !fill && wt.SwapExpiry > time.Now().UnixMilli() {
		return ErrMustFill
	}

	// Generate transaction
	_, _, err = sendAndWait(ctx, msg, &actions.ImportAsset{
		Fill: fill,
	}, dcli, dscli, dtcli, factory, true)
	return err
}

var exportAssetCmd = &cobra.Command{
	Use: "export-asset",
	RunE: func(*cobra.Command, []string) error {
		ctx := context.Background()
		currentChainID, priv, factory, cli, scli, tcli, err := handler.DefaultActor()
		if err != nil {
			return err
		}

		// Select token to send
		assetID, err := handler.Root().PromptAsset("assetID", true)
		if err != nil {
			return err
		}
		_, decimals, balance, sourceChainID, err := handler.GetAssetInfo(ctx, tcli, priv.Address, assetID, true)
		if balance == 0 || err != nil {
			return err
		}

		// Select recipient
		recipient, err := handler.Root().PromptAddress("recipient")
		if err != nil {
			return err
		}

		// Select amount
		amount, err := handler.Root().PromptAmount("amount", decimals, balance, nil)
		if err != nil {
			return err
		}

		// Determine return
		var ret bool
		if sourceChainID != ids.Empty {
			ret = true
		}

		// Select reward
		reward, err := handler.Root().PromptAmount("reward", decimals, balance-amount, nil)
		if err != nil {
			return err
		}

		// Determine destination
		destination := sourceChainID
		if !ret {
			destination, _, err = handler.Root().PromptChain("destination", set.Of(currentChainID))
			if err != nil {
				return err
			}
		}

		// Determine if swap in
		swap, err := handler.Root().PromptBool("swap on import")
		if err != nil {
			return err
		}
		var (
			swapIn     uint64
			assetOut   ids.ID
			swapOut    uint64
			swapExpiry int64
		)
		if swap {
			swapIn, err = handler.Root().PromptAmount("swap in", decimals, amount, nil)
			if err != nil {
				return err
			}
			assetOut, err = handler.Root().PromptAsset("asset out (on destination)", true)
			if err != nil {
				return err
			}
			uris, err := handler.Root().GetChain(destination)
			if err != nil {
				return err
			}
			networkID, _, _, err := cli.Network(ctx)
			if err != nil {
				return err
			}
			dcli := trpc.NewJSONRPCClient(uris[0], networkID, destination)
			_, decimals, _, _, err := handler.GetAssetInfo(ctx, dcli, priv.Address, assetOut, false)
			if err != nil {
				return err
			}
			swapOut, err = handler.Root().PromptAmount(
				"swap out (on destination, no decimals)",
				decimals,
				consts.MaxUint64,
				nil,
			)
			if err != nil {
				return err
			}
			swapExpiry, err = handler.Root().PromptTime("swap expiry")
			if err != nil {
				return err
			}
		}

		// Confirm action
		cont, err := handler.Root().PromptContinue()
		if !cont || err != nil {
			return err
		}

		// Generate transaction
		success, txID, err := sendAndWait(ctx, nil, &actions.ExportAsset{
			To:          recipient,
			Asset:       assetID,
			Value:       amount,
			Return:      ret,
			Reward:      reward,
			SwapIn:      swapIn,
			AssetOut:    assetOut,
			SwapOut:     swapOut,
			SwapExpiry:  swapExpiry,
			Destination: destination,
		}, cli, scli, tcli, factory, true)
		if err != nil {
			return err
		}
		if !success {
			return errors.New("not successful")
		}

		// Perform import
		imp, err := handler.Root().PromptBool("perform import on destination")
		if err != nil {
			return err
		}
		if imp {
			uris, err := handler.Root().GetChain(destination)
			if err != nil {
				return err
			}
			networkID, _, _, err := cli.Network(ctx)
			if err != nil {
				return err
			}
			dscli, err := rpc.NewWebSocketClient(uris[0], rpc.DefaultHandshakeTimeout, pubsub.MaxPendingMessages, pubsub.MaxReadMessageSize)
			if err != nil {
				return err
			}
			if err := performImport(ctx, cli, rpc.NewJSONRPCClient(uris[0]), dscli, trpc.NewJSONRPCClient(uris[0], networkID, destination), txID, factory); err != nil {
				return err
			}
		}

		// Ask if user would like to switch to destination chain
		sw, err := handler.Root().PromptBool("switch default chain to destination")
		if err != nil {
			return err
		}
		if !sw {
			return nil
		}
		return handler.Root().StoreDefaultChain(destination)
	},
}
