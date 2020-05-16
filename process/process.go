package process

import (
	"io"
	"sync"
	"time"

	"github.com/renproject/hyperdrive/block"
	"github.com/renproject/hyperdrive/schedule"
	"github.com/renproject/id"
	"github.com/renproject/surge"
	"github.com/sirupsen/logrus"
)

// Step in the consensus algorithm.
type Step uint8

// SizeHint of how many bytes will be needed to represent steps in
// binary.
func (Step) SizeHint() int {
	return 1
}

// Marshal this step into binary.
func (step Step) Marshal(w io.Writer, m int) (int, error) {
	return surge.Marshal(w, uint8(step), m)
}

// Unmarshal into this step from binary.
func (step *Step) Unmarshal(r io.Reader, m int) (int, error) {
	return surge.Unmarshal(r, (*uint8)(step), m)
}

// Define all Steps.
const (
	StepNil Step = iota
	StepPropose
	StepPrevote
	StepPrecommit
)

// NilReasons can be used to provide contextual information alongside an error
// upon validating blocks.
type NilReasons map[string][]byte

// A Blockchain defines a storage interface for Blocks that is based around
// Height.
type Blockchain interface {
	InsertBlockAtHeight(block.Height, block.Block)
	BlockAtHeight(block.Height) (block.Block, bool)
	BlockExistsAtHeight(block.Height) bool
	LatestBaseBlock() block.Block
}

// A SaveRestorer defines a storage interface for the State.
type SaveRestorer interface {
	Save(*State)
	Restore(*State)
}

// A Proposer builds a `block.Block` for proposals.
type Proposer interface {
	BlockProposal(block.Height, block.Round) block.Block
}

// A Validator validates a `block.Block` that has been proposed.
//
// When the Validator decides that a `block.Block` is invalid, it must provide
// `NilReasons` (that is, reasons explaining why the `block.Block` is invalid).
// This is expected to be used for debugging, but it can also be used to help
// the proposer "prune" invalid transactions from future `Propose` messages
// (when `DidReceiveSufficientNilPrevotes` triggers). This can be useful in
// fast-forwarding, because a proposer might have missed a `block.Block` and be
// attempting to `Propose` transactions that are already committed in that
// `block.Block`.
//
// The `checkHistory` argument tells the Validator whether, or not, it must also
// check the parental history of the `block.Block` (that is, are all parent
// blocks present and known to be valid too). This is important, because when
// fast-forwarding, it is (by definition) not required to check all parental
// history. An implementor can "disable" fast-forwarding by also validating as
// if `checkHistory` was true.
type Validator interface {
	IsBlockValid(block block.Block, checkHistory bool) (NilReasons, error)
}

// An Observer is notified when note-worthy events happen for the first time.
type Observer interface {
	DidCommitBlock(block.Height)
	DidReceiveSufficientNilPrevotes(messages Messages, f int)
}

// A Broadcaster is used to send signed, shard-specific, Messages to one or all
// Replicas in the network.
//
// For the consensus algorithm to work correctly, it is assumed that all honest
// processes will eventually deliver all messages to all other honest processes.
// The specific message ordering is not important. In practice, the Prevote
// messages are the only messages that must guarantee delivery when guaranteeing
// correctness.
//
// For crash-resilience, the implementation of Broadcast should ensure that two
// different Propose messages are not broadcast for the same shard, height, and
// round. This can be done by storing Proposes (and checking for their
// existence) before sending the Propose to the network proper.
type Broadcaster interface {
	Broadcast(Message)
	Cast(id.Signatory, Message)
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
	logger logrus.FieldLogger
	mu     *sync.Mutex

	signatory  id.Signatory
	blockchain Blockchain
	state      State

	saveRestorer SaveRestorer
	proposer     Proposer
	validator    Validator
	scheduler    schedule.Scheduler
	broadcaster  Broadcaster
	timer        Timer
	observer     Observer
	catcher      Catcher

	shard Shard
}

