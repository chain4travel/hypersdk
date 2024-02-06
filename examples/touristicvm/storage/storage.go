// Copyright (C) 2023, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package storage

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"sync"

	"github.com/ava-labs/avalanchego/database"
	"github.com/ava-labs/avalanchego/ids"
	smath "github.com/ava-labs/avalanchego/utils/math"
	"github.com/ava-labs/hypersdk/chain"
	"github.com/ava-labs/hypersdk/codec"
	"github.com/ava-labs/hypersdk/consts"
	"github.com/ava-labs/hypersdk/state"

	mconsts "github.com/ava-labs/hypersdk/examples/touristicvm/consts"
)

type ReadState func(context.Context, [][]byte) ([][]byte, []error)

// Metadata
// 0x0/ (tx)
//   -> [txID] => timestamp
//
// State
// / (height) => store in root
//   -> [heightPrefix] => height
// 0x0/ (balance)
//   -> [owner] => balance
// 0x1/ (hypersdk-height)
// 0x2/ (hypersdk-timestamp)
// 0x3/ (hypersdk-fee)
// 0x4/ (hypersdk-incoming warp)
// 0x5/ (hypersdk-outgoing warp)

const (
	// metaDB
	txPrefix = 0x0

	// stateDB
	balancePrefix      = 0x0
	assetPrefix        = 0x1
	nftPrefix          = 0x2
	loanPrefix         = 0x3
	heightPrefix       = 0x4
	timestampPrefix    = 0x5
	feePrefix          = 0x6
	incomingWarpPrefix = 0x7
	outgoingWarpPrefix = 0x8
)

const (
	BalanceChunks uint16 = 1
	AssetChunks   uint16 = 5
	NFTChunks     uint16 = 5
	LoanChunks    uint16 = 1
)

var (
	failureByte  = byte(0x0)
	successByte  = byte(0x1)
	heightKey    = []byte{heightPrefix}
	timestampKey = []byte{timestampPrefix}
	feeKey       = []byte{feePrefix}

	balanceKeyPool = sync.Pool{
		New: func() any {
			return make([]byte, 1+codec.AddressLen+consts.IDLen+consts.Uint16Len)
		},
	}
)

// [txPrefix] + [txID]
func TxKey(id ids.ID) (k []byte) {
	k = make([]byte, 1+consts.IDLen)
	k[0] = txPrefix
	copy(k[1:], id[:])
	return
}

func StoreTransaction(
	_ context.Context,
	db database.KeyValueWriter,
	id ids.ID,
	t int64,
	success bool,
	units chain.Dimensions,
	fee uint64,
) error {
	k := TxKey(id)
	v := make([]byte, consts.Uint64Len+1+chain.DimensionsLen+consts.Uint64Len)
	binary.BigEndian.PutUint64(v, uint64(t))
	if success {
		v[consts.Uint64Len] = successByte
	} else {
		v[consts.Uint64Len] = failureByte
	}
	copy(v[consts.Uint64Len+1:], units.Bytes())
	binary.BigEndian.PutUint64(v[consts.Uint64Len+1+chain.DimensionsLen:], fee)
	return db.Put(k, v)
}

func GetTransaction(
	_ context.Context,
	db database.KeyValueReader,
	id ids.ID,
) (bool, int64, bool, chain.Dimensions, uint64, error) {
	k := TxKey(id)
	v, err := db.Get(k)
	if errors.Is(err, database.ErrNotFound) {
		return false, 0, false, chain.Dimensions{}, 0, nil
	}
	if err != nil {
		return false, 0, false, chain.Dimensions{}, 0, err
	}
	t := int64(binary.BigEndian.Uint64(v))
	success := true
	if v[consts.Uint64Len] == failureByte {
		success = false
	}
	d, err := chain.UnpackDimensions(v[consts.Uint64Len+1 : consts.Uint64Len+1+chain.DimensionsLen])
	if err != nil {
		return false, 0, false, chain.Dimensions{}, 0, err
	}
	fee := binary.BigEndian.Uint64(v[consts.Uint64Len+1+chain.DimensionsLen:])
	return true, t, success, d, fee, nil
}

