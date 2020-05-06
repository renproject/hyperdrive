// Package proc implements the Byzantine fault tolerant consensus algorithm
// described by "The latest gossip of BFT consensus" (Buchman et al.), which can
// be found at https://arxiv.org/pdf/1807.04938.pdf. It makes extensive use of
// dependency injection, and concrete implementions  must be careful to meet all
// of the requirements specified by the interface, otherwise the correctness of
// the consensus algorithm can be broken.
package proc

// A Scheduler is used to determine which Process should be proposing a Vaue at
// the given Height and Round. A Scheduler must be derived solely from the
// Height, Round, and Values on which all correct Processes have already
// achieved consensus.
type Scheduler interface {
	Schedule(height Height, round Round) Pid
}

// A Proposer is used to propose new Values for consensus. A Proposer must only
// ever return a valid Value, and once it returns a Value, it must never return
// a different Value for the same Height and Round.
type Proposer interface {
	Propose(Height, Round) Value
}

// A Timer is used to schedule timeout events.
type Timer interface {
	// TimeoutPropose is called when the Process needs its OnTimeoutPropose
	// method called after a timeout. The timeout should be proportional to the
	// Round.
	TimeoutPropose(Height, Round)
	// TimeoutPrevote is called when the Process needs its OnTimeoutPrevote
	// method called after a timeout. The timeout should be proportional to the
	// Round.
	TimeoutPrevote(Height, Round)
	// TimeoutPrecommit is called when the Process needs its OnTimeoutPrecommit
	// method called after a timeout. The timeout should be proportional to the
	// Round.
	TimeoutPrecommit(Height, Round)
}

// A Broadcaster is used to broadcast Propose, Prevote, and Precommit messages
// to all Processes in the consensus algorithm, including the Process that
// initiated the broadcast. It is assumed that all messages between correct
// Processes are eventually delivered, although no specific order is assumed.
//
// Once a Value has been broadcast as part of a Propose, Prevote, or Precommit
// message, different Values must not be broadcast for that same message type
// with the same Height and Round. The same restriction applies to valid Rounds
// broadcast with a Propose message.
type Broadcaster interface {
	BroadcastPropose(Height, Round, Value, Round)
	BroadcastPrevote(Height, Round, Value)
	BroadcastPrecommit(Height, Round, Value)
}

// A Validator is used to validate a proposed Value. Processes are not required
// to agree on the validity of a Value.
type Validator interface {
	Valid(Value) bool
}

// A Committer is used to emit Values that are committed. The commitment of a
// new Value implies that all correct Processes agree on this Value at this
// Height, and will never revert.
type Committer interface {
	Commit(Height, Value)
}

// A Process is a deterministic finite state automaton that communicates with
// other Processes to implement a Byzantine fault tolerant consensus algorithm.
// It is intended to be used as part of a larger component that implements a
// Byzantine fault tolerant replicated state machine.
//
// All messages from previous and future Heights will be ignored. The component
// using the Process should buffer all messages from future Heights so that they
// are not lost. It is assumed that this component will also handle the
// authentication and rate-limiting of messages.
//
// Processes are not safe for concurrent use. All methods must be called by the
// same goroutine that allocates and starts the Process.
type Process struct {

	// Input interface that provide data to the Process.
	scheduler Scheduler
	proposer  Proposer
	validator Validator

	// Output interfaces that received data from the Process.
	timer       Timer
	broadcaster Broadcaster
	committer   Committer

	// ProposeLogs store the Proposes for all Rounds.
	ProposeLogs map[Round]Propose `json:"proposeLogs"`
	// PrevoteLogs store the Prevotes for all Processes in all Rounds.
	PrevoteLogs map[Round]map[Pid]Prevote `json:"prevoteLogs"`
	// PrecommitLogs store the Precommits for all Processes in all Rounds.
	PrecommitLogs map[Round]map[Pid]Precommit `json:"precommitLogs"`
	// F is the maximum number of malicious adversaries that the Process can
	// withstand while still maintaining safety and liveliness.
	F int `json:"f"`

	// OnceFlags prevents events from happening more than once.
	OnceFlags map[Round]OnceFlag `json:"onceFlags"`

	// Whoami represnts the Pid of this Process. It is assumed that the ECDSA
	// privkey required to prove ownership of this Pid is known.
	Whoami Pid `json:"whoami"`

	// State of the Process.
	State `json:"state"`
}

