package replica_test

import (
	"testing/quick"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/renproject/hyperdrive/block"
	. "github.com/renproject/hyperdrive/replica"
)

var conf = quick.Config{
	MaxCount:      256,
	MaxCountScale: 0,
	Rand:          nil,
	Values:        nil,
}

var _ = Describe("replica TransitionBuffer", func() {
	Context("When only given Proposed Transitions", func() {
		It("Number of Enqueue matches Dequeue for same height", func() {
			test := func(num uint8, incrementHeight uint8) bool {
				tb := NewTransitionBuffer()
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
						Expect(ok).To(Equal(true))
					}
					_, ok := tb.Dequeue(k)
					Expect(ok).To(Equal(false))
				}
				return true
			}
			Expect(quick.Check(test, &conf)).ShouldNot(HaveOccurred())
		})
	})
	Context("When given TimedOut -> Proposed -> Proposed -> PreVoted", func() {
		tb := NewTransitionBuffer()

		tb.Enqueue(TimedOut{Time: time.Now()})
		tb.Enqueue(Proposed{Block: block.Genesis()})
		tb.Enqueue(Proposed{Block: block.Genesis()})
		prevote := PreVoted{}
		prevote.Height = 0
		tb.Enqueue(prevote)
		It("First TimedOut should come out", func() {
			tran, ok := tb.Dequeue(0)
			Expect(ok).To(Equal(true))
			switch tranType := tran.(type) {
			case TimedOut:
			default:
				Expect("TimedOut type").To(Equal(""), "Type is: %T", tranType)
			}
		})
		It("Second is PreVoted", func() {
			tran, ok := tb.Dequeue(0)
			Expect(ok).To(Equal(true))
			switch tranType := tran.(type) {
			case PreVoted:
			default:
				Expect("PreVoted type").To(Equal(""), "Type is: %T", tranType)
			}
		})
		It("Third should have nothing", func() {
			tran, ok := tb.Dequeue(0)
			Expect(ok).To(Equal(false), "dequeued type %T", tran)
		})
	})
	Context("When given PreVoted -> PreVoted -> PreCommitted -> PreCommitted", func() {
		tb := NewTransitionBuffer()

		tb.Enqueue(Proposed{Block: block.Genesis()})
		prevote := PreVoted{}
		prevote.Height = 0
		tb.Enqueue(prevote)
		tb.Enqueue(prevote)
		precom := PreCommitted{}
		precom.Polka.Height = 0
		tb.Enqueue(precom)
		tb.Enqueue(precom)
		It("First and Second should be a PreCommitted", func() {
			tran, ok := tb.Dequeue(0)
			Expect(ok).To(Equal(true))
			switch tranType := tran.(type) {
			case PreCommitted:
			default:
				Expect("PreCommitted type").To(Equal(""),
					"Type is: %T", tranType)
			}
			tran, ok = tb.Dequeue(0)
			Expect(ok).To(Equal(true))
			switch tranType := tran.(type) {
			case PreCommitted:
			default:
				Expect("PreCommitted type").To(Equal(""),
					"Type is: %T", tranType)
			}
		})
		It("Third should have nothing", func() {
			tran, ok := tb.Dequeue(0)
			Expect(ok).To(Equal(false), "dequeued type %t", tran)
		})
	})
	Context("When given Proposed -> PreCommitted -> PreCommitted", func() {
		tb := NewTransitionBuffer()

		tb.Enqueue(Proposed{Block: block.Genesis()})
		precom := PreCommitted{}
		precom.Polka.Height = 0
		tb.Enqueue(precom)
		tb.Enqueue(precom)
		It("First and Second should be a PreCommitted", func() {
			tran, ok := tb.Dequeue(0)
			Expect(ok).To(Equal(true))
			switch tranType := tran.(type) {
			case PreCommitted:
			default:
				Expect("PreCommitted type").To(Equal(""),
					"Type is: %T", tranType)
			}
			tran, ok = tb.Dequeue(0)
			Expect(ok).To(Equal(true))
			switch tranType := tran.(type) {
			case PreCommitted:
			default:
				Expect("PreCommitted type").To(Equal(""),
					"Type is: %T", tranType)
			}
		})
		It("Fourth should have nothing", func() {
			tran, ok := tb.Dequeue(0)
			Expect(ok).To(Equal(false), "dequeued type %t", tran)
		})
	})
	Context("When given Transitions in incorrect order, panics", func() {
		It("Should panic when PreCommitted -> Proposed", func() {
			tb := NewTransitionBuffer()

			precom := PreCommitted{}
			precom.Polka.Height = 0
			tb.Enqueue(precom)
			Expect(func() {
				tb.Enqueue(Proposed{Block: block.Genesis()})
			}).Should(Panic())
			_, ok := tb.Dequeue(0)
			Expect(ok).To(Equal(true))
			tran, ok := tb.Dequeue(0)
			Expect(ok).To(Equal(false), "dequeued type %T", tran)
		})
		It("Should panic when PreVoted -> Proposed", func() {
			tb := NewTransitionBuffer()

			prevote := PreVoted{}
			prevote.Height = 0
			tb.Enqueue(prevote)
			Expect(func() {
				tb.Enqueue(Proposed{Block: block.Genesis()})
			}).Should(Panic())
			_, ok := tb.Dequeue(0)
			Expect(ok).To(Equal(true))
			tran, ok := tb.Dequeue(0)
			Expect(ok).To(Equal(false), "dequeued type %T", tran)
		})
		It("Should panic when PreCommitted -> PreVoted", func() {
			tb := NewTransitionBuffer()

			precom := PreCommitted{}
			precom.Polka.Height = 0
			tb.Enqueue(precom)
			Expect(func() {
				prevote := PreVoted{}
				prevote.Height = 0
				tb.Enqueue(prevote)
			}).Should(Panic())
			_, ok := tb.Dequeue(0)
			Expect(ok).To(Equal(true))
			tran, ok := tb.Dequeue(0)
			Expect(ok).To(Equal(false), "dequeued type %T", tran)
		})
	})
	Context("When Drop is called", func() {
		It("Should remove everything below the given height", func() {
			tb := NewTransitionBuffer()

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

// type mockInput struct {
// 	height block.Height
// 	rnd    *rand.Rand
// 	state  mockState
// 	Map    map[block.Height][]mockState
// }

// type mockState uint8

// const (
// 	ProposedState mockState = iota
// 	PreVotedState
// 	PreCommittedState
// 	Immediate
// )

// func newMock() *mockInput {
// 	return &mockInput{
// 		height: 0,
// 		rnd:    rand.New(rand.NewSource(time.Now().UnixNano())),
// 		state:  ProposedState,
// 		Map:    make(map[block.Height][]mockState),
// 	}
// }

// func (m *mockInput) nextTransition() Transition {
// 	var rndTransition Transition
// 	mState := m.state

// 	// maybe increment height
// 	if m.rnd.Intn(6) == 1 {
// 		m.height++
// 	}

// 	// maybe change state
// 	if m.rnd.Intn(3) == 1 {
// 		if m.state == 2 {
// 			m.state = 0
// 		} else {
// 			m.state++
// 		}
// 	}

// 	switch m.state {
// 	case ProposedState:
// 		gen := Proposed{Block: block.Genesis()}
// 		gen.Height = m.height
// 		rndTransition = gen
// 	case PreVotedState:
// 		prevote := PreVoted{}
// 		prevote.Height = m.height
// 		rndTransition = prevote
// 	case PreCommittedState:
// 		precom := PreCommitted{}
// 		precom.Polka.Height = m.height
// 		rndTransition = precom
// 	}

// 	// maybe send other message
// 	if m.rnd.Intn(6) == 1 {
// 		rndTransition = TimedOut{Time: time.Now()}
// 		mState = Immediate
// 	}

// 	// keep track of what we did
// 	if _, ok := m.Map[m.height]; !ok {
// 		states := make([]mockState, 1)
// 		states[0] = mState
// 		m.Map[m.height] = states
// 	} else {
// 		states := m.Map[m.height]
// 		states = append(states, mState)
// 	}

// 	return rndTransition
// }