// [accountPrefix] + [address] + [asset]
func BalanceKey(addr codec.Address, asset ids.ID) (k []byte) {
	k = balanceKeyPool.Get().([]byte)
	k[0] = balancePrefix
	copy(k[1:], addr[:])
	copy(k[1+codec.AddressLen:], asset[:])
	binary.BigEndian.PutUint16(k[1+codec.AddressLen+consts.IDLen:], BalanceChunks)
	return
}

// If locked is 0, then account does not exist
func GetBalance(
	ctx context.Context,
	im state.Immutable,
	addr codec.Address,
	asset ids.ID,
) (uint64, error) {
	key, bal, _, err := getBalance(ctx, im, addr, asset)
	balanceKeyPool.Put(key)
	return bal, err
}
func getBalance(
	ctx context.Context,
	im state.Immutable,
	addr codec.Address,
	asset ids.ID,
) ([]byte, uint64, bool, error) {
	k := BalanceKey(addr, asset)
	bal, exists, err := innerGetBalance(im.GetValue(ctx, k))
	return k, bal, exists, err
}
func GetAssetFromState(
	ctx context.Context,
	f ReadState,
	asset ids.ID,
) (bool, []byte, uint8, []byte, uint64, uint64, codec.Address, bool, error) {
	values, errs := f(ctx, [][]byte{AssetKey(asset)})
	return innerGetAsset(values[0], errs[0])
}

// Used to serve RPC queries
func GetBalanceFromState(
	ctx context.Context,
	f ReadState,
	addr codec.Address,
	asset ids.ID,
) (uint64, error) {
	k := BalanceKey(addr, asset)
	values, errs := f(ctx, [][]byte{k})
	bal, _, err := innerGetBalance(values[0], errs[0])
	balanceKeyPool.Put(k)
	return bal, err
}

func innerGetBalance(
	v []byte,
	err error,
) (uint64, bool, error) {
	if errors.Is(err, database.ErrNotFound) {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, err
	}
	return binary.BigEndian.Uint64(v), true, nil
}

func SetBalance(
	ctx context.Context,
	mu state.Mutable,
	addr codec.Address,
	asset ids.ID,
	balance uint64,
) error {
	k := BalanceKey(addr, asset)
	return setBalance(ctx, mu, k, balance)
}

func setBalance(
	ctx context.Context,
	mu state.Mutable,
	key []byte,
	balance uint64,
) error {
	return mu.Insert(ctx, key, binary.BigEndian.AppendUint64(nil, balance))
}

func AddBalance(
	ctx context.Context,
	mu state.Mutable,
	addr codec.Address,
	asset ids.ID,
	amount uint64,
	create bool,
) error {
	key, bal, exists, err := getBalance(ctx, mu, addr, asset)
	if err != nil {
		return err
	}
	// Don't add balance if account doesn't exist. This
	// can be useful when processing fee refunds.
	if !exists && !create {
		return nil
	}
	nbal, err := smath.Add64(bal, amount)
	if err != nil {
		return fmt.Errorf(
			"%w: could not add balance (asset=%s, bal=%d, addr=%v, amount=%d)",
			ErrInvalidBalance,
			asset,
			bal,
			codec.MustAddressBech32(mconsts.HRP, addr),
			amount,
		)
	}
	return setBalance(ctx, mu, key, nbal)
}