// New Process initialised to the default state, starting in the first round.
func New(logger logrus.FieldLogger, signatory id.Signatory, blockchain Blockchain, state State, saveRestorer SaveRestorer, proposer Proposer, validator Validator, observer Observer, broadcaster Broadcaster, scheduler schedule.Scheduler, timer Timer, catcher Catcher, shard Shard) *Process {
	p := &Process{
		logger: logger,
		mu:     new(sync.Mutex),

		signatory:  signatory,
		blockchain: blockchain,
		state:      state,

		saveRestorer: saveRestorer,
		proposer:     proposer,
		validator:    validator,
		scheduler:    scheduler,
		broadcaster:  broadcaster,
		timer:        timer,
		observer:     observer,
		catcher:      catcher,
		shard:        shard,
	}
	return p
}

// CurrentHeight of the Process.
func (p *Process) CurrentHeight() block.Height {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.state.CurrentHeight
}

// Save the current state of the process using the saveRestorer.
func (p *Process) Save() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.saveRestorer.Save(&p.state)
}

// Restore the current state of the process using the saveRestorer.
func (p *Process) Restore() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.saveRestorer.Restore(&p.state)
}

// SizeHint returns the number of bytes required to store this process in
// binary.
func (p *Process) SizeHint() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.state.SizeHint()
}

// Marshal the process into binary.
func (p *Process) Marshal(w io.Writer, m int) (int, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.state.Marshal(w, m)
}

// Unmarshal into this process from binary.
func (p *Process) Unmarshal(r io.Reader, m int) (int, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.state.Unmarshal(r, m)
}

// Start the process.
func (p *Process) Start() {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Log the starting state of process for debugging purpose.
	p.logger.Debugf("🎰 starting process at height=%v, round=%v, step=%v", p.state.CurrentHeight, p.state.CurrentRound, p.state.CurrentStep)
	numProposes := p.state.Proposals.QueryByHeightRound(p.state.CurrentHeight, p.state.CurrentRound)
	numPrevotes := p.state.Prevotes.QueryByHeightRound(p.state.CurrentHeight, p.state.CurrentRound)
	numPrecommits := p.state.Precommits.QueryByHeightRound(p.state.CurrentHeight, p.state.CurrentRound)
	p.logger.Debugf("propose inbox len=%v, prevote inbox len=%v, precommit inbox len=%v", numProposes, numPrevotes, numPrecommits)

	// Resend our latest messages to others.
	p.resendLatestMessages(nil)

	// Query others for previous messages.
	resync := NewResync(p.shard, p.state.CurrentHeight, p.state.CurrentRound)
	p.broadcaster.Broadcast(resync)

	// Start the Process from previous state.
	if p.state.CurrentStep == StepNil || p.state.CurrentStep == StepPropose {
		p.startRound(p.state.CurrentRound)
	}
	if numPrevotes >= 2*p.state.Prevotes.f+1 && p.state.CurrentStep == StepPrevote {
		p.scheduleTimeoutPrevote(p.state.CurrentHeight, p.state.CurrentRound, p.timer.Timeout(StepPrevote, p.state.CurrentRound))
	}
	if numPrecommits >= 2*p.state.Precommits.f+1 {
		p.scheduleTimeoutPrecommit(p.state.CurrentHeight, p.state.CurrentRound, p.timer.Timeout(StepPrecommit, p.state.CurrentRound))
	}
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
	case *Resync:
		p.handleResync(m)
	}
}

// resend sends any messages stored at the given height and round to the `to`
// signatory. If no signatory is provided, we broadcast the message to all known
// peers.
func (p *Process) resend(to *id.Signatory, height block.Height, round block.Round) {
	proposal := p.state.Proposals.QueryByHeightRoundSignatory(height, round, p.signatory)
	prevote := p.state.Prevotes.QueryByHeightRoundSignatory(height, round, p.signatory)
	precommit := p.state.Precommits.QueryByHeightRoundSignatory(height, round, p.signatory)
	if proposal != nil {
		// Resend messages to all peers if no signatory is provided.
		if to == nil {
			p.broadcaster.Broadcast(proposal)
		} else {
			p.broadcaster.Cast(*to, proposal)
		}
	}
	if prevote != nil {
		if to == nil {
			p.broadcaster.Broadcast(prevote)
		} else {
			p.broadcaster.Cast(*to, prevote)
		}
	}
	if precommit != nil {
		if to == nil {
			p.broadcaster.Broadcast(precommit)
		} else {
			p.broadcaster.Cast(*to, precommit)
		}
	}
}

