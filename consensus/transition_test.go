package consensus_test

import (
	"math/rand"
	"testing/quick"
	"time"

	"github.com/renproject/hyperdrive/block"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/renproject/hyperdrive/consensus"
)

var conf = quick.Config{
	MaxCount:      256,
	MaxCountScale: 0,
	Rand:          nil,
	Values:        nil,
}

var _ = Describe("Transition buffer", func() {
	Context("when only given Proposed Transitions", func() {
		It("should enqueue the same number of times as it dequeues for a given height", func() {
			test := func(num uint8, incrementHeight uint8) bool {
				tb := NewTransitionBuffer(20)
				// cannot do (x % 0)
				if incrementHeight == 0 {
					incrementHeight++
				}

				genesis := Proposed{Block: block.Genesis()}
				var height block.Height
				height = 0
				scratch := make(map[block.Height]uint8)

				for i := uint8(0); i < num; i++ {
					if i%incrementHeight == 0 {
						genesis.Height++
						height++
					}
					tb.Enqueue(genesis)
					if _, ok := scratch[height]; !ok {
						scratch[height] = 0
					}
					scratch[height]++
				}

				for k, v := range scratch {
					for i := uint8(0); i < v; i++ {
						_, ok := tb.Dequeue(k)
						Expect(ok).To(Equal(true),
							"Dequeue was empty! should have %v, had %v",
							v, i)
					}
					_, ok := tb.Dequeue(k)
					Expect(ok).To(Equal(false),
						"Dequeue was Not empty when it shouldn't")
				}
				return true
			}
			Expect(quick.Check(test, &conf)).ShouldNot(HaveOccurred())
		})
	})
	Context("when given semi-random Transitions", func() {
		It("should always give you back the most relevant Transition", func() {
			test := func(size uint32, numInputs uint8) bool {
				size = size % 100
				tb := NewTransitionBuffer(size % 67)

				mock := newMock()

				for i := uint8(0); i < numInputs; i++ {
					tb.Enqueue(mock.nextTransition())
				}

				for height, mockTran := range mock.Map {
					tran, ok := tb.Dequeue(height)
					Expect(ok).To(Equal(true))
					switch tranType := tran.(type) {
					case Proposed:
						Expect(mockTran).To(Equal(mockProposed),
							"expected %v, got: %T", show(mockTran), tranType)
					case PreVoted:
						Expect(mockTran).To(Equal(mockPreVoted),
							"expected %v, got: %T", show(mockTran), tranType)
					case PreCommitted:
						Expect(mockTran).To(Equal(mockPreCommitted),
							"expected %v, got: %T", show(mockTran), tranType)
					case TimedOut:
						Expect(mock.GotImmediate).To(Equal(true),
							"expected %v, got: %T", show(mockTran), tranType)
					default:
						Expect(false).To(Equal(true),
							"unexpected Transition type: %T FIXME!", tranType)
					}
				}
				return true
			}
			Expect(quick.Check(test, &conf)).ShouldNot(HaveOccurred())
		})
	})
	Context("when Drop is called", func() {
		It("should remove everything below the given height", func() {
			tb := NewTransitionBuffer(5)

			precom := PreCommitted{}
			precom.Polka.Height = 0
			tb.Enqueue(precom)
			tb.Enqueue(precom)
			precom.Polka.Height = 1
			tb.Enqueue(precom)
			tb.Drop(1)
			tran, ok := tb.Dequeue(0)
			Expect(ok).To(Equal(false), "dequeued type %T", tran)
			_, ok = tb.Dequeue(1)
			Expect(ok).To(Equal(true))
		})
	})
})

type mockInput struct {
	height block.Height
	rnd    *rand.Rand
	// This keeps track of the Transition the TransitionBuffer at the
	// given Height should return with Dequeue
	Map          map[block.Height]mockTran
	GotImmediate bool
}

type mockTran uint8

const (
	mockProposed mockTran = iota
	mockPreVoted
	mockPreCommitted
	mockImmediate
)

func show(tran mockTran) string {
	switch tran {
	case mockProposed:
		return "Proposed"
	case mockPreVoted:
		return "PreVoted"
	case mockPreCommitted:
		return "PreCommitted"
	case mockImmediate:
		return "TimedOut"
	default:
		return "FIXME, Not a mockTran"
	}
}

func newMock() *mockInput {
	return &mockInput{
		height:       0,
		rnd:          rand.New(rand.NewSource(time.Now().UnixNano())),
		Map:          make(map[block.Height]mockTran),
		GotImmediate: false,
	}
}

func (m *mockInput) nextTransition() Transition {
	var rndTransition Transition

	// maybe increase height
	if m.rnd.Intn(6) == 1 {
		m.height = m.height + block.Height(m.rnd.Intn(4))
	}
	// maybe decrease height
	if m.rnd.Intn(6) == 1 {
		tmp := m.height - block.Height(m.rnd.Intn(4))
		if tmp > 0 {
			m.height = tmp
		}
	}

	// pick Transition
	nextTran := mockTran(m.rnd.Intn(3))

	// rarely pick a timout
	if m.rnd.Intn(100) == 1 {
		nextTran = mockImmediate
	}

	switch nextTran {
	case mockProposed:
		gen := Proposed{Block: block.Genesis()}
		gen.Height = m.height
		rndTransition = gen
		// The only time I would ever Dequeue a Propose is if either
		// there already was a Propose or nothing else in the queue at
		// that height
		if _, ok := m.Map[m.height]; !ok {
			m.Map[m.height] = mockProposed
		}
	case mockPreVoted:
		prevote := PreVoted{}
		prevote.Height = m.height
		rndTransition = prevote
		// The only time I would change what I Dequeue to a PreVoted
		// is if a Propose was the last thing in the queue
		if mock, ok := m.Map[m.height]; ok {
			switch mock {
			case mockProposed:
				m.Map[m.height] = mockPreVoted
			default:
			}
		} else {
			m.Map[m.height] = mockPreVoted
		}
	case mockPreCommitted:
		precom := PreCommitted{}
		precom.Polka.Height = m.height
		rndTransition = precom
		// The only time I would change what I Dequeue to a
		// PreCommitted is if either a Propose or a PreVoted
		// was at the top of the queue
		if mock, ok := m.Map[m.height]; ok {
			switch mock {
			case mockProposed:
				m.Map[m.height] = mockPreCommitted
			case mockPreVoted:
				m.Map[m.height] = mockPreCommitted
			default:
			}
		} else {
			m.Map[m.height] = mockPreCommitted
		}
	case mockImmediate:
		rndTransition = TimedOut{Time: time.Now()}
		m.GotImmediate = true
	}

	return rndTransition
}
