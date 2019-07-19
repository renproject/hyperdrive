package block

import (
	"bytes"
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

// Equal compares one Signature with another.
func (sig Signature) Equal(other Signature) bool {
	return bytes.Equal(sig[:], other[:])
}

// Equal compares one Signatory with another.
func (sig Signatory) Equal(other Signatory) bool {
	return bytes.Equal(sig[:], other[:])
}

// Kind defines the different kinds of Block that exist.
type Kind uint8

const (
	// Invalid define an invalid Kind that should not be used.
	Invalid = iota
	// Basic defines the Kind used for Blocks that represents the need for
	// consensus on application-specific Data. Blocks of this Kind must have nil
	// Signatories in the Header. This is the most common Kind.
	Basic
	// Rebase defines the Kind used for Blocks that change the Signatories
	// overseeing the consensus algorithm. Blocks of this Kind must have nil
	// Data, and must not have nil Signatories in the Header.
	Rebase
	// Base defines the Kind used for Blocks that immediately proceed a Rebase.
	// Blocks of this Kind are only used to reach finality on a Rebase; must
	// have nil Data, nil Signatories in the Header, nil Note.
	Base
)

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

// Data stores application-specific information used in Blocks and Notes (must
// be nil in Rebase Blocks and Base Blocks).
type Data []byte

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

// Hash of the Header and content.
func (block Block) Hash() Hash {
	return block.hash
}

// Equal compares one Block with another by checking that their Hashes are the
// equal, and their Notes are equal.
func (block Block) Equal(other Block) bool {
	return block.hash.Equal(other.hash) && block.note.Equal(other.note)
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
