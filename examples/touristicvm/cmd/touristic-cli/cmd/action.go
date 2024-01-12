// Copyright (C) 2023, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package cmd

import (
	"context"
	"github.com/ava-labs/hypersdk/codec"
	"github.com/ava-labs/hypersdk/consts"
	"github.com/ava-labs/hypersdk/examples/touristicvm/actions"
	tconsts "github.com/ava-labs/hypersdk/examples/touristicvm/consts"
	hutils "github.com/ava-labs/hypersdk/utils"
	"github.com/manifoldco/promptui"
	"github.com/spf13/cobra"
	"strconv"
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
		URL, err := handler.Root().PromptString("Asset Url", 1, 256)
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
		_, priv, _, _, _, tcli, err := handler.DefaultActor()
		if err != nil {
			return err
		}

		// Add symbol to token
		//nftID, err := handler.Root().PromptString("NFT ID", 1, 256)
		//if err != nil {
		//	return err
		//}
		nftID, err := handler.Root().PromptAsset("nftID", false)
		if err != nil {
			return err
		}
		exists, symbol, metadata, owner, url, err := tcli.NFT(ctx, nftID, false)
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
			"{{yellow}}symbol:{{/}} %s {{yellow}}decimals:{{/}} %s {{yellow}}metadata:{{/}} %s {{yellow}}supply:{{/}} %d {{yellow}}max-supply:{{/}} %d {{yellow}}url:{{/}} %s\n",
			string(symbol),
			string(metadata),
			string(url),
		)
		return err
	},
}
