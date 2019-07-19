package process

import (
	"sync"
	"time"

	"github.com/renproject/hyperdrive/process/block"
	"github.com/renproject/hyperdrive/process/message"
)

// Step in the consensus algorithm.
type Step uint8

// Define all Steps.
const (
	StepNil       = Step(0)
	StepPropose   = Step(1)
	StepPrevote   = Step(2)
	StepPrecommit = Step(3)
)

// A Proposer builds a `block.Block` for proposals.
type Proposer interface {
	Propose() block.Block
}

// A Validator validates a `block.Block` that has been proposed.
type Validator interface {
	Validate(block.Block) bool
}

// A Scheduler determines which `block.Signatory` should be broadcasting
// proposals in at a given `block.Height` and `block.Round`.
type Scheduler interface {
	Schedule(block.Height, block.Round) block.Signatory
}

// A Broadcaster sends a `message.Message` to as many Processes in the network
// as possible.
type Broadcaster interface {
	Broadcast(message.Message)
}

// A Timer determines the timeout duration at a given Step and `block.Round`.
type Timer interface {
	Timeout(step Step, round block.Round) time.Duration
}

// An Observer is notified when note-worthy events happen for the first time.
type Observer interface {
	OnBlockCommitted(blockHash block.Hash)
}

// Processes defines a wrapper type around the []Process type.
type Processes []Process

// A Process defines a state machine in the distributed replicated state
// machine. See https://arxiv.org/pdf/1807.04938.pdf for more information.
type Process struct {
	mu *sync.Mutex

	signatory block.Signatory

	currentHeight block.Height
	currentRound  block.Round
	currentStep   Step

	lockedBlock block.Block
	lockedRound block.Round
	validBlock  block.Block
	validRound  block.Round

	blockchain block.Blockchain
	proposals  message.Inbox
	prevotes   message.Inbox
	precommits message.Inbox

	proposer    Proposer
	validator   Validator
	scheduler   Scheduler
	broadcaster Broadcaster
	timer       Timer
	observer    Observer
}

// New Process initialised to the default state, starting in the first round.
func New(signatory block.Signatory, blockchain block.Blockchain, proposals, prevotes, precommits message.Inbox, proposer Proposer, validator Validator, scheduler Scheduler, broadcaster Broadcaster, timer Timer, observer Observer) Process {
	p := Process{
		mu: new(sync.Mutex),

		signatory: signatory,

		currentHeight: 0,
		currentRound:  0,
		currentStep:   StepPropose,

		lockedBlock: block.InvalidBlock,
		lockedRound: block.InvalidRound,
		validBlock:  block.InvalidBlock,
		validRound:  block.InvalidRound,

		blockchain: blockchain,
		proposals:  proposals,
		prevotes:   prevotes,
		precommits: precommits,

		proposer:    proposer,
		validator:   validator,
		scheduler:   scheduler,
		broadcaster: broadcaster,
		timer:       timer,
		observer:    observer,
	}
	p.StartRound(0)
	return p
}

// StartRound is safe for concurrent use. See
// https://arxiv.org/pdf/1807.04938.pdf for more information.
func (p *Process) StartRound(round block.Round) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.startRound(round)
}

// HandleMessage is safe for concurrent use. See
// https://arxiv.org/pdf/1807.04938.pdf for more information.
func (p *Process) HandleMessage(m message.Message) {
	p.mu.Lock()
	defer p.mu.Unlock()

	switch m := m.(type) {
	case message.Propose:
		p.handlePropose(m)
	case message.Prevote:
		p.handlePrevote(m)
	case message.Precommit:
		p.handlePrecommit(m)
	}
}

func (p *Process) startRound(round block.Round) {
	p.currentRound = round
	p.currentStep = StepPropose
	if p.signatory.Equal(p.scheduler.Schedule(p.currentHeight, p.currentRound)) {
		var proposal block.Block
		if p.validBlock.Hash() != block.InvalidHash {
			proposal = p.validBlock
		} else {
			proposal = p.proposer.Propose()
		}
		p.broadcaster.Broadcast(message.NewPropose(
			p.signatory,
			p.currentHeight,
			p.currentRound,
			proposal,
			p.validRound,
		))
	} else {
		p.scheduleTimeoutPropose(p.currentHeight, p.currentRound, p.timer.Timeout(StepPropose, p.currentRound))
	}
}