// Start the Process.
func (p *Process) Start() {
	p.StartRound(0)
}

// StartRound will progress the Process to a new Round.
func (p *Process) StartRound(round Round) {
	defer func() {
		p.tryTimeoutPrevoteUponSufficientPrevotes()
		p.tryTimeoutPrecommitUponSufficientPrecommits()
		p.tryPrevoteUponPropose()
		p.tryPrevoteUponSufficientPrevotes()
		p.tryPrecommitUponSufficientPrevotes()
		p.tryPrecommitNilUponSufficientPrevotes()
	}()

	// Set the state the new round, and set the step to the first step in the
	// sequence.
	p.CurrentRound = round
	p.CurrentStep = Proposing

	// If we are not the proposer, then we trigger the propose timeout.
	proposer := p.scheduler.Schedule(p.CurrentHeight, p.CurrentRound)
	if !p.Whoami.Equal(&proposer) {
		p.timer.TimeoutPropose(p.CurrentHeight, p.CurrentRound)
		return
	}

	// If we are the proposer, then we emit a propose.
	proposeValue := p.ValidValue
	if proposeValue.Equal(&NilValue) {
		proposeValue = p.proposer.Propose(p.CurrentHeight, p.CurrentRound)
	}
	p.broadcaster.BroadcastPropose(
		p.CurrentHeight,
		p.CurrentRound,
		proposeValue,
		p.ValidRound,
	)
}

// OnTimeoutPropose is used to notify the Process that a timeout has been
// activated. It must only be called after the TimeoutPropose method in the
// Timer has been called.
//
// L57:
//	Function OnTimeoutPropose(height, round)
//		if height = currentHeight ∧ round = currentRound ∧ currentStep = propose then
//			broadcast〈PREVOTE, currentHeight, currentRound, nil
//			currentStep ← prevote
func (p *Process) OnTimeoutPropose(height Height, round Round) {
	if height == p.CurrentHeight && round == p.CurrentRound && p.CurrentStep == Proposing {
		p.broadcaster.BroadcastPrevote(p.CurrentHeight, p.CurrentRound, NilValue)
		p.stepToPrevoting()
	}
}

// OnTimeoutPrevote is used to notify the Process that a timeout has been
// activated. It must only be called after the TimeoutPrevote method in the
// Timer has been called.
//
// L61:
//	Function OnTimeoutPrevote(height, round)
//		if height = currentHeight ∧ round = currentRound ∧ currentStep = prevote then
//			broadcast〈PREVOTE, currentHeight, currentRound, nil
//			currentStep ← prevote
func (p *Process) OnTimeoutPrevote(height Height, round Round) {
	if height == p.CurrentHeight && round == p.CurrentRound && p.CurrentStep == Prevoting {
		p.broadcaster.BroadcastPrecommit(p.CurrentHeight, p.CurrentRound, NilValue)
		p.stepToPrecommitting()
	}
}

// OnTimeoutPrecommit is used to notify the Process that a timeout has been
// activated. It must only be called after the TimeoutPrecommit method in the
// Timer has been called.
//
// L65:
//	Function OnTimeoutPrecommit(height, round)
//		if height = currentHeight ∧ round = currentRound then
//			StartRound(currentRound + 1)
func (p *Process) OnTimeoutPrecommit(height Height, round Round) {
	if height == p.CurrentHeight && round == p.CurrentRound {
		p.StartRound(round + 1)
	}
}

// Propose is used to notify the Process that a Propose message has been
// received (this includes Propose messages that the Process itself has
// broadcast). All conditions that could be opened by the receipt of a Propose
// message will be tried.
func (p *Process) Propose(propose Propose) {
	if !p.insertPropose(propose) {
		return
	}

	p.trySkipToFutureRound(propose.Round)
	p.tryCommitUponSufficientPrecommits(propose.Round)

	p.tryPrecommitUponSufficientPrevotes()
	p.tryPrevoteUponPropose()
}

// Prevote is used to notify the Process that a Prevote message has been
// received (this includes Prevote messages that the Process itself has
// broadcast). All conditions that could be opened by the receipt of a Prevote
// message will be tried.
func (p *Process) Prevote(prevote Prevote) {
	if !p.insertPrevote(prevote) {
		return
	}

	p.trySkipToFutureRound(prevote.Round)

	p.tryPrecommitUponSufficientPrevotes()
	p.tryPrecommitNilUponSufficientPrevotes()
	p.tryPrevoteUponSufficientPrevotes()
	p.tryTimeoutPrevoteUponSufficientPrevotes()
}

