package state

// A timer handles all timer related operations. It will keep track of the ticks
// that it has seen and let the user know if the timer has expired. It also starts
// updating ticks and issuing expiry when it has been explicitly activated.
type timer struct {
	active           bool
	numTicks         int
	numTicksToExpiry int
}

// NewTimer returns a timer object that is initialized with the number of ticks
// required for expiry of the timer.
func NewTimer(numTicksToExpiry int) timer {
	return timer{
		active:           false,
		numTicks:         0,
		numTicksToExpiry: numTicksToExpiry,
	}
}

// IsActive returns true if the timer is active, false otherwise.
func (timer *timer) IsActive() bool {
	return timer.active
}

// Activate must be called explicitly to start the timer.
func (timer *timer) Activate() {
	timer.active = true
}

// Deactivate deactivates the timer
func (timer *timer) Deactivate() {
	timer.active = false
}

// Tick increments `numTicks` if the timer is active. If number of ticks
// exceeds `numTicksToExpiry`, this function returns true to indicate expiry.
func (timer *timer) Tick() bool {
	if timer.active {
		timer.numTicks++
	}
	return timer.numTicks >= timer.numTicksToExpiry
}

// Reset resets the timer values and deactivates it.
func (timer *timer) Reset() {
	timer.active = false
	timer.numTicks = 0
}

// SetExpiry updates the expiration period.
func (timer *timer) SetExpiry(numTicksToExpiry int) {
	timer.numTicksToExpiry = numTicksToExpiry
}