func SubBalance(
	ctx context.Context,
	mu state.Mutable,
	addr codec.Address,
	asset ids.ID,
	amount uint64,
) error {
	key, bal, _, err := getBalance(ctx, mu, addr, asset)
	if err != nil {
		return err
	}
	nbal, err := smath.Sub(bal, amount)
	if err != nil {
		return fmt.Errorf(
			"%w: could not subtract balance (asset=%s, bal=%d, addr=%v, amount=%d)",
			ErrInvalidBalance,
			asset,
			bal,
			codec.MustAddressBech32(mconsts.HRP, addr),
			amount,
		)
	}
	if nbal == 0 {
		// If there is no balance left, we should delete the record instead of
		// setting it to 0.
		return mu.Remove(ctx, key)
	}
	return setBalance(ctx, mu, key, nbal)
}

// [assetPrefix] + [address]
func AssetKey(asset ids.ID) (k []byte) {
	k = make([]byte, 1+consts.IDLen+consts.Uint16Len)
	k[0] = assetPrefix
	copy(k[1:], asset[:])
	binary.BigEndian.PutUint16(k[1+consts.IDLen:], AssetChunks)
	return
}

// [nftPrefix] + [nftID]
func NFTKey(nftID ids.ID) (k []byte) {
	k = make([]byte, 1+consts.IDLen+consts.Uint16Len)
	k[0] = nftPrefix
	copy(k[1:], nftID[:])
	binary.BigEndian.PutUint16(k[1+consts.IDLen:], NFTChunks)
	return
}
func GetNFT(
	ctx context.Context,
	im state.Immutable,
	id ids.ID,
) (bool, []byte, codec.Address, []byte, bool, error) {
	k := NFTKey(id)
	return innerGetNFT(im.GetValue(ctx, k))
}
func GetNFTFromState(
	ctx context.Context,
	f ReadState,
	id ids.ID,
) (bool, []byte, codec.Address, []byte, bool, error) {
	values, errs := f(ctx, [][]byte{NFTKey(id)})
	return innerGetNFT(values[0], errs[0])
}

func innerGetNFT(
	v []byte,
	err error,
) (bool, []byte, codec.Address, []byte, bool, error) {
	if errors.Is(err, database.ErrNotFound) {
		return false, nil, codec.EmptyAddress, nil, false, nil
	}
	if err != nil {
		return false, nil, codec.EmptyAddress, nil, false, err
	}

	metaDataLen := binary.BigEndian.Uint16(v)
	metaData := v[consts.Uint16Len : consts.Uint16Len+metaDataLen]

	var addr codec.Address
	copy(addr[:], v[consts.Uint16Len+metaDataLen:])

	urlLen := binary.BigEndian.Uint16(v[consts.Uint16Len+metaDataLen+codec.AddressLen:])
	urlBytes := v[consts.Uint16Len+metaDataLen+codec.AddressLen+consts.Uint16Len : consts.Uint16Len+metaDataLen+codec.AddressLen+consts.Uint16Len+urlLen]

	return true, metaData, addr, urlBytes, false, nil
}
func SetNFT(
	ctx context.Context,
	mu state.Mutable,
	nftID ids.ID,
	metadata []byte,
	owner codec.Address,
	url string, // URL to store asset metadata on IPFS
) error {
	k := NFTKey(nftID)
	metadataLen := len(metadata)
	urlBytes := []byte(url)
	urlLen := len(urlBytes)
	v := make([]byte, consts.Uint16Len+metadataLen+codec.AddressLen+consts.Uint16Len+urlLen)

	// metadata
	binary.BigEndian.PutUint16(v, uint16(metadataLen))
	copy(v[metadataLen:], metadata)

	// owner address
	copy(v[consts.Uint16Len+metadataLen:], owner[:])

	// url
	binary.BigEndian.PutUint16(v[consts.Uint16Len+metadataLen+codec.AddressLen:], uint16(urlLen))
	copy(v[consts.Uint16Len+metadataLen+codec.AddressLen+consts.Uint16Len:], url)
	return mu.Insert(ctx, k, v)
}
func HeightKey() (k []byte) {
	return heightKey
}

