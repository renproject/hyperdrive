package process

import (
	"sync"
	"time"

	"github.com/renproject/hyperdrive/process/block"
	"github.com/renproject/hyperdrive/process/message"
)

type Step uint8

const (
	StepPropose   = Step(1)
	StepPrevote   = Step(2)
	StepPrecommit = Step(3)
)

type Broadcaster interface {
	Broadcast(message.Message)
}

type Scheduler interface {
	Schedule(block.Height, block.Round) message.Signatory
}

type Proposer interface {
	Propose() block.Block
}

type Validator interface {
	Validate(block.Block) bool
}

type Timer interface {
	Timeout(step Step, round block.Round) time.Duration
}

type Process struct {
	mu *sync.Mutex

	id message.Signatory

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

	broadcaster Broadcaster
	scheduler   Scheduler
	proposer    Proposer
	validator   Validator
	timer       Timer
}

func New(id message.Signatory, blockchain block.Blockchain, proposals, prevotes, precommits message.Inbox, broadcaster Broadcaster, scheduler Scheduler, proposer Proposer, validator Validator, timer Timer) Process {
	p := Process{
		mu: new(sync.Mutex),

		id: id,

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

		broadcaster: broadcaster,
		scheduler:   scheduler,
		proposer:    proposer,
		validator:   validator,
		timer:       timer,
	}
	p.StartRound(0)
	return p
}