func (p *Process) resendLatestMessages(to *id.Signatory) {
	if !p.state.Equal(DefaultState(p.state.Prevotes.f)) {
		p.logger.Debugf("resending messages at current height=%v and current round=%v", p.state.CurrentHeight, p.state.CurrentRound)
		p.resend(to, p.state.CurrentHeight, p.state.CurrentRound)
		if p.state.CurrentRound > 0 {
			p.logger.Debugf("resending messages at current height=%v and previous round=%v", p.state.CurrentHeight, p.state.CurrentRound-1)
			p.resend(to, p.state.CurrentHeight, p.state.CurrentRound-1)
		} else if p.state.CurrentHeight > 0 {
			maxRound := block.Round(0)
			for round := range p.state.Proposals.messages[p.state.CurrentHeight-1] {
				if round > maxRound {
					maxRound = round
				}
			}
			p.logger.Debugf("resending messages at previous height=%v and previous round=%v", p.state.CurrentHeight-1, maxRound)
			p.resend(to, p.state.CurrentHeight-1, maxRound)
		}
	}
}

func (p *Process) startRound(round block.Round) {
	p.state.CurrentRound = round
	p.state.CurrentStep = StepPropose

	// If process p is the proposer.
	if p.signatory.Equal(p.scheduler.Schedule(p.state.CurrentHeight, p.state.CurrentRound)) {
		var proposal block.Block
		if p.state.ValidBlock.Hash() != block.InvalidHash {
			proposal = p.state.ValidBlock
		} else {
			proposal = p.proposer.BlockProposal(p.state.CurrentHeight, p.state.CurrentRound)
		}
		propose := NewPropose(
			p.shard,
			p.state.CurrentHeight,
			p.state.CurrentRound,
			proposal,
			p.state.ValidRound,
		)

		// Include the previous block for nodes to catch up
		previousBlock, ok := p.blockchain.BlockAtHeight(p.state.CurrentHeight - 1)
		if !ok {
			panic("fail to get previous block from storage")
		}
		messages := p.state.Precommits.QueryMessagesByHeightWithHighestRound(p.state.CurrentHeight - 1)
		commits := make([]Precommit, 0, 2*p.state.Precommits.F()+1)
		for _, message := range messages {
			commit := message.(*Precommit)
			if commit.blockHash.Equal(previousBlock.Hash()) {
				commits = append(commits, *commit)
				if len(commits) >= 2*p.state.Precommits.F()+1 {
					// Restrict the len of commits to 2F+1, as is expected by
					// the nodes that will be receiving this message.
					break
				}
			}
		}
		if len(commits) < 2*p.state.Precommits.F()+1 {
			commits = []Precommit{}
		}
		propose.latestCommit = LatestCommit{
			Block:      previousBlock,
			Precommits: commits,
		}
		p.logger.Infof("🔊 proposed block=%v at height=%v and round=%v", propose.BlockHash(), propose.height, propose.round)
		p.broadcaster.Broadcast(propose)
	} else {
		p.scheduleTimeoutPropose(p.state.CurrentHeight, p.state.CurrentRound, p.timer.Timeout(StepPropose, p.state.CurrentRound))
	}
}