// Precommit is used to notify the Process that a Precommit message has been
// received (this includes Precommit messages that the Process itself has
// broadcast). All conditions that could be opened by the receipt of a Precommit
// message will be tried.
func (p *Process) Precommit(precommit Precommit) {
	if !p.insertPrecommit(precommit) {
		return
	}

	p.trySkipToFutureRound(precommit.Round)

	p.tryCommitUponSufficientPrecommits(precommit.Round)
	p.tryTimeoutPrecommitUponSufficientPrecommits()
}

// L22:
//	upon〈PROPOSAL, currentHeight, currentRound, v, −1〉from proposer(currentHeight, currentRound)
//	while currentStep = propose do
//		if valid(v) ∧ (lockedRound = −1 ∨ lockedValue = v) then
//			broadcast〈PREVOTE, currentHeight, currentRound, id(v)
//		else
//			broadcast〈PREVOTE, currentHeight, currentRound, nil
//		currentStep ← prevote
func (p *Process) tryPrevoteUponPropose() {
	if p.CurrentStep != Proposing {
		return
	}

	propose, ok := p.ProposeLogs[p.CurrentRound]
	if !ok {
		return
	}
	if propose.ValidRound != InvalidRound {
		return
	}

	if p.LockedRound == InvalidRound || p.LockedValue.Equal(&propose.Value) {
		p.broadcaster.BroadcastPrevote(p.CurrentHeight, p.CurrentRound, propose.Value)
	} else {
		p.broadcaster.BroadcastPrevote(p.CurrentHeight, p.CurrentRound, NilValue)
	}
	p.stepToPrevoting()
}

// L28:
//	upon〈PROPOSAL, currentHeight, currentRound, v, vr〉from proposer(currentHeight, currentRound) AND 2f+ 1〈PREVOTE, currentHeight, vr, id(v)〉
//	while currentStep = propose ∧ (vr ≥ 0 ∧ vr < currentRound) do
//		if valid(v) ∧ (lockedRound ≤ vr ∨ lockedValue = v) then
//			broadcast〈PREVOTE, currentHeight, currentRound, id(v)〉
//		else
//			broadcast〈PREVOTE, currentHeight, currentRound, nil〉
//		stepp←prevote
func (p *Process) tryPrevoteUponSufficientPrevotes() {
	if p.CurrentStep != Proposing {
		return
	}

	propose, ok := p.ProposeLogs[p.CurrentRound]
	if !ok {
		return
	}
	if propose.ValidRound == InvalidRound || propose.ValidRound >= p.CurrentRound {
		return
	}

	prevotesInValidRound := 0
	for _, prevote := range p.PrevoteLogs[propose.ValidRound] {
		if prevote.Value.Equal(&propose.Value) {
			prevotesInValidRound++
		}
	}
	if prevotesInValidRound < 2*p.F+1 {
		return
	}

	if p.LockedRound <= propose.ValidRound || p.LockedValue.Equal(&propose.Value) {
		p.broadcaster.BroadcastPrevote(p.CurrentHeight, p.CurrentRound, propose.Value)
	} else {
		p.broadcaster.BroadcastPrevote(p.CurrentHeight, p.CurrentRound, NilValue)
	}
	p.stepToPrevoting()
}

// L34:
//	upon 2f+ 1〈PREVOTE, currentHeight, currentRound, ∗〉
//	while currentStep = prevote for the first time do
//		scheduleOnTimeoutPrevote(currentHeight, currentRound) to be executed after timeoutPrevote(currentRound)
func (p *Process) tryTimeoutPrevoteUponSufficientPrevotes() {
	if p.checkOnceFlag(p.CurrentRound, OnceFlagTimeoutPrevoteUponSufficientPrevotes) {
		return
	}
	if p.CurrentStep != Prevoting {
		return
	}
	if len(p.PrevoteLogs[p.CurrentRound]) == 2*p.F+1 {
		p.timer.TimeoutPrevote(p.CurrentHeight, p.CurrentRound)
	}
	p.setOnceFlag(p.CurrentRound, OnceFlagTimeoutPrevoteUponSufficientPrevotes)
}