func (p *Process) StartRound(round block.Round) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.startRound(round)
}

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
	if p.id.Equal(p.scheduler.Schedule(p.currentHeight, p.currentRound)) {
		var proposal block.Block
		if p.validBlock != nil {
			proposal = p.validBlock
		} else {
			proposal = p.proposer.Propose()
		}
		p.broadcaster.Broadcast(message.NewPropose(
			p.id,
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
	// NOTICE: It is required that F=1 for the proposals inbox.
	_, didExceedF, _ := p.proposals.Insert(propose)

	// upon Propose(currentHeight, currentRound, block, -1)
	if propose.Height() == p.currentHeight && propose.Round() == p.currentRound && propose.ValidRound() == block.InvalidRound {
		// from Schedule(currentHeight, currentRound)
		if propose.Signatory().Equal(p.scheduler.Schedule(p.currentHeight, p.currentRound)) {
			// while step = StepPropose
			if p.currentStep == StepPropose {
				if p.validator.Validate(propose.Block()) && (p.lockedRound == block.InvalidRound || p.lockedBlock.Equal(propose.Block())) {
					p.broadcaster.Broadcast(message.NewPrevote(
						p.id,
						p.currentHeight,
						p.currentRound,
						propose.Block().Hash(),
					))
				} else {
					p.broadcaster.Broadcast(message.NewPrevote(
						p.id,
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
	if didExceedF {
		p.checkProposeInCurrentHeightAndRoundWithPrevotesForTheFirstTime()
	}
	p.checkProposeInCurrentHeightWithPrecommits(propose.Round())
}

func (p *Process) handlePrevote(prevote message.Prevote) {
	n, _, didExceed2F := p.prevotes.Insert(prevote)
	if didExceed2F && prevote.Height() == p.currentHeight && prevote.Round() == p.currentRound && p.currentStep == StepPrevote {
		// upon 2f + 1 Prevote(currentHeight, currentRound, *) while step = StepPrevote, for the first time
		p.scheduleTimeoutPrevote(p.currentHeight, p.currentRound, p.timer.Timeout(StepPrevote, p.currentRound))
	}

	// upon f + 1 *(currentHeight, round, *, *) and round > currentRound
	if n > p.prevotes.F() && prevote.Height() == p.currentHeight && prevote.Round() > p.currentRound {
		p.startRound(prevote.Round())
	}

	p.checkProposeInCurrentHeightAndRoundWithPrevotes()
	if didExceed2F {
		p.checkProposeInCurrentHeightAndRoundWithPrevotesForTheFirstTime()
	}
}

func (p *Process) handlePrecommit(precommit message.Precommit) {
	// upon 2f + 1 Precommit(currentHeight, currentRound, *) for the first time
	n, _, didExceed2F := p.precommits.Insert(precommit)
	if didExceed2F && precommit.Height() == p.currentHeight && precommit.Round() == p.currentRound {
		p.scheduleTimeoutPrecommit(p.currentHeight, p.currentRound, p.timer.Timeout(StepPrecommit, p.currentRound))
	}

	// upon f + 1 *(currentHeight, round, *, *) and round > currentRound
	if n > p.precommits.F() && precommit.Height() == p.currentHeight && precommit.Round() > p.currentRound {
		p.startRound(precommit.Round())
	}

	p.checkProposeInCurrentHeightWithPrecommits(precommit.Round())
}

func (p *Process) timeoutPropose(height block.Height, round block.Round) {
	if height == p.currentHeight && round == p.currentRound && p.currentStep == StepPropose {
		p.broadcaster.Broadcast(message.NewPrevote(
			p.id,
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
			p.id,
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
	// upon Propose(currentHeight, currentRound, block, validRound)
	// from Schedule(p.currentHeight, p.currentRound)
	m := p.proposals.QueryByHeightRoundSignatory(p.currentHeight, p.currentRound, p.scheduler.Schedule(p.currentHeight, p.currentRound))
	if m == nil {
		return
	}
	propose := m.(message.Propose)

	if propose.ValidRound() > block.InvalidRound {
		// and 2f + 1 Prevote(currentHeight, validRound, blockHash)
		n := p.prevotes.QueryByHeightRoundBlockHash(p.currentHeight, propose.ValidRound(), propose.BlockHash())
		if n > 2*p.prevotes.F() {
			// while step = StepPropose and validRound >= 0 and validRound < currentRound
			if p.currentStep == StepPropose && propose.ValidRound() < p.currentRound {
				if p.validator.Validate(propose.Block()) && (p.lockedRound <= propose.ValidRound() || p.lockedBlock.Equal(propose.Block())) {
					p.broadcaster.Broadcast(message.NewPrevote(
						p.id,
						p.currentHeight,
						p.currentRound,
						propose.Block().Hash(),
					))
				} else {
					p.broadcaster.Broadcast(message.NewPrevote(
						p.id,
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

func (p *Process) checkProposeInCurrentHeightAndRoundWithPrevotesForTheFirstTime() {
	// NOTICE: It is required that this function is only called when a proposal
	// is received for the first time, or, 2f + 1 Prevote(currentHeight,
	// currentRound, *) are seen for the first time

	// upon Propose(currentHeight, currentRound, block, *)
	// from Schedule(p.currentHeight, p.currentRound)
	m := p.proposals.QueryByHeightRoundSignatory(p.currentHeight, p.currentRound, p.scheduler.Schedule(p.currentHeight, p.currentRound))
	if m == nil {
		return
	}
	propose := m.(message.Propose)

	// and 2f + 1 Prevote(currentHeight, currentRound, blockHash) while valid(block) and step >= StepPrevote, for the first time
	n := p.prevotes.QueryByHeightRoundBlockHash(p.currentHeight, p.currentRound, propose.BlockHash())
	if n > 2*p.prevotes.F() {
		if p.currentStep >= StepPrevote && p.validator.Validate(propose.Block()) {
			if p.currentStep == StepPrevote {
				p.lockedBlock = propose.Block()
				p.lockedRound = p.currentRound
				p.broadcaster.Broadcast(message.NewPrecommit(
					p.id,
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
	// upon Propose(currentHeight, round, block, *)
	// from Schedule(p.currentHeight, round)
	m := p.proposals.QueryByHeightRoundSignatory(p.currentHeight, round, p.scheduler.Schedule(p.currentHeight, round))
	if m == nil {
		return
	}
	propose := m.(message.Propose)

	// and 2f + 1 Precommits(currentHeight, round, blockHash)
	n := p.precommits.QueryByHeightRoundBlockHash(p.currentHeight, round, propose.BlockHash())
	if n > 2*p.precommits.F() {
		// while blockchain[currentHeight] = nil
		if p.blockchain.Block(p.currentHeight) == nil {
			if p.validator.Validate(propose.Block()) {
				p.blockchain.InsertBlock(p.currentHeight, propose.Block())
				p.currentHeight++
				p.lockedBlock = block.InvalidBlock
				p.lockedRound = block.InvalidRound
				p.validBlock = block.InvalidBlock
				p.validRound = block.InvalidRound
				p.startRound(0)
			}
		}
	}
}
