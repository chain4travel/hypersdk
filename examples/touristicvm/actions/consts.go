// Copyright (C) 2023, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package actions

const (
	// Action TypeIDs
	transferID    uint8 = 0
	mintAssetID   uint8 = 1
	createAssetID uint8 = 2
	createNFTID   uint8 = 3
	getNFTID      uint8 = 4
)

const (
	TransferComputeUnits  = 1
	MintAssetComputeUnits = 2

	CreateNFTComputeUnits   = 10
	CreateAssetComputeUnits = 10

	MaxSymbolSize   = 8
	MaxMemoSize     = 256
	MaxMetadataSize = 256
	MaxDecimals     = 9

	MaxNFTIDSize  = 8
	MaxNFTURLSize = 1000
	MaxOwnerSize  = 32
)