func (p *Process) handlePropose(propose message.Propose) {
	_, firstTime, _, _ := p.proposals.Insert(propose)

	// upon Propose{currentHeight, currentRound, block, -1}
	if propose.Height() == p.currentHeight && propose.Round() == p.currentRound && propose.ValidRound() == block.InvalidRound {
		// from Schedule{currentHeight, currentRound}
		if propose.Signatory().Equal(p.scheduler.Schedule(p.currentHeight, p.currentRound)) {
			// while step = StepPropose
			if p.currentStep == StepPropose {
				if p.validator.Validate(propose.Block()) && (p.lockedRound == block.InvalidRound || p.lockedBlock.Equal(propose.Block())) {
					p.broadcaster.Broadcast(message.NewPrevote(
						p.signatory,
						p.currentHeight,
						p.currentRound,
						propose.Block().Hash(),
					))
				} else {
					p.broadcaster.Broadcast(message.NewPrevote(
						p.signatory,
						p.currentHeight,
						p.currentRound,
						block.InvalidHash,
					))
				}
				p.currentStep = StepPrevote
			}
		}
	}

	p.checkProposeInCurrentHeightAndRoundWithPrevotes()
	if firstTime {
		p.checkProposeInCurrentHeightAndRoundWithPrevotesForTheFirstTime()
	}
	p.checkProposeInCurrentHeightWithPrecommits(propose.Round())
}

func (p *Process) handlePrevote(prevote message.Prevote) {
	n, _, _, firstTimeExceeding2F := p.prevotes.Insert(prevote)
	if firstTimeExceeding2F && prevote.Height() == p.currentHeight && prevote.Round() == p.currentRound && p.currentStep == StepPrevote {
		// upon 2f+1 Prevote{currentHeight, currentRound, *} while step = StepPrevote for the first time
		p.scheduleTimeoutPrevote(p.currentHeight, p.currentRound, p.timer.Timeout(StepPrevote, p.currentRound))
	}

	// upon f+1 *{currentHeight, round, *, *} and round > currentRound
	if n > p.prevotes.F() && prevote.Height() == p.currentHeight && prevote.Round() > p.currentRound {
		p.startRound(prevote.Round())
	}

	p.checkProposeInCurrentHeightAndRoundWithPrevotes()
	if firstTimeExceeding2F {
		p.checkProposeInCurrentHeightAndRoundWithPrevotesForTheFirstTime()
	}
}

func (p *Process) handlePrecommit(precommit message.Precommit) {
	// upon 2f+1 Precommit{currentHeight, currentRound, *} for the first time
	n, _, _, firstTimeExceeding2F := p.precommits.Insert(precommit)
	if firstTimeExceeding2F && precommit.Height() == p.currentHeight && precommit.Round() == p.currentRound {
		p.scheduleTimeoutPrecommit(p.currentHeight, p.currentRound, p.timer.Timeout(StepPrecommit, p.currentRound))
	}

	// upon f+1 *{currentHeight, round, *, *} and round > currentRound
	if n > p.precommits.F() && precommit.Height() == p.currentHeight && precommit.Round() > p.currentRound {
		p.startRound(precommit.Round())
	}

	p.checkProposeInCurrentHeightWithPrecommits(precommit.Round())
}

func (p *Process) timeoutPropose(height block.Height, round block.Round) {
	if height == p.currentHeight && round == p.currentRound && p.currentStep == StepPropose {
		p.broadcaster.Broadcast(message.NewPrevote(
			p.signatory,
			p.currentHeight,
			p.currentRound,
			block.InvalidHash,
		))
		p.currentStep = StepPrevote
	}
}

func (p *Process) timeoutPrevote(height block.Height, round block.Round) {
	if height == p.currentHeight && round == p.currentRound && p.currentStep == StepPrevote {
		p.broadcaster.Broadcast(message.NewPrecommit(
			p.signatory,
			p.currentHeight,
			p.currentRound,
			block.InvalidHash,
		))
		p.currentStep = StepPrecommit
	}
}

func (p *Process) timeoutPrecommit(height block.Height, round block.Round) {
	if height == p.currentHeight && round == p.currentRound {
		p.startRound(p.currentRound + 1)
	}
}

func (p *Process) scheduleTimeoutPropose(height block.Height, round block.Round, duration time.Duration) {
	go func() {
		time.Sleep(duration)

		p.mu.Lock()
		defer p.mu.Unlock()

		p.timeoutPropose(height, round)
	}()
}

func (p *Process) scheduleTimeoutPrevote(height block.Height, round block.Round, duration time.Duration) {
	go func() {
		time.Sleep(duration)

		p.mu.Lock()
		defer p.mu.Unlock()

		p.timeoutPrevote(height, round)
	}()
}

