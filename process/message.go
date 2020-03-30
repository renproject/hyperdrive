package process

import (
	"crypto/ecdsa"
	"crypto/sha256"
	"encoding"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/renproject/hyperdrive/block"
	"github.com/renproject/id"
)

// MessageType distinguished between the three valid (and one invalid) messages
// types that are supported during consensus rounds.
type MessageType uint64

const (
	// NilMessageType is invalid and must not be used.
	NilMessageType = 0
	// ProposeMessageType is used by messages that propose blocks for consensus.
	ProposeMessageType = 1
	// PrevoteMessageType is used by messages that are prevoting for block
	// hashes (or nil prevoting).
	PrevoteMessageType = 2
	// PrecommitMessageType is used by messages that are precommitting for block
	// hashes (or nil precommitting).
	PrecommitMessageType = 3
	// ResyncMessageType is used by messages that query others for previous
	// messages.
	ResyncMessageType = 4
)

// Messages is a wrapper around the `[]Message` type.
type Messages []Message

// The Message interface defines the common behaviour of all messages that are
// broadcast throughout the network during consensus rounds.
type Message interface {
	fmt.Stringer
	json.Marshaler
	json.Unmarshaler
	encoding.BinaryMarshaler
	encoding.BinaryUnmarshaler

	// The Signatory that sent the message.
	Signatory() id.Signatory
	// The SigHash that is expected to by signed by the signatory. This is used
	// to authenticate the claimed signatory.
	SigHash() id.Hash
	// The Signature produced by the signatory signing the sighash. This is used
	// to authenticate the claimed signatory.
	Sig() id.Signature

	// The Height of the blockchain in which this message was broadcast.
	Height() block.Height
	// The Round of consensus in which this message was broadcast.
	Round() block.Round
	// The BlockHash of the block to this message concerns. Proposals will be
	// proposing the block identified  by this hash, prevotes will be prevoting
	// for the block identified by this hash (nil prevotes will use
	// `InvalidBlockHash`), and precommits will be precommitting for block
	// identified by this hash (nil precommits will also use
	// `InvalidBlockHash`).
	BlockHash() id.Hash

	// Type returns the message type of this message. This is useful for
	// marshaling/unmarshaling when type information is elided.
	Type() MessageType
}

var (
	// ErrBlockHashNotProvided is returned when querying the block hash for a
	// message type that does not implement the function.
	ErrBlockHashNotProvided = errors.New("block hash not provided")
)

// Sign a message using an ECDSA private key. The resulting signature will be
// stored inside the message.
func Sign(m Message, privKey ecdsa.PrivateKey) error {
	sigHash := m.SigHash()
	signatory := id.NewSignatory(privKey.PublicKey)
	sig, err := crypto.Sign(sigHash[:], &privKey)
	if err != nil {
		return fmt.Errorf("invariant violation: error signing message: %v", err)
	}
	if len(sig) != id.SignatureLength {
		return fmt.Errorf("invariant violation: invalid signed message, expected = %v, got = %v", id.SignatureLength, len(sig))
	}

	switch m := m.(type) {
	case *Propose:
		m.signatory = signatory
		copy(m.sig[:], sig)
	case *Prevote:
		m.signatory = signatory
		copy(m.sig[:], sig)
	case *Precommit:
		m.signatory = signatory
		copy(m.sig[:], sig)
	case *Resync:
		m.signatory = signatory
		copy(m.sig[:], sig)
	default:
		panic(fmt.Errorf("invariant violation: unexpected message type=%T", m))
	}
	return nil
}

// Verify that the signature in a message is from the expected signatory. This
// is done by checking the `Message.Sig()` against the `Message.SigHash()` and
// `Message.Signatory()`.
func Verify(m Message) error {
	sigHash := m.SigHash()
	sig := m.Sig()
	pubKey, err := crypto.SigToPub(sigHash[:], sig[:])
	if err != nil {
		return fmt.Errorf("error verifying message: %v", err)
	}

	signatory := id.NewSignatory(*pubKey)
	if !m.Signatory().Equal(signatory) {
		return fmt.Errorf("bad signatory: expected signatory=%v, got signatory=%v", m.Signatory(), signatory)
	}
	return nil
}

// Proposes is a wrapper around the `[]Propose` type.
type Proposes []Propose

