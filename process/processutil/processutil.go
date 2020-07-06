package processutil

import (
	"math/rand"

	"github.com/renproject/hyperdrive/process"
	"github.com/renproject/id"
)

type BroadcasterCallbacks struct {
	BroadcastProposeCallback   func(process.Propose)
	BroadcastPrevoteCallback   func(process.Prevote)
	BroadcastPrecommitCallback func(process.Precommit)
}

func (broadcaster BroadcasterCallbacks) BroadcastPropose(propose process.Propose) {
	if broadcaster.BroadcastProposeCallback == nil {
		return
	}
	broadcaster.BroadcastProposeCallback(propose)
}

func (broadcaster BroadcasterCallbacks) BroadcastPrevote(prevote process.Prevote) {
	if broadcaster.BroadcastPrevoteCallback == nil {
		return
	}
	broadcaster.BroadcastPrevoteCallback(prevote)
}

func (broadcaster BroadcasterCallbacks) BroadcastPrecommit(precommit process.Precommit) {
	if broadcaster.BroadcastPrecommitCallback == nil {
		return
	}
	broadcaster.BroadcastPrecommitCallback(precommit)
}

type CommitterCallback struct {
	Callback func(process.Height, process.Value)
}

func (committer CommitterCallback) Commit(height process.Height, value process.Value) {
	if committer.Callback == nil {
		return
	}
	committer.Callback(height, value)
}

type MockProposer struct {
	MockValue process.Value
}

func (p MockProposer) Propose(height process.Height, round process.Round) process.Value {
	return p.MockValue
}

type MockValidator struct {
	MockValid bool
}

func (v MockValidator) Valid(process.Value) bool {
	return v.MockValid
}

func RandomHeight(r *rand.Rand) process.Height {
	switch r.Int() % 10 {
	case 0:
		return process.Height(-1)
	case 1:
		return process.Height(0)
	case 2:
		return process.Height(9223372036854775807)
	default:
		return process.Height(r.Int63())
	}
}

func RandomRound(r *rand.Rand) process.Round {
	switch r.Int() % 10 {
	case 0:
		return process.Round(-1)
	case 1:
		return process.Round(0)
	case 2:
		return process.Round(9223372036854775807)
	default:
		return process.Round(r.Int63())
	}
}

func RandomStep(r *rand.Rand) process.Step {
	switch r.Int() % 10 {
	case 0:
		return process.Proposing
	case 1:
		return process.Prevoting
	case 2:
		return process.Prevoting
	case 3:
		return process.Step(255)
	default:
		return process.Step(uint8(r.Int63()))
	}
}

func RandomValue(r *rand.Rand) process.Value {
	switch r.Int() % 10 {
	case 0:
		return process.Value{}
	case 1:
		return process.Value{0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF}
	default:
		v := process.Value{}
		for i := range v {
			v[i] = byte(r.Int())
		}
		return v
	}
}

func RandomState(r *rand.Rand) process.State {
	switch r.Int() % 10 {
	case 0:
		return process.DefaultState()
	default:
		return process.State{
			CurrentHeight: RandomHeight(r),
			CurrentRound:  RandomRound(r),
			CurrentStep:   RandomStep(r),
			LockedRound:   RandomRound(r),
			LockedValue:   RandomValue(r),
			ValidRound:    RandomRound(r),
			ValidValue:    RandomValue(r),

			ProposeLogs:   make(map[process.Round]process.Propose),
			PrevoteLogs:   make(map[process.Round]map[id.Signatory]process.Prevote),
			PrecommitLogs: make(map[process.Round]map[id.Signatory]process.Precommit),
			OnceFlags:     make(map[process.Round]process.OnceFlag),
		}
	}
}

func RandomPropose(r *rand.Rand) process.Propose {
	switch r.Int() % 10 {
	case 0:
		return process.Propose{}
	case 1:
		return process.Propose{
			Height:     RandomHeight(r),
			Round:      RandomRound(r),
			ValidRound: RandomRound(r),
			Value:      RandomValue(r),
			From:       id.Signatory{},
			Signature:  id.Signature{},
		}
	case 2:
		signatory := id.Signatory{}
		for i := range signatory {
			signatory[i] = byte(r.Int())
		}
		signature := id.Signature{}
		for i := range signature {
			signature[i] = byte(r.Int())
		}
		return process.Propose{
			Height:     RandomHeight(r),
			Round:      RandomRound(r),
			ValidRound: RandomRound(r),
			Value:      RandomValue(r),
			From:       signatory,
			Signature:  signature,
		}
	default:
		msg := process.Propose{
			Height:     RandomHeight(r),
			Round:      RandomRound(r),
			ValidRound: RandomRound(r),
			Value:      RandomValue(r),
		}
		privKey := id.NewPrivKey()
		hash, err := process.NewProposeHash(msg.Height, msg.Round, msg.ValidRound, msg.Value)
		if err != nil {
			panic(err)
		}
		signature, err := privKey.Sign(&hash)
		if err != nil {
			panic(err)
		}
		msg.From = privKey.Signatory()
		msg.Signature = signature
		return msg
	}
}

func RandomPrevote(r *rand.Rand) process.Prevote {
	switch r.Int() % 10 {
	case 0:
		return process.Prevote{}
	case 1:
		return process.Prevote{
			Height:    RandomHeight(r),
			Round:     RandomRound(r),
			Value:     RandomValue(r),
			From:      id.Signatory{},
			Signature: id.Signature{},
		}
	case 2:
		signatory := id.Signatory{}
		for i := range signatory {
			signatory[i] = byte(r.Int())
		}
		signature := id.Signature{}
		for i := range signature {
			signature[i] = byte(r.Int())
		}
		return process.Prevote{
			Height:    RandomHeight(r),
			Round:     RandomRound(r),
			Value:     RandomValue(r),
			From:      signatory,
			Signature: signature,
		}
	default:
		msg := process.Prevote{
			Height: RandomHeight(r),
			Round:  RandomRound(r),
			Value:  RandomValue(r),
		}
		privKey := id.NewPrivKey()
		hash, err := process.NewPrevoteHash(msg.Height, msg.Round, msg.Value)
		if err != nil {
			panic(err)
		}
		signature, err := privKey.Sign(&hash)
		if err != nil {
			panic(err)
		}
		msg.From = privKey.Signatory()
		msg.Signature = signature
		return msg
	}
}

func RandomPrecommit(r *rand.Rand) process.Precommit {
	switch r.Int() % 10 {
	case 0:
		return process.Precommit{}
	case 1:
		return process.Precommit{
			Height:    RandomHeight(r),
			Round:     RandomRound(r),
			Value:     RandomValue(r),
			From:      id.Signatory{},
			Signature: id.Signature{},
		}
	case 2:
		signatory := id.Signatory{}
		for i := range signatory {
			signatory[i] = byte(r.Int())
		}
		signature := id.Signature{}
		for i := range signature {
			signature[i] = byte(r.Int())
		}
		return process.Precommit{
			Height:    RandomHeight(r),
			Round:     RandomRound(r),
			Value:     RandomValue(r),
			From:      signatory,
			Signature: signature,
		}
	default:
		msg := process.Precommit{
			Height: RandomHeight(r),
			Round:  RandomRound(r),
			Value:  RandomValue(r),
		}
		privKey := id.NewPrivKey()
		hash, err := process.NewPrecommitHash(msg.Height, msg.Round, msg.Value)
		if err != nil {
			panic(err)
		}
		signature, err := privKey.Sign(&hash)
		if err != nil {
			panic(err)
		}
		msg.From = privKey.Signatory()
		msg.Signature = signature
		return msg
	}
}
