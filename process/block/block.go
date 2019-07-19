package block

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"time"

	"golang.org/x/crypto/sha3"
)

type (
	// Hash defines the output of the 256-bit SHA3 hashing function.
	Hash [32]byte
	// Hashes defines a wrapper type around the []Hash type.
	Hashes [32]byte
	// Signature defines the ECDSA signature of a Hash.
	Signature [65]byte
	// Signatures defines a wrapper type around the []Signature type.
	Signatures []Signatory
	// Signatory defines the Hash of the ECDSA pubkey that is recovered from a
	// Signature.
	Signatory [32]byte
	// Signatories defines a wrapper type around the []Signatory type.
	Signatories []Signatory
)

// Equal compares one Hash with another.
func (hash Hash) Equal(other Hash) bool {
	return bytes.Equal(hash[:], other[:])
}

// String implements the `fmt.Stringer` interface for the Hash type.
func (hash Hash) String() string {
	return base64.StdEncoding.WithPadding(base64.NoPadding).EncodeToString(hash[:])
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
	copy(hash[:], v)
	return nil
}

// Equal compares one Signature with another.
func (sig Signature) Equal(other Signature) bool {
	return bytes.Equal(sig[:], other[:])
}

// String implements the `fmt.Stringer` interface for the Hash type.
func (sig Signature) String() string {
	return base64.StdEncoding.WithPadding(base64.NoPadding).EncodeToString(sig[:])
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
	copy(sig[:], v)
	return nil
}

func (sigs Signatures) Hash() Hash {
	data := make([]byte, 0, 65*len(sigs))
	for _, sig := range sigs {
		data = append(data, sig[:]...)
	}
	return Hash(sha3.Sum256(data))
}

// String implements the `fmt.Stringer` interface for the Signatures type.
func (sigs Signatures) String() string {
	hash := sigs.Hash()
	return base64.StdEncoding.WithPadding(base64.NoPadding).EncodeToString(hash[:])
}

// Equal compares one Signatory with another.
func (sig Signatory) Equal(other Signatory) bool {
	return bytes.Equal(sig[:], other[:])
}

// String implements the `fmt.Stringer` interface for the Hash type.
func (sig Signatory) String() string {
	return base64.StdEncoding.WithPadding(base64.NoPadding).EncodeToString(sig[:])
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
	copy(sig[:], v)
	return nil
}

func (sigs Signatories) Hash() Hash {
	data := make([]byte, 0, 32*len(sigs))
	for _, sig := range sigs {
		data = append(data, sig[:]...)
	}
	return Hash(sha3.Sum256(data))
}

// String implements the `fmt.Stringer` interface for the Signatories type.
func (sigs Signatories) String() string {
	hash := sigs.Hash()
	return base64.StdEncoding.WithPadding(base64.NoPadding).EncodeToString(hash[:])
}

// Kind defines the different kinds of Block that exist.
type Kind uint8

const (
	// Invalid define an invalid Kind that should not be used.
	Invalid = iota
	// Standard defines the Kind used for Blocks that represents the need for
	// consensus on application-specific Data. Blocks of this Kind must have nil
	// Header Signatories. This is the most common Kind.
	Standard
	// Rebase defines the Kind used for Blocks that change the Signatories
	// overseeing the consensus algorithm. Blocks of this Kind must have nil
	// Data, and must not have nil Header Signatories.
	Rebase
	// Base defines the Kind used for Blocks that immediately proceed a Rebase.
	// Blocks of this Kind are only used to reach finality on a Rebase; must
	// have nil Data, and must have the same Header Signatories as their parent
	// Block.
	Base
)

// String implements the `fmt.Stringer` interface for the Kind type.
func (kind Kind) String() string {
	switch kind {
	case Standard:
		return "standard"
	case Rebase:
		return "rebase"
	case Base:
		return "base"
	default:
		panic(fmt.Errorf("invariant violation: unexpected kind=%d", uint8(kind)))
	}
}

