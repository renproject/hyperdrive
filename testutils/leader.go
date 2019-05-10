package testutils

import (
	"time"

	"github.com/renproject/hyperdrive"
	"github.com/renproject/hyperdrive/block"
	"github.com/renproject/hyperdrive/replica"
	"github.com/renproject/hyperdrive/shard"
	"github.com/renproject/hyperdrive/sig"
	"github.com/renproject/hyperdrive/state"
	"github.com/renproject/hyperdrive/tx"
)

// This file implements a malicious leader that sends proposals even when it is not its own turn
type faultyLeader struct {
	dispatcher replica.Dispatcher
	signer     sig.SignerVerifier

	prevotes   map[sig.Hash]map[sig.Signatory]sig.Signature
	precommits map[sig.Hash]map[sig.Signatory]sig.Signature

	consensusThreshold int
}

func NewFaultyLeader(signer sig.SignerVerifier, dispatcher replica.Dispatcher, consensusThreshold int) hyperdrive.Hyperdrive {
	return &faultyLeader{
		dispatcher: dispatcher,
		signer:     signer,

		prevotes:   map[sig.Hash]map[sig.Signatory]sig.Signature{},
		precommits: map[sig.Hash]map[sig.Signatory]sig.Signature{},

		consensusThreshold: consensusThreshold,
	}
}

func (faultyLeader *faultyLeader) AcceptTick(t time.Time) {
	return
}

func (faultyLeader *faultyLeader) AcceptPropose(shardHash sig.Hash, proposed block.SignedBlock) {
	action := state.PreVote{
		PreVote: block.PreVote{
			Block:  &proposed,
			Round:  proposed.Round,
			Height: proposed.Height,
		},
	}
	signedPreVote, err := action.PreVote.Sign(faultyLeader.signer)
	if err != nil {
		panic(err)
	}
	if _, ok := faultyLeader.prevotes[proposed.Header]; ok {
		faultyLeader.prevotes[proposed.Header][signedPreVote.Signatory] = signedPreVote.Signature
	} else {
		faultyLeader.prevotes[proposed.Header] = map[sig.Signatory]sig.Signature{}
		faultyLeader.prevotes[proposed.Header][signedPreVote.Signatory] = signedPreVote.Signature
	}

	faultyLeader.dispatcher.Dispatch(shardHash, state.SignedPreVote{
		SignedPreVote: signedPreVote,
	})
}

func (faultyLeader *faultyLeader) AcceptPreVote(shardHash sig.Hash, preVote block.SignedPreVote) {
	header := sig.Hash{}
	if preVote.Block != nil {
		header = preVote.Block.Header
	}
	if _, ok := faultyLeader.prevotes[header]; ok {
		faultyLeader.prevotes[header][preVote.Signatory] = preVote.Signature
	} else {
		faultyLeader.prevotes[header] = map[sig.Signatory]sig.Signature{}
		faultyLeader.prevotes[header][preVote.Signatory] = preVote.Signature
	}
	if len(faultyLeader.prevotes[header]) >= faultyLeader.consensusThreshold {
		sigMap := faultyLeader.prevotes[header]
		sigs := sig.Signatures{}
		signatories := sig.Signatories{}
		for signatory, signature := range sigMap {
			sigs = append(sigs, signature)
			signatories = append(signatories, signatory)
		}
		action := state.PreCommit{
			PreCommit: block.PreCommit{
				Polka: block.Polka{
					Block:       preVote.Block,
					Round:       preVote.Round,
					Height:      preVote.Height,
					Signatures:  sigs,
					Signatories: signatories,
				},
			},
		}
		signedPreCommit, err := action.PreCommit.Sign(faultyLeader.signer)
		if err != nil {
			panic(err)
		}
		faultyLeader.dispatcher.Dispatch(shardHash, state.SignedPreCommit{
			SignedPreCommit: signedPreCommit,
		})
	}
}

func (faultyLeader *faultyLeader) AcceptPreCommit(shardHash sig.Hash, preCommit block.SignedPreCommit) {
	header := sig.Hash{}
	if preCommit.Polka.Block != nil {
		header = preCommit.Polka.Block.Header
	}
	if _, ok := faultyLeader.precommits[header]; ok {
		faultyLeader.precommits[header][preCommit.Signatory] = preCommit.Signature
	} else {
		faultyLeader.precommits[header] = map[sig.Signatory]sig.Signature{}
		faultyLeader.precommits[header][preCommit.Signatory] = preCommit.Signature
	}

}

func (faultyLeader *faultyLeader) AcceptCommit(shardHash sig.Hash, commit block.Commit) {
	panic("unimplemented")
}

func (faultyLeader *faultyLeader) AcceptShard(shard shard.Shard, head block.SignedBlock, pool tx.Pool) {
	return
}
