package mq

import (
	"fmt"
	"sort"

	"github.com/renproject/hyperdrive/process"
	"github.com/renproject/id"
)

// A MessageQueue is used to sort incoming messages by their height and round,
// where messages with lower heights/rounds are found at the beginning of the
// queue. Every sender, identified by their pid, has their own dedicated queue
// with its own dedicated maximum capacity. This limits how far in the future
// the MessageQueue will buffer messages, to prevent running out of memory.
// However, this also means that explicit resynchronisation is needed, because
// not all messages that are received are guaranteed to be kept. MessageQueues
// do not handle de-duplication, and are not safe for concurrent use.
type MessageQueue struct {
	opts        Options
	queuesByPid map[id.Signatory][]interface{}
}

// New returns an empty MessageQueue.
func New(opts Options) MessageQueue {
	return MessageQueue{
		opts:        opts,
		queuesByPid: make(map[id.Signatory][]interface{}),
	}
}

// Consume Propose, Prevote, and Precommit messages from the MessageQueue that
// have heights up to (and including) the given height. The appropriate callback
// will be called for every message that is consumed. All consumed messages will
// be dropped from the MessageQueue.
func (mq *MessageQueue) Consume(h process.Height, propose func(process.Propose), prevote func(process.Prevote), precommit func(process.Precommit)) (n int) {
	for from, q := range mq.queuesByPid {
		for len(q) > 0 {
			if q[0] == nil || height(q[0]) > h {
				break
			}
			switch msg := q[0].(type) {
			case process.Propose:
				propose(msg)
			case process.Prevote:
				prevote(msg)
			case process.Precommit:
				precommit(msg)
			}
			n++
			q = q[1:]
		}
		mq.queuesByPid[from] = q
	}
	return
}

// InsertPropose message into the MessageQueue. This method assumes that the
// sender has already been authenticated and filtered.
func (mq *MessageQueue) InsertPropose(propose process.Propose) {
	mq.insert(propose)
}

// InsertPrevote message into the MessageQueue. This method assumes that the
// sender has already been authenticated and filtered.
func (mq *MessageQueue) InsertPrevote(prevote process.Prevote) {
	mq.insert(prevote)
}

// InsertPrecommit message into the MessageQueue. This method assumes that the
// sender has already been authenticated and filtered.
func (mq *MessageQueue) InsertPrecommit(precommit process.Precommit) {
	mq.insert(precommit)
}

func (mq *MessageQueue) insert(msg interface{}) {
	// Initialise the queue for the sender of the message, to avoid nil-pointer
	// errors. This makes the assumption that messages that have not already
	// passed authentication checks will not be placed into the MessageQueue.
	msgFrom := from(msg)
	if _, ok := mq.queuesByPid[msgFrom]; !ok {
		mq.queuesByPid[msgFrom] = make([]interface{}, mq.opts.MaxCapacity)
	}

	// Load the queue from the map, and defer saving it back to the map.
	q := mq.queuesByPid[msgFrom]
	defer func() { mq.queuesByPid[msgFrom] = q }()

	// Find the index at which the message should be inserted to maintain
	// height/round ordering.
	msgHeight := height(msg)
	msgRound := round(msg)
	insertAt := sort.Search(len(q), func(i int) bool {
		if q[i] == nil {
			return true
		}

		height := height(q[i])
		round := round(q[i])
		return height > msgHeight || (height == msgHeight && round > msgRound)
	})

	// Insert into the slice using the trick described at
	// https://github.com/golang/go/wiki/SliceTricks (which minimises
	// allocations and copying).
	q = append(q, nil)
	copy(q[insertAt+1:], q[insertAt:])
	q[insertAt] = msg

	// If the queue for this sender has exceeded its maximum capacity, then we
	// drop excess elements. This protects against adversaries that might seek
	// to cause an OOM by sending messages "from the far future".
	if len(q) > mq.opts.MaxCapacity {
		q = q[:mq.opts.MaxCapacity]
	}
}

func height(msg interface{}) process.Height {
	switch msg := msg.(type) {
	case process.Propose:
		return msg.Height
	case process.Prevote:
		return msg.Height
	case process.Precommit:
		return msg.Height
	default:
		panic(fmt.Errorf("non-exhaustive pattern: %T", msg))
	}
}

func round(msg interface{}) process.Round {
	switch msg := msg.(type) {
	case process.Propose:
		return msg.Round
	case process.Prevote:
		return msg.Round
	case process.Precommit:
		return msg.Round
	default:
		panic(fmt.Errorf("non-exhaustive pattern: %T", msg))
	}
}

func from(msg interface{}) id.Signatory {
	switch msg := msg.(type) {
	case process.Propose:
		return msg.From
	case process.Prevote:
		return msg.From
	case process.Precommit:
		return msg.From
	default:
		panic(fmt.Errorf("non-exhaustive pattern: %T", msg))
	}
}
