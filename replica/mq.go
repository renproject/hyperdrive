package replica

import (
	"fmt"
	"sort"
	"sync"

	"github.com/renproject/hyperdrive/block"
	"github.com/renproject/hyperdrive/process"
	"github.com/renproject/id"
)

// A MessageQueue is used to queue messages, and order them by height. It will
// drop messages that are too far in the future. This helps protect honest
// Replicas from being spammed by malicious Replicas with lots of "future"
// messages in an attempt to waste processing time. It also makes the Process
// easier to analyse, because it can assume that all messages will come in for
// height H before height >H.
//
// The MessageQueue is not expected to protect against malicious Replicas that
// spam lots of "future" rounds in any given height (this must be done
// externally to the MessageQueue), and it is not expected to protect against
// malicious Replicas that spam lots of "past" heights.
type MessageQueue interface {
	Push(process.Message)
	PopUntil(block.Height) []process.Message
	Peek() block.Height
	Len() int
}

type messageQueue struct {
	// mu protects the entire message queue.
	mu *sync.Mutex

	// once is used to make sure that no signatory sends multiple messages with
	// the same type at the same height and round.
	once map[process.MessageType]map[block.Height]map[block.Round]map[id.Signatory]bool

	// queue stores all messages, ordered first by height and then by round.
	queue    []process.Message
	queueMax int
}

// NewMessageQueue returns a new MessageQueue with a maximum capacity. Above
// this capacity, messages with the greater height will be dropped.
func NewMessageQueue(cap int) MessageQueue {
	if cap <= 0 {
		panic(fmt.Sprintf("message queue capacity too low: expected cap>0, got cap=%v", cap))
	}
	return &messageQueue{
		mu:       new(sync.Mutex),
		once:     map[process.MessageType]map[block.Height]map[block.Round]map[id.Signatory]bool{},
		queue:    make([]process.Message, 0, cap),
		queueMax: cap,
	}
}

func (mq *messageQueue) Push(message process.Message) {
	mq.mu.Lock()
	defer mq.mu.Unlock()

	if _, ok := mq.once[message.Type()][message.Height()][message.Round()][message.Signatory()]; ok {
		// Ignore messages that appear in the "once" filter to protecte against
		// malicious duplicates.
		return
	}

	// If the queue is at maximum capacity, then we need to make room for the
	// new message, or ignore the new message.
	if len(mq.queue) >= mq.queueMax {
		if mq.queue[len(mq.queue)-1].Height() < message.Height() {
			// Drop the new message because it is too far in the future.
			return
		}
		if mq.queue[len(mq.queue)-1].Height() == message.Height() {
			if mq.queue[len(mq.queue)-1].Round() <= message.Round() {
				// Drop the new message because it is too far in the future. We
				// use the "or equals" check here to favour messages that are
				// already in the queue.
				return
			}
		}
		// Truncate the queue to make room for the newer message.
		mq.queue = mq.queue[:len(mq.queue)-1]
	}

	// Write the message to the "once" filter to protect against malicious
	// duplicates.
	if _, ok := mq.once[message.Type()]; !ok {
		mq.once[message.Type()] = map[block.Height]map[block.Round]map[id.Signatory]bool{}
	}
	if _, ok := mq.once[message.Type()][message.Height()]; !ok {
		mq.once[message.Type()][message.Height()] = map[block.Round]map[id.Signatory]bool{}
	}
	if _, ok := mq.once[message.Type()][message.Height()][message.Round()]; !ok {
		mq.once[message.Type()][message.Height()][message.Round()] = map[id.Signatory]bool{}
	}
	mq.once[message.Type()][message.Height()][message.Round()][message.Signatory()] = true

	// Find a place to insert the message in the queue.
	insertAt := sort.Search(len(mq.queue), func(i int) bool {
		if message.Height() < mq.queue[i].Height() {
			return true
		}
		if message.Height() > mq.queue[i].Height() {
			return false
		}
		return message.Round() < mq.queue[i].Round()
	})
	mq.queue = append(mq.queue[:insertAt], append([]process.Message{message}, mq.queue[insertAt:]...)...)
}

// PopUntil returns all messages up to, and including, the given height.
func (mq *messageQueue) PopUntil(height block.Height) []process.Message {
	mq.mu.Lock()
	defer mq.mu.Unlock()

	// There is nothing to return if the queue is empty.
	if len(mq.queue) == 0 || mq.queue[0].Height() > height {
		return []process.Message{}
	}

	// Store the beginning and end heights that will be dropped from the queue
	// so that, at the end, we can drop them from the "once" filter.
	beginHeight := mq.queue[0].Height()
	endHeight := height

	// Avoid allocations by returning the front end of the queue.
	n := len(mq.queue)
	for i, message := range mq.queue {
		if message.Height() > height {
			n = i
			break
		}
	}
	messages := mq.queue[:n]
	mq.queue = mq.queue[n:]

	// Drop all space in the filter.
	for h := beginHeight; h < endHeight; h++ {
		delete(mq.once[process.ProposeMessageType], h)
		delete(mq.once[process.PrevoteMessageType], h)
		delete(mq.once[process.PrecommitMessageType], h)
	}
	return messages
}

// Peek returns the smallest height in the queue. If there are no messages in
// the queue, it returns an invalid height.
func (mq *messageQueue) Peek() block.Height {
	mq.mu.Lock()
	defer mq.mu.Unlock()

	if len(mq.queue) == 0 {
		return block.InvalidHeight
	}
	return mq.queue[0].Height()
}

func (mq *messageQueue) Len() int {
	mq.mu.Lock()
	defer mq.mu.Unlock()

	return len(mq.queue)
}
