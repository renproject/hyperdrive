package id

import (
	"bytes"
	"crypto/ecdsa"
	"encoding/base64"
	"encoding/json"
	"fmt"

	"golang.org/x/crypto/sha3"
)

// Constants represent the length of the variables.
const (
	HashLength = 32

	SignatureLength = 65

	SignatoryLength = 32
)

// ErrInvalidJsonBytes returns a error which is returned when unable to
// unmarshal the json bytes because of the incorrect length of the bytes.
func ErrInvalidJsonBytes(t interface{}, expected, got int) error {
	return fmt.Errorf("fail to unmarshal json bytes to %T, expect bytes length %v, got %v", t, expected, got)
}

// Hashes defines a wrapper type around the []Hash type.
type Hashes []Hash

// Equal compares one Hashes with another.
func (hashes Hashes) Equal(other Hashes) bool {
	if len(hashes) != len(other) {
		return false
	}
	for i := range hashes {
		if !hashes[i].Equal(other[i]) {
			return false
		}
	}
	return true
}

// Hash defines the output of the 256-bit SHA3 hashing function.
type Hash [32]byte

// Equal compares one Hash with another.
func (hash Hash) Equal(other Hash) bool {
	return bytes.Equal(hash[:], other[:])
}

// String implements the `fmt.Stringer` interface for the Hash type.
func (hash Hash) String() string {
	return base64.RawStdEncoding.EncodeToString(hash[:])
}

// MarshalJSON implements the `json.Marshaler` interface for the Hash type.
func (hash Hash) MarshalJSON() ([]byte, error) {
	return json.Marshal(hash[:])
}

// UnmarshalJSON implements the `json.Unmarshaler` interface for the Hash type.
func (hash *Hash) UnmarshalJSON(data []byte) error {
	v := []byte{}
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}
	if len(v) != HashLength {
		return ErrInvalidJsonBytes(*hash, HashLength, len(v))
	}
	copy(hash[:], v)
	return nil
}

// Signatures defines a wrapper type around the []Signature type.
type Signatures []Signature

// Equal compares one Hashes with another.
func (sigs Signatures) Equal(other Signatures) bool {
	if len(sigs) != len(other) {
		return false
	}
	for i := range sigs {
		if !sigs[i].Equal(other[i]) {
			return false
		}
	}
	return true
}

// Hash returns a 256-bit SHA3 hash of the Signatures by converting them into
// bytes and concatenating them to each other.
func (sigs Signatures) Hash() Hash {
	data := make([]byte, 0, 64*len(sigs))
	for _, sig := range sigs {
		data = append(data, sig[:]...)
	}
	return sha3.Sum256(data)
}

// String implements the `fmt.Stringer` interface for the Signatures type.
func (sigs Signatures) String() string {
	hash := sigs.Hash()
	return base64.RawStdEncoding.EncodeToString(hash[:])
}

// Signature defines the ECDSA signature of a Hash. Encoded as R, S, V.
type Signature [65]byte

// Equal compares one Signature with another.
func (sig Signature) Equal(other Signature) bool {
	return bytes.Equal(sig[:], other[:])
}

// String implements the `fmt.Stringer` interface for the Hash type.
func (sig Signature) String() string {
	return base64.RawStdEncoding.EncodeToString(sig[:])
}

// MarshalJSON implements the `json.Marshaler` interface for the Signature type.
func (sig Signature) MarshalJSON() ([]byte, error) {
	return json.Marshal(sig[:])
}

// UnmarshalJSON implements the `json.Unmarshaler` interface for the Signature type.
func (sig *Signature) UnmarshalJSON(data []byte) error {
	v := []byte{}
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}
	if len(v) != SignatureLength {
		return ErrInvalidJsonBytes(*sig, SignatureLength, len(v))
	}
	copy(sig[:], v)
	return nil
}

// Signatories defines a wrapper type around the []Signatory type.
type Signatories []Signatory

// Signatory defines the Hash of the ECDSA public key that is recovered from a
// Signature.
type Signatory [32]byte

// NewSignatory returns the the Signatory of the given ECSDA.PublicKey
func NewSignatory(pubKey ecdsa.PublicKey) Signatory {
	pubKeyBytes := append(pubKey.X.Bytes(), pubKey.Y.Bytes()...)
	return sha3.Sum256(pubKeyBytes)
}

// Equal compares one Signatory with another.
func (sig Signatory) Equal(other Signatory) bool {
	return bytes.Equal(sig[:], other[:])
}

// String implements the `fmt.Stringer` interface for the Signatory type.
func (sig Signatory) String() string {
	return base64.RawStdEncoding.EncodeToString(sig[:])
}

// MarshalJSON implements the `json.Marshaler` interface for the Signatory type.
func (sig Signatory) MarshalJSON() ([]byte, error) {
	return json.Marshal(sig[:])
}

// UnmarshalJSON implements the `json.Unmarshaler` interface for the Signatory type.
func (sig *Signatory) UnmarshalJSON(data []byte) error {
	v := []byte{}
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}
	if len(v) != SignatoryLength {
		return ErrInvalidJsonBytes(*sig, SignatoryLength, len(v))
	}
	copy(sig[:], v)
	return nil
}

// Hash returns a 256-bit SHA3 hash of the Signatories by converting them into
// bytes and concatenating them to each other.
func (sigs Signatories) Hash() Hash {
	data := make([]byte, 0, 32*len(sigs))
	for _, sig := range sigs {
		data = append(data, sig[:]...)
	}
	return sha3.Sum256(data)
}

func (sigs Signatories) Equal(other Signatories) bool {
	if len(sigs) != len(other) {
		return false
	}
	for i := range sigs {
		if !sigs[i].Equal(other[i]) {
			return false
		}
	}
	return true
}

// String implements the `fmt.Stringer` interface for the Signatories type.
func (sigs Signatories) String() string {
	hash := sigs.Hash()
	return base64.RawStdEncoding.EncodeToString(hash[:])
}