func (p *Process) scheduleTimeoutPrecommit(height block.Height, round block.Round, duration time.Duration) {
	go func() {
		time.Sleep(duration)

		p.mu.Lock()
		defer p.mu.Unlock()

		p.timeoutPrecommit(height, round)
	}()
}

func (p *Process) checkProposeInCurrentHeightAndRoundWithPrevotes() {
	// upon Propose{currentHeight, currentRound, block, validRound} from Schedule(currentHeight, currentRound)
	m := p.proposals.QueryByHeightRoundSignatory(p.currentHeight, p.currentRound, p.scheduler.Schedule(p.currentHeight, p.currentRound))
	if m == nil {
		return
	}
	propose := m.(message.Propose)

	if propose.ValidRound() > block.InvalidRound {
		// and 2f+1 Prevote{currentHeight, validRound, blockHash}
		n := p.prevotes.QueryByHeightRoundBlockHash(p.currentHeight, propose.ValidRound(), propose.BlockHash())
		if n > 2*p.prevotes.F() {
			// while step = StepPropose and validRound >= 0 and validRound < currentRound
			if p.currentStep == StepPropose && propose.ValidRound() < p.currentRound {
				if p.validator.Validate(propose.Block()) && (p.lockedRound <= propose.ValidRound() || p.lockedBlock.Equal(propose.Block())) {
					p.broadcaster.Broadcast(message.NewPrevote(
						p.signatory,
						p.currentHeight,
						p.currentRound,
						propose.Block().Hash(),
					))
				} else {
					p.broadcaster.Broadcast(message.NewPrevote(
						p.signatory,
						p.currentHeight,
						p.currentRound,
						block.InvalidHash,
					))
				}
				p.currentStep = StepPrevote
			}
		}
	}
}

// checkProposeInCurrentHeightAndRoundWithPrevotesForTheFirstTime must only be
// called when a `message.Propose` and 2f+1 `message.Prevotes` has been seen for
// the first time at the current `block.Height` and `block.Round`. This can
// happen when a `message.Propose` is seen for the first time at the current
// `block.Height` and `block.Round`, or, when a `message.Prevote` is seen for
// the first time at the current `block.Height` and `block.Round`.
func (p *Process) checkProposeInCurrentHeightAndRoundWithPrevotesForTheFirstTime() {
	// upon Propose{currentHeight, currentRound, block, *} from Schedule(currentHeight, currentRound)
	m := p.proposals.QueryByHeightRoundSignatory(p.currentHeight, p.currentRound, p.scheduler.Schedule(p.currentHeight, p.currentRound))
	if m == nil {
		return
	}
	propose := m.(message.Propose)

	// and 2f+1 Prevote{currentHeight, currentRound, blockHash} while Validate(block) and step >= StepPrevote for the first time
	n := p.prevotes.QueryByHeightRoundBlockHash(p.currentHeight, p.currentRound, propose.BlockHash())
	if n > 2*p.prevotes.F() {
		if p.currentStep >= StepPrevote && p.validator.Validate(propose.Block()) {
			if p.currentStep == StepPrevote {
				p.lockedBlock = propose.Block()
				p.lockedRound = p.currentRound
				p.broadcaster.Broadcast(message.NewPrecommit(
					p.signatory,
					p.currentHeight,
					p.currentRound,
					propose.Block().Hash(),
				))
			}
			p.validBlock = propose.Block()
			p.validRound = p.currentRound
		}
	}
}

func (p *Process) checkProposeInCurrentHeightWithPrecommits(round block.Round) {
	// upon Propose{currentHeight, round, block, *} from Schedule(currentHeight, round)
	m := p.proposals.QueryByHeightRoundSignatory(p.currentHeight, round, p.scheduler.Schedule(p.currentHeight, round))
	if m == nil {
		return
	}
	propose := m.(message.Propose)

	// and 2f+1 Precommits{currentHeight, round, blockHash}
	n := p.precommits.QueryByHeightRoundBlockHash(p.currentHeight, round, propose.BlockHash())
	if n > 2*p.precommits.F() {
		// while !BlockExistsAtHeight(currentHeight)
		if p.blockchain.BlockExistsAtHeight(p.currentHeight) {
			if p.validator.Validate(propose.Block()) {
				p.blockchain.InsertBlockAtHeight(p.currentHeight, propose.Block())
				p.currentHeight++
				p.lockedBlock = block.InvalidBlock
				p.lockedRound = block.InvalidRound
				p.validBlock = block.InvalidBlock
				p.validRound = block.InvalidRound
				p.startRound(0)

				if p.observer != nil {
					p.observer.OnBlockCommitted(propose.Block().Hash())
				}
			}
		}
	}
}