// L36:
//	upon〈PROPOSAL, currentHeight, currentRound, v, ∗〉from proposer(currentHeight, currentRound) AND 2f+ 1〈PREVOTE, currentHeight, currentRound, id(v)〉
//	while valid(v) ∧ currentStep ≥ prevote for the first time do
//		if currentStep = prevote then
//			lockedValue ← v
//			lockedRound ← currentRound
//			broadcast〈PRECOMMIT, currentHeight, currentRound, id(v))〉
//			currentStep ← precommit
//		validValue ← v
//		validRound ← currentRound
func (p *Process) tryPrecommitUponSufficientPrevotes() {
	if p.checkOnceFlag(p.CurrentRound, OnceFlagPrecommitUponSufficientPrevotes) {
		return
	}
	if p.CurrentStep < Prevoting {
		return
	}

	propose, ok := p.ProposeLogs[p.CurrentRound]
	if !ok {
		return
	}
	prevotesForValue := 0
	for _, prevote := range p.PrevoteLogs[p.CurrentRound] {
		if prevote.Value.Equal(&propose.Value) {
			prevotesForValue++
		}
	}
	if prevotesForValue < 2*p.F+1 {
		return
	}

	if p.CurrentStep == Prevoting {
		p.LockedValue = propose.Value
		p.LockedRound = p.CurrentRound
		p.broadcaster.BroadcastPrecommit(p.CurrentHeight, p.CurrentRound, propose.Value)
		p.stepToPrecommitting()
	}
	p.ValidValue = propose.Value
	p.ValidRound = p.CurrentRound
	p.setOnceFlag(p.CurrentRound, OnceFlagPrecommitUponSufficientPrevotes)
}

// L44:
//	upon 2f+ 1〈PREVOTE, currentHeight, currentRound, nil〉
//	while currentStep = prevote do
//		broadcast〈PRECOMMIT, currentHeight, currentRound, nil〉
//		currentStep ← precommit
func (p *Process) tryPrecommitNilUponSufficientPrevotes() {
	if p.CurrentStep != Prevoting {
		return
	}
	prevotesForNil := 0
	for _, prevote := range p.PrevoteLogs[p.CurrentRound] {
		if prevote.Value.Equal(&NilValue) {
			prevotesForNil++
		}
	}
	if prevotesForNil == 2*p.F+1 {
		p.broadcaster.BroadcastPrecommit(p.CurrentHeight, p.CurrentRound, NilValue)
		p.stepToPrecommitting()
	}
}

// L47:
//	upon 2f+ 1〈PRECOMMIT, currentHeight, currentRound, ∗〉for the first time do
//		scheduleOnTimeoutPrecommit(currentHeight, currentRound) to be executed after timeoutPrecommit(currentRound)
func (p *Process) tryTimeoutPrecommitUponSufficientPrecommits() {
	if p.checkOnceFlag(p.CurrentRound, OnceFlagTimeoutPrecommitUponSufficientPrecommits) {
		return
	}
	if len(p.PrecommitLogs[p.CurrentRound]) == 2*p.F+1 {
		p.timer.TimeoutPrecommit(p.CurrentHeight, p.CurrentRound)
		p.setOnceFlag(p.CurrentRound, OnceFlagTimeoutPrecommitUponSufficientPrecommits)
	}
}

// L49:
//	upon〈PROPOSAL, currentHeight, r, v, ∗〉from proposer(currentHeight, r) AND 2f+ 1〈PRECOMMIT, currentHeight, r, id(v)〉
//	while decision[currentHeight] = nil do
//		if valid(v) then
//			decisionp[currentHeight] = v
//			currentHeight ← currentHeight + 1
//			reset
//			StartRound(0)
func (p *Process) tryCommitUponSufficientPrecommits(round Round) {
	propose, ok := p.ProposeLogs[round]
	if !ok {
		return
	}
	precommitsForValue := 0
	for _, precommit := range p.PrecommitLogs[round] {
		if precommit.Value.Equal(&propose.Value) {
			precommitsForValue++
		}
	}
	if precommitsForValue == 2*p.F+1 {
		p.committer.Commit(p.CurrentHeight, propose.Value)
		p.CurrentHeight++
		p.Reset()
		p.StartRound(0)
	}
}

