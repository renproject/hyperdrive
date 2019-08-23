package block

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"time"

	"github.com/renproject/hyperdrive/id"
	"golang.org/x/crypto/sha3"
)

// Kind defines the different kinds of Block that exist.
type Kind uint8

const (
	// Invalid defines an invalid Kind that must not be used.
	Invalid = iota
	// Standard Blocks are used when reaching consensus on the ordering of
	// application-specific data. Standard Blocks must have nil Header
	// Signatories. This is the most common Block Kind.
	Standard
	// Rebase Blocks are used when reaching consensus about a change to the
	// Header Signatories that oversee the consensus algorithm. Rebase Blocks
	// must include non-empty Header Signatories.
	Rebase
	// Base Blocks are used to finalise Rebase Blocks. Base Blocks must come
	// immediately after a Rebase Block, must have no Content, and must have the
	// same Header Signatories as their parent.
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
	parentHash id.Hash   // Hash of the Block parent
	baseHash   id.Hash   // Hash of the Block base
	height     Height    // Height at which the Block was committed
	round      Round     // Round at which the Block was committed
	timestamp  Timestamp // Seconds since Unix Epoch

	// Signatories oversee the consensus algorithm (must be nil unless the Block
	// is a Rebase/Base Block)
	signatories id.Signatories
}

// NewHeader returns a Header. It will panic a pre-condition for Header validity
// is violated.
func NewHeader(kind Kind, parentHash, baseHash id.Hash, height Height, round Round, timestamp Timestamp, signatories id.Signatories) Header {
	switch kind {
	case Standard:
		if signatories != nil {
			panic("pre-condition violation: standard blocks must not declare signatories")
		}
	case Rebase:
		if signatories == nil || len(signatories) == 0 {
			panic("pre-condition violation: rebase blocks must declare signatories")
		}
	case Base:
		if signatories == nil || len(signatories) == 0 {
			panic("pre-condition violation: base blocks must declare signatories")
		}
	default:
		panic(fmt.Errorf("pre-condition violation: unexpected block kind=%v", kind))
	}
	if parentHash.Equal(InvalidHash) {
		panic(fmt.Errorf("pre-condition violation: invalid parent hash=%v", parentHash))
	}
	if baseHash.Equal(InvalidHash) {
		panic(fmt.Errorf("pre-condition violation: invalid base hash=%v", baseHash))
	}
	if height <= InvalidHeight {
		panic(fmt.Errorf("pre-condition violation: invalid height=%v", height))
	}
	if round <= InvalidRound {
		panic(fmt.Errorf("pre-condition violation: invalid round=%v", round))
	}
	if Timestamp(time.Now().Unix()) < timestamp {
		panic("pre-condition violation: timestamp has not passed")
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

// ParentHash of the Block.
func (header Header) ParentHash() id.Hash {
	return header.parentHash
}

// BaseHash of the Block.
func (header Header) BaseHash() id.Hash {
	return header.baseHash
}

// Height of the Block.
func (header Header) Height() Height {
	return header.height
}

// Round of the Block.
func (header Header) Round() Round {
	return header.round
}

// Timestamp of the Block in seconds since Unix Epoch.
func (header Header) Timestamp() Timestamp {
	return header.timestamp
}

// Signatories of the Block.
func (header Header) Signatories() id.Signatories {
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
		Kind        Kind           `json:"kind"`
		ParentHash  id.Hash        `json:"parentHash"`
		BaseHash    id.Hash        `json:"baseHash"`
		Height      Height         `json:"height"`
		Round       Round          `json:"round"`
		Timestamp   Timestamp      `json:"timestamp"`
		Signatories id.Signatories `json:"signatories"`
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
		Kind        Kind           `json:"kind"`
		ParentHash  id.Hash        `json:"parentHash"`
		BaseHash    id.Hash        `json:"baseHash"`
		Height      Height         `json:"height"`
		Round       Round          `json:"round"`
		Timestamp   Timestamp      `json:"timestamp"`
		Signatories id.Signatories `json:"signatories"`
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

// State stores application-specific state after the execution of a Block.
type State []byte

// String implements the `fmt.Stringer` interface for the State type.
func (state State) String() string {
	return base64.StdEncoding.WithPadding(base64.NoPadding).EncodeToString(state)
}

// Blocks defines a wrapper type around the []Block type.
type Blocks []Block

// A Block is the atomic unit upon which consensus is reached. Consensus
// guarantees a consistent ordering of Blocks that is agreed upon by all members
// in a distributed network, even when some of the members are malicious.
type Block struct {
	hash      id.Hash // Hash of the Header, Data, and State
	header    Header
	data      Data
	prevState State
}

// New Block with the Header, Data, and State of the Block parent. The Block
// Hash will automatically be computed and set.
func New(header Header, data Data, prevState State) Block {
	return Block{
		hash:      ComputeHash(header, data, prevState),
		header:    header,
		data:      data,
		prevState: prevState,
	}
}

// Hash returns the 256-bit SHA3 Hash of the Header and Data.
func (block Block) Hash() id.Hash {
	return block.hash
}

// Header of the Block.
func (block Block) Header() Header {
	return block.header
}

// Data embedded in the Block for application-specific purposes.
func (block Block) Data() Data {
	return block.data
}

// PreviousState embedded in the Block for application-specific state after the
// execution of the Block parent.
func (block Block) PreviousState() State {
	return block.prevState
}

// String implements the `fmt.Stringer` interface for the Block type.
func (block Block) String() string {
	return fmt.Sprintf("Block(Hash=%v,Header=%v,Data=%v,PreviousState=%v)", block.hash, block.header, block.data, block.prevState)
}

// Equal compares one Block with another by checking that their Hashes are the
// equal, and their Notes are equal.
func (block Block) Equal(other Block) bool {
	return block.hash.Equal(other.hash)
}

// MarshalJSON implements the `json.Marshaler` interface for the Block type.
func (block Block) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Hash      id.Hash `json:"hash"`
		Header    Header  `json:"header"`
		Data      Data    `json:"data"`
		PrevState State   `json:"prevState"`
	}{
		block.hash,
		block.header,
		block.data,
		block.prevState,
	})
}

// UnmarshalJSON implements the `json.Unmarshaler` interface for the Block type.
func (block *Block) UnmarshalJSON(data []byte) error {
	tmp := struct {
		Hash      id.Hash `json:"hash"`
		Header    Header  `json:"header"`
		Data      Data    `json:"data"`
		PrevState State   `json:"prevState"`
	}{}
	if err := json.Unmarshal(data, &tmp); err != nil {
		return err
	}
	block.hash = tmp.Hash
	block.header = tmp.Header
	block.data = tmp.Data
	block.prevState = tmp.PrevState
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
	InvalidHash      = id.Hash{}
	InvalidSignature = id.Signature{}
	InvalidSignatory = id.Signatory{}
	InvalidBlock     = Block{}
	InvalidRound     = Round(-1)
	InvalidHeight    = Height(-1)
)

func ComputeHash(header Header, data Data, prevState State) id.Hash {
	return id.Hash(sha3.Sum256([]byte(fmt.Sprintf("BlockHash(Header=%v,Data=%v,PreviousState=%v)", header, data, prevState))))
}
