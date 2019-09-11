package process

import (
	"crypto/ecdsa"
	"encoding/json"
	"fmt"
	"reflect"

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

// MarshalJSON implements the `json.Marshaler` interface for the Propose type.
func (propose Propose) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Sig        id.Signature `json:"sig"`
		Signatory  id.Signatory `json:"signatory"`
		Height     block.Height `json:"height"`
		Round      block.Round  `json:"round"`
		Block      block.Block  `json:"block"`
		ValidRound block.Round  `json:"validRound"`
	}{
		propose.sig,
		propose.signatory,
		propose.height,
		propose.round,
		propose.block,
		propose.validRound,
	})
}

// UnmarshalJSON implements the `json.Unmarshaler` interface for the Propose type.
func (propose *Propose) UnmarshalJSON(data []byte) error {
	tmp := struct {
		Sig        id.Signature `json:"sig"`
		Signatory  id.Signatory `json:"signatory"`
		Height     block.Height `json:"height"`
		Round      block.Round  `json:"round"`
		Block      block.Block  `json:"block"`
		ValidRound block.Round  `json:"validRound"`
	}{}
	if err := json.Unmarshal(data, &tmp); err != nil {
		return err
	}
	propose.sig = tmp.Sig
	propose.signatory = tmp.Signatory
	propose.height = tmp.Height
	propose.round = tmp.Round
	propose.block = tmp.Block
	propose.validRound = tmp.ValidRound
	return nil
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

// MarshalJSON implements the `json.Marshaler` interface for the Prevote type.
func (prevote Prevote) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Sig       id.Signature `json:"sig"`
		Signatory id.Signatory `json:"signatory"`
		Height    block.Height `json:"height"`
		Round     block.Round  `json:"round"`
		BlockHash id.Hash      `json:"blockHash"`
	}{
		prevote.sig,
		prevote.signatory,
		prevote.height,
		prevote.round,
		prevote.blockHash,
	})
}

// UnmarshalJSON implements the `json.Unmarshaler` interface for the Prevote type.
func (prevote *Prevote) UnmarshalJSON(data []byte) error {
	tmp := struct {
		Sig       id.Signature `json:"sig"`
		Signatory id.Signatory `json:"signatory"`
		Height    block.Height `json:"height"`
		Round     block.Round  `json:"round"`
		BlockHash id.Hash      `json:"blockHash"`
	}{}
	if err := json.Unmarshal(data, &tmp); err != nil {
		return err
	}
	prevote.sig = tmp.Sig
	prevote.signatory = tmp.Signatory
	prevote.height = tmp.Height
	prevote.round = tmp.Round
	prevote.blockHash = tmp.BlockHash
	return nil
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

// MarshalJSON implements the `json.Marshaler` interface for the Precommit type.
func (precommit Precommit) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Sig       id.Signature `json:"sig"`
		Signatory id.Signatory `json:"signatory"`
		Height    block.Height `json:"height"`
		Round     block.Round  `json:"round"`
		BlockHash id.Hash      `json:"blockHash"`
	}{
		precommit.sig,
		precommit.signatory,
		precommit.height,
		precommit.round,
		precommit.blockHash,
	})
}

// UnmarshalJSON implements the `json.Unmarshaler` interface for the Precommit type.
func (precommit *Precommit) UnmarshalJSON(data []byte) error {
	tmp := struct {
		Sig       id.Signature `json:"sig"`
		Signatory id.Signatory `json:"signatory"`
		Height    block.Height `json:"height"`
		Round     block.Round  `json:"round"`
		BlockHash id.Hash      `json:"blockHash"`
	}{}
	if err := json.Unmarshal(data, &tmp); err != nil {
		return err
	}
	precommit.sig = tmp.Sig
	precommit.signatory = tmp.Signatory
	precommit.height = tmp.Height
	precommit.round = tmp.Round
	precommit.blockHash = tmp.BlockHash
	return nil
}

func (precommit *Precommit) String() string {
	return fmt.Sprintf("Precommit(Height=%v,Round=%v,BlockHash=%v)", precommit.Height(), precommit.Round(), precommit.BlockHash())
}

type Inbox struct {
	f           int
	messages    map[block.Height]map[block.Round]map[id.Signatory]Message
	messageType reflect.Type
}

func NewInbox(f int, messageType reflect.Type) *Inbox {
	if f <= 0 {
		panic(fmt.Sprintf("invariant violation: f = %v needs to be a positive number", f))
	}
	if messageType == nil {
		panic("invariant violation: message type cannot be nil")
	}
	return &Inbox{
		f:           f,
		messages:    map[block.Height]map[block.Round]map[id.Signatory]Message{},
		messageType: messageType,
	}
}

func (inbox *Inbox) Insert(message Message) (n int, firstTime, firstTimeExceedingF, firstTimeExceeding2F bool) {
	if reflect.TypeOf(message) != reflect.PtrTo(inbox.messageType) {
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
	inbox.messages[height][round][signatory] = message
	n = len(inbox.messages[height][round])

	firstTime = (previousN == 0) && (n == 1)
	firstTimeExceedingF = (previousN < inbox.F()+1) && (n > inbox.F())
	firstTimeExceeding2F = (previousN < 2*inbox.F()+1) && (n > 2*inbox.F())
	return
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

func (inbox *Inbox) MessageType() reflect.Type {
	return inbox.messageType
}

func (inbox *Inbox) Reset(height block.Height) {
	for blockHeight := range inbox.messages {
		if blockHeight <= height {
			delete(inbox.messages, blockHeight)
		}
	}
}

// MarshalJSON implements the `json.Marshaler` interface for the Inbox type.
func (inbox Inbox) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		F        int                                                       `json:"f"`
		Messages map[block.Height]map[block.Round]map[id.Signatory]Message `json:"messages"`
	}{
		inbox.f,
		inbox.messages,
	})
}

// UnmarshalJSON implements the `json.Unmarshaler` interface for the Inbox type.
// Note : you need to be really careful when doing unmarshaling, specifically you need
// to initialize the inbox with the expected messageType. Otherwise it would panic.
func (inbox *Inbox) UnmarshalJSON(data []byte) error {
	tmp := struct {
		F        int                                                               `json:"f"`
		Messages map[block.Height]map[block.Round]map[id.Signatory]json.RawMessage `json:"messages"`
	}{}
	if err := json.Unmarshal(data, &tmp); err != nil {
		return err
	}
	inbox.f = tmp.F
	inbox.messages = map[block.Height]map[block.Round]map[id.Signatory]Message{}

	for height, roundMap := range tmp.Messages {
		if roundMap != nil {
			inbox.messages[height] = map[block.Round]map[id.Signatory]Message{}
		}
		for round, sigMap := range roundMap {
			if sigMap != nil {
				inbox.messages[height][round] = map[id.Signatory]Message{}
			}
			for sig, raw := range sigMap {
				message := reflect.New(inbox.messageType).Interface()
				if err := json.Unmarshal(raw, message); err != nil {
					return err
				}
				msg := message.(Message)
				inbox.messages[height][round][sig] = msg
			}
		}
	}

	return nil
}
