package auth

import (
	"context"
	"fmt"
	"github.com/ava-labs/avalanchego/utils/crypto/secp256k1"
	"github.com/ava-labs/avalanchego/vms/platformvm/warp"
	"github.com/ava-labs/hypersdk/chain"
	"github.com/ava-labs/hypersdk/codec"
	"github.com/ava-labs/hypersdk/crypto"
	"github.com/ava-labs/hypersdk/utils"
	consts "github.com/chain4travel/hypersdk/examples/touristicvm/consts"
)

const (
	SECP256K1ComputeUnits = 10 // can't be batched like ed25519
	SECP256K1Size         = secp256k1.PublicKeyLen + secp256k1.SignatureLen
)

type SECP256K1 struct {
	Signer    secp256k1.PublicKey `json:"signer"`
	Signature []byte              `json:"signature"`

	addr codec.Address
}

func (s *SECP256K1) address() codec.Address {
	if s.addr == codec.EmptyAddress {
		s.addr = NewSECP256K1Address(s.Signer)
	}
	return s.addr
}
func (s *SECP256K1) GetTypeID() uint8 {
	return consts.SECP256K1ID
}

func (s *SECP256K1) Verify(ctx context.Context, msg []byte) error {
	if !s.Signer.Verify(msg, s.Signature) {
		return crypto.ErrInvalidSignature
	}
	return nil
}

func (s *SECP256K1) ValidRange(rules chain.Rules) (start int64, end int64) {
	return -1, -1
}

func (s *SECP256K1) Marshal(p *codec.Packer) {
	p.PackFixedBytes(s.Signer.Bytes())
	p.PackFixedBytes(s.Signature[:])
}

func UnmarshalSECP256K1(p *codec.Packer, _ *warp.Message) (chain.Auth, error) {
	var d SECP256K1
	signerBytes := make([]byte, secp256k1.PublicKeyLen) // avoid allocating additional memory
	p.UnpackFixedBytes(secp256k1.PublicKeyLen, &signerBytes)
	fmt.Println("UnmarshalSECP256K1", signerBytes)
	fmt.Println("len(signerBytes)", len(signerBytes))
	signer, err := secp256k1.ToPublicKey(signerBytes)
	if err != nil {
		return nil, err
	}
	d.Signer = *signer
	signature := make([]byte, secp256k1.SignatureLen) // avoid allocating additional memory
	p.UnpackFixedBytes(secp256k1.SignatureLen, &signature)
	d.Signature = signature
	return &d, p.Err()
}
func (s *SECP256K1) Size() int {
	return SECP256K1Size
}

func (s *SECP256K1) ComputeUnits(rules chain.Rules) uint64 {
	return SECP256K1ComputeUnits
}

func (s *SECP256K1) Actor() codec.Address {
	return s.address()
}

func (s *SECP256K1) Sponsor() codec.Address {
	return s.address()
}

var _ chain.AuthFactory = (*SECP256K1Factory)(nil)

type SECP256K1Factory struct {
	priv secp256k1.PrivateKey
}

func NewSECP256K1Factory(priv secp256k1.PrivateKey) *SECP256K1Factory {
	return &SECP256K1Factory{priv: priv}
}

func (f *SECP256K1Factory) Sign(msg []byte) (chain.Auth, error) {
	sig, err := f.priv.Sign(msg)
	if err != nil {
		return nil, err
	}
	return &SECP256K1{Signer: *f.priv.PublicKey(), Signature: sig}, nil
}

func (*SECP256K1Factory) MaxUnits() (uint64, uint64) {
	return SECP256K1Size, SECP256K1ComputeUnits
}

func NewSECP256K1Address(pk secp256k1.PublicKey) codec.Address {
	return codec.CreateAddress(consts.SECP256K1ID, utils.ToID(pk.Bytes()))
}
