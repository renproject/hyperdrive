package state

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/renproject/hyperdrive/block"
	"github.com/renproject/hyperdrive/shard"
	"github.com/renproject/hyperdrive/sig"
	"github.com/renproject/hyperdrive/tx"
)

// NumTicksToTriggerTimeOut specifies the maximum number of Ticks to wait before
// triggering a TimedOut  transition.
const NumTicksToTriggerTimeOut = 2

// A Machine is a deterministic state machine that is used to implement all
// states and transitions required to participate in a BFT replicated state
// machine using the Tendermint consensus algorithm. For more information, see
// https://arxiv.org/pdf/1807.04938.pdf.
type Machine interface {

	// State returns the current state of the Machine. Generally, the Machine
	// will be waiting for a Propose, waiting for a Polka, or waiting for a
	// Commit.
	State() State

	// Height returns the current height on which the Machine is trying to reach
	// consensus on a new Block.
	Height() block.Height

	// Round returns the current round in which the Machine is trying tp reach
	// consensus on a new Block.
	Round() block.Round

	// LastBlock returns the last Block to which the Machine has committed. If
	// no such Block exists, the genesis block is returned.
	LastBlock() block.SignedBlock

	// StartRound is called once when the StateMachine starts operating and
	// everytime a round is completed. Actions returned by StartRound are
	// expected to be dispatched to all other Replicas in the system. If the
	// action returned by StartRound is not nil, it must be sent back to the
	// same StateMachine for it to progress.
	StartRound(round block.Round, commit *block.Commit) Action

	// Commit the Machine to a Block that has already been committed to by the
	// BFT replicated state machine.
	Commit(commit block.Commit)

	// Transition the Machine from one state to another. The state might not
	// actually change, depending on the Transition and the current state of the
	// Machine.
	Transition(transition Transition) Action
}

type machine struct {
	currentState  State
	currentHeight block.Height
	currentRound  block.Round

	lockedRound block.Round
	lockedValue *block.SignedBlock
	validRound  block.Round
	validValue  *block.SignedBlock

	lastCommit *block.Commit

	polkaBuilder  block.PolkaBuilder
	commitBuilder block.CommitBuilder

	signer sig.Signer
	shard  shard.Shard
	txPool tx.Pool

	proposeTimer   timer
	preVoteTimer   timer
	preCommitTimer timer

	bufferedMessages map[block.Round]map[sig.Signatory]struct{}
}

func NewMachine(state State, polkaBuilder block.PolkaBuilder, commitBuilder block.CommitBuilder, signer sig.Signer, shard shard.Shard, txPool tx.Pool, lastCommit *block.Commit) Machine {
	machine := machine{
		currentState: state,
		currentRound: 0,

		lockedRound: -1,
		lockedValue: nil,
		validRound:  -1,
		validValue:  nil,

		lastCommit: lastCommit,

		polkaBuilder:  polkaBuilder,
		commitBuilder: commitBuilder,

		signer: signer,
		shard:  shard,
		txPool: txPool,

		proposeTimer:   NewTimer(NumTicksToTriggerTimeOut),
		preVoteTimer:   NewTimer(NumTicksToTriggerTimeOut),
		preCommitTimer: NewTimer(NumTicksToTriggerTimeOut),

		bufferedMessages: map[block.Round]map[sig.Signatory]struct{}{},
	}

	machine.currentHeight = 0
	if lastCommit != nil {
		machine.currentHeight = lastCommit.Polka.Height + 1
	}

	return &machine
}

func (machine *machine) State() State {
	return machine.currentState
}

func (machine *machine) Height() block.Height {
	return machine.currentHeight
}

func (machine *machine) Round() block.Round {
	return machine.currentRound
}

func (machine *machine) LastBlock() block.SignedBlock {
	if machine.lastCommit != nil {
		if machine.lastCommit.Polka.Block == nil {
			panic("invariant violated: nil block committed")
		}
		return *machine.lastCommit.Polka.Block
	}
	return block.Genesis()
}

