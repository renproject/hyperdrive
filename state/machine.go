package state

import (
	"fmt"

	"github.com/renproject/hyperdrive/block"
	"github.com/renproject/hyperdrive/shard"
	"github.com/renproject/hyperdrive/sig"
	"github.com/renproject/hyperdrive/tx"
)

// NumTicksToTriggerTimeOut specifies the maximum number of Ticks to wait before
// triggering a TimedOut  transition.
const NumTicksToTriggerTimeOut = 2

type Machine interface {
	// StartRound is called once when the StateMachine starts operating and everytime a round is completed.
	// Actions returned by StartRound are expected to be dispatched to all other Replicas in the system.
	// If the action returned by StartRound is not nil, it must be sent back to the same StateMachine for it to progress.
	StartRound(round block.Round, commit *block.Commit) Action

	Transition(transition Transition) Action

	Height() block.Height
	Round() block.Round
	SyncCommit(commit block.Commit)
	Drop()
}

type machine struct {
	index         int
	currentState  State
	currentHeight block.Height
	currentRound  block.Round

	lockedRound block.Round
	lockedValue *block.SignedBlock
	validRound  block.Round
	validValue  *block.SignedBlock

	polkaBuilder       block.PolkaBuilder
	commitBuilder      block.CommitBuilder
	consensusThreshold int

	signer sig.Signer
	shard  shard.Shard
	txPool tx.Pool

	ticksAtProposeState   int
	ticksAtPrevoteState   int
	ticksAtPrecommitState int

	bufferedMessages map[block.Round]map[sig.Signatory]struct{}
}

func NewMachine(index int, state State, polkaBuilder block.PolkaBuilder, commitBuilder block.CommitBuilder, signer sig.Signer, shard shard.Shard, txPool tx.Pool, consensusThreshold int) Machine {
	return &machine{
		index:         index,
		currentState:  state,
		currentHeight: 0,
		currentRound:  0,

		lockedRound: -1,
		lockedValue: nil,
		validRound:  -1,
		validValue:  nil,

		polkaBuilder:       polkaBuilder,
		commitBuilder:      commitBuilder,
		consensusThreshold: consensusThreshold,

		signer: signer,
		shard:  shard,
		txPool: txPool,

		ticksAtProposeState:   -1,
		ticksAtPrevoteState:   -1,
		ticksAtPrecommitState: -1,

		bufferedMessages: map[block.Round]map[sig.Signatory]struct{}{},
	}
}

func (machine *machine) Height() block.Height {
	return machine.currentHeight
}

func (machine *machine) Round() block.Round {
	return machine.currentRound
}

func (machine *machine) StartRound(round block.Round, commit *block.Commit) Action {
	machine.currentRound = round
	machine.currentState = WaitingForPropose{}

	machine.ticksAtProposeState = 0
	machine.ticksAtPrevoteState = -1
	machine.ticksAtPrecommitState = -1
	if round == 0 {
		machine.bufferedMessages = map[block.Round]map[sig.Signatory]struct{}{}
	}

	if machine.shouldProposeBlock() {
		signedBlock := block.SignedBlock{}
		if machine.validValue != nil {
			signedBlock = *machine.validValue
		} else {
			signedBlock = machine.buildSignedBlock(commit)
		}

		if signedBlock.Height != machine.currentHeight {
			panic("unexpected block")
		}

		propose := block.Propose{
			Block:      signedBlock,
			Round:      round,
			ValidRound: machine.validRound,
		}

		signedPropose, err := propose.Sign(machine.signer)
		if err != nil {
			panic(err)
		}

		committed := Commit{}
		if commit != nil {
			committed = Commit{
				Commit: *commit,
			}
		}

		return Propose{
			SignedPropose: signedPropose,
			Commit:        committed,
		}
	}
	if commit != nil {
		return Commit{
			Commit: *commit,
		}
	}
	return nil
}

func (machine *machine) SyncCommit(commit block.Commit) {
	if commit.Polka.Height > machine.currentHeight {
		machine.currentState = WaitingForPropose{}
		machine.currentHeight = commit.Polka.Height + 1
		machine.currentRound = 0
		machine.lockedValue = nil
		machine.lockedRound = -1
	}
}

func (machine *machine) Drop() {
	machine.polkaBuilder.Drop(machine.currentHeight)
	machine.commitBuilder.Drop(machine.currentHeight)
}