// L55:
//	upon f+ 1〈∗, currentHeight, r, ∗, ∗〉with r > currentRound do
//		StartRound(r)
func (p *Process) trySkipToFutureRound(round Round) {
	if round <= p.CurrentRound {
		return
	}

	msgsInRound := 0
	if _, ok := p.ProposeLogs[round]; ok {
		msgsInRound = 1
	}
	msgsInRound += len(p.PrevoteLogs[round])
	msgsInRound += len(p.PrecommitLogs[round])

	if msgsInRound == p.F+1 {
		p.StartRound(round)
	}
}

// insertPropose after validating it and checking for duplicates. If the Propose
// was accepted and inserted, then it return true, otherwise it returns false.
func (p *Process) insertPropose(propose Propose) bool {
	if propose.Height != p.CurrentHeight {
		return false
	}

	existingPropose, ok := p.ProposeLogs[propose.Round]
	if ok {
		if !propose.Equal(&existingPropose) {
			// FIXME: Punish the proposer.
		}
		return false
	}
	proposer := p.scheduler.Schedule(propose.Height, propose.Round)
	if !proposer.Equal(&propose.From) {
		return false
	}
	if !p.validator.Valid(propose.Value) {
		return false
	}

	p.ProposeLogs[propose.Round] = propose
	return true
}

// insertPrevote after validating it and checking for duplicates. If the Prevote
// was accepted and inserted, then it return true, otherwise it returns false.
func (p *Process) insertPrevote(prevote Prevote) bool {
	if prevote.Height != p.CurrentHeight {
		return false
	}
	if _, ok := p.PrevoteLogs[prevote.Round]; !ok {
		p.PrevoteLogs[prevote.Round] = map[Pid]Prevote{}
	}

	existingPrevote, ok := p.PrevoteLogs[prevote.Round][prevote.From]
	if ok {
		if !prevote.Equal(&existingPrevote) {
			// FIXME: Punish the prevoter.
		}
		return false
	}

	p.PrevoteLogs[prevote.Round][prevote.From] = prevote
	return true
}

// insertPrecommit after validating it and checking for duplicates. If the
// Precommit was accepted and inserted, then it return true, otherwise it
// returns false.
func (p *Process) insertPrecommit(precommit Precommit) bool {
	if precommit.Height != p.CurrentHeight {
		return false
	}
	if _, ok := p.PrecommitLogs[precommit.Round]; !ok {
		p.PrecommitLogs[precommit.Round] = map[Pid]Precommit{}
	}

	existingPrecommit, ok := p.PrecommitLogs[precommit.Round][precommit.From]
	if ok {
		if !precommit.Equal(&existingPrecommit) {
			// FIXME: Punish the precommitr.
		}
		return false
	}

	p.PrecommitLogs[precommit.Round][precommit.From] = precommit
	return true
}

// stepToPrevoting puts the Process into the Prevoting Step. This will also try
// other methods that might now have passing conditions.
func (p *Process) stepToPrevoting() {
	p.CurrentStep = Prevoting
	p.tryPrecommitUponSufficientPrevotes()
	p.tryPrecommitNilUponSufficientPrevotes()
	p.tryTimeoutPrevoteUponSufficientPrevotes()
}

// stepToPrecommitting puts the Process into the Precommitting Step. This will
// also try other methods that might now have passing conditions.
func (p *Process) stepToPrecommitting() {
	p.CurrentStep = Precommitting
	p.tryPrecommitUponSufficientPrevotes()
}

// checkOnceFlag returns true if the OnceFlag has already been set for the given
// Round. Otherwise, it returns false.
func (p *Process) checkOnceFlag(round Round, flag OnceFlag) bool {
	return p.OnceFlags[round]&flag == flag
}

// setOnceFlag set the OnceFlag for the given Round.
func (p *Process) setOnceFlag(round Round, flag OnceFlag) {
	p.OnceFlags[round] |= flag
}

// A OnceFlag is used to guarantee that events only happen once in any given
// Round.
type OnceFlag uint16

// Enumerate all OnceFlag values.
const (
	OnceFlagTimeoutPrecommitUponSufficientPrecommits = OnceFlag(1)
	OnceFlagTimeoutPrevoteUponSufficientPrevotes     = OnceFlag(2)
	OnceFlagPrecommitUponSufficientPrevotes          = OnceFlag(4)
)
