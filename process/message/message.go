package message

import (
	"github.com/renproject/hyperdrive/process/block"
)

type Messages []Message

type Message interface {
	Signatory() block.Signatory
	Height() block.Height
	Round() block.Round
	BlockHash() block.Hash
}

type Proposes []Propose

type Propose struct {
	signatory  block.Signatory
	height     block.Height
	round      block.Round
	block      block.Block
	validRound block.Round
}

func NewPropose(signatory block.Signatory, height block.Height, round block.Round, block block.Block, validRound block.Round) Propose {
	return Propose{
		signatory:  signatory,
		height:     height,
		round:      round,
		block:      block,
		validRound: validRound,
	}
}

func (propose Propose) Signatory() block.Signatory {
	return propose.signatory
}

func (propose Propose) Height() block.Height {
	return propose.height
}

func (propose Propose) Round() block.Round {
	return propose.round
}

func (propose Propose) BlockHash() block.Hash {
	return propose.block.Hash()
}

func (propose Propose) Block() block.Block {
	return propose.block
}

func (propose Propose) ValidRound() block.Round {
	return propose.validRound
}

type Prevotes []Prevote

type Prevote struct {
	signatory block.Signatory
	height    block.Height
	round     block.Round
	blockHash block.Hash
}

func NewPrevote(signatory block.Signatory, height block.Height, round block.Round, blockHash block.Hash) Prevote {
	return Prevote{
		signatory: signatory,
		height:    height,
		round:     round,
		blockHash: blockHash,
	}
}

func (prevote Prevote) Signatory() block.Signatory {
	return prevote.signatory
}

func (prevote Prevote) Height() block.Height {
	return prevote.height
}

func (prevote Prevote) Round() block.Round {
	return prevote.round
}

func (prevote Prevote) BlockHash() block.Hash {
	return prevote.blockHash
}

type Precommits []Precommit

type Precommit struct {
	signatory block.Signatory
	height    block.Height
	round     block.Round
	blockHash block.Hash
}

func NewPrecommit(signatory block.Signatory, height block.Height, round block.Round, blockHash block.Hash) Precommit {
	return Precommit{
		signatory: signatory,
		height:    height,
		round:     round,
		blockHash: blockHash,
	}
}

func (precommit Precommit) Signatory() block.Signatory {
	return precommit.signatory
}

func (precommit Precommit) Height() block.Height {
	return precommit.height
}

func (precommit Precommit) Round() block.Round {
	return precommit.round
}

func (precommit Precommit) BlockHash() block.Hash {
	return precommit.blockHash
}

type Inbox struct {
	f        int
	messages map[block.Height]map[block.Round]map[block.Signatory]Message
}

func (inbox *Inbox) Insert(message Message) (n int, firstTime, firstTimeExceedingF bool, firstTimeExceeding2F bool) {
	if _, ok := inbox.messages[message.Height()]; !ok {
		inbox.messages[message.Height()] = map[block.Round]map[block.Signatory]Message{}
	}
	if _, ok := inbox.messages[message.Height()][message.Round()]; !ok {
		inbox.messages[message.Height()][message.Round()] = map[block.Signatory]Message{}
	}

	previousN := len(inbox.messages[message.Height()][message.Round()])
	inbox.messages[message.Height()][message.Round()][message.Signatory()] = message
	n = len(inbox.messages[message.Height()][message.Round()])
	firstTime = (previousN == 0) && (n == 1)
	firstTimeExceedingF = (previousN < inbox.f+1) && (n > inbox.f)
	firstTimeExceeding2F = (previousN < 2*inbox.f+1) && (n > 2*inbox.f)
	return
}

func (inbox *Inbox) QueryByHeightRoundBlockHash(height block.Height, round block.Round, blockHash block.Hash) (n int) {
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

func (inbox *Inbox) QueryByHeightRoundSignatory(height block.Height, round block.Round, sig block.Signatory) Message {
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