func TimestampKey() (k []byte) {
	return timestampKey
}

func FeeKey() (k []byte) {
	return feeKey
}

func IncomingWarpKeyPrefix(sourceChainID ids.ID, msgID ids.ID) (k []byte) {
	k = make([]byte, 1+consts.IDLen*2)
	k[0] = incomingWarpPrefix
	copy(k[1:], sourceChainID[:])
	copy(k[1+consts.IDLen:], msgID[:])
	return k
}

func OutgoingWarpKeyPrefix(txID ids.ID) (k []byte) {
	k = make([]byte, 1+consts.IDLen)
	k[0] = outgoingWarpPrefix
	copy(k[1:], txID[:])
	return k
}

func GetAsset(
	ctx context.Context,
	im state.Immutable,
	asset ids.ID,
) (bool, []byte, uint8, []byte, uint64, uint64, codec.Address, bool, error) {
	k := AssetKey(asset)
	return innerGetAsset(im.GetValue(ctx, k))
}

func innerGetAsset(
	v []byte,
	err error,
) (bool, []byte, uint8, []byte, uint64, uint64, codec.Address, bool, error) {
	if errors.Is(err, database.ErrNotFound) {
		return false, nil, 0, nil, 0, 0, codec.EmptyAddress, false, nil
	}
	if err != nil {
		return false, nil, 0, nil, 0, 0, codec.EmptyAddress, false, err
	}
	symbolLen := binary.BigEndian.Uint16(v)
	symbol := v[consts.Uint16Len : consts.Uint16Len+symbolLen]
	decimals := v[consts.Uint16Len+symbolLen]
	metadataLen := binary.BigEndian.Uint16(v[consts.Uint16Len+symbolLen+consts.Uint8Len:])
	metadata := v[consts.Uint16Len+symbolLen+consts.Uint8Len+consts.Uint16Len : consts.Uint16Len+symbolLen+consts.Uint8Len+consts.Uint16Len+metadataLen]
	supply := binary.BigEndian.Uint64(v[consts.Uint16Len+symbolLen+consts.Uint8Len+consts.Uint16Len+metadataLen:])
	maxSupply := binary.BigEndian.Uint64(v[consts.Uint16Len+symbolLen+consts.Uint8Len+consts.Uint16Len+metadataLen+consts.Uint64Len:])

	var addr codec.Address
	copy(addr[:], v[consts.Uint16Len+symbolLen+consts.Uint8Len+consts.Uint16Len+metadataLen+consts.Uint64Len+consts.Uint64Len:])
	warp := v[consts.Uint16Len+symbolLen+consts.Uint8Len+consts.Uint16Len+metadataLen+consts.Uint64Len+consts.Uint64Len+codec.AddressLen] == 0x1
	return true, symbol, decimals, metadata, supply, maxSupply, addr, warp, nil
}

func SetAsset(
	ctx context.Context,
	mu state.Mutable,
	asset ids.ID,
	symbol []byte,
	decimals uint8,
	metadata []byte,
	supply uint64,
	maxSupply uint64,
	owner codec.Address,
	warp bool,
) error {
	k := AssetKey(asset)
	symbolLen := len(symbol)
	metadataLen := len(metadata)
	v := make([]byte, consts.Uint16Len+symbolLen+consts.Uint8Len+consts.Uint16Len+metadataLen+consts.Uint64Len+consts.Uint64Len+codec.AddressLen+1)
	binary.BigEndian.PutUint16(v, uint16(symbolLen))
	copy(v[consts.Uint16Len:], symbol)
	v[consts.Uint16Len+symbolLen] = decimals
	binary.BigEndian.PutUint16(v[consts.Uint16Len+symbolLen+consts.Uint8Len:], uint16(metadataLen))
	copy(v[consts.Uint16Len+symbolLen+consts.Uint8Len+consts.Uint16Len:], metadata)
	binary.BigEndian.PutUint64(v[consts.Uint16Len+symbolLen+consts.Uint8Len+consts.Uint16Len+metadataLen:], supply)
	binary.BigEndian.PutUint64(v[consts.Uint16Len+symbolLen+consts.Uint8Len+consts.Uint16Len+metadataLen+consts.Uint64Len:], maxSupply)
	copy(v[consts.Uint16Len+symbolLen+consts.Uint8Len+consts.Uint16Len+metadataLen+consts.Uint64Len+consts.Uint64Len:], owner[:])
	b := byte(0x0)
	if warp {
		b = 0x1
	}
	v[consts.Uint16Len+symbolLen+consts.Uint8Len+consts.Uint16Len+metadataLen+consts.Uint64Len+consts.Uint64Len+codec.AddressLen] = b
	return mu.Insert(ctx, k, v)
}

