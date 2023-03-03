package wallet

import (
	"crypto/ecdsa"
	"crypto/rand"
	"encoding/json"
	"errors"

	"github.com/btcsuite/btcd/btcec"

	"github.com/defiweb/go-eth/crypto"
	"github.com/defiweb/go-eth/types"
)

var (
	ErrMissingChainID = errors.New("missing chain ID")
	ErrInvalidSender  = errors.New("transaction sender does not match signer address")
)

var s256 = btcec.S256()

type PrivateKey struct {
	private *ecdsa.PrivateKey
	public  *ecdsa.PublicKey
	address types.Address
}

// NewKeyFromECDSA creates a new private key from an ecdsa.PrivateKey.
func NewKeyFromECDSA(priv *ecdsa.PrivateKey) *PrivateKey {
	return &PrivateKey{
		private: priv,
		public:  &priv.PublicKey,
		address: crypto.PublicKeyToAddress(&priv.PublicKey),
	}
}

// NewKeyFromBytes creates a new private key from private key bytes.
func NewKeyFromBytes(b []byte) *PrivateKey {
	priv, _ := btcec.PrivKeyFromBytes(s256, b)
	return NewKeyFromECDSA((*ecdsa.PrivateKey)(priv))
}

// NewRandomKey creates a random private key.
func NewRandomKey() *PrivateKey {
	priv, err := ecdsa.GenerateKey(s256, rand.Reader)
	if err != nil {
		panic(err)
	}
	return NewKeyFromECDSA(priv)
}

// PublicKey returns the ECDSA public key.
func (k *PrivateKey) PublicKey() *ecdsa.PublicKey {
	return k.public
}

// PrivateKey returns the ECDSA private key.
func (k *PrivateKey) PrivateKey() *ecdsa.PrivateKey {
	return k.private
}

// JSON returns the JSON representation of the private key.
func (k *PrivateKey) JSON(passphrase string, scryptN, scryptP int) ([]byte, error) {
	key, err := encryptV3Key(k.private, passphrase, scryptN, scryptP)
	if err != nil {
		return nil, err
	}
	return json.Marshal(key)
}

// Address implements the Key interface.
func (k *PrivateKey) Address() types.Address {
	return k.address
}

// SignHash implements the Key interface.
func (k *PrivateKey) SignHash(hash types.Hash) (types.Signature, error) {
	return crypto.SignHash(k.private, hash)
}

// SignMessage implements the Key interface.
func (k *PrivateKey) SignMessage(data []byte) (types.Signature, error) {
	return crypto.SignMessage(k.private, data)
}

// SignTransaction implements the Key interface.
func (k *PrivateKey) SignTransaction(tx *types.Transaction) error {
	if tx.ChainID == nil {
		return ErrMissingChainID
	}
	if tx.From != nil && *tx.From != k.Address() {
		return ErrInvalidSender
	}
	r, err := tx.SigningHash(crypto.Keccak256)
	if err != nil {
		return err
	}
	s, err := k.SignHash(r)
	if err != nil {
		return err
	}
	v := uint64(s[types.SignatureLength-1])
	if tx.Type == 0 {
		v = v + 35 + tx.ChainID.Uint64()*2
	}
	addr := k.Address()
	tx.From = &addr
	tx.Signature = &s
	return nil
}

// VerifyHash implements the Key interface.
func (k *PrivateKey) VerifyHash(hash types.Hash, sig types.Signature) bool {
	addr, err := crypto.Ecrecover(hash, sig)
	if err != nil {
		return false
	}
	return addr == k.address
}

// VerifyMessage implements the Key interface.
func (k *PrivateKey) VerifyMessage(data []byte, sig types.Signature) bool {
	addr, err := crypto.EcrecoverMessage(data, sig)
	if err != nil {
		return false
	}
	return addr == k.address
}
