package process

import (
	"encoding/json"
	"sync"
	"time"

	"github.com/renproject/hyperdrive/block"
	"github.com/renproject/hyperdrive/id"
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

// A Blockchain defines a storage interface for Blocks that is based around
// Height.
type Blockchain interface {
	InsertBlockAtHeight(block.Height, block.Block)
	BlockAtHeight(block.Height) (block.Block, bool)
	BlockExistsAtHeight(block.Height) bool
}

// A Proposer builds a `block.Block` for proposals.
type Proposer interface {
	BlockProposal(block.Height, block.Round) block.Block
}

// A Validator validates a `block.Block` that has been proposed.
type Validator interface {
	IsBlockValid(block.Block) bool
}

// An Observer is notified when note-worthy events happen for the first time.
type Observer interface {
	DidCommitBlock(block.Height)
}

// A Scheduler determines which `id.Signatory` should be broadcasting
// proposals in at a given `block.Height` and `block.Round`.
type Scheduler interface {
	Schedule(block.Height, block.Round) id.Signatory
}

// A Broadcaster sends a Message to as many Processes in the network as
// possible.
type Broadcaster interface {
	Broadcast(Message)
}

// A Timer determines the timeout duration at a given Step and `block.Round`.
type Timer interface {
	Timeout(step Step, round block.Round) time.Duration
}

// Processes defines a wrapper type around the []Process type.
type Processes []Process

// A Process defines a state machine in the distributed replicated state
// machine. See https://arxiv.org/pdf/1807.04938.pdf for more information.
type Process struct {
	mu *sync.Mutex

	signatory  id.Signatory
	blockchain Blockchain
	state      State

	proposer    Proposer
	validator   Validator
	scheduler   Scheduler
	broadcaster Broadcaster
	timer       Timer
	observer    Observer
}

// New Process initialised to the default state, starting in the first round.
func New(signatory id.Signatory, blockchain Blockchain, state State, proposer Proposer, validator Validator, observer Observer, broadcaster Broadcaster, scheduler Scheduler, timer Timer) Process {
	p := Process{
		mu: new(sync.Mutex),

		signatory:  signatory,
		blockchain: blockchain,
		state:      state,

		proposer:    proposer,
		validator:   validator,
		observer:    observer,
		broadcaster: broadcaster,
		scheduler:   scheduler,
		timer:       timer,
	}
	if state.Equal(DefaultState()) {
		// Only call StartRound when the State has not been initialised, to
		// prevent resetting a State that has been restored from persistent
		// storage
		p.StartRound(0)
	}
	return p
}

// MarshalJSON implements the `json.Marshaler` interface for the Process type,
// by marshaling its isolated State.
func (p Process) MarshalJSON() ([]byte, error) {
	return json.Marshal(p.state)
}

// UnmarshalJSON implements the `json.Unmarshaler` interface for the Process
// type, by unmarshaling its isolated State.
func (p *Process) UnmarshalJSON(data []byte) error {
	return json.Unmarshal(data, &p.state)
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
func (p *Process) HandleMessage(m Message) {
	p.mu.Lock()
	defer p.mu.Unlock()

	switch m := m.(type) {
	case *Propose:
		p.handlePropose(m)
	case *Prevote:
		p.handlePrevote(m)
	case *Precommit:
		p.handlePrecommit(m)
	}
}

func (p *Process) startRound(round block.Round) {
	p.state.CurrentRound = round
	p.state.CurrentStep = StepPropose
	if p.signatory.Equal(p.scheduler.Schedule(p.state.CurrentHeight, p.state.CurrentRound)) {
		var proposal block.Block
		if p.state.ValidBlock.Hash() != block.InvalidHash {
			proposal = p.state.ValidBlock
		} else {
			proposal = p.proposer.BlockProposal(p.state.CurrentHeight, p.state.CurrentRound)
		}
		p.broadcaster.Broadcast(NewPropose(
			p.state.CurrentHeight,
			p.state.CurrentRound,
			proposal,
			p.state.ValidRound,
		))
	} else {
		p.scheduleTimeoutPropose(p.state.CurrentHeight, p.state.CurrentRound, p.timer.Timeout(StepPropose, p.state.CurrentRound))
	}
}

func (p *Process) handlePropose(propose *Propose) {
	_, firstTime, _, _ := p.state.Proposals.Insert(propose)

	// upon Propose{currentHeight, currentRound, block, -1}
	if propose.Height() == p.state.CurrentHeight && propose.Round() == p.state.CurrentRound && propose.ValidRound() == block.InvalidRound {
		// from Schedule{currentHeight, currentRound}
		if propose.Signatory().Equal(p.scheduler.Schedule(p.state.CurrentHeight, p.state.CurrentRound)) {
			// while currentStep = StepPropose
			if p.state.CurrentStep == StepPropose {
				if p.validator.IsBlockValid(propose.Block()) && (p.state.LockedRound == block.InvalidRound || p.state.LockedBlock.Equal(propose.Block())) {
					p.broadcaster.Broadcast(NewPrevote(
						p.state.CurrentHeight,
						p.state.CurrentRound,
						propose.Block().Hash(),
					))
				} else {
					p.broadcaster.Broadcast(NewPrevote(
						p.state.CurrentHeight,
						p.state.CurrentRound,
						block.InvalidHash,
					))
				}
				p.state.CurrentStep = StepPrevote
			}
		}
	}

	p.checkProposeInCurrentHeightAndRoundWithPrevotes()
	if firstTime {
		p.checkProposeInCurrentHeightAndRoundWithPrevotesForTheFirstTime()
	}
	p.checkProposeInCurrentHeightWithPrecommits(propose.Round())
}

func (p *Process) handlePrevote(prevote *Prevote) {
	n, _, _, firstTimeExceeding2F := p.state.Prevotes.Insert(prevote)
	if firstTimeExceeding2F && prevote.Height() == p.state.CurrentHeight && prevote.Round() == p.state.CurrentRound && p.state.CurrentStep == StepPrevote {
		// upon 2f+1 Prevote{currentHeight, currentRound, *} while step = StepPrevote for the first time
		p.scheduleTimeoutPrevote(p.state.CurrentHeight, p.state.CurrentRound, p.timer.Timeout(StepPrevote, p.state.CurrentRound))
	}

	// upon 2f+1 Prevote{currentHeight, currentRound, nil} while currentStep = StepPrevote
	if n := p.state.Prevotes.QueryByHeightRoundBlockHash(p.state.CurrentHeight, p.state.CurrentRound, block.InvalidHash); n > 2*p.state.Prevotes.F() && p.state.CurrentStep == StepPrevote {
		p.broadcaster.Broadcast(NewPrecommit(
			p.state.CurrentHeight,
			p.state.CurrentRound,
			block.InvalidHash,
		))
		p.state.CurrentStep = StepPrecommit
	}

	// upon f+1 *{currentHeight, round, *, *} and round > currentRound
	if n > p.state.Prevotes.F() && prevote.Height() == p.state.CurrentHeight && prevote.Round() > p.state.CurrentRound {
		p.startRound(prevote.Round())
	}

	p.checkProposeInCurrentHeightAndRoundWithPrevotes()
	if firstTimeExceeding2F {
		p.checkProposeInCurrentHeightAndRoundWithPrevotesForTheFirstTime()
	}
}

func (p *Process) handlePrecommit(precommit *Precommit) {
	// upon 2f+1 Precommit{currentHeight, currentRound, *} for the first time
	n, _, _, firstTimeExceeding2F := p.state.Precommits.Insert(precommit)
	if firstTimeExceeding2F && precommit.Height() == p.state.CurrentHeight && precommit.Round() == p.state.CurrentRound {
		p.scheduleTimeoutPrecommit(p.state.CurrentHeight, p.state.CurrentRound, p.timer.Timeout(StepPrecommit, p.state.CurrentRound))
	}

	// upon f+1 *{currentHeight, round, *, *} and round > currentRound
	if n > p.state.Precommits.F() && precommit.Height() == p.state.CurrentHeight && precommit.Round() > p.state.CurrentRound {
		p.startRound(precommit.Round())
	}

	p.checkProposeInCurrentHeightWithPrecommits(precommit.Round())
}

func (p *Process) timeoutPropose(height block.Height, round block.Round) {
	if height == p.state.CurrentHeight && round == p.state.CurrentRound && p.state.CurrentStep == StepPropose {
		p.broadcaster.Broadcast(NewPrevote(
			p.state.CurrentHeight,
			p.state.CurrentRound,
			block.InvalidHash,
		))
		p.state.CurrentStep = StepPrevote
	}
}

func (p *Process) timeoutPrevote(height block.Height, round block.Round) {
	if height == p.state.CurrentHeight && round == p.state.CurrentRound && p.state.CurrentStep == StepPrevote {
		p.broadcaster.Broadcast(NewPrecommit(
			p.state.CurrentHeight,
			p.state.CurrentRound,
			block.InvalidHash,
		))
		p.state.CurrentStep = StepPrecommit
	}
}

func (p *Process) timeoutPrecommit(height block.Height, round block.Round) {
	if height == p.state.CurrentHeight && round == p.state.CurrentRound {
		p.startRound(p.state.CurrentRound + 1)
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
	m := p.state.Proposals.QueryByHeightRoundSignatory(p.state.CurrentHeight, p.state.CurrentRound, p.scheduler.Schedule(p.state.CurrentHeight, p.state.CurrentRound))
	if m == nil {
		return
	}
	propose := m.(*Propose)

	if propose.ValidRound() > block.InvalidRound {
		// and 2f+1 Prevote{currentHeight, validRound, blockHash}
		n := p.state.Prevotes.QueryByHeightRoundBlockHash(p.state.CurrentHeight, propose.ValidRound(), propose.BlockHash())
		if n > 2*p.state.Prevotes.F() {
			// while step = StepPropose and validRound >= 0 and validRound < currentRound
			if p.state.CurrentStep == StepPropose && propose.ValidRound() < p.state.CurrentRound {
				if p.validator.IsBlockValid(propose.Block()) && (p.state.LockedRound <= propose.ValidRound() || p.state.LockedBlock.Equal(propose.Block())) {
					p.broadcaster.Broadcast(NewPrevote(
						p.state.CurrentHeight,
						p.state.CurrentRound,
						propose.Block().Hash(),
					))
				} else {
					p.broadcaster.Broadcast(NewPrevote(
						p.state.CurrentHeight,
						p.state.CurrentRound,
						block.InvalidHash,
					))
				}
				p.state.CurrentStep = StepPrevote
			}
		}
	}
}

// checkProposeInCurrentHeightAndRoundWithPrevotesForTheFirstTime must only be
// called when a Propose and 2f+1 Prevotes has been seen for the first time at
// the current `block.Height` and `block.Round`. This can happen when a Propose
// is seen for the first time at the current `block.Height` and `block.Round`,
// or, when a Prevote is seen for the first time at the current `block.Height`
// and `block.Round`.
func (p *Process) checkProposeInCurrentHeightAndRoundWithPrevotesForTheFirstTime() {
	// upon Propose{currentHeight, currentRound, block, *} from Schedule(currentHeight, currentRound)
	m := p.state.Proposals.QueryByHeightRoundSignatory(p.state.CurrentHeight, p.state.CurrentRound, p.scheduler.Schedule(p.state.CurrentHeight, p.state.CurrentRound))
	if m == nil {
		return
	}
	propose := m.(*Propose)

	// and 2f+1 Prevote{currentHeight, currentRound, blockHash} while Validate(block) and step >= StepPrevote for the first time
	n := p.state.Prevotes.QueryByHeightRoundBlockHash(p.state.CurrentHeight, p.state.CurrentRound, propose.BlockHash())
	if n > 2*p.state.Prevotes.F() {
		if p.state.CurrentStep >= StepPrevote && p.validator.IsBlockValid(propose.Block()) {
			if p.state.CurrentStep == StepPrevote {
				p.state.LockedBlock = propose.Block()
				p.state.LockedRound = p.state.CurrentRound
				p.state.CurrentStep = StepPrecommit
				p.broadcaster.Broadcast(NewPrecommit(
					p.state.CurrentHeight,
					p.state.CurrentRound,
					propose.Block().Hash(),
				))
			}
			p.state.ValidBlock = propose.Block()
			p.state.ValidRound = p.state.CurrentRound
		}
	}
}

func (p *Process) checkProposeInCurrentHeightWithPrecommits(round block.Round) {
	// upon Propose{currentHeight, round, block, *} from Schedule(currentHeight, round)
	m := p.state.Proposals.QueryByHeightRoundSignatory(p.state.CurrentHeight, round, p.scheduler.Schedule(p.state.CurrentHeight, round))
	if m == nil {
		return
	}
	propose := m.(*Propose)

	// and 2f+1 Precommits{currentHeight, round, blockHash}
	n := p.state.Precommits.QueryByHeightRoundBlockHash(p.state.CurrentHeight, round, propose.BlockHash())
	if n > 2*p.state.Precommits.F() {
		// while !BlockExistsAtHeight(currentHeight)
		if !p.blockchain.BlockExistsAtHeight(p.state.CurrentHeight) {
			if p.validator.IsBlockValid(propose.Block()) {
				p.blockchain.InsertBlockAtHeight(p.state.CurrentHeight, propose.Block())
				p.state.CurrentHeight++
				p.state.Reset()
				p.startRound(0)

				if p.observer != nil {
					p.observer.DidCommitBlock(p.state.CurrentHeight - 1)
				}
			}
		}
	}
}