func (p *Process) handlePropose(propose *Propose) {
	// Before inserting the Propose, we need to check whether or not the Propose
	// is from the scheduled Proposer. Otherwise, we can safely ignore it.
	var firstTime bool
	var conflicting Message
	if propose.Signatory().Equal(p.scheduler.Schedule(propose.Height(), propose.Round())) {
		p.logger.Debugf("received propose at height=%v and round=%v", propose.height, propose.round)
		_, firstTime, _, _, _, conflicting = p.state.Proposals.Insert(propose)
		if conflicting != nil && p.catcher != nil {
			p.catcher.DidReceiveMessageConflict(conflicting, propose)
		}
	} else {
		// Ignore out-of-turn Proposes.
		p.logger.Warnf("received propose at height=%v and round=%v from out-of-turn proposer=%v", propose.height, propose.round, propose.signatory)
		return
	}

	p.syncLatestCommit(propose.latestCommit)

	// upon Propose{currentHeight, currentRound, block, -1}
	if propose.Height() == p.state.CurrentHeight && propose.Round() == p.state.CurrentRound && propose.ValidRound() == block.InvalidRound {
		// from Schedule{currentHeight, currentRound}
		if propose.Signatory().Equal(p.scheduler.Schedule(p.state.CurrentHeight, p.state.CurrentRound)) {
			// while currentStep = StepPropose
			if p.state.CurrentStep == StepPropose {
				var prevote *Prevote
				nilReasons, err := p.validator.IsBlockValid(propose.Block(), true)
				if err == nil && (p.state.LockedRound == block.InvalidRound || p.state.LockedBlock.Equal(propose.Block())) {
					prevote = NewPrevote(
						p.shard,
						p.state.CurrentHeight,
						p.state.CurrentRound,
						propose.Block().Hash(),
						nilReasons,
					)
					p.logger.Debugf("prevoted=%v at height=%v and round=%v", propose.BlockHash(), propose.height, propose.round)
				} else {
					prevote = NewPrevote(
						p.shard,
						p.state.CurrentHeight,
						p.state.CurrentRound,
						block.InvalidHash,
						nilReasons,
					)
					p.logger.Warnf("prevoted=<nil> at height=%v and round=%v (invalid propose: %v)", propose.height, propose.round, err)
				}
				p.state.CurrentStep = StepPrevote
				p.broadcaster.Broadcast(prevote)
			}
		}
	}

	// Resend our prevote from the valid round if it exists in case of missed
	// messages.
	if propose.ValidRound() > block.InvalidRound {
		prevote := p.state.Prevotes.QueryByHeightRoundSignatory(propose.Height(), propose.ValidRound(), p.signatory)
		if prevote != nil {
			p.broadcaster.Broadcast(prevote)
		}
	}

	// upon f+1 *{currentHeight, round, *, *} and round > currentRound
	n := p.numberOfMessagesAtCurrentHeight(propose.Round())
	if n > p.state.Prevotes.F() && propose.Height() == p.state.CurrentHeight && propose.Round() > p.state.CurrentRound {
		p.startRound(propose.Round())
	}

	if propose.Height() == p.state.CurrentHeight {
		if propose.Round() == p.state.CurrentRound {
			// These conditions can only be true when the Propose was for the
			// current height and round, so we only call them if the Propose was
			// in fact for the current height and round.
			p.checkProposeInCurrentHeightAndRoundWithPrevotes()
			if firstTime {
				p.checkProposeInCurrentHeightAndRoundWithPrevotesForTheFirstTime()
			}
		}
		// This condition can only be true when the Propose was for the
		// current height, so we only call it if the Propose was in fact for
		// the current height.
		p.checkProposeInCurrentHeightWithPrecommits(propose.Round())
	}
}

func (p *Process) handlePrevote(prevote *Prevote) {
	prevoteDebugStr := "<nil>"
	if !prevote.blockHash.Equal(block.InvalidHash) {
		prevoteDebugStr = prevote.blockHash.String()
	}
	p.logger.Debugf("received prevote=%v at height=%v and round=%v", prevoteDebugStr, prevote.height, prevote.round)
	_, _, _, firstTimeExceeding2F, firstTimeExceeding2FOnBlockHash, conflicting := p.state.Prevotes.Insert(prevote)
	if conflicting != nil && p.catcher != nil {
		p.catcher.DidReceiveMessageConflict(conflicting, prevote)
	}
	if firstTimeExceeding2F && prevote.Height() == p.state.CurrentHeight && prevote.Round() == p.state.CurrentRound && p.state.CurrentStep == StepPrevote {
		// upon 2f+1 Prevote{currentHeight, currentRound, *} while step = StepPrevote for the first time
		p.scheduleTimeoutPrevote(p.state.CurrentHeight, p.state.CurrentRound, p.timer.Timeout(StepPrevote, p.state.CurrentRound))
	}

	// upon f+1 Prevote{currentHeight, currentRound, nil}
	if n := p.state.Prevotes.QueryByHeightRoundBlockHash(p.state.CurrentHeight, p.state.CurrentRound, block.InvalidHash); n > p.state.Prevotes.F() {
		// if we are the proposer
		if p.signatory.Equal(p.scheduler.Schedule(p.state.CurrentHeight, p.state.CurrentRound)) {
			p.observer.DidReceiveSufficientNilPrevotes(p.state.Prevotes.QueryMessagesByHeightRound(p.state.CurrentHeight, p.state.CurrentRound), p.state.Prevotes.F())
		}
	}

	// upon 2f+1 Prevote{currentHeight, currentRound, nil} while currentStep = StepPrevote
	if n := p.state.Prevotes.QueryByHeightRoundBlockHash(p.state.CurrentHeight, p.state.CurrentRound, block.InvalidHash); n > 2*p.state.Prevotes.F() && p.state.CurrentStep == StepPrevote {
		precommit := NewPrecommit(
			p.shard,
			p.state.CurrentHeight,
			p.state.CurrentRound,
			block.InvalidHash,
		)
		p.logger.Debugf("precommited=<nil> at height=%v and round=%v (2f+1 prevote=<nil>)", precommit.height, precommit.round)
		p.state.CurrentStep = StepPrecommit
		p.broadcaster.Broadcast(precommit)
	}

	// upon f+1 *{currentHeight, round, *, *} and round > currentRound
	n := p.numberOfMessagesAtCurrentHeight(prevote.Round())
	if n > p.state.Prevotes.F() && prevote.Height() == p.state.CurrentHeight && prevote.Round() > p.state.CurrentRound {
		p.startRound(prevote.Round())
	}

	if prevote.Height() == p.state.CurrentHeight {
		// These conditions can only be true when the Prevote was for the
		// current height (not necessarily the current round), so we only call
		// them if the Prevote was in fact for the current height and round.
		p.checkProposeInCurrentHeightAndRoundWithPrevotes()
		if prevote.Round() == p.state.CurrentRound && firstTimeExceeding2FOnBlockHash {
			p.checkProposeInCurrentHeightAndRoundWithPrevotesForTheFirstTime()
		}
	}
}