func (machine *machine) Transition(transition Transition) Action {
	// Check pre-conditions
	if machine.lockedRound < 0 {
		if machine.lockedValue != nil {
			panic("expected locked block to be nil")
		}
	}
	if machine.lockedRound >= 0 {
		if machine.lockedValue == nil {
			panic("expected locked round to be nil")
		}
	}

	if machine.validRound < 0 {
		if machine.validValue != nil {
			panic("expected valid block to be nil")
		}
	}
	if machine.validRound >= 0 {
		if machine.validValue == nil {
			panic("expected valid round to be nil")
		}
	}

	if transition.Round() > machine.currentRound {
		if _, ok := machine.bufferedMessages[transition.Round()]; !ok {
			machine.bufferedMessages[transition.Round()] = map[sig.Signatory]struct{}{}
		}
		if _, ok := machine.bufferedMessages[transition.Round()][transition.Signer()]; !ok {
			machine.bufferedMessages[transition.Round()][transition.Signer()] = struct{}{}
		}
	}

	updateRound := machine.UpdateRound()
	if updateRound != nil && *updateRound > machine.currentRound {
		switch transition := transition.(type) {
		case PreVoted:
			_ = machine.polkaBuilder.Insert(transition.SignedPreVote)
		case PreCommitted:
			_ = machine.commitBuilder.Insert(transition.SignedPreCommit)
		}
		return machine.StartRound(*updateRound, nil)
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
		// TODO: Verify proposer is for the current round
		if transition.ValidRound < 0 {
			machine.ticksAtProposeState = -1
			machine.currentState = WaitingForPolka{}
			if machine.lockedRound == -1 || machine.lockedValue.Block.Equal(transition.Block.Block) {
				return machine.broadcastPreVote(&transition.Block)
			}
			return machine.broadcastPreVote(nil)
		}
		if _, polkaRound := machine.polkaBuilder.Polka(machine.currentHeight, machine.consensusThreshold); polkaRound != nil {
			if transition.ValidRound < machine.currentRound {
				machine.ticksAtProposeState = -1
				machine.currentState = WaitingForPolka{}
				if machine.lockedRound <= transition.ValidRound || machine.lockedValue.Block.Equal(transition.Block.Block) {
					return machine.broadcastPreVote(&transition.Block)
				}
				return machine.broadcastPreVote(nil)
			}
		}

	case PreVoted:
		machine.polkaBuilder.Insert(transition.SignedPreVote)

	case PreCommitted:
		machine.commitBuilder.Insert(transition.SignedPreCommit)

	case Ticked:
		machine.ticksAtProposeState++
		maxTicksToTimeout := int(NumTicksToTriggerTimeOut + machine.currentRound)
		if machine.ticksAtProposeState > maxTicksToTimeout {
			machine.ticksAtProposeState = -1
			machine.currentState = WaitingForPolka{}
			return machine.broadcastPreVote(nil)
		}

	default:
		panic(fmt.Errorf("unexpected transition type %T", transition))
	}

	return nil
}

