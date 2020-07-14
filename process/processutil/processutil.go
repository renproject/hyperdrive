package processutil

import (
	"math/rand"

	"github.com/renproject/hyperdrive/process"
	"github.com/renproject/id"
)

// BroadcasterCallbacks provide callback functions to test the broadcaster
// behaviour required by a Process
type BroadcasterCallbacks struct {
	BroadcastProposeCallback   func(process.Propose)
	BroadcastPrevoteCallback   func(process.Prevote)
	BroadcastPrecommitCallback func(process.Precommit)
}

// BroadcastPropose passes the propose message to the propose callback, if present
func (broadcaster BroadcasterCallbacks) BroadcastPropose(propose process.Propose) {
	if broadcaster.BroadcastProposeCallback == nil {
		return
	}
	broadcaster.BroadcastProposeCallback(propose)
}

// BroadcastPrevote passes the prevote message to the prevote callback, if present
func (broadcaster BroadcasterCallbacks) BroadcastPrevote(prevote process.Prevote) {
	if broadcaster.BroadcastPrevoteCallback == nil {
		return
	}
	broadcaster.BroadcastPrevoteCallback(prevote)
}

// BroadcastPrecommit passes the precommit message to the precommit callback, if present
func (broadcaster BroadcasterCallbacks) BroadcastPrecommit(precommit process.Precommit) {
	if broadcaster.BroadcastPrecommitCallback == nil {
		return
	}
	broadcaster.BroadcastPrecommitCallback(precommit)
}

// CommitterCallback provides a callback function to test the Committer
// behaviour required by a Process
type CommitterCallback struct {
	Callback func(process.Height, process.Value)
}

// Commit passes the commitment parameters height and round to the commit callback, if present
func (committer CommitterCallback) Commit(height process.Height, value process.Value) {
	if committer.Callback == nil {
		return
	}
	committer.Callback(height, value)
}

// MockProposer is a mock implementation of the Proposer interface
// It always proposes the value MockValue
type MockProposer struct {
	MockValue func() process.Value
}

// Propose implements the propose behaviour as required by the Proposer interface
// The MockProposer's propose method does not take into consideration the
// consensus parameters height and round, but simply returns the value MockValue
func (p MockProposer) Propose(height process.Height, round process.Round) process.Value {
	return p.MockValue()
}

// MockValidator is a mock implementation of the Validator interface
// It always returns the MockValid value as its validation check
type MockValidator struct {
	MockValid func(value process.Value) bool
}

// Valid implements the validation behaviour as required by the Validator interface
// The MockValidator's valid method does not take into consideration the
// received propose message, but simply returns the MockValid value as its
// validation check
func (v MockValidator) Valid(value process.Value) bool {
	return v.MockValid(value)
}

// CatcherCallbacks provide callback functions to test the Catcher interface
// required by a Process
type CatcherCallbacks struct {
	CatchDoubleProposeCallback    func(process.Propose, process.Propose)
	CatchDoublePrevoteCallback    func(process.Prevote, process.Prevote)
	CatchDoublePrecommitCallback  func(process.Precommit, process.Precommit)
	CatchOutOfTurnProposeCallback func(process.Propose)
}

// CatchDoublePropose implements the interface method of handling the event when
// two different propose messages were received from the same process.
// In this case, it simply passes those to the appropriate callback function
func (catcher CatcherCallbacks) CatchDoublePropose(propose1 process.Propose, propose2 process.Propose) {
	if catcher.CatchDoubleProposeCallback == nil {
		return
	}
	catcher.CatchDoubleProposeCallback(propose1, propose2)
}

// CatchDoublePrevote implements the interface method of handling the event when
// two different prevote messages were received from the same process.
// In this case, it simply passes those to the appropriate callback function
func (catcher CatcherCallbacks) CatchDoublePrevote(prevote1 process.Prevote, prevote2 process.Prevote) {
	if catcher.CatchDoublePrevoteCallback == nil {
		return
	}
	catcher.CatchDoublePrevoteCallback(prevote1, prevote2)
}

// CatchDoublePrecommit implements the interface method of handling the event when
// two different precommit messages were received from the same process.
// In this case, it simply passes those to the appropriate callback function
func (catcher CatcherCallbacks) CatchDoublePrecommit(precommit1 process.Precommit, precommit2 process.Precommit) {
	if catcher.CatchDoublePrecommitCallback == nil {
		return
	}
	catcher.CatchDoublePrecommitCallback(precommit1, precommit2)
}

// CatchOutOfTurnPropose implements the interface method of handling the event when
// a process not scheduled to propose for the current height/round broadcasts a propose.
// In this case, it simply passes those to the appropriate callback function
func (catcher CatcherCallbacks) CatchOutOfTurnPropose(propose process.Propose) {
	if catcher.CatchOutOfTurnProposeCallback == nil {
		return
	}
	catcher.CatchOutOfTurnProposeCallback(propose)
}

// RandomHeight consumes a source of randomness and returns a random height
// for the consensus mechanism. It returns a truly random height 70% of the times,
// whereas for the other 30% of the times it returns heights for edge scenarios
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

// RandomRound consumes a source of randomness and returns a random round
// for the consensus mechanism. It returns a truly random round 70% of the times,
// whereas for the other 30% of the times it returns rounds for edge scenarios
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

// RandomStep consumes a source of randomness and returns a random step
// for the consensus mechanism. A random step could be a valid or invalid step
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

// RandomGoodValue consumes a source of randomness and returns a truly random
// value which is proposed by the proposer, on which consensus can be reached
func RandomGoodValue(r *rand.Rand) process.Value {
	v := process.Value{}
	for i := range v {
		v[i] = byte(r.Int())
	}
	return v
}

// RandomValue consumes a source of randomness and returns a random value
// for the consensus mechanism. It returns a truly random round 80% of the times,
// whereas for the other 20% of the times it returns rounds for edge scenarios
func RandomValue(r *rand.Rand) process.Value {
	switch r.Int() % 10 {
	case 0:
		return process.Value{}
	case 1:
		return process.Value{0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF}
	default:
		return RandomGoodValue(r)
	}
}

// RandomState consumes a source of randomness and returns a random state of
// a process in the consensus mechanism.
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

// RandomPropose consumes a source of randomness and returns a random propose
// message. The message is a valid message 70% of the times, and other times
// this function returns some edge scenarios, including empty message
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

// RandomPrevote consumes a source of randomness and returns a random prevote
// message. The message is a valid message 70% of the times, and other times
// this function returns some edge scenarios, including empty message
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

// RandomPrecommit consumes a source of randomness and returns a random precommit
// message. The message is a valid message 70% of the times, and other times
// this function returns some edge scenarios, including empty message
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