// A Header defines properties of a Block that are not application-specific.
// These properties are required by, or produced by, the consensus algorithm.
type Header struct {
	kind       Kind      // Kind of Block
	parentHash Hash      // Hash of the Block parent
	baseHash   Hash      // Hash of the Block base
	height     Height    // Height at which the Block was committed
	round      Round     // Round at which the Block was committed
	timestamp  Timestamp // Seconds since Unix Epoch

	// Signatories oversee the consensus algorithm (must be nil unless the Block
	// is a Rebase Block)
	signatories Signatories
}

func NewHeader(kind Kind, parentHash Hash, baseHash Hash, height Height, round Round, timestamp Timestamp, signatories Signatories) Header {
	if kind == Standard && signatories != nil {
		panic("pre-condition violation: standard blocks must have nil signatories")
	}
	if Timestamp(time.Now().Unix()) > timestamp {
		panic("pre-condition violation: now must be after timestamp")
	}

	return Header{
		kind:        kind,
		parentHash:  parentHash,
		baseHash:    baseHash,
		height:      height,
		round:       round,
		timestamp:   timestamp,
		signatories: signatories,
	}
}

// Kind of the Block.
func (header Header) Kind() Kind {
	return header.kind
}

// Signatories of the Block.
func (header Header) Signatories() Signatories {
	return header.signatories
}

// String implements the `fmt.Stringer` interface for the Header type.
func (header Header) String() string {
	return fmt.Sprintf(
		"Header(Kind=%v,ParentHash=%v,BaseHash=%v,Height=%v,Round=%v,Timestamp=%v,Signatories=%v)",
		header.kind,
		header.parentHash,
		header.baseHash,
		header.height,
		header.round,
		header.timestamp,
		header.signatories,
	)
}

// MarshalJSON implements the `json.Marshaler` interface for the Header type.
func (header Header) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Kind        Kind        `json:"kind"`
		ParentHash  Hash        `json:"parentHash"`
		BaseHash    Hash        `json:"baseHash"`
		Height      Height      `json:"height"`
		Round       Round       `json:"round"`
		Timestamp   Timestamp   `json:"timestamp"`
		Signatories Signatories `json:"signatories"`
	}{
		header.kind,
		header.parentHash,
		header.baseHash,
		header.height,
		header.round,
		header.timestamp,
		header.signatories,
	})
}

// UnmarshalJSON implements the `json.Unmarshaler` interface for the Header type.
func (header *Header) UnmarshalJSON(data []byte) error {
	tmp := struct {
		Kind        Kind        `json:"kind"`
		ParentHash  Hash        `json:"parentHash"`
		BaseHash    Hash        `json:"baseHash"`
		Height      Height      `json:"height"`
		Round       Round       `json:"round"`
		Timestamp   Timestamp   `json:"timestamp"`
		Signatories Signatories `json:"signatories"`
	}{}
	if err := json.Unmarshal(data, &tmp); err != nil {
		return err
	}
	header.kind = tmp.Kind
	header.parentHash = tmp.ParentHash
	header.baseHash = tmp.BaseHash
	header.height = tmp.Height
	header.round = tmp.Round
	header.timestamp = tmp.Timestamp
	header.signatories = tmp.Signatories
	return nil
}

// Data stores application-specific information used in Blocks and Notes (must
// be nil in Rebase Blocks and Base Blocks).
type Data []byte

// String implements the `fmt.Stringer` interface for the Data type.
func (data Data) String() string {
	return base64.StdEncoding.WithPadding(base64.NoPadding).EncodeToString(data)
}

// Notes defines a wrapper type around the []Note type.
type Notes []Note

// A Note is used as a finality mechanism for committed Blocks, after the Block
// has been executed. This is useful in scenarios where the execution of a Block
// happens independently from the committment of a Block (common in sMPC where
// execution requires long-running interactive processes). A Block is proposed
// and committed without a Note. A Block is only finalised once it has a valid
// Note. A Block is only valid if its parent is finalised. It is expected that
// the Note will contain application-specific state that has resulted from
// execution (along with proofs of correctness of the transition).
type Note struct {
	hash    Hash // Hash of the Note content
	content Data // Application-specific data stored in the Note
}