func (machine *machine) StartRound(round block.Round, lastCommit *block.Commit) Action {
	machine.currentRound = round
	machine.currentState = WaitingForPropose{}

	machine.resetTimersOnNewRound()

	if round == 0 {
		machine.bufferedMessages = map[block.Round]map[sig.Signatory]struct{}{}
	}

	committed := Commit{}
	if lastCommit != nil {
		machine.lastCommit = lastCommit
		committed = Commit{
			Commit: *lastCommit,
		}
	}

	if machine.shouldProposeBlock() {
		// Introduce an artificial random delay of maximum 5 seconds
		time.Sleep(time.Duration(rand.Intn(5)) * time.Second)

		signedBlock := block.SignedBlock{}
		if machine.validValue != nil {
			signedBlock = *machine.validValue
		} else {
			signedBlock = machine.buildSignedBlock()
		}

		if signedBlock.Height != machine.currentHeight {
			panic("unexpected block")
		}

		propose := block.Propose{
			Block:      signedBlock,
			Round:      round,
			ValidRound: machine.validRound,
			LastCommit: machine.lastCommit,
		}

		signedPropose, err := propose.Sign(machine.signer)
		if err != nil {
			// FIXME: Should an error cause a panic? I am confident that there
			// is not a sane alternative, and that letting the state Machine
			// panic (and therefore reboot), is reasonable. One potential
			// alternative is logging the error and returning nil. This will
			// cause the state Machine to not need to reboot and the network
			// will naturally progress to the next round.
			panic(err)
		}

		return Propose{
			SignedPropose: signedPropose,
			LastCommit:    committed,
		}
	}

	if len(committed.Signatures) > 0 {
		return committed
	}
	return nil
}

func (machine *machine) Commit(commit block.Commit) {
	if commit.Polka.Height >= machine.currentHeight {
		machine.currentState = WaitingForPropose{}
		machine.currentHeight = commit.Polka.Height + 1
		machine.currentRound = 0
		machine.lockedValue = nil
		machine.lockedRound = -1
		machine.validValue = nil
		machine.validRound = -1
		machine.lastCommit = &commit
		machine.bufferedMessages = map[block.Round]map[sig.Signatory]struct{}{}
		machine.resetTimersOnNewRound()
		machine.drop()
	}
}

func (machine *machine) Transition(transition Transition) Action {
	// Check pre-conditions
	machine.preconditionCheck()

	// Handle messages for rounds greater than the current round. If there are f+1
	// transitions for a round higher than the current round, progress to the new round.
	if transition.Round() > machine.currentRound {
		if _, ok := machine.bufferedMessages[transition.Round()]; !ok {
			machine.bufferedMessages[transition.Round()] = map[sig.Signatory]struct{}{}
		}
		machine.bufferedMessages[transition.Round()][transition.Signer()] = struct{}{}
	}

	higherRound := machine.checkForHigherRounds()
	if higherRound > machine.currentRound {
		// Found f+1 messages for a higher round, progress to the new round.
		switch transition := transition.(type) {
		case PreVoted:
			// At this stage, we want to buffer all preVotes. It doesn't matter if
			// the preVote is new or not, because we want to start a new round regardless.
			_ = machine.polkaBuilder.Insert(transition.SignedPreVote)
		case PreCommitted:
			// At this stage, we want to buffer all preCommits. It doesn't matter if
			// the preCommit is new or not, because we want to start a new round regardless.
			_ = machine.commitBuilder.Insert(transition.SignedPreCommit)
		}
		return machine.StartRound(higherRound, nil)
	}

	// If a Proposal is received for a height higher than the currentHeight, progress
	// to new height if the attached commit is valid.
	if propose, ok := transition.(Proposed); ok {
		if propose.SignedPropose.Block.Height > machine.currentHeight {
			if propose.LastCommit == nil {
				return nil
			}
			machine.Commit(*propose.LastCommit)
			machine.currentRound = propose.Round()
		}
	}

	// Always check to see if the machine has received 2f+1 preCommits for the
	// current round and activate preCommitTimer if it has.
	machine.checkAndActivatePreCommitTimer()

	// Handle ticks
	if ticked, ok := transition.(Ticked); ok {
		if timeoutAction := machine.handleTick(ticked); timeoutAction != nil {
			return timeoutAction
		}
	}

	commit, _ := machine.commitBuilder.Commit(machine.currentHeight, machine.shard.ConsensusThreshold())
	if action := machine.handleCommit(commit); action != nil {
		return action
	}

	switch machine.currentState.(type) {
	case WaitingForPropose:
		return machine.waitForPropose(transition)
	case WaitingForPolka:
		return machine.waitForPolka(transition)
	case WaitingForCommit:
		return machine.waitForCommit(transition)
	default:
		panic(fmt.Errorf("unexpected state type %T", machine.currentState))
	}
}