func (machine *machine) waitForPolka(transition Transition) Action {
	machine.checkAndSchedulePreVoteTimeout()

	switch transition := transition.(type) {
	case Proposed:

	case PreVoted:
		if !machine.polkaBuilder.Insert(transition.SignedPreVote) {
			return nil
		}

		polka, polkaRound := machine.polkaBuilder.Polka(machine.currentHeight, machine.consensusThreshold)
		if polkaRound != nil && *polkaRound == machine.currentRound && machine.ticksAtPrevoteState < 0 {
			machine.ticksAtPrevoteState = 0
		}

		if polka != nil && polka.Round == machine.currentRound {
			if polka.Block == nil {
				machine.ticksAtPrevoteState = -1
				machine.currentState = WaitingForCommit{}
				return machine.broadcastPreCommit(*polka)
			}
			machine.lockedRound = machine.currentRound
			machine.lockedValue = polka.Block
			machine.validRound = machine.currentRound
			machine.validValue = polka.Block
			machine.ticksAtPrevoteState = -1
			machine.currentState = WaitingForCommit{}
			return machine.broadcastPreCommit(*polka)
		}
	case PreCommitted:
		polka, _ := machine.polkaBuilder.Polka(machine.currentHeight, machine.consensusThreshold)
		if polka != nil && polka.Round == machine.currentRound {
			if polka.Block == nil {
				machine.ticksAtPrevoteState = -1
				machine.currentState = WaitingForCommit{}
				return machine.broadcastPreCommit(*polka)
			}
			machine.lockedRound = machine.currentRound
			machine.lockedValue = polka.Block
			machine.validRound = machine.currentRound
			machine.validValue = polka.Block
			machine.ticksAtPrevoteState = -1
			machine.currentState = WaitingForCommit{}
			return machine.broadcastPreCommit(*polka)
		}

		machine.commitBuilder.Insert(transition.SignedPreCommit)

	case Ticked:
		polka, _ := machine.polkaBuilder.Polka(machine.currentHeight, machine.consensusThreshold)
		if polka != nil && polka.Round == machine.currentRound {
			if polka.Block == nil {
				machine.ticksAtPrevoteState = -1
				machine.currentState = WaitingForCommit{}
				return machine.broadcastPreCommit(*polka)
			}
			machine.lockedRound = machine.currentRound
			machine.lockedValue = polka.Block
			machine.validRound = machine.currentRound
			machine.validValue = polka.Block
			machine.ticksAtPrevoteState = -1
			machine.currentState = WaitingForCommit{}
			return machine.broadcastPreCommit(*polka)
		}
		if machine.ticksAtPrevoteState >= 0 {
			machine.ticksAtPrevoteState++
			maxTicksToTimeout := int(NumTicksToTriggerTimeOut + machine.currentRound)
			if machine.ticksAtPrevoteState > maxTicksToTimeout {
				machine.ticksAtPrevoteState = -1
				machine.currentState = WaitingForCommit{}
				polka := block.Polka{
					Round:  machine.currentRound,
					Height: machine.currentHeight,
				}
				return machine.broadcastPreCommit(polka)
			}
		}

	default:
		panic(fmt.Errorf("unexpected transition type %T", transition))
	}

	return nil
}

