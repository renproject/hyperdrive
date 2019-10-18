package block

import (
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"time"

	"github.com/renproject/id"
)

// Kind defines the different kinds of Block that exist.
type Kind uint8

const (
	// Invalid defines an invalid Kind that must not be used.
	Invalid Kind = iota

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
	kind         Kind      // Kind of Block
	parentHash   id.Hash   // Hash of the Block parent
	baseHash     id.Hash   // Hash of the Block base
	txsRef       id.Hash   // Reference to the Block txs
	planRef      id.Hash   // Reference to the Block plan
	prevStateRef id.Hash   // Reference to the Block previous state
	height       Height    // Height at which the Block was committed
	round        Round     // Round at which the Block was committed
	timestamp    Timestamp // Seconds since Unix Epoch

	// Signatories oversee the consensus algorithm (must be nil unless the Block
	// is a Rebase/Base Block)
	signatories id.Signatories
}

// NewHeader returns a Header. It will panic if a pre-condition for Header
// validity is violated.
func NewHeader(kind Kind, parentHash, baseHash, txsRef, planRef, prevStateRef id.Hash, height Height, round Round, timestamp Timestamp, signatories id.Signatories) Header {
	switch kind {
	case Standard:
		if signatories != nil {
			panic("pre-condition violation: standard blocks must not declare signatories")
		}
	case Rebase, Base:
		if len(signatories) == 0 {
			panic(fmt.Sprintf("pre-condition violation: %v blocks must declare signatories", kind))
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
		kind:         kind,
		parentHash:   parentHash,
		baseHash:     baseHash,
		txsRef:       txsRef,
		planRef:      planRef,
		prevStateRef: prevStateRef,
		height:       height,
		round:        round,
		timestamp:    timestamp,
		signatories:  signatories,
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

// TxsRef of the Block.
func (header Header) TxsRef() id.Hash {
	return header.txsRef
}

// PlanRef of the Block.
func (header Header) PlanRef() id.Hash {
	return header.planRef
}

// PrevStateRef of the Block.
func (header Header) PrevStateRef() id.Hash {
	return header.prevStateRef
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
		"Header(Kind=%v,ParentHash=%v,BaseHash=%v,TxsRef=%v,PlanRef=%v,PrevStateRef=%v,Height=%v,Round=%v,Timestamp=%v,Signatories=%v)",
		header.kind,
		header.parentHash,
		header.baseHash,
		header.txsRef,
		header.planRef,
		header.prevStateRef,
		header.height,
		header.round,
		header.timestamp,
		header.signatories,
	)
}

// Txs stores application-specific transaction data used in Blocks and Notes
// (must be nil in Rebase Blocks and Base Blocks).
type Txs []byte

// Hash of a Txs object.
func (txs Txs) Hash() id.Hash {
	return sha256.Sum256(txs)
}

// String implements the `fmt.Stringer` interface for the Txs type.
func (txs Txs) String() string {
	return base64.RawStdEncoding.EncodeToString(txs)
}

// Plan stores application-specific plan data used in Blocks and Notes (must be
// nil in Rebase Blocks and Base Blocks).
type Plan []byte

// Hash of a Plan object.
func (plan Plan) Hash() id.Hash {
	return sha256.Sum256(plan)
}

// String implements the `fmt.Stringer` interface for the Plan type.
func (plan Plan) String() string {
	return base64.RawStdEncoding.EncodeToString(plan)
}

// State stores application-specific state after the execution of a Block.
type State []byte

// Hash of a State object.
func (state State) Hash() id.Hash {
	return sha256.Sum256(state)
}

// String implements the `fmt.Stringer` interface for the State type.
func (state State) String() string {
	return base64.RawStdEncoding.EncodeToString(state)
}

// Blocks defines a wrapper type around the []Block type.
type Blocks []Block

// A Block is the atomic unit upon which consensus is reached. Consensus
// guarantees a consistent ordering of Blocks that is agreed upon by all members
// in a distributed network, even when some of the members are malicious.
type Block struct {
	hash      id.Hash // Hash of the Header, Txs, Plan, and State
	header    Header
	txs       Txs
	plan      Plan
	prevState State
}

// New Block with the Header, Txs, Plan, and State of the Block parent. The
// Block Hash will automatically be computed and set.
func New(header Header, txs Txs, plan Plan, prevState State) Block {
	return Block{
		hash:      ComputeHash(header, txs, plan, prevState),
		header:    header,
		txs:       txs,
		plan:      plan,
		prevState: prevState,
	}
}

// Hash returns the 256-bit SHA2 Hash of the Header and Data.
func (block Block) Hash() id.Hash {
	return block.hash
}

// Header of the Block.
func (block Block) Header() Header {
	return block.header
}

// Txs embedded in the Block for application-specific purposes.
func (block Block) Txs() Txs {
	return block.txs
}

// Plan embedded in the Block for application-specific purposes.
func (block Block) Plan() Plan {
	return block.plan
}

// PreviousState embedded in the Block for application-specific state after the
// execution of the Block parent.
func (block Block) PreviousState() State {
	return block.prevState
}

// String implements the `fmt.Stringer` interface for the Block type.
func (block Block) String() string {
	return fmt.Sprintf("Block(Hash=%v,Header=%v,Txs=%v,Plan=%v,PreviousState=%v)", block.hash, block.header, block.txs, block.plan, block.prevState)
}

// Equal compares one Block with another by checking that their Hashes are the
// equal, and their Notes are equal.
func (block Block) Equal(other Block) bool {
	return block.String() == other.String()
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

// ComputeHash of a block basing on its header, data and previous state.
func ComputeHash(header Header, txs Txs, plan Plan, prevState State) id.Hash {
	return sha256.Sum256([]byte(fmt.Sprintf("BlockHash(Header=%v,Txs=%v,Plan=%v,PreviousState=%v)", header, txs, plan, prevState)))
}