// Propose a block for committment.
type Propose struct {
	signatory  id.Signatory
	sig        id.Signature
	height     block.Height
	round      block.Round
	block      block.Block
	validRound block.Round

	latestCommit LatestCommit
}

// The LatestCommit can be attached to a proposal. It stores the latest
// committed block, and a set of precommits that prove this block was committed.
// This is useful for allowing processes that have fallen out-of-sync to fast
// forward. See https://github.com/renproject/hyperdrive/wiki/Consensus for more
// information about fast fowarding.
type LatestCommit struct {
	Block      block.Block
	Precommits []Precommit
}

func NewPropose(height block.Height, round block.Round, block block.Block, validRound block.Round) *Propose {
	return &Propose{
		height:     height,
		round:      round,
		block:      block,
		validRound: validRound,
	}
}

func (propose *Propose) Signatory() id.Signatory {
	return propose.signatory
}

func (propose *Propose) SigHash() id.Hash {
	return sha256.Sum256([]byte(propose.String()))
}

func (propose *Propose) Sig() id.Signature {
	return propose.sig
}

func (propose *Propose) Height() block.Height {
	return propose.height
}

func (propose *Propose) Round() block.Round {
	return propose.round
}

func (propose *Propose) BlockHash() id.Hash {
	return propose.block.Hash()
}

func (propose *Propose) Type() MessageType {
	return ProposeMessageType
}

func (propose *Propose) Block() block.Block {
	return propose.block
}

func (propose *Propose) ValidRound() block.Round {
	return propose.validRound
}

func (propose *Propose) String() string {
	return fmt.Sprintf("Propose(Height=%v,Round=%v,BlockHash=%v,ValidRound=%v)", propose.Height(), propose.Round(), propose.BlockHash(), propose.ValidRound())
}

// Prevotes is a wrapper around the `[]Prevote` type.
type Prevotes []Prevote

// Prevote for a block hash.
type Prevote struct {
	signatory  id.Signatory
	sig        id.Signature
	height     block.Height
	round      block.Round
	blockHash  id.Hash
	nilReasons NilReasons
}

func NewPrevote(height block.Height, round block.Round, blockHash id.Hash, nilReasons NilReasons) *Prevote {
	return &Prevote{
		height:     height,
		round:      round,
		blockHash:  blockHash,
		nilReasons: nilReasons,
	}
}

func (prevote *Prevote) Signatory() id.Signatory {
	return prevote.signatory
}

func (prevote *Prevote) SigHash() id.Hash {
	return sha256.Sum256([]byte(prevote.String()))
}

func (prevote *Prevote) Sig() id.Signature {
	return prevote.sig
}

func (prevote *Prevote) Height() block.Height {
	return prevote.height
}

func (prevote *Prevote) Round() block.Round {
	return prevote.round
}

func (prevote *Prevote) BlockHash() id.Hash {
	return prevote.blockHash
}

func (prevote *Prevote) NilReasons() NilReasons {
	return prevote.nilReasons
}

func (prevote *Prevote) Type() MessageType {
	return PrevoteMessageType
}

func (prevote *Prevote) String() string {
	nilReasonsBytes, err := prevote.NilReasons().MarshalBinary()
	if err != nil {
		return fmt.Sprintf("Prevote(Height=%v,Round=%v,BlockHash=%v)", prevote.Height(), prevote.Round(), prevote.BlockHash())
	}
	nilReasonsHash := id.Hash(sha256.Sum256(nilReasonsBytes))
	return fmt.Sprintf("Prevote(Height=%v,Round=%v,BlockHash=%v,NilReasons=%v)", prevote.Height(), prevote.Round(), prevote.BlockHash(), nilReasonsHash.String())
}

// Precommits is a wrapper around the `[]Precommit` type.
type Precommits []Precommit

// Precommit a block hash.
type Precommit struct {
	signatory id.Signatory
	sig       id.Signature
	height    block.Height
	round     block.Round
	blockHash id.Hash
}

func NewPrecommit(height block.Height, round block.Round, blockHash id.Hash) *Precommit {
	return &Precommit{
		height:    height,
		round:     round,
		blockHash: blockHash,
	}
}

func (precommit *Precommit) Signatory() id.Signatory {
	return precommit.signatory
}

func (precommit *Precommit) SigHash() id.Hash {
	return sha256.Sum256([]byte(precommit.String()))
}