func (machine *machine) waitForCommit(transition Transition) Action {
	machine.checkAndSchedulePreCommitTimeout()
	polka, _ := machine.polkaBuilder.Polka(machine.currentHeight, machine.consensusThreshold)
	if polka != nil && polka.Round == machine.currentRound && polka.Block != nil {
		machine.validRound = machine.currentRound
		machine.validValue = polka.Block
	}
	switch transition := transition.(type) {
	case Proposed:
		commit, _ := machine.commitBuilder.Commit(machine.currentHeight, machine.consensusThreshold)
		if commit != nil && commit.Polka.Round == machine.currentRound {
			if commit.Polka.Block != nil {
				machine.currentHeight++
				machine.Drop()
				machine.lockedRound = -1
				machine.lockedValue = nil
				machine.validRound = -1
				machine.validValue = nil
				return machine.StartRound(0, commit)
			}
			machine.ticksAtPrecommitState = -1
			return machine.StartRound(machine.currentRound+1, nil)
		}

	case PreVoted:
		commit, _ := machine.commitBuilder.Commit(machine.currentHeight, machine.consensusThreshold)
		if commit != nil && commit.Polka.Round == machine.currentRound {
			if commit.Polka.Block != nil {
				machine.currentHeight++
				machine.Drop()
				machine.lockedRound = -1
				machine.lockedValue = nil
				machine.validRound = -1
				machine.validValue = nil
				return machine.StartRound(0, commit)
			}
			machine.ticksAtPrecommitState = -1
			return machine.StartRound(machine.currentRound+1, nil)
		}

		if !machine.polkaBuilder.Insert(transition.SignedPreVote) {
			return nil
		}

		polka, _ := machine.polkaBuilder.Polka(machine.currentHeight, machine.consensusThreshold)
		if polka != nil && polka.Round == machine.currentRound && polka.Block != nil {
			machine.validRound = machine.currentRound
			machine.validValue = polka.Block
		}
		return nil

	case PreCommitted:
		if !machine.commitBuilder.Insert(transition.SignedPreCommit) {
			return nil
		}

		commit, commitRound := machine.commitBuilder.Commit(machine.currentHeight, machine.consensusThreshold)
		if commitRound != nil && *commitRound == machine.currentRound && machine.ticksAtPrecommitState < 0 {
			machine.ticksAtPrecommitState = 0
		}

		if commit != nil && commit.Polka.Round == machine.currentRound {
			if commit.Polka.Block != nil {
				machine.currentHeight++
				machine.Drop()
				machine.lockedRound = -1
				machine.lockedValue = nil
				machine.validRound = -1
				machine.validValue = nil
				return machine.StartRound(0, commit)
			}
			machine.ticksAtPrecommitState = -1
			return machine.StartRound(machine.currentRound+1, nil)
		}

	case Ticked:
		commit, commitRound := machine.commitBuilder.Commit(machine.currentHeight, machine.consensusThreshold)
		if commitRound != nil && *commitRound == machine.currentRound && machine.ticksAtPrecommitState < 0 {
			machine.ticksAtPrecommitState = 0
		}

		if commit != nil && commit.Polka.Round == machine.currentRound {
			if commit.Polka.Block != nil {
				machine.currentHeight++
				machine.Drop()
				machine.lockedRound = -1
				machine.lockedValue = nil
				machine.validRound = -1
				machine.validValue = nil
				return machine.StartRound(0, commit)
			}
			machine.ticksAtPrecommitState = -1
			return machine.StartRound(machine.currentRound+1, nil)
		}

		if machine.ticksAtPrecommitState >= 0 {
			machine.ticksAtPrecommitState++
			maxTicksToTimeout := int(NumTicksToTriggerTimeOut + machine.currentRound)

			if machine.ticksAtPrecommitState > maxTicksToTimeout {
				machine.ticksAtPrecommitState = -1
				return machine.StartRound(machine.currentRound+1, nil)
			}
		}

	default:
		panic(fmt.Errorf("unexpected transition type %T", transition))
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
	precommit := block.PreCommit{
		Polka: polka,
	}

	signedPreCommit, err := precommit.Sign(machine.signer)
	if err != nil {
		panic(err)
	}
	machine.commitBuilder.Insert(signedPreCommit)

	return SignedPreCommit{
		SignedPreCommit: signedPreCommit,
	}
}

func (machine *machine) shouldProposeBlock() bool {
	return machine.signer.Signatory().Equal(machine.shard.Leader(machine.currentRound))
}

func (machine *machine) buildSignedBlock(commit *block.Commit) block.SignedBlock {
	transactions := make(tx.Transactions, 0, block.MaxTransactions)
	transaction, ok := machine.txPool.Dequeue()
	for ok && len(transactions) < block.MaxTransactions {
		transactions = append(transactions, transaction)
		transaction, ok = machine.txPool.Dequeue()
	}

	parentHeader := sig.Hash{}
	if commit != nil && commit.Polka.Block != nil {
		parentHeader = commit.Polka.Block.Header
	}
	block := block.New(
		machine.currentHeight,
		parentHeader,
		transactions,
	)
	signedBlock, err := block.Sign(machine.signer)
	if err != nil {
		// FIXME: We should handle this error properly. It would not make sense to propagate it, but there should at
		// least be some sane logging and recovery.
		panic(err)
	}
	return signedBlock
}

func (machine *machine) UpdateRound() *block.Round {
	currentRound := &machine.currentRound
	for round, sigMap := range machine.bufferedMessages {
		if round > *currentRound && len(sigMap) > machine.consensusThreshold/2 {
			currentRound = &round
		}
	}
	if *currentRound > machine.currentRound {
		for round := range machine.bufferedMessages {
			if round < *currentRound {
				delete(machine.bufferedMessages, round)
			}
		}
		return currentRound
	}
	return nil
}

func (machine *machine) checkAndSchedulePreCommitTimeout() {
	_, commitRound := machine.commitBuilder.Commit(machine.currentHeight, machine.consensusThreshold)
	if commitRound != nil && *commitRound == machine.currentRound && machine.ticksAtPrecommitState < 0 {
		machine.ticksAtPrecommitState = 0
	}
}

func (machine *machine) checkAndSchedulePreVoteTimeout() {
	_, polkaRound := machine.polkaBuilder.Polka(machine.currentHeight, machine.consensusThreshold)
	if polkaRound != nil && *polkaRound == machine.currentRound && machine.ticksAtPrevoteState < 0 {
		machine.ticksAtPrevoteState = 0
	}
}