// MarshalJSON implements the `json.Marshaler` interface for the Note type.
func (note Note) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Hash    Hash `json:"hash"`
		Content Data `json:"content"`
	}{
		note.hash,
		note.content,
	})
}

// UnmarshalJSON implements the `json.Unmarshaler` interface for the Note type.
func (note *Note) UnmarshalJSON(data []byte) error {
	tmp := struct {
		Hash    Hash `json:"hash"`
		Content Data `json:"content"`
	}{}
	if err := json.Unmarshal(data, &tmp); err != nil {
		return err
	}
	note.hash = tmp.Hash
	note.content = tmp.Content
	return nil
}

// Equal compares one Note with another by checking that their Hashes are the
// equal.
func (note Note) Equal(other Note) bool {
	return note.hash.Equal(other.hash)
}

// Blocks defines a wrapper type around the []Block type.
type Blocks []Block

// A Block is the atomic unit upon which consensus is reached. Consensus
// guarantees a consistent ordering of Blocks that is agreed upon by all members
// in a distributed network, even when some of the members are malicious.
type Block struct {
	hash    Hash // Hash of the Header and content
	header  Header
	content Data
	note    Note // A valid Note is required before proceeding Blocks can proposed
}

func New(header Header, content Data) Block {
	block := Block{
		header:  header,
		content: content,
	}
	block.hash = Hash(sha3.Sum256([]byte(block.String())))
	return block
}

func (block *Block) AppendNote(note Note) {
	block.note = note
}

// Hash of the Header and content.
func (block Block) Hash() Hash {
	return block.hash
}

// Header of the Block.
func (block Block) Header() Header {
	return block.header
}

// Note appended to the Block.
func (block Block) Note() Note {
	return block.note
}

// String implements the `fmt.Stringer` interface for the Block type.
func (block Block) String() string {
	return fmt.Sprintf("Block(Header=%v,Content=%v)", block.header, block.content)
}

// Equal compares one Block with another by checking that their Hashes are the
// equal, and their Notes are equal.
func (block Block) Equal(other Block) bool {
	return block.hash.Equal(other.hash) && block.note.Equal(other.note)
}

// MarshalJSON implements the `json.Marshaler` interface for the Block type.
func (block Block) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Hash    Hash   `json:"hash"`
		Header  Header `json:"header"`
		Content Data   `json:"content"`
		Note    Note   `json:"note"`
	}{
		block.hash,
		block.header,
		block.content,
		block.note,
	})
}

// UnmarshalJSON implements the `json.Unmarshaler` interface for the Block type.
func (block *Block) UnmarshalJSON(data []byte) error {
	tmp := struct {
		Hash    Hash   `json:"hash"`
		Header  Header `json:"header"`
		Content Data   `json:"content"`
		Note    Note   `json:"note"`
	}{}
	if err := json.Unmarshal(data, &tmp); err != nil {
		return err
	}
	block.hash = tmp.Hash
	block.header = tmp.Header
	block.content = tmp.Content
	block.note = tmp.Note
	return nil
}

// Timestamp represents seconds since Unix Epoch.
type Timestamp uint64

// Height of a Block.
type Height int64

// Round in which a Block was proposed.
type Round int64

// Define some default invalid values.
var (
	InvalidHash      = Hash{}
	InvalidSignature = Signature{}
	InvalidSignatory = Signatory{}
	InvalidBlock     = Block{}
	InvalidRound     = Round(-1)
	InvalidHeight    = Height(-1)
)

// A Blockchain defines a storage interface for Blocks that is based around
// Height.
type Blockchain interface {
	InsertBlockAtHeight(Height, Block)
	BlockAtHeight(Height) (Block, bool)
	BlockExistsAtHeight(Height) bool
}
