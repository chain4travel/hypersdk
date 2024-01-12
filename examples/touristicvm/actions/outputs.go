// Copyright (C) 2023, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package actions

var (
	OutputValueZero     = []byte("value is zero")
	OutputMemoTooLarge  = []byte("memo is too large")
	OutputAssetIsNative = []byte("cannot mint native asset")
	OutputAssetMissing  = []byte("asset missing")
	OutputWarpAsset     = []byte("warp asset")
	OutputWrongOwner    = []byte("wrong owner")

	OutputSymbolEmpty       = []byte("symbol is empty")
	OutputSymbolIncorrect   = []byte("symbol is incorrect")
	OutputSymbolTooLarge    = []byte("symbol is too large")
	OutputDecimalsIncorrect = []byte("decimal is incorrect")
	OutputDecimalsTooLarge  = []byte("decimal is too large")
	OutputMetadataEmpty     = []byte("metadata is empty")
	OutputMetadataTooLarge  = []byte("metadata is too large")
	MaxSupplyTooLarge       = []byte("max supply is too large")
)