func (machine *machine) waitForPropose(transition Transition) Action {
	switch transition := transition.(type) {
	case Proposed:
		// Precondition check: is transition for current round?
		if transition.Round() != machine.currentRound {
			panic("proposal round should be equal to currentRound of the state machine")
		}

		// Precondition check: is transition for current height?
		if transition.Block.Height != machine.currentHeight {
			panic("proposal height should be equal to currentHeight of the state machine")
		}

		// Precondition check: is proposer valid?
		if machine.shard.Leader(machine.currentRound).Equal(transition.Signatory) {
			if transition.ValidRound < 0 {
				// Reset propose timer and update state
				machine.proposeTimer.Reset()

				machine.currentState = WaitingForPolka{}

				// Broadcast PreVote
				if machine.lockedRound == -1 || machine.lockedValue.Block.Equal(transition.Block.Block) {
					return machine.broadcastPreVote(&transition.Block)
				}
				return machine.broadcastPreVote(nil)
			}
			if polka, polkaRound := machine.polkaBuilder.Polka(machine.currentHeight, machine.shard.ConsensusThreshold()); polkaRound != nil {
				if polka.Block != nil && polka.Block.Block.Equal(transition.Block.Block) && transition.ValidRound < machine.currentRound {
					// Reset propose timer and update state
					machine.proposeTimer.Reset()

					machine.currentState = WaitingForPolka{}

					// Broadcast PreVote
					if machine.lockedRound <= transition.ValidRound || machine.lockedValue.Block.Equal(transition.Block.Block) {
						return machine.broadcastPreVote(&transition.Block)
					}
					return machine.broadcastPreVote(nil)
				}
			}
		}

	case PreVoted:
		// Insert all preVotes. We explicitly ignore whether the preVote was already
		// added because we don't need that information at this stage.
		_ = machine.polkaBuilder.Insert(transition.SignedPreVote)

	case PreCommitted:
		// Insert all preCommits. If the preCommit was new information, check and update
		// the preCommit timer.
		if machine.commitBuilder.Insert(transition.SignedPreCommit) {
			machine.checkAndActivatePreCommitTimer()
			commit, _ := machine.commitBuilder.Commit(machine.currentHeight, machine.shard.ConsensusThreshold())
			return machine.handleCommit(commit)
		}

	case Ticked:
		// Ignore

	default:
		panic(fmt.Errorf("unexpected transition type %T", transition))
	}
	return nil
}

func (machine *machine) waitForPolka(transition Transition) Action {
	machine.checkAndActivatePreVoteTimer()
	polka := &block.Polka{}

	switch transition := transition.(type) {
	case Proposed:
		// Ignore all proposals at this stage.
		return nil

	case PreVoted:
		// If preVote received is not new information, return immediately.
		if !machine.polkaBuilder.Insert(transition.SignedPreVote) {
			return nil
		}

		var polkaRound *block.Round
		polka, polkaRound = machine.polkaBuilder.Polka(machine.currentHeight, machine.shard.ConsensusThreshold())
		if polkaRound != nil && *polkaRound == machine.currentRound && !machine.preVoteTimer.IsActive() {
			machine.activateTimerWithExpiry(&machine.preVoteTimer)
		}

	case PreCommitted:
		// Insert all preCommits. If the preCommit was new information, check and update
		// the preCommit timer.
		if machine.commitBuilder.Insert(transition.SignedPreCommit) {
			machine.checkAndActivatePreCommitTimer()
			commit, _ := machine.commitBuilder.Commit(machine.currentHeight, machine.shard.ConsensusThreshold())
			if action := machine.handleCommit(commit); action != nil {
				return action
			}
		}

		polka, _ = machine.polkaBuilder.Polka(machine.currentHeight, machine.shard.ConsensusThreshold())

	case Ticked:
		polka, _ = machine.polkaBuilder.Polka(machine.currentHeight, machine.shard.ConsensusThreshold())

	default:
		panic(fmt.Errorf("unexpected transition type %T", transition))
	}

	return machine.handlePolka(polka)
}