func (precommit *Precommit) Sig() id.Signature {
	return precommit.sig
}

func (precommit *Precommit) Height() block.Height {
	return precommit.height
}

func (precommit *Precommit) Round() block.Round {
	return precommit.round
}

func (precommit *Precommit) BlockHash() id.Hash {
	return precommit.blockHash
}

func (precommit *Precommit) Type() MessageType {
	return PrecommitMessageType
}

func (precommit *Precommit) String() string {
	return fmt.Sprintf("Precommit(Height=%v,Round=%v,BlockHash=%v)", precommit.Height(), precommit.Round(), precommit.BlockHash())
}

// Resyncs is a wrapper around the `[]Resync` type.
type Resyncs []Resync

// Resync previous messages.
type Resync struct {
	signatory id.Signatory
	sig       id.Signature
	height    block.Height
	round     block.Round
}

func NewResync(height block.Height, round block.Round) *Resync {
	return &Resync{
		height: height,
		round:  round,
	}
}

func (resync *Resync) Signatory() id.Signatory {
	return resync.signatory
}

func (resync *Resync) SigHash() id.Hash {
	return sha256.Sum256([]byte(resync.String()))
}

func (resync *Resync) Sig() id.Signature {
	return resync.sig
}

func (resync *Resync) Height() block.Height {
	return resync.height
}

func (resync *Resync) Round() block.Round {
	return resync.round
}

func (resync *Resync) BlockHash() id.Hash {
	panic(ErrBlockHashNotProvided)
}

func (resync *Resync) Type() MessageType {
	return ResyncMessageType
}

func (resync *Resync) String() string {
	return fmt.Sprintf("Resync(Height=%v,Round=%v)", resync.Height(), resync.Round())
}

// An Inbox is storage container for one type message. Any type of message can
// be stored, but an attempt to store messages of different types in one inbox
// will cause a panic. Inboxes are used extensively by the consensus algorithm
// to track how many messages (of particular types) have been received, and
// under what conditions. For example, inboxes are used to track when `2F+1`
// prevote messages have been received for a specific block hash for the first
// time.
type Inbox struct {
	f           int
	messages    map[block.Height]map[block.Round]map[id.Signatory]Message
	messageType MessageType
}

// NewInbox returns an inbox for one type of message. It assumes at most `F`
// adversaries are present.
func NewInbox(f int, messageType MessageType) *Inbox {
	if f <= 0 {
		panic(fmt.Sprintf("invariant violation: f = %v needs to be a positive number", f))
	}
	if messageType == NilMessageType {
		panic("invariant violation: message type cannot be nil")
	}
	return &Inbox{
		f:           f,
		messages:    map[block.Height]map[block.Round]map[id.Signatory]Message{},
		messageType: messageType,
	}
}

// Insert a message into the inbox. It returns:
//
// - `n` the number of unique messages at the height and round of the inserted
//    message (not necessarily for the same block hash),
// - `firstTime` whether, or not, this is the first time this message has been
//   seen,
// - `firstTimeExceedingF` whether, or not, `n` has exceeded `F` for the first
//   time as a result of this message being inserted,
// - `firstTimeExceeding2F` whether, or not, `n` has exceeded `2F` for the first
//   time as a result of this message being inserted, and
// - `firstTimeExceeding2FOnBlockHash` whether, or not, this is the first
//   time that more than `2F` unique messages have been seen for the same block.
//
// This method is used extensively for tracking the different conditions under
// which the state machine is allowed to transition between various states. Its
// correctness is fundamental to the correctness of the overall implementation.
func (inbox *Inbox) Insert(message Message) (n int, firstTime, firstTimeExceedingF, firstTimeExceeding2F, firstTimeExceeding2FOnBlockHash bool) {
	if message.Type() != inbox.messageType {
		panic(fmt.Sprintf("pre-condition violation: expected type %v, got type %T", inbox.messageType, message))
	}

	height, round, signatory := message.Height(), message.Round(), message.Signatory()
	if _, ok := inbox.messages[height]; !ok {
		inbox.messages[height] = map[block.Round]map[id.Signatory]Message{}
	}
	if _, ok := inbox.messages[height][round]; !ok {
		inbox.messages[height][round] = map[id.Signatory]Message{}
	}

	previousN := len(inbox.messages[height][round])
	_, ok := inbox.messages[height][round][signatory]

	inbox.messages[height][round][signatory] = message

	n = len(inbox.messages[height][round])
	nOnBlockHash := 0
	if !ok {
		nOnBlockHash = inbox.QueryByHeightRoundBlockHash(height, round, message.BlockHash())
	}

	firstTime = (previousN == 0) && (n == 1)
	firstTimeExceedingF = (previousN == inbox.F()) && (n == inbox.F()+1)
	firstTimeExceeding2F = (previousN == 2*inbox.F()) && (n == 2*inbox.F()+1)
	firstTimeExceeding2FOnBlockHash = !ok && (nOnBlockHash == 2*inbox.F()+1)
	return
}