func (p *Process) handlePrecommit(precommit *Precommit) {
	precommitDebugStr := "<nil>"
	if !precommit.blockHash.Equal(block.InvalidHash) {
		precommitDebugStr = precommit.blockHash.String()
	}
	p.logger.Debugf("received precommit=%v at height=%v and round=%v", precommitDebugStr, precommit.height, precommit.round)
	// upon 2f+1 Precommit{currentHeight, currentRound, *} for the first time
	_, _, _, firstTimeExceeding2F, _, conflicting := p.state.Precommits.Insert(precommit)
	if conflicting != nil && p.catcher != nil {
		p.catcher.DidReceiveMessageConflict(conflicting, precommit)
	}
	if firstTimeExceeding2F && precommit.Height() == p.state.CurrentHeight && precommit.Round() == p.state.CurrentRound {
		p.scheduleTimeoutPrecommit(p.state.CurrentHeight, p.state.CurrentRound, p.timer.Timeout(StepPrecommit, p.state.CurrentRound))
	}

	// upon f+1 *{currentHeight, round, *, *} and round > currentRound
	n := p.numberOfMessagesAtCurrentHeight(precommit.Round())
	if n > p.state.Precommits.F() && precommit.Height() == p.state.CurrentHeight && precommit.Round() > p.state.CurrentRound {
		p.startRound(precommit.Round())
	}

	if precommit.Height() == p.state.CurrentHeight {
		// This condition can only be true when the Precommit was for the
		// current height, so we only call it if the Precommit was in fact for
		// the current height.
		p.checkProposeInCurrentHeightWithPrecommits(precommit.Round())
	}
}

func (p *Process) handleResync(resync *Resync) {
	p.logger.Debugf("received resync at height=%v and round=%v", resync.height, resync.round)

	// If we need to resend messages after a height and round that are ahead of
	// us, then there is nothing to resend.
	if p.state.CurrentHeight < resync.height {
		return
	}
	if p.state.CurrentHeight == resync.height && p.state.CurrentRound < resync.round {
		return
	}

	// Resend our latest messages to the requestor.
	p.resendLatestMessages(&resync.signatory)
}

// timeoutPropose checks if we have move to a new height, a new round or a new
// step after the timeout. If not, prevote for a invalid block and broadcast
// the vote, then move to prevote step.
func (p *Process) timeoutPropose(height block.Height, round block.Round) {
	if height == p.state.CurrentHeight && round == p.state.CurrentRound && p.state.CurrentStep == StepPropose {
		prevote := NewPrevote(
			p.shard,
			p.state.CurrentHeight,
			p.state.CurrentRound,
			block.InvalidHash,
			nil,
		)
		p.logger.Warnf("prevoted=<nil> at height=%v and round=%v (timeout)", prevote.height, prevote.round)
		p.state.CurrentStep = StepPrevote
		p.broadcaster.Broadcast(prevote)
	}
}