func (machine *machine) waitForCommit(transition Transition) Action {
	var commit *block.Commit

	switch transition := transition.(type) {
	case Proposed:
		// Retrieve commits for processing later. We ignore the round at which the commit was found
		// because that information is only needed by the state machine to activate preCommit timers,
		// something that should have already been completed prior to this stage.
		commit, _ = machine.commitBuilder.Commit(machine.currentHeight, machine.shard.ConsensusThreshold())

	case PreVoted:
		// Insert all preVotes. It doesn't matter to this stage if the preVote is not new.
		_ = machine.polkaBuilder.Insert(transition.SignedPreVote)

		// Retrieve commits for processing later. We ignore the round at which the commit was found
		// because that information is only needed by the state machine to activate preCommit timers,
		// something that should have already been completed prior to this stage.
		commit, _ = machine.commitBuilder.Commit(machine.currentHeight, machine.shard.ConsensusThreshold())

	case PreCommitted:
		// If preCommit received is not new information, return immediately.
		if !machine.commitBuilder.Insert(transition.SignedPreCommit) {
			return nil
		}

		// At this point, we have received a new preCommit. Here, we need to check if there are 2f+1
		// preCommits for the current height and activate the preCommit timer, if required.
		var commitRound *block.Round
		commit, commitRound = machine.commitBuilder.Commit(machine.currentHeight, machine.shard.ConsensusThreshold())
		if commitRound != nil && *commitRound == machine.currentRound && !machine.preCommitTimer.IsActive() {
			machine.activateTimerWithExpiry(&machine.preCommitTimer)
		}

	case Ticked:
		// Retrieve commit for processing later. We ignore the round at which the commit was found
		// because that information is only needed by the state machine to activate preCommit timers,
		// something that should have already been completed prior to this stage.
		commit, _ = machine.commitBuilder.Commit(machine.currentHeight, machine.shard.ConsensusThreshold())

	default:
		panic(fmt.Errorf("unexpected transition type %T", transition))
	}

	machine.updateValidBlockWithPolka()
	return machine.handleCommit(commit)
}

func (machine *machine) resetTimersOnNewRound() {
	machine.proposeTimer.Reset()
	machine.preVoteTimer.Reset()
	machine.preCommitTimer.Reset()

	machine.activateTimerWithExpiry(&machine.proposeTimer)
}

func (machine *machine) shouldProposeBlock() bool {
	return machine.signer.Signatory().Equal(machine.shard.Leader(machine.currentRound))
}

func (machine *machine) buildSignedBlock() block.SignedBlock {
	transactions := make(tx.Transactions, 0, block.MaxTransactions)
	for len(transactions) < block.MaxTransactions {
		transaction, ok := machine.txPool.Dequeue()
		if ok {
			transactions = append(transactions, transaction)
			continue
		}
		break
	}

	parentHeader := machine.LastBlock().Header
	block := block.New(
		machine.currentHeight,
		parentHeader,
		transactions,
	)
	signedBlock, err := block.Sign(machine.signer)
	if err != nil {
		// FIXME: Should an error cause a panic? I am confident that there is
		// not a sane alternative, and that letting the state Machine panic (and
		// therefore reboot), is reasonable. One potential alternative is
		// logging the error and returning nil. This will cause the state
		// Machine to not need to reboot and the network will naturally progress
		// to the next round.
		panic(err)
	}
	return signedBlock
}

func (machine *machine) preconditionCheck() {
	if machine.lockedRound < 0 {
		if machine.lockedValue != nil {
			panic("expected locked block to be nil")
		}
	}
	if machine.lockedRound >= 0 {
		if machine.lockedValue == nil {
			panic("expected locked block to not be nil")
		}
	}

	if machine.validRound < 0 {
		if machine.validValue != nil {
			panic("expected valid block to be nil")
		}
	}
	if machine.validRound >= 0 {
		if machine.validValue == nil {
			panic("expected valid block to not be nil")
		}
	}
}

func (machine *machine) handleTick(tick Ticked) Action {
	// Check for preCommit timeouts first to ensure the machine
	// doesn't have to progress rounds.
	if machine.preCommitTimer.Tick() {
		return machine.onTimeoutPrecommit()
	}
	if machine.proposeTimer.Tick() {
		return machine.onTimeoutPropose()
	}
	if machine.preVoteTimer.Tick() {
		return machine.onTimeoutPrevote()
	}
	return nil
}

func (machine *machine) broadcastPreVote(proposedBlock *block.SignedBlock) Action {
	preVote := block.PreVote{
		Block:  proposedBlock,
		Height: machine.currentHeight,
		Round:  machine.currentRound,
	}

	signedPrevote, err := preVote.Sign(machine.signer)
	if err != nil {
		panic(err)
	}
	machine.polkaBuilder.Insert(signedPrevote)

	return SignedPreVote{
		SignedPreVote: signedPrevote,
	}
}

