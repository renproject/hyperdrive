package automaton

import (
	"sync"
	"time"

	"github.com/renproject/hyperdrive/automaton/block"
	"github.com/renproject/hyperdrive/automaton/message"
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

type Automaton struct {
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

func New(id message.Signatory, blockchain block.Blockchain, proposals, prevotes, precommits message.Inbox, broadcaster Broadcaster, scheduler Scheduler, proposer Proposer, validator Validator, timer Timer) Automaton {
	auto := Automaton{
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
	auto.StartRound(0)
	return auto
}

func (auto *Automaton) StartRound(round block.Round) {
	auto.mu.Lock()
	defer auto.mu.Unlock()

	auto.startRound(round)
}

func (auto *Automaton) HandleMessage(m message.Message) {
	auto.mu.Lock()
	defer auto.mu.Unlock()

	switch m := m.(type) {
	case message.Propose:
		auto.handlePropose(m)
	case message.Prevote:
		auto.handlePrevote(m)
	case message.Precommit:
		auto.handlePrecommit(m)
	}
}

func (auto *Automaton) startRound(round block.Round) {
	auto.currentRound = round
	auto.currentStep = StepPropose
	if auto.id.Equal(auto.scheduler.Schedule(auto.currentHeight, auto.currentRound)) {
		var proposal block.Block
		if auto.validBlock != nil {
			proposal = auto.validBlock
		} else {
			proposal = auto.proposer.Propose()
		}
		auto.broadcaster.Broadcast(message.NewPropose(
			auto.id,
			auto.currentHeight,
			auto.currentRound,
			proposal,
			auto.validRound,
		))
	} else {
		auto.scheduleTimeoutPropose(auto.currentHeight, auto.currentRound, auto.timer.Timeout(StepPropose, auto.currentRound))
	}
}

func (auto *Automaton) handlePropose(propose message.Propose) {
	// NOTICE: It is required that F=1 for the proposals inbox.
	_, didExceedF, _ := auto.proposals.Insert(propose)

	// upon Propose(currentHeight, currentRound, block, -1)
	if propose.Height() == auto.currentHeight && propose.Round() == auto.currentRound && propose.ValidRound() == block.InvalidRound {
		// from Schedule(currentHeight, currentRound)
		if propose.Signatory().Equal(auto.scheduler.Schedule(auto.currentHeight, auto.currentRound)) {
			// while step = StepPropose
			if auto.currentStep == StepPropose {
				if auto.validator.Validate(propose.Block()) && (auto.lockedRound == block.InvalidRound || auto.lockedBlock.Equal(propose.Block())) {
					auto.broadcaster.Broadcast(message.NewPrevote(
						auto.id,
						auto.currentHeight,
						auto.currentRound,
						propose.Block().Hash(),
					))
				} else {
					auto.broadcaster.Broadcast(message.NewPrevote(
						auto.id,
						auto.currentHeight,
						auto.currentRound,
						block.InvalidHash,
					))
				}
				auto.currentStep = StepPrevote
			}
		}
	}

	auto.checkProposeInCurrentHeightAndRoundWithPrevotes()
	if didExceedF {
		auto.checkProposeInCurrentHeightAndRoundWithPrevotesForTheFirstTime()
	}
	auto.checkProposeInCurrentHeightWithPrecommits(propose.Round())
}

func (auto *Automaton) handlePrevote(prevote message.Prevote) {
	n, _, didExceed2F := auto.prevotes.Insert(prevote)
	if didExceed2F && prevote.Height() == auto.currentHeight && prevote.Round() == auto.currentRound && auto.currentStep == StepPrevote {
		// upon 2f + 1 Prevote(currentHeight, currentRound, *) while step = StepPrevote, for the first time
		auto.scheduleTimeoutPrevote(auto.currentHeight, auto.currentRound, auto.timer.Timeout(StepPrevote, auto.currentRound))
	}

	// upon f + 1 *(currentHeight, round, *, *) and round > currentRound
	if n > auto.prevotes.F() && prevote.Height() == auto.currentHeight && prevote.Round() > auto.currentRound {
		auto.startRound(prevote.Round())
	}

	auto.checkProposeInCurrentHeightAndRoundWithPrevotes()
	if didExceed2F {
		auto.checkProposeInCurrentHeightAndRoundWithPrevotesForTheFirstTime()
	}
}

func (auto *Automaton) handlePrecommit(precommit message.Precommit) {
	// upon 2f + 1 Precommit(currentHeight, currentRound, *) for the first time
	n, _, didExceed2F := auto.precommits.Insert(precommit)
	if didExceed2F && precommit.Height() == auto.currentHeight && precommit.Round() == auto.currentRound {
		auto.scheduleTimeoutPrecommit(auto.currentHeight, auto.currentRound, auto.timer.Timeout(StepPrecommit, auto.currentRound))
	}

	// upon f + 1 *(currentHeight, round, *, *) and round > currentRound
	if n > auto.precommits.F() && precommit.Height() == auto.currentHeight && precommit.Round() > auto.currentRound {
		auto.startRound(precommit.Round())
	}

	auto.checkProposeInCurrentHeightWithPrecommits(precommit.Round())
}

func (auto *Automaton) timeoutPropose(height block.Height, round block.Round) {
	if height == auto.currentHeight && round == auto.currentRound && auto.currentStep == StepPropose {
		auto.broadcaster.Broadcast(message.NewPrevote(
			auto.id,
			auto.currentHeight,
			auto.currentRound,
			block.InvalidHash,
		))
		auto.currentStep = StepPrevote
	}
}

func (auto *Automaton) timeoutPrevote(height block.Height, round block.Round) {
	if height == auto.currentHeight && round == auto.currentRound && auto.currentStep == StepPrevote {
		auto.broadcaster.Broadcast(message.NewPrecommit(
			auto.id,
			auto.currentHeight,
			auto.currentRound,
			block.InvalidHash,
		))
		auto.currentStep = StepPrecommit
	}
}

func (auto *Automaton) timeoutPrecommit(height block.Height, round block.Round) {
	if height == auto.currentHeight && round == auto.currentRound {
		auto.startRound(auto.currentRound + 1)
	}
}

func (auto *Automaton) scheduleTimeoutPropose(height block.Height, round block.Round, duration time.Duration) {
	go func() {
		time.Sleep(duration)

		auto.mu.Lock()
		defer auto.mu.Unlock()

		auto.timeoutPropose(height, round)
	}()
}

func (auto *Automaton) scheduleTimeoutPrevote(height block.Height, round block.Round, duration time.Duration) {
	go func() {
		time.Sleep(duration)

		auto.mu.Lock()
		defer auto.mu.Unlock()

		auto.timeoutPrevote(height, round)
	}()
}

func (auto *Automaton) scheduleTimeoutPrecommit(height block.Height, round block.Round, duration time.Duration) {
	go func() {
		time.Sleep(duration)

		auto.mu.Lock()
		defer auto.mu.Unlock()

		auto.timeoutPrecommit(height, round)
	}()
}

func (auto *Automaton) checkProposeInCurrentHeightAndRoundWithPrevotes() {
	// upon Propose(currentHeight, currentRound, block, validRound)
	// from Schedule(auto.currentHeight, auto.currentRound)
	m := auto.proposals.QueryByHeightRoundSignatory(auto.currentHeight, auto.currentRound, auto.scheduler.Schedule(auto.currentHeight, auto.currentRound))
	if m == nil {
		return
	}
	propose := m.(message.Propose)

	if propose.ValidRound() > block.InvalidRound {
		// and 2f + 1 Prevote(currentHeight, validRound, blockHash)
		n := auto.prevotes.QueryByHeightRoundBlockHash(auto.currentHeight, propose.ValidRound(), propose.BlockHash())
		if n > 2*auto.prevotes.F() {
			// while step = StepPropose and validRound >= 0 and validRound < currentRound
			if auto.currentStep == StepPropose && propose.ValidRound() < auto.currentRound {
				if auto.validator.Validate(propose.Block()) && (auto.lockedRound <= propose.ValidRound() || auto.lockedBlock.Equal(propose.Block())) {
					auto.broadcaster.Broadcast(message.NewPrevote(
						auto.id,
						auto.currentHeight,
						auto.currentRound,
						propose.Block().Hash(),
					))
				} else {
					auto.broadcaster.Broadcast(message.NewPrevote(
						auto.id,
						auto.currentHeight,
						auto.currentRound,
						block.InvalidHash,
					))
				}
				auto.currentStep = StepPrevote
			}
		}
	}
}

func (auto *Automaton) checkProposeInCurrentHeightAndRoundWithPrevotesForTheFirstTime() {
	// NOTICE: It is required that this function is only called when a proposal
	// is received for the first time, or, 2f + 1 Prevote(currentHeight,
	// currentRound, *) are seen for the first time

	// upon Propose(currentHeight, currentRound, block, *)
	// from Schedule(auto.currentHeight, auto.currentRound)
	m := auto.proposals.QueryByHeightRoundSignatory(auto.currentHeight, auto.currentRound, auto.scheduler.Schedule(auto.currentHeight, auto.currentRound))
	if m == nil {
		return
	}
	propose := m.(message.Propose)

	// and 2f + 1 Prevote(currentHeight, currentRound, blockHash) while valid(block) and step >= StepPrevote, for the first time
	n := auto.prevotes.QueryByHeightRoundBlockHash(auto.currentHeight, auto.currentRound, propose.BlockHash())
	if n > 2*auto.prevotes.F() {
		if auto.currentStep >= StepPrevote && auto.validator.Validate(propose.Block()) {
			if auto.currentStep == StepPrevote {
				auto.lockedBlock = propose.Block()
				auto.lockedRound = auto.currentRound
				auto.broadcaster.Broadcast(message.NewPrecommit(
					auto.id,
					auto.currentHeight,
					auto.currentRound,
					propose.Block().Hash(),
				))
			}
			auto.validBlock = propose.Block()
			auto.validRound = auto.currentRound
		}
	}
}

func (auto *Automaton) checkProposeInCurrentHeightWithPrecommits(round block.Round) {
	// upon Propose(currentHeight, round, block, *)
	// from Schedule(auto.currentHeight, round)
	m := auto.proposals.QueryByHeightRoundSignatory(auto.currentHeight, round, auto.scheduler.Schedule(auto.currentHeight, round))
	if m == nil {
		return
	}
	propose := m.(message.Propose)

	// and 2f + 1 Precommits(currentHeight, round, blockHash)
	n := auto.precommits.QueryByHeightRoundBlockHash(auto.currentHeight, round, propose.BlockHash())
	if n > 2*auto.precommits.F() {
		// while blockchain[currentHeight] = nil
		if auto.blockchain.Block(auto.currentHeight) == nil {
			if auto.validator.Validate(propose.Block()) {
				auto.blockchain.InsertBlock(auto.currentHeight, propose.Block())
				auto.currentHeight++
				auto.lockedBlock = block.InvalidBlock
				auto.lockedRound = block.InvalidRound
				auto.validBlock = block.InvalidBlock
				auto.validRound = block.InvalidRound
				auto.startRound(0)
			}
		}
	}
}
