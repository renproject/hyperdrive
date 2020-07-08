package mq_test

import (
	"math/rand"
	"testing/quick"
	"time"

	"github.com/renproject/hyperdrive/mq"
	"github.com/renproject/hyperdrive/process"
	"github.com/renproject/hyperdrive/process/processutil"

	"github.com/renproject/id"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("MQ", func() {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))

	randomMsg := func(r *rand.Rand, from id.Signatory, height process.Height, round process.Round) interface{} {
		switch r.Int() % 3 {
		case 0:
			// propose msg
			msg := processutil.RandomPropose(r)
			msg.From = from
			msg.Height = height
			msg.Round = round
			return msg
		case 1:
			// prevote msg
			msg := processutil.RandomPrevote(r)
			msg.From = from
			msg.Height = height
			msg.Round = round
			return msg
		case 2:
			// precommit msg
			msg := processutil.RandomPrecommit(r)
			msg.From = from
			msg.Height = height
			msg.Round = round
			return msg
		default:
			panic("this should not happen")
		}
	}

	Context("when we instantiate a new message queue", func() {
		It("should return an empty mq with the given options", func() {
			opts := mq.DefaultOptions()
			queue := mq.New(opts)

			// since the queue is empty, we don't expect any message
			proposeCallback := func(propose process.Propose) {
				Expect(true).ToNot(BeTrue())
			}
			prevoteCallback := func(prevote process.Prevote) {
				Expect(true).ToNot(BeTrue())
			}
			precommitCallback := func(precommit process.Precommit) {
				Expect(true).ToNot(BeTrue())
			}

			n := queue.Consume(
				process.Height(9223372036854775807),
				proposeCallback,
				prevoteCallback,
				precommitCallback,
			)

			Expect(n).To(Equal(0))
		})
	})

	Context("when we can insert new messages", func() {
		Context("when two messages have different heights", func() {
			It("should correctly sort the messages based on height", func() {
				opts := mq.DefaultOptions()
				queue := mq.New(opts)

				loop := func() bool {
					sender := id.NewPrivKey().Signatory()
					lowerHeight := process.Height(r.Int63())
					higherHeight := lowerHeight + 1 + process.Height(r.Intn(100))

					// send msg1
					msg1 := randomMsg(r, sender, lowerHeight, processutil.RandomRound(r))
					switch msg1 := msg1.(type) {
					case process.Propose:
						queue.InsertPropose(msg1)
					case process.Prevote:
						queue.InsertPrevote(msg1)
					case process.Precommit:
						queue.InsertPrecommit(msg1)
					}

					// send msg2
					msg2 := randomMsg(r, sender, higherHeight, processutil.RandomRound(r))
					switch msg2 := msg2.(type) {
					case process.Propose:
						queue.InsertPropose(msg2)
					case process.Prevote:
						queue.InsertPrevote(msg2)
					case process.Precommit:
						queue.InsertPrecommit(msg2)
					}

					// we should first consume msg1 and then msg2
					i := 0
					proposeCallback := func(propose process.Propose) {
						switch i {
						case 0:
							Expect(propose.From.Equal(&sender)).To(BeTrue())
							Expect(propose.Height).To(Equal(lowerHeight))
						case 1:
							Expect(propose.From.Equal(&sender)).To(BeTrue())
							Expect(propose.Height).To(Equal(higherHeight))
						case 2:
							// we have only 2 msgs
							Expect(true).ToNot(BeTrue())
						}

						i++
					}

					prevoteCallback := func(prevote process.Prevote) {
						switch i {
						case 0:
							Expect(prevote.From.Equal(&sender)).To(BeTrue())
							Expect(prevote.Height).To(Equal(lowerHeight))
						case 1:
							Expect(prevote.From.Equal(&sender)).To(BeTrue())
							Expect(prevote.Height).To(Equal(higherHeight))
						case 2:
							// we have only 2 msgs
							Expect(true).ToNot(BeTrue())
						}

						i++
					}

					precommitCallback := func(precommit process.Precommit) {
						switch i {
						case 0:
							Expect(precommit.From.Equal(&sender)).To(BeTrue())
							Expect(precommit.Height).To(Equal(lowerHeight))
						case 1:
							Expect(precommit.From.Equal(&sender)).To(BeTrue())
							Expect(precommit.Height).To(Equal(higherHeight))
						case 2:
							// we have only 2 msgs
							Expect(true).ToNot(BeTrue())
						}

						i++
					}

					// cannot consume msgs of height less than lowerHeight
					evenLowerHeight := lowerHeight - 1 - process.Height(r.Intn(100))
					n := queue.Consume(evenLowerHeight, proposeCallback, prevoteCallback, precommitCallback)
					Expect(n).To(Equal(0))
					Expect(i).To(Equal(0))

					// consume all messages
					n = queue.Consume(higherHeight, proposeCallback, prevoteCallback, precommitCallback)
					Expect(n).To(Equal(2))
					Expect(i).To(Equal(2))

					return true
				}
				Expect(quick.Check(loop, nil)).To(Succeed())
			})
		})

		Context("when two messages have the same height", func() {
			It("should correctly sort the messages based on round", func() {
				opts := mq.DefaultOptions()
				queue := mq.New(opts)

				loop := func() bool {
					sender := id.NewPrivKey().Signatory()
					height := process.Height(r.Int63())
					// at the most 20 rounds
					rounds := make([]process.Round, 1+r.Intn(20))
					for t := 0; t < cap(rounds); t++ {
						rounds[t] = process.Round(t)
					}

					for t := range rounds {
						msg := randomMsg(r, sender, height, rounds[t])
						switch msg := msg.(type) {
						case process.Propose:
							queue.InsertPropose(msg)
						case process.Prevote:
							queue.InsertPrevote(msg)
						case process.Precommit:
							queue.InsertPrecommit(msg)
						}
					}

					// we should first consume msg1 and then msg2
					t := 0
					proposeCallback := func(propose process.Propose) {
						if t > cap(rounds) {
							Expect(true).ToNot(BeTrue())
						}

						Expect(propose.From.Equal(&sender)).To(BeTrue())
						Expect(propose.Height).To(Equal(height))
						Expect(propose.Round).To(Equal(rounds[t]))

						t++
					}

					prevoteCallback := func(prevote process.Prevote) {
						if t > cap(rounds) {
							Expect(true).ToNot(BeTrue())
						}

						Expect(prevote.From.Equal(&sender)).To(BeTrue())
						Expect(prevote.Height).To(Equal(height))
						Expect(prevote.Round).To(Equal(rounds[t]))

						t++
					}

					precommitCallback := func(precommit process.Precommit) {
						if t > cap(rounds) {
							Expect(true).ToNot(BeTrue())
						}

						Expect(precommit.From.Equal(&sender)).To(BeTrue())
						Expect(precommit.Height).To(Equal(height))
						Expect(precommit.Round).To(Equal(rounds[t]))

						t++
					}

					// cannot consume msgs of height less than lowerHeight
					lowerHeight := height - 1 - process.Height(r.Intn(100))
					n := queue.Consume(lowerHeight, proposeCallback, prevoteCallback, precommitCallback)
					Expect(n).To(Equal(0))
					Expect(t).To(Equal(0))

					// consume all messages
					n = queue.Consume(height, proposeCallback, prevoteCallback, precommitCallback)
					Expect(n).To(Equal(cap(rounds)))
					Expect(t).To(Equal(cap(rounds)))

					return true
				}
				Expect(quick.Check(loop, nil)).To(Succeed())
			})
		})

		Context("when messages with different heights and rounds are inserted", func() {
			It("should correctly sort the messages, first by height, then by round", func() {
				opts := mq.DefaultOptions()
				queue := mq.New(opts)

				loop := func() bool {
					sender := id.NewPrivKey().Signatory()
					// at the most 20 heights and rounds in increasing order
					heights := make([]process.Height, 1+r.Intn(10))
					nextHeight := 1
					nextRound := 0
					for s := 0; s < cap(heights); s++ {
						nextHeight = nextHeight + r.Intn(10)
						heights[s] = process.Height(nextHeight)
					}
					rounds := make([]process.Round, 1+r.Intn(10))
					for t := 0; t < cap(rounds); t++ {
						nextRound = nextRound + r.Intn(10)
						rounds[t] = process.Round(nextRound)
					}

					// append all messages and shuffle them
					msgsCount := cap(heights) * cap(rounds)
					msgs := make([]interface{}, 0, msgsCount)
					for s := range heights {
						for t := range rounds {
							msg := randomMsg(r, sender, heights[s], rounds[t])
							msgs = append(msgs, msg)
						}
					}
					r.Shuffle(len(msgs), func(i, j int) { msgs[i], msgs[j] = msgs[j], msgs[i] })

					// insert all msgs
					for _, msg := range msgs {
						switch msg := msg.(type) {
						case process.Propose:
							queue.InsertPropose(msg)
						case process.Prevote:
							queue.InsertPrevote(msg)
						case process.Precommit:
							queue.InsertPrecommit(msg)
						}
					}

					// we should first consume msg1 and then msg2
					prevHeight := process.Height(-1)
					prevRound := process.Round(-1)
					i := 0
					proposeCallback := func(propose process.Propose) {
						// if we're starting an increased height
						if propose.Height > prevHeight {
							prevRound = process.Round(-1)
						}

						if i > msgsCount {
							Expect(true).ToNot(BeTrue())
						}

						Expect(propose.From.Equal(&sender)).To(BeTrue())
						Expect(propose.Height >= prevHeight).To(BeTrue())
						Expect(propose.Round >= prevRound).To(BeTrue())

						prevHeight = propose.Height
						prevRound = propose.Round

						i++
					}

					prevoteCallback := func(prevote process.Prevote) {
						// if we're starting an increased height
						if prevote.Height > prevHeight {
							prevRound = process.Round(-1)
						}

						if i > msgsCount {
							Expect(true).ToNot(BeTrue())
						}

						Expect(prevote.From.Equal(&sender)).To(BeTrue())
						Expect(prevote.Height >= prevHeight).To(BeTrue())
						Expect(prevote.Round >= prevRound).To(BeTrue())

						prevHeight = prevote.Height
						prevRound = prevote.Round

						i++
					}

					precommitCallback := func(precommit process.Precommit) {
						// if we're starting an increased height
						if precommit.Height > prevHeight {
							prevRound = process.Round(-1)
						}

						if i > msgsCount {
							Expect(true).ToNot(BeTrue())
						}

						Expect(precommit.From.Equal(&sender)).To(BeTrue())
						Expect(precommit.Height >= prevHeight).To(BeTrue())
						Expect(precommit.Round >= prevRound).To(BeTrue())

						prevHeight = precommit.Height
						prevRound = precommit.Round

						i++
					}

					// cannot consume msgs of height less than lowerHeight
					lowerHeight := heights[0] - 1
					n := queue.Consume(lowerHeight, proposeCallback, prevoteCallback, precommitCallback)
					Expect(n).To(Equal(0))
					Expect(i).To(Equal(0))

					// consume all messages
					n = queue.Consume(heights[len(heights)-1], proposeCallback, prevoteCallback, precommitCallback)
					Expect(n).To(Equal(msgsCount))
					Expect(i).To(Equal(msgsCount))

					return true
				}
				Expect(quick.Check(loop, nil)).To(Succeed())
			})
		})
	})

	Context("when we have reached the queue's max capacity", func() {
		It("trivial case when max capacity is 1", func() {
			loop := func() bool {
				opts := mq.DefaultOptions().WithMaxCapacity(1)
				queue := mq.New(opts)

				// insert a msg
				originalSender := id.NewPrivKey().Signatory()
				originalMsg := processutil.RandomPropose(r)
				originalMsg.From = originalSender
				originalMsg.Height = process.Height(1)
				originalMsg.Round = process.Round(1)
				queue.InsertPropose(originalMsg)

				// any message in height > 1 or (height = 1 || round > 1) will be dropped
				// since every sender has a separate max capacity sized queue
				// this msg should be consumed
				msg := processutil.RandomPropose(r)
				msg.From = id.NewPrivKey().Signatory()
				msg.Height = process.Height(1)
				msg.Round = process.Round(2)
				queue.InsertPropose(msg)

				// so consuming will only return the first msg
				proposeCallback := func(propose process.Propose) {}
				n := queue.Consume(process.Height(1), proposeCallback, nil, nil)
				Expect(n).To(Equal(2))

				// re-insert the original msg
				queue.InsertPropose(originalMsg)

				// any message in height > 1 or (height = 1 || round > 1) will be dropped
				// since this msg has the same original sender, the max capacity is
				// applicable and this msg is dropped
				msg = processutil.RandomPropose(r)
				msg.From = originalSender
				msg.Height = process.Height(1)
				msg.Round = process.Round(2)
				queue.InsertPropose(msg)

				// so consuming will only return the original msg
				proposeCallback = func(propose process.Propose) {
					Expect(propose.Round).To(Equal(originalMsg.Round))
					Expect(propose.From).To(Equal(originalSender))
				}
				n = queue.Consume(process.Height(1), proposeCallback, nil, nil)
				Expect(n).To(Equal(1))

				// re-insert the original msg
				queue.InsertPropose(originalMsg)

				// any message in height <= 1 or (height = 1 && round < 1) will drop
				// the original msg
				msg = processutil.RandomPropose(r)
				msg.From = originalSender
				msg.Height = process.Height(1)
				msg.Round = process.Round(0)
				queue.InsertPropose(msg)

				// so consuming will only return the new msg
				proposeCallback = func(propose process.Propose) {
					Expect(propose.Round).To(Equal(msg.Round))
					Expect(propose.From).To(Equal(originalSender))
				}
				n = queue.Consume(process.Height(1), proposeCallback, nil, nil)
				Expect(n).To(Equal(1))

				return true
			}
			Expect(quick.Check(loop, nil)).To(Succeed())
		})

		It("should drop the excess messages", func() {
			loop := func() bool {
				// max capacity
				c := 5 + r.Intn(20)
				opts := mq.DefaultOptions().WithMaxCapacity(c)
				queue := mq.New(opts)

				// construct msgs
				// more messages than the queue's capacity
				// msgsCount > c
				sender := id.NewPrivKey().Signatory()
				height := process.Height(1)
				msgsCount := c + 5 + r.Intn(20)
				rounds := make([]process.Round, msgsCount)
				msgs := make([]interface{}, msgsCount)
				for i := range rounds {
					rounds[i] = process.Round(i)
					msgs[i] = randomMsg(r, sender, height, rounds[i])
				}

				// insert all msgs
				for _, msg := range msgs {
					switch msg := msg.(type) {
					case process.Propose:
						queue.InsertPropose(msg)
					case process.Prevote:
						queue.InsertPrevote(msg)
					case process.Precommit:
						queue.InsertPrecommit(msg)
					}
				}

				// at the end of insertion, only the lowest round msgs should be in
				// the queue. The ones above the capacity will have been dropped
				// msg.Round < c
				// total msgs consumed = total capacity of queue
				i := 0
				maxMsgRound := process.Round(c)
				proposeCallback := func(propose process.Propose) {
					if i > msgsCount {
						Expect(true).ToNot(BeTrue())
					}

					Expect(propose.Round < maxMsgRound).To(BeTrue())

					i++
				}

				prevoteCallback := func(prevote process.Prevote) {
					if i > msgsCount {
						Expect(true).ToNot(BeTrue())
					}

					Expect(prevote.Round < maxMsgRound).To(BeTrue())

					i++
				}

				precommitCallback := func(precommit process.Precommit) {
					if i > msgsCount {
						Expect(true).ToNot(BeTrue())
					}

					Expect(precommit.Round < maxMsgRound).To(BeTrue())

					i++
				}

				n := queue.Consume(height, proposeCallback, prevoteCallback, precommitCallback)
				Expect(n).To(Equal(c))
				Expect(i).To(Equal(c))

				return true
			}
			Expect(quick.Check(loop, nil)).To(Succeed())
		})
	})
})