func (machine *machine) broadcastPreCommit(polka block.Polka) Action {
	preCommit := block.PreCommit{
		Polka: polka,
	}

	signedPreCommit, err := preCommit.Sign(machine.signer)
	if err != nil {
		panic(err)
	}
	machine.commitBuilder.Insert(signedPreCommit)

	return SignedPreCommit{
		SignedPreCommit: signedPreCommit,
	}
}

func (machine *machine) checkForHigherRounds() block.Round {
	highestRound := machine.currentRound
	for round, sigMap := range machine.bufferedMessages {
		if round > highestRound && len(sigMap) >= machine.shard.ConsensusThreshold()/2 {
			highestRound = round
		}
	}
	if highestRound > machine.currentRound {
		for round := range machine.bufferedMessages {
			if round < highestRound {
				delete(machine.bufferedMessages, round)
			}
		}
	}
	return highestRound
}

func (machine *machine) checkAndActivatePreVoteTimer() {
	_, polkaRound := machine.polkaBuilder.Polka(machine.currentHeight, machine.shard.ConsensusThreshold())
	if polkaRound != nil && *polkaRound == machine.currentRound && !machine.preVoteTimer.IsActive() {
		machine.activateTimerWithExpiry(&machine.preVoteTimer)
	}
}

func (machine *machine) checkAndActivatePreCommitTimer() {
	_, commitRound := machine.commitBuilder.Commit(machine.currentHeight, machine.shard.ConsensusThreshold())
	if commitRound != nil && *commitRound == machine.currentRound && !machine.preCommitTimer.IsActive() {
		machine.activateTimerWithExpiry(&machine.preCommitTimer)
	}
}

func (machine *machine) handlePolka(polka *block.Polka) Action {
	if polka != nil && polka.Round == machine.currentRound {
		if polka.Block == nil {
			machine.preVoteTimer.Reset()
			machine.currentState = WaitingForCommit{}
			return machine.broadcastPreCommit(*polka)
		}
		machine.lockedRound = machine.currentRound
		machine.lockedValue = polka.Block
		machine.validRound = machine.currentRound
		machine.validValue = polka.Block
		machine.preVoteTimer.Reset()
		machine.currentState = WaitingForCommit{}
		return machine.broadcastPreCommit(*polka)
	}
	return nil
}

func (machine *machine) handleCommit(commit *block.Commit) Action {
	if commit != nil && commit.Polka.Round == machine.currentRound {
		if commit.Polka.Block != nil && machine.currentHeight <= commit.Polka.Height {
			machine.currentHeight = commit.Polka.Height + 1
			machine.drop()
			machine.lockedRound = -1
			machine.lockedValue = nil
			machine.validRound = -1
			machine.validValue = nil
			return machine.StartRound(0, commit)
		}
		return machine.StartRound(machine.currentRound+1, nil)
	}
	return nil
}

func (machine *machine) updateValidBlockWithPolka() {
	polka, _ := machine.polkaBuilder.Polka(machine.currentHeight, machine.shard.ConsensusThreshold())
	if polka != nil && polka.Round == machine.currentRound && polka.Block != nil {
		machine.validRound = machine.currentRound
		machine.validValue = polka.Block
	}
}

func (machine *machine) drop() {
	machine.polkaBuilder.Drop(machine.currentHeight)
	machine.commitBuilder.Drop(machine.currentHeight)
}

func (machine *machine) activateTimerWithExpiry(timer *timer) {
	timer.Reset()
	timer.Activate()
	timer.SetExpiry(int(NumTicksToTriggerTimeOut + machine.currentRound))
}

func (machine *machine) onTimeoutPropose() Action {
	if _, ok := machine.currentState.(WaitingForPropose); !ok {
		panic("expected state machine to timeoutPropose while waiting on propose")
	}

	machine.proposeTimer.Reset()
	machine.currentState = WaitingForPolka{}
	return machine.broadcastPreVote(nil)
}

func (machine *machine) onTimeoutPrevote() Action {
	if _, ok := machine.currentState.(WaitingForPolka); !ok {
		panic("expected state machine to timeoutPrevote while waiting on polka")
	}

	machine.preVoteTimer.Reset()

	machine.currentState = WaitingForCommit{}
	return machine.broadcastPreCommit(block.Polka{
		Round:  machine.currentRound,
		Height: machine.currentHeight,
	})
}

func (machine *machine) onTimeoutPrecommit() Action {
	return machine.StartRound(machine.currentRound+1, nil)
}
