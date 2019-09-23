package process

import (
	"crypto/ecdsa"
	"encoding"
	"encoding/json"
	"fmt"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/renproject/hyperdrive/block"
	"github.com/renproject/id"
	"golang.org/x/crypto/sha3"
)

type MessageType uint64

const (
	NilMessageType       = 0
	ProposeMessageType   = 1
	PrevoteMessageType   = 2
	PrecommitMessageType = 3
)

type Messages []Message

type Message interface {
	fmt.Stringer
	json.Marshaler
	json.Unmarshaler
	encoding.BinaryMarshaler
	encoding.BinaryUnmarshaler

	Signatory() id.Signatory
	SigHash() id.Hash
	Sig() id.Signature

	Height() block.Height
	Round() block.Round
	BlockHash() id.Hash

	Type() MessageType
}

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
	default:
		panic(fmt.Errorf("invariant violation: unexpected message type=%T", m))
	}
	return nil
}

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

type Proposes []Propose

type Propose struct {
	signatory  id.Signatory
	sig        id.Signature
	height     block.Height
	round      block.Round
	block      block.Block
	validRound block.Round

	latestCommit LatestCommit
}

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
	return sha3.Sum256([]byte(propose.String()))
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

type Prevotes []Prevote

type Prevote struct {
	signatory id.Signatory
	sig       id.Signature
	height    block.Height
	round     block.Round
	blockHash id.Hash
}

func NewPrevote(height block.Height, round block.Round, blockHash id.Hash) *Prevote {
	return &Prevote{
		height:    height,
		round:     round,
		blockHash: blockHash,
	}
}

func (prevote *Prevote) Signatory() id.Signatory {
	return prevote.signatory
}

func (prevote *Prevote) SigHash() id.Hash {
	return sha3.Sum256([]byte(prevote.String()))
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

func (prevote *Prevote) Type() MessageType {
	return PrevoteMessageType
}

func (prevote *Prevote) String() string {
	return fmt.Sprintf("Prevote(Height=%v,Round=%v,BlockHash=%v)", prevote.Height(), prevote.Round(), prevote.BlockHash())
}

type Precommits []Precommit

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
	return sha3.Sum256([]byte(precommit.String()))
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

type Inbox struct {
	f           int
	messages    map[block.Height]map[block.Round]map[id.Signatory]Message
	messageType MessageType
}

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

func (inbox *Inbox) QueryMessagesByHeightWithHigestRound(height block.Height) []Message {
	if _, ok := inbox.messages[height]; !ok {
		return nil
	}
	highestRound := block.Round(-1)
	for round := range inbox.messages[height] {
		if round > highestRound {
			highestRound = round
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

func (inbox *Inbox) QueryByHeightRoundSignatory(height block.Height, round block.Round, sig id.Signatory) Message {
	if _, ok := inbox.messages[height]; !ok {
		return nil
	}
	if _, ok := inbox.messages[height][round]; !ok {
		return nil
	}
	return inbox.messages[height][round][sig]
}

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

func (inbox *Inbox) Reset(height block.Height) {
	for blockHeight := range inbox.messages {
		if blockHeight < height {
			delete(inbox.messages, blockHeight)
		}
	}
}
