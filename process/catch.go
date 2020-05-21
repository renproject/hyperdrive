package process

// A Catcher is used to publish events about potential malicious behaviour by
// other Processes.
type Catcher interface {
	// DidReceiveMessageConflict is called when a new Message is received that
	// conflicts with an existing Message. Messages of the same type are defined
	// to be in conflict when they are from the same Signatory, Height, and
	// Round, but have different contents.
	//
	// For example, when proposing a Block in any given Height and Round, an
	// honest Process should only ever propose one Block. A malicious Process
	// might try to break consensus by proposing two different Blocks, resulting
	// in two conflicting Proposes in the same Height and Round.
	DidReceiveMessageConflict(conflicting, message Message)
}

type catchAndIgnore struct{}

// CatchAndIgnore returns a Catcher that ignores all potentiall malicious
// behaviour. It should only be used during testing, or when Processes are known
// to be honest.
func CatchAndIgnore() Catcher {
	return catchAndIgnore{}
}

// DidReceiveMessageConflict does nothing.
func (catchAndIgnore) DidReceiveMessageConflict(conflicting, message Message) {}