// Delete removes all messages at a given height.
func (inbox *Inbox) Delete(height block.Height) {
	delete(inbox.messages, height)
}

// QueryMessagesByHeightRound returns all unique messages that have been
// received at the specified height and round. The specific block hash of the
// messages are ignored and might be different from each other.
func (inbox *Inbox) QueryMessagesByHeightRound(height block.Height, round block.Round) []Message {
	if _, ok := inbox.messages[height]; !ok {
		return nil
	}
	if _, ok := inbox.messages[height][round]; !ok {
		return nil
	}
	messages := make([]Message, 0, len(inbox.messages[height][round]))
	for _, message := range inbox.messages[height][round] {
		messages = append(messages, message)
	}
	return messages
}

// QueryMessagesByHeightWithHighestRound returns all unique messages that have
// been received at the specified height and at the heighest round observed (for
// the specified height). Only rounds with >2F messages are considered. The
// specific block hash of the messages are ignored and might be different from
// each other.
func (inbox *Inbox) QueryMessagesByHeightWithHighestRound(height block.Height) []Message {
	if _, ok := inbox.messages[height]; !ok {
		return nil
	}
	highestRound := block.Round(-1)
	for round := range inbox.messages[height] {
		if len(inbox.messages[height][round]) > 2*inbox.f {
			if round > highestRound {
				highestRound = round
			}
		}
	}
	if highestRound == -1 {
		return nil
	}
	if _, ok := inbox.messages[height][highestRound]; !ok {
		return nil
	}
	messages := make([]Message, 0, len(inbox.messages[height][highestRound]))
	for _, message := range inbox.messages[height][highestRound] {
		messages = append(messages, message)
	}
	return messages
}

// QueryByHeightRoundBlockHash returns the number of unique messages that have
// been received at the specified height and round. Only messages that reference
// the specified block hash are considered.
func (inbox *Inbox) QueryByHeightRoundBlockHash(height block.Height, round block.Round, blockHash id.Hash) (n int) {
	if _, ok := inbox.messages[height]; !ok {
		return
	}
	if _, ok := inbox.messages[height][round]; !ok {
		return
	}
	for _, message := range inbox.messages[height][round] {
		if blockHash.Equal(message.BlockHash()) {
			n++
		}
	}
	return
}

// QueryByHeightRoundSignatory the message (or nil) sent by a specific signatory
// at a specific height and round.
func (inbox *Inbox) QueryByHeightRoundSignatory(height block.Height, round block.Round, sig id.Signatory) Message {
	if _, ok := inbox.messages[height]; !ok {
		return nil
	}
	if _, ok := inbox.messages[height][round]; !ok {
		return nil
	}
	return inbox.messages[height][round][sig]
}

// QueryByHeightRound returns the number of unique messages that have been
// received at the specified height and round. The specific block hash of the
// messages are ignored and might be different from each other.
func (inbox *Inbox) QueryByHeightRound(height block.Height, round block.Round) (n int) {
	if _, ok := inbox.messages[height]; !ok {
		return
	}
	if _, ok := inbox.messages[height][round]; !ok {
		return
	}
	n = len(inbox.messages[height][round])
	return
}

func (inbox *Inbox) F() int {
	return inbox.f
}

func (inbox *Inbox) MessageType() MessageType {
	return inbox.messageType
}

// Reset the inbox to a specific height. All messages for height lower than the
// specified height are dropped. This is necessary to ensure that, over time,
// the storage space of the inbox is bounded.
func (inbox *Inbox) Reset(height block.Height) {
	for blockHeight := range inbox.messages {
		if blockHeight < height {
			delete(inbox.messages, blockHeight)
		}
	}
}