func (p *Process) timeoutPrevote(height block.Height, round block.Round) {
	if height == p.state.CurrentHeight && round == p.state.CurrentRound && p.state.CurrentStep == StepPrevote {
		precommit := NewPrecommit(
			p.shard,
			p.state.CurrentHeight,
			p.state.CurrentRound,
			block.InvalidHash,
		)
		p.logger.Warnf("precommitted=<nil> at height=%v and round=%v (timeout)", precommit.height, precommit.round)
		p.state.CurrentStep = StepPrecommit
		p.broadcaster.Broadcast(precommit)
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

// checkProposeInCurrentHeightAndRoundWithPrevotes must only be called when a
// Propose or Prevote has been seen for the first time, and it is possible that
// a Propose and 2f+1 Prevotes have been seen where the Propose is at the
// current `block.Height` and `block.Round`, and the 2f+1 Prevotes are at the
// current `block.Height` and valid `block.Round` of the Propose. This can
// happen when a Propose is seen for the first time at the current
// `block.Height` and `block.Round`, or, when a Prevote is seen for the first
// time at the current `block.Height` and any `block.Round`. It is ok to call
// this function multiple times.
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
				var prevote *Prevote
				nilReasons, err := p.validator.IsBlockValid(propose.Block(), true)
				if err == nil && (p.state.LockedRound <= propose.ValidRound() || p.state.LockedBlock.Equal(propose.Block())) {
					prevote = NewPrevote(
						p.shard,
						p.state.CurrentHeight,
						p.state.CurrentRound,
						propose.Block().Hash(),
						nilReasons,
					)
					p.logger.Debugf("prevoted=%v at height=%v and round=%v (2f+1 valid prevotes)", prevote.blockHash, prevote.height, prevote.round)
				} else {
					prevote = NewPrevote(
						p.shard,
						p.state.CurrentHeight,
						p.state.CurrentRound,
						block.InvalidHash,
						nilReasons,
					)
					p.logger.Warnf("prevoted=<nil> at height=%v and round=%v (invalid propose: %v)", prevote.height, prevote.round, err)
				}

				p.state.CurrentStep = StepPrevote
				p.broadcaster.Broadcast(prevote)
			}
		}
	}
}

// checkProposeInCurrentHeightAndRoundWithPrevotesForTheFirstTime checks and
// reacts to a Propose and Prevote 2f+1 Prevotes having been seen for the first
// time at the current `block.Height` and `block.Round`. This can happen when a
// Propose is seen for the first time at the current `block.Height` and
// `block.Round`, or, when a Prevote is seen for the first time at the current
// `block.Height` and `block.Round`. This function can be called multiple times
// pre-emptively (when it is not yet the case that a Propose and 2f+1 Prevotes
// has been seen for the first time at the current `block.Height` and
// `block.Round`), but it must only be called once when the condition is true.
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
		_, err := p.validator.IsBlockValid(propose.Block(), true)
		if p.state.CurrentStep >= StepPrevote && err == nil {
			p.state.ValidBlock = propose.Block()
			p.state.ValidRound = p.state.CurrentRound
			if p.state.CurrentStep == StepPrevote {
				p.state.LockedBlock = propose.Block()
				p.state.LockedRound = p.state.CurrentRound
				p.state.CurrentStep = StepPrecommit
				precommit := NewPrecommit(
					p.shard,
					p.state.CurrentHeight,
					p.state.CurrentRound,
					propose.Block().Hash(),
				)
				p.logger.Debugf("precommitted=%v at height=%v and round=%v", precommit.blockHash, p.state.CurrentHeight, p.state.CurrentRound)
				p.broadcaster.Broadcast(precommit)
			}
		} else {
			p.logger.Warnf("nothing precommitted at height=%v, round=%v and step=%v (invalid block: %v)", propose.height, propose.round, p.state.CurrentStep, err)
		}
	}
}

