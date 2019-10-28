// Package block defines the `Block` type, and all of the related types. This
// package does not implement any kind of consensus logic; it is concerned with
// defining data types, and serialization/deserialization of those data types.
package block

import (
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"time"

	"github.com/renproject/id"
)

// Kind defines the different kinds of blocks that exist. This is used for
// differentiating between blocks that are for consensus on application-specific
// data, blocks that suggest changing the signatories of a shard, and blocks
// that change the signatories of a shard.
type Kind uint8

const (
	// Invalid defines an invalid kind that must not be used.
	Invalid Kind = iota
	// Standard blocks are used when reaching consensus on the ordering of
	// application-specific data. Standard blocks must have nil header
	// signatories. This is the most common block kind.
	Standard
	// Rebase blocks are used when reaching consensus about a change to the
	// header signatories that oversee the consensus algorithm. Rebase blocks
	// must include non-empty header signatories.
	Rebase
	// Base blocks are used to finalise rebase blocks. Base blocks must come
	// immediately after a rebase block, must have no content, and must have the
	// same header signatories as their parent.
	Base
)

// String implements the `fmt.Stringer` interface for the `Kind` type.
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

// A Header defines properties of a block that are not application-specific.
// These properties are required by, or produced by, the consensus algorithm and
// are not subject modification by the user.
type Header struct {
	kind         Kind      // Kind of block
	parentHash   id.Hash   // Hash of the block parent
	baseHash     id.Hash   // Hash of the block base
	txsRef       id.Hash   // Reference to the block txs
	planRef      id.Hash   // Reference to the block plan
	prevStateRef id.Hash   // Reference to the block previous state
	height       Height    // Height at which the block was committed
	round        Round     // Round at which the block was committed
	timestamp    Timestamp // Seconds since Unix epoch

	// Signatories oversee the consensus algorithm (must be nil unless the block
	// is a rebase/base block)
	signatories id.Signatories
}

// NewHeader with all pre-conditions and invariants checked. It will panic if a
// pre-condition or invariant is violated.
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

// Kind of the block.
func (header Header) Kind() Kind {
	return header.kind
}

// ParentHash of the block.
func (header Header) ParentHash() id.Hash {
	return header.parentHash
}

// BaseHash of the block.
func (header Header) BaseHash() id.Hash {
	return header.baseHash
}

// TxsRef of the block.
func (header Header) TxsRef() id.Hash {
	return header.txsRef
}

// PlanRef of the block.
func (header Header) PlanRef() id.Hash {
	return header.planRef
}

// PrevStateRef of the block.
func (header Header) PrevStateRef() id.Hash {
	return header.prevStateRef
}

// Height of the block.
func (header Header) Height() Height {
	return header.height
}

// Round of the block.
func (header Header) Round() Round {
	return header.round
}

// Timestamp of the block in seconds since Unix epoch.
func (header Header) Timestamp() Timestamp {
	return header.timestamp
}

// Signatories of the block.
func (header Header) Signatories() id.Signatories {
	return header.signatories
}

// String implements the `fmt.Stringer` interface for the `Header` type. Two
// headers must not have the same string representation, unless the headers are
// equal.
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

// Txs stores application-specific transaction data used in blocks.
type Txs []byte

// Hash the transactions using SHA256.
func (txs Txs) Hash() id.Hash {
	return sha256.Sum256(txs)
}

// String implements the `fmt.Stringer` interface for the `Txs` type.
func (txs Txs) String() string {
	return base64.RawStdEncoding.EncodeToString(txs)
}

// Plan stores application-specific data used that is required for the execution
// of transactions in the block. This plan is usually pre-computed data that is
// needed by players in the secure multi-party computation. It is separated from
// transactions for clearer semantic representation (sometimes plans can be
// needed without transactions, and some transactions can be executed without
// plans).
type Plan []byte

// Hash the plan using SHA256.
func (plan Plan) Hash() id.Hash {
	return sha256.Sum256(plan)
}

// String implements the `fmt.Stringer` interface for the `Plan` type.
func (plan Plan) String() string {
	return base64.RawStdEncoding.EncodeToString(plan)
}

// State stores application-specific state after the execution of a block. The
// block at height H+1 will store the state after the execution of block H. This
// is required because of the nature of execution when using an interactive
// secure multi-party computations.
type State []byte

// Hash the state using SHA256.
func (state State) Hash() id.Hash {
	return sha256.Sum256(state)
}

// String implements the `fmt.Stringer` interface for the `State` type.
func (state State) String() string {
	return base64.RawStdEncoding.EncodeToString(state)
}

// Blocks is a wrapper around the `[]Block` type.
type Blocks []Block

// A Block is the atomic unit upon which consensus is reached. Consensus
// guarantees a consistent ordering of blocks that is agreed upon by all members
// in a distributed network, even when some of the members are malicious.
type Block struct {
	hash id.Hash

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

// String implements the `fmt.Stringer` interface for the `Block` type.  Two
// blocks must not have the same string representation, unless the blocks are
// equal.
func (block Block) String() string {
	return fmt.Sprintf("Block(Hash=%v,Header=%v,Txs=%v,Plan=%v,PreviousState=%v)", block.hash, block.header, block.txs, block.plan, block.prevState)
}

// Equal compares one block with another.
func (block Block) Equal(other Block) bool {
	return block.String() == other.String()
}

// Timestamp represents seconds since Unix epoch.
type Timestamp uint64

// Height of a block.
type Height int64

// Round in which a block was proposed.
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

// ComputeHash of a block based on its header, transactions, execution plan and
// state before execution (i.e. state after execution of its parent). This
// function returns a hash that can be used when creating a block.
func ComputeHash(header Header, txs Txs, plan Plan, prevState State) id.Hash {
	return sha256.Sum256([]byte(fmt.Sprintf("BlockHash(Header=%v,Txs=%v,Plan=%v,PreviousState=%v)", header, txs, plan, prevState)))
}
