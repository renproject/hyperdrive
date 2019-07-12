package sig

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
)

// Hash is the result of Keccak256
type Hash [32]byte

// Equal compares two `Hash`
func (hash Hash) Equal(other Hash) bool {
	return bytes.Equal(hash[:], other[:])
}

// String prints the Hash as a Base64 encoded string.
func (hash Hash) String() string {
	return base64.StdEncoding.EncodeToString(hash[:])
}

func (hash Hash) MarshalBinary() ([]byte, error) {
	return hash[:], nil
}

func (hash *Hash) UnmarshalBinary(data []byte) error {
	copy(hash[:], data[:32])
	return nil
}

// Signature produced by `Sign`
type Signature [65]byte

// Equal compares two `Signatory`
func (sig Signature) Equal(other Signature) bool {
	return bytes.Equal(sig[:], other[:])
}

func (sig Signature) MarshalBinary() ([]byte, error) {
	return sig[:], nil
}

func (sig *Signature) UnmarshalBinary(data []byte) error {
	copy(sig[:], data[:65])
	return nil
}

// Signatures is an array of Signature
type Signatures []Signature

// Equal checks for set equality of Signatures, order does not matter
func (sigs Signatures) Equal(other Signatures) bool {
	if len(sigs) != len(other) {
		return false
	}
	// create a map of string -> int
	diff := make(map[Signature]int, len(sigs))
	for _, _x := range sigs {
		// 0 value for int is 0, so just increment a counter for the string
		diff[_x]++
	}
	for _, _y := range other {
		// If the string _y is not in diff bail out early
		if _, ok := diff[_y]; !ok {
			return false
		}
		diff[_y]--
		if diff[_y] == 0 {
			delete(diff, _y)
		}
	}
	if len(diff) == 0 {
		return true
	}
	return false
}

func (sigs Signatures) MarshalBinary() ([]byte, error) {
	buf := new(bytes.Buffer)
	if err := binary.Write(buf, binary.LittleEndian, uint64(len(sigs))); err != nil {
		return nil, err
	}
	for _, sig := range sigs {
		data, err := sig.MarshalBinary()
		if err != nil {
			return nil, err
		}
		if err := binary.Write(buf, binary.LittleEndian, data); err != nil {
			return nil, err
		}
	}
	return buf.Bytes(), nil
}

func (sigs *Signatures) UnmarshalBinary(data []byte) error {
	buf := bytes.NewBuffer(data)

	var n uint64
	if err := binary.Read(buf, binary.LittleEndian, &n); err != nil {
		return err
	}

	*sigs = make(Signatures, n)
	for i := range *sigs {
		if err := (*sigs)[i].UnmarshalBinary(buf.Bytes()); err != nil {
			return err
		}
	}
	return nil
}

// Signatory is the last 20 bytes of a Public Key
type Signatory [20]byte

// Equal compares two `Signatory`
func (signatory Signatory) Equal(other Signatory) bool {
	return bytes.Equal(signatory[:], other[:])
}

// String prints the Signatory in a Base64 encoding.
func (signatory Signatory) String() string {
	return base64.StdEncoding.EncodeToString(signatory[:])
}

func (signatory Signatory) MarshalBinary() ([]byte, error) {
	return signatory[:], nil
}

func (signatory *Signatory) UnmarshalBinary(data []byte) error {
	copy(signatory[:], data[:20])
	return nil
}

// Signatories is an array of Signatory
type Signatories []Signatory

// Equal checks for set equality of Signatories, order does not matter
func (sig Signatories) Equal(other Signatories) bool {
	if len(sig) != len(other) {
		return false
	}
	// create a map of string -> int
	diff := make(map[Signatory]int, len(sig))
	for _, _x := range sig {
		// 0 value for int is 0, so just increment a counter for the string
		diff[_x]++
	}
	for _, _y := range other {
		// If the string _y is not in diff bail out early
		if _, ok := diff[_y]; !ok {
			return false
		}
		diff[_y]--
		if diff[_y] == 0 {
			delete(diff, _y)
		}
	}
	if len(diff) == 0 {
		return true
	}
	return false
}

func (sigs Signatories) MarshalBinary() ([]byte, error) {
	buf := new(bytes.Buffer)
	if err := binary.Write(buf, binary.LittleEndian, uint64(len(sigs))); err != nil {
		return nil, err
	}
	for _, sig := range sigs {
		data, err := sig.MarshalBinary()
		if err != nil {
			return nil, err
		}
		if err := binary.Write(buf, binary.LittleEndian, data); err != nil {
			return nil, err
		}
	}
	return buf.Bytes(), nil
}

func (sigs *Signatories) UnmarshalBinary(data []byte) error {
	buf := bytes.NewBuffer(data)

	var n uint64
	if err := binary.Read(buf, binary.LittleEndian, &n); err != nil {
		return err
	}

	*sigs = make(Signatories, n)
	for i := range *sigs {
		if err := (*sigs)[i].UnmarshalBinary(buf.Bytes()); err != nil {
			return err
		}
	}
	return nil
}

// Signer signs the provided hash, returning a signature
type Signer interface {
	// Sign a `Hash` and return the resulting `Signature`.
	Sign(hash Hash) (Signature, error)

	// Signatory returns the `Signatory` of your PublicKey
	Signatory() Signatory
}

// A Verifier can return the `Signatory` that produced a `Signature`.
type Verifier interface {
	// Note: Verify will not return an error if the hash and signature
	// do not match, but the returned public key is effectively random.
	Verify(hash Hash, signature Signature) (Signatory, error)
}

// A SignerVerifier combines the `Signer` and `Verifier` interfaces.
type SignerVerifier interface {
	Signer
	Verifier
}