func DeleteAsset(ctx context.Context, mu state.Mutable, asset ids.ID) error {
	k := AssetKey(asset)
	return mu.Remove(ctx, k)
}

// [loanPrefix] + [asset] + [destination]
func LoanKey(asset ids.ID, destination ids.ID) (k []byte) {
	k = make([]byte, 1+consts.IDLen*2+consts.Uint16Len)
	k[0] = loanPrefix
	copy(k[1:], asset[:])
	copy(k[1+consts.IDLen:], destination[:])
	binary.BigEndian.PutUint16(k[1+consts.IDLen*2:], LoanChunks)
	return
}

// Used to serve RPC queries
func GetLoanFromState(
	ctx context.Context,
	f ReadState,
	asset ids.ID,
	destination ids.ID,
) (uint64, error) {
	values, errs := f(ctx, [][]byte{LoanKey(asset, destination)})
	return innerGetLoan(values[0], errs[0])
}
func innerGetLoan(v []byte, err error) (uint64, error) {
	if errors.Is(err, database.ErrNotFound) {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	return binary.BigEndian.Uint64(v), nil
}
func GetLoan(
	ctx context.Context,
	im state.Immutable,
	asset ids.ID,
	destination ids.ID,
) (uint64, error) {
	k := LoanKey(asset, destination)
	v, err := im.GetValue(ctx, k)
	return innerGetLoan(v, err)
}

func SetLoan(
	ctx context.Context,
	mu state.Mutable,
	asset ids.ID,
	destination ids.ID,
	amount uint64,
) error {
	k := LoanKey(asset, destination)
	return mu.Insert(ctx, k, binary.BigEndian.AppendUint64(nil, amount))
}
func AddLoan(
	ctx context.Context,
	mu state.Mutable,
	asset ids.ID,
	destination ids.ID,
	amount uint64,
) error {
	loan, err := GetLoan(ctx, mu, asset, destination)
	if err != nil {
		return err
	}
	nloan, err := smath.Add64(loan, amount)
	if err != nil {
		return fmt.Errorf(
			"%w: could not add loan (asset=%s, destination=%s, amount=%d)",
			ErrInvalidBalance,
			asset,
			destination,
			amount,
		)
	}
	return SetLoan(ctx, mu, asset, destination, nloan)
}

func SubLoan(
	ctx context.Context,
	mu state.Mutable,
	asset ids.ID,
	destination ids.ID,
	amount uint64,
) error {
	loan, err := GetLoan(ctx, mu, asset, destination)
	if err != nil {
		return err
	}
	nloan, err := smath.Sub(loan, amount)
	if err != nil {
		return fmt.Errorf(
			"%w: could not subtract loan (asset=%s, destination=%s, amount=%d)",
			ErrInvalidBalance,
			asset,
			destination,
			amount,
		)
	}
	if nloan == 0 {
		// If there is no balance left, we should delete the record instead of
		// setting it to 0.
		return mu.Remove(ctx, LoanKey(asset, destination))
	}
	return SetLoan(ctx, mu, asset, destination, nloan)
}