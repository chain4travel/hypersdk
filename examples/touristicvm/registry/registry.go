// Copyright (C) 2023, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package registry

import (
	"github.com/ava-labs/avalanchego/utils/wrappers"
	"github.com/ava-labs/avalanchego/vms/platformvm/warp"
	"github.com/ava-labs/hypersdk/chain"
	"github.com/ava-labs/hypersdk/codec"

	"github.com/chain4travel/hypersdk/examples/touristicvm/actions"
	"github.com/chain4travel/hypersdk/examples/touristicvm/auth"
	"github.com/chain4travel/hypersdk/examples/touristicvm/consts"
)

// Setup types
func init() {
	consts.ActionRegistry = codec.NewTypeParser[chain.Action, *warp.Message]()
	consts.AuthRegistry = codec.NewTypeParser[chain.Auth, *warp.Message]()

	errs := &wrappers.Errs{}
	errs.Add(
		// When registering new actions, ALWAYS make sure to append at the end.
		consts.ActionRegistry.Register((&actions.Transfer{}).GetTypeID(), actions.UnmarshalTransfer, false),

		consts.ActionRegistry.Register((&actions.CreateAsset{}).GetTypeID(), actions.UnmarshalCreateAsset, false),
		consts.ActionRegistry.Register((&actions.MintAsset{}).GetTypeID(), actions.UnmarshalMintAsset, false),
		consts.ActionRegistry.Register((&actions.CreateNFT{}).GetTypeID(), actions.UnmarshalCreateNFT, false),
		consts.ActionRegistry.Register((&actions.GetNFT{}).GetTypeID(), actions.UnmarshalGetNFT, false),
		consts.ActionRegistry.Register((&actions.TransferNFT{}).GetTypeID(), actions.UnmarshalTransferNFT, false),

		consts.ActionRegistry.Register((&actions.ImportAsset{}).GetTypeID(), actions.UnmarshalImportAsset, true),
		consts.ActionRegistry.Register((&actions.ExportAsset{}).GetTypeID(), actions.UnmarshalExportAsset, false),

		// When registering new auth, ALWAYS make sure to append at the end.
		consts.AuthRegistry.Register((&auth.ED25519{}).GetTypeID(), auth.UnmarshalED25519, false),
		consts.AuthRegistry.Register((&auth.SECP256R1{}).GetTypeID(), auth.UnmarshalSECP256R1, false),
	)
	if errs.Errored() {
		panic(errs.Err)
	}
}