// checkProposeInCurrentHeightWithPrecommits must only be called when a Propose
// or Precommit has been seen for the first time, and it is possible that a
// Propose and 2f+1 Precommits have been seen where the Propose is at the
// current `block.Height` and any `block.Round`, and the 2f+1 Precommits are at
// the current `block.Height` and the as `block.Round` as the Propose. This can
// happen when a Propose is seen for the first time at the current
// `block.Height`, or when a Precommit is seen for the first time at the current
// `block.Height`. It is ok to call this function multiple times.
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
			_, err := p.validator.IsBlockValid(propose.Block(), true)
			if err == nil {
				p.blockchain.InsertBlockAtHeight(p.state.CurrentHeight, propose.Block())
				p.state.CurrentHeight++
				p.state.Reset(p.state.CurrentHeight - 1)
				if p.observer != nil {
					p.observer.DidCommitBlock(p.state.CurrentHeight - 1)
				}
				p.logger.Infof("✅ committed block=%v at height=%v", propose.BlockHash(), propose.height)
				p.startRound(0)

				// If we just committed a base block, then we need to
				// resynchronise with other nodes, in case we have dropped
				// Proposes from this new base that arrived bfeore the new base.
				if propose.Block().Header().Kind() == block.Base {
					p.broadcaster.Broadcast(NewResync(p.shard, p.state.CurrentHeight, p.state.CurrentRound))
				}
			} else {
				p.logger.Warnf("nothing committed at height=%v and round=%v (invalid block: %v)", propose.height, propose.round, err)
			}
		}
	}
}

func (p *Process) numberOfMessagesAtCurrentHeight(round block.Round) int {
	numUniqueProposals := p.state.Proposals.QueryByHeightRound(p.state.CurrentHeight, round)
	numUniquePrevotes := p.state.Prevotes.QueryByHeightRound(p.state.CurrentHeight, round)
	numUniquePrecommits := p.state.Precommits.QueryByHeightRound(p.state.CurrentHeight, round)
	return numUniqueProposals + numUniquePrevotes + numUniquePrecommits
}

func (p *Process) syncLatestCommit(latestCommit LatestCommit) {
	// Check that they have not included too many signatories. This is required
	// to protect against DoS attacks performed by including a massive number of
	// Precommits.
	if latestCommit.Precommits == nil || len(latestCommit.Precommits) != 2*p.state.Precommits.F()+1 {
		return
	}

	// Check that the latest commit is from the future.
	if latestCommit.Block.Header().Height() <= p.state.CurrentHeight {
		return
	}

	// Check the proposed block and previous block without historical data. It
	// needs the validator to store the previous execute state.
	_, err := p.validator.IsBlockValid(latestCommit.Block, false)
	if err != nil {
		p.logger.Warnf("error syncing to height=%v and round=%v (invalid block: %v)", latestCommit.Block.Header().Height(), latestCommit.Block.Header().Round(), err)
		return
	}

	// Validate the commits
	signatories := map[id.Signatory]struct{}{}
	baseBlock := p.blockchain.LatestBaseBlock()
	for _, sig := range baseBlock.Header().Signatories() {
		signatories[sig] = struct{}{}
	}
	for _, commit := range latestCommit.Precommits {
		if err := Verify(&commit); err != nil {
			return
		}
		if _, ok := signatories[commit.signatory]; !ok {
			return
		}
		if !commit.blockHash.Equal(latestCommit.Block.Hash()) {
			return
		}
		if commit.height != latestCommit.Block.Header().Height() {
			return
		}
		if commit.round != latestCommit.Block.Header().Round() {
			return
		}
		if !commit.shard.Equal(p.shard) {
			return
		}
	}

	// Check we have 2f+1 distinct commits
	signatories = map[id.Signatory]struct{}{}
	for _, commit := range latestCommit.Precommits {
		signatories[commit.Signatory()] = struct{}{}
	}
	if len(signatories) < 2*p.state.Proposals.f+1 {
		return
	}

	// if the commits are valid, store the block if we don't have one
	if !p.blockchain.BlockExistsAtHeight(latestCommit.Block.Header().Height()) {
		p.blockchain.InsertBlockAtHeight(latestCommit.Block.Header().Height(), latestCommit.Block)
	}
	if p.observer != nil {
		p.observer.DidCommitBlock(latestCommit.Block.Header().Height())
	}
	p.logger.Infof("syncing from height=%v to height=%v", p.state.CurrentHeight, latestCommit.Block.Header().Height()+1)
	p.state.CurrentHeight = latestCommit.Block.Header().Height() + 1
	p.state.CurrentRound = 0
	p.state.Reset(latestCommit.Block.Header().Height())
	p.startRound(p.state.CurrentRound)
}
