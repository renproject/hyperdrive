package replica_test

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/renproject/hyperdrive/process"
	"github.com/renproject/hyperdrive/process/processutil"
	"github.com/renproject/hyperdrive/replica"
	"github.com/renproject/hyperdrive/timer"
	"github.com/renproject/id"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

type Message struct {
	to    int
	value interface{}
}

// TODO: Implement the other scenarios.
//
// 1. 3F+1 replicas online (done)
// 2. 2F+1 replicas online (done)
// 3. 3F+1 replicas online, and F replicas slowly go offline (wip)
// 4. 3F+1 replicas online, and F replicas slowly go offline and then come back online
// 5. 3F+1 replicas online, and F replicas are behaving maliciously
//    - Proposal: malformed, missing, fork-attempt, out-of-turn
//    - Prevote: yes-to-malformed, no-to-valid, missing, fork-attempt, out-of-turn
//    - Precommit: yes-to-malformed, no-to-valid, missing, fork-attempt, out-of-turn

var _ = Describe("Replica", func() {
	Context("with 3f+1 replicas online", func() {
		It("should be able to reach consensus", func() {
			// randomness seed
			rSeed := time.Now().UnixNano()
			r := rand.New(rand.NewSource(rSeed))

			// f is the maximum no. of adversaries
			// n is the number of honest replicas online
			// h is the target minimum consensus height
			f := 10 + r.Intn(30)
			n := 3*f + 1
			targetHeight := process.Height(30)

			// commits from replicas
			commits := make(map[int]map[process.Height]process.Value)

			// setup private keys for the replicas
			// and their signatories
			privKeys := make([]*id.PrivKey, n)
			signatories := make([]id.Signatory, n)
			for i := range privKeys {
				privKeys[i] = id.NewPrivKey()
				signatories[i] = privKeys[i].Signatory()
				commits[i] = make(map[process.Height]process.Value)
			}

			// every replica sends this signal when they reach the target
			// consensus height
			completionSignal := make(chan bool, n)

			// slice of messages to be broadcasted between replicas.
			// messages from mq are popped and handled whenever mqSignal is signalled
			mq := []Message{}
			mqSignal := make(chan struct{}, n*n)
			mqSignal <- struct{}{}

			// build replicas
			replicas := make([]*replica.Replica, n)
			for i := range replicas {
				replicaIndex := i

				replicas[i] = replica.New(
					replica.DefaultOptions(),
					signatories[i],
					signatories,
					// Proposer
					processutil.MockProposer{
						MockValue: func() process.Value {
							v := processutil.RandomGoodValue(r)
							return v
						},
					},
					// Validator
					processutil.MockValidator{
						MockValid: func(process.Value) bool {
							return true
						},
					},
					// Committer
					processutil.CommitterCallback{
						Callback: func(height process.Height, value process.Value) {
							// add commit to the commits map
							commits[replicaIndex][height] = value
							// signal for completion if this is the target height
							if height == targetHeight {
								completionSignal <- true
							}
						},
					},
					// Catcher
					nil,
					// Broadcaster
					processutil.BroadcasterCallbacks{
						BroadcastProposeCallback: func(propose process.Propose) {
							for j := 0; j < n; j++ {
								mq = append(mq, Message{
									to:    j,
									value: propose,
								})
							}
						},
						BroadcastPrevoteCallback: func(prevote process.Prevote) {
							for j := 0; j < n; j++ {
								mq = append(mq, Message{
									to:    j,
									value: prevote,
								})
							}
						},
						BroadcastPrecommitCallback: func(precommit process.Precommit) {
							for j := 0; j < n; j++ {
								mq = append(mq, Message{
									to:    j,
									value: precommit,
								})
							}
						},
					},
					// Flusher
					func() {
						mqSignal <- struct{}{}
					},
				)
			}

			// Create a global context that can be used to cancel the running of all
			// replicas.
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			// From the global context, create per-replica contexts that can be used to
			// cancel replicas independently of one another. This is useful when we want
			// to simulate replicas "crashing" and "restarting" during consensus.
			replicaCtxs, replicaCtxCancels := make([]context.Context, n), make([]context.CancelFunc, n)
			for i := range replicaCtxs {
				replicaCtxs[i], replicaCtxCancels[i] = context.WithCancel(ctx)
			}

			// Run all of the replicas in independent background goroutines.
			for i := range replicas {
				go replicas[i].Run(replicaCtxs[i])
			}

			completion := func() {
				// cancel the replica contexts
				for i := range replicaCtxs {
					replicaCtxCancels[i]()
				}

				// ensure that all replicas have the same commits
				referenceCommits := commits[0]
				for j := 0; j < n; j++ {
					for h := process.Height(1); h <= targetHeight; {
						Expect(commits[j][h]).To(Equal(referenceCommits[h]))
						h++
					}
				}
			}

			time.Sleep(100 * time.Millisecond)
			for range mqSignal {
				// this means the target consensus has been reached on every replica
				if len(completionSignal) == n {
					completion()
					break
				}

				if len(mq) == 0 {
					continue
				}

				// pop the first message
				m := mq[0]
				mq = mq[1:]

				// ignore if its a nil message
				if m.value == nil {
					continue
				}

				// handle the message
				replica := replicas[m.to]
				switch value := m.value.(type) {
				case process.Propose:
					replica.Propose(context.Background(), value)
				case process.Prevote:
					replica.Prevote(context.Background(), value)
				case process.Precommit:
					replica.Precommit(context.Background(), value)
				default:
					panic(fmt.Errorf("non-exhaustive pattern: message.value has type %T", value))
				}
			}
		})
	})

	Context("with 2f+1 replicas online", func() {
		It("should be able to reach consensus", func() {
			// randomness seed
			rSeed := time.Now().UnixNano()
			r := rand.New(rand.NewSource(rSeed))

			// f is the maximum no. of adversaries
			// n is the number of honest replicas online
			// h is the target minimum consensus height
			f := 10 + r.Intn(30)
			n := 2*f + 1
			targetHeight := process.Height(30)

			// commits from replicas
			commits := make(map[int]map[process.Height]process.Value)

			// setup private keys for the replicas
			// and their signatories
			privKeys := make([]*id.PrivKey, 3*f+1)
			signatories := make([]id.Signatory, 3*f+1)
			for i := range privKeys {
				privKeys[i] = id.NewPrivKey()
				signatories[i] = privKeys[i].Signatory()
				commits[i] = make(map[process.Height]process.Value)
			}

			// every replica sends this signal when they reach the target
			// consensus height
			completionSignal := make(chan bool, n)

			// slice of messages to be broadcasted between replicas.
			// messages from mq are popped and handled whenever mqSignal is signalled
			mq := []Message{}
			mqSignal := make(chan struct{}, n*n)
			mqSignal <- struct{}{}

			// build replicas
			replicas := make([]*replica.Replica, n)
			for i := range replicas {
				replicaIndex := i

				replicas[i] = replica.New(
					replica.DefaultOptions(),
					signatories[i],
					signatories,
					// Proposer
					processutil.MockProposer{
						MockValue: func() process.Value {
							v := processutil.RandomGoodValue(r)
							return v
						},
					},
					// Validator
					processutil.MockValidator{
						MockValid: func(process.Value) bool {
							return true
						},
					},
					// Committer
					processutil.CommitterCallback{
						Callback: func(height process.Height, value process.Value) {
							// add commit to the commits map
							commits[replicaIndex][height] = value
							// signal for completion if this is the target height
							if height == targetHeight {
								completionSignal <- true
							}
						},
					},
					// Catcher
					nil,
					// Broadcaster
					processutil.BroadcasterCallbacks{
						BroadcastProposeCallback: func(propose process.Propose) {
							for j := 0; j < n; j++ {
								mq = append(mq, Message{
									to:    j,
									value: propose,
								})
							}
						},
						BroadcastPrevoteCallback: func(prevote process.Prevote) {
							for j := 0; j < n; j++ {
								mq = append(mq, Message{
									to:    j,
									value: prevote,
								})
							}
						},
						BroadcastPrecommitCallback: func(precommit process.Precommit) {
							for j := 0; j < n; j++ {
								mq = append(mq, Message{
									to:    j,
									value: precommit,
								})
							}
						},
					},
					// Flusher
					func() {
						mqSignal <- struct{}{}
					},
				)
			}

			// Create a global context that can be used to cancel the running of all
			// replicas.
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			// From the global context, create per-replica contexts that can be used to
			// cancel replicas independently of one another. This is useful when we want
			// to simulate replicas "crashing" and "restarting" during consensus.
			replicaCtxs, replicaCtxCancels := make([]context.Context, n), make([]context.CancelFunc, n)
			for i := range replicaCtxs {
				replicaCtxs[i], replicaCtxCancels[i] = context.WithCancel(ctx)
			}

			// Run all of the replicas in independent background goroutines.
			for i := range replicas {
				go replicas[i].Run(replicaCtxs[i])
			}

			completion := func() {
				// cancel the replica contexts
				for i := range replicaCtxs {
					replicaCtxCancels[i]()
				}

				// ensure that all replicas have the same commits
				referenceCommits := commits[0]
				for j := 0; j < n; j++ {
					for h := process.Height(1); h <= targetHeight; {
						Expect(commits[j][h]).To(Equal(referenceCommits[h]))
						h++
					}
				}
			}

			time.Sleep(100 * time.Millisecond)
			for range mqSignal {
				// this means the target consensus has been reached on every replica
				if len(completionSignal) == n {
					completion()
					break
				}

				if len(mq) == 0 {
					continue
				}

				// pop the first message
				m := mq[0]
				mq = mq[1:]

				// handle the message
				replica := replicas[m.to]
				switch value := m.value.(type) {
				case process.Propose:
					replica.Propose(context.Background(), value)
				case process.Prevote:
					replica.Prevote(context.Background(), value)
				case process.Precommit:
					replica.Precommit(context.Background(), value)
				default:
					panic(fmt.Errorf("non-exhaustive pattern: message.value has type %T", value))
				}
			}
		})
	})

	Context("with 3f+1 replicas online, f replicas slowly go offline", func() {
		It("should be able to reach consensus", func() {
			// randomness seed
			rSeed := time.Now().UnixNano()
			r := rand.New(rand.NewSource(rSeed))

			// f is the maximum no. of adversaries
			// n is the number of honest replicas online
			// h is the target minimum consensus height
			f := 3
			n := 3*f + 1
			targetHeight := process.Height(12)

			// commits from replicas
			commits := make(map[int]map[process.Height]process.Value)

			// setup private keys for the replicas
			// and their signatories
			privKeys := make([]*id.PrivKey, n)
			signatories := make([]id.Signatory, n)
			for i := range privKeys {
				privKeys[i] = id.NewPrivKey()
				signatories[i] = privKeys[i].Signatory()
				commits[i] = make(map[process.Height]process.Value)
			}

			// every replica sends this signal when they reach the target
			// consensus height
			completionSignal := make(chan bool, n)

			// replicas randomly send this signal that kills them
			// this is to simulate a behaviour where f replicas go offline
			killedReplicas := make(map[int]bool)
			killSignal := make(chan int, f)

			// lastKilled stores the last height at which a replica was killed
			// killInterval is the number of blocks progressed after which
			// a replica should be killed
			lastKilled := process.Height(0)
			killInterval := process.Height(2)

			// slice of messages to be broadcasted between replicas.
			// messages from mq are popped and handled whenever mqSignal is signalled
			mq := []Message{}
			mqSignal := make(chan struct{}, n*n)
			mqSignal <- struct{}{}

			// build replicas
			replicas := make([]*replica.Replica, n)
			for i := range replicas {
				replicaIndex := i

				replicas[i] = replica.New(
					replica.DefaultOptions().
						WithTimerOptions(
							timer.DefaultOptions().
								WithTimeout(1*time.Second),
						),
					signatories[i],
					signatories,
					// Proposer
					processutil.MockProposer{
						MockValue: func() process.Value {
							v := processutil.RandomGoodValue(r)
							return v
						},
					},
					// Validator
					processutil.MockValidator{
						MockValid: func(process.Value) bool {
							return true
						},
					},
					// Committer
					processutil.CommitterCallback{
						Callback: func(height process.Height, value process.Value) {
							// add commit to the commits map
							commits[replicaIndex][height] = value

							// signal for completion if this is the target height
							if height == targetHeight {
								completionSignal <- true
								return
							}

							// kill this replica if we've progressed the killInterval number
							// of blocks, and there are more than 2f+1 replica alive
							if height-lastKilled > killInterval {
								if len(killedReplicas) < f {
									killSignal <- replicaIndex
									lastKilled = height
								}
							}
						},
					},
					// Catcher
					nil,
					// Broadcaster
					processutil.BroadcasterCallbacks{
						BroadcastProposeCallback: func(propose process.Propose) {
							for j := 0; j < n; j++ {
								if _, ok := killedReplicas[j]; !ok {
									mq = append(mq, Message{
										to:    j,
										value: propose,
									})
								}
							}
						},
						BroadcastPrevoteCallback: func(prevote process.Prevote) {
							for j := 0; j < n; j++ {
								if _, ok := killedReplicas[j]; !ok {
									mq = append(mq, Message{
										to:    j,
										value: prevote,
									})
								}
							}
						},
						BroadcastPrecommitCallback: func(precommit process.Precommit) {
							for j := 0; j < n; j++ {
								if _, ok := killedReplicas[j]; !ok {
									mq = append(mq, Message{
										to:    j,
										value: precommit,
									})
								}
							}
						},
					},
					// Flusher
					func() {
						mqSignal <- struct{}{}
					},
				)
			}

			// Create a global context that can be used to cancel the running of all
			// replicas.
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			// in case the test fails to proceed to completion, we have a signal
			// to mark it as failed
			// this test should take 30 seconds to complete
			failTestSignal := make(chan bool, 1)
			go func() {
				time.Sleep(45 * time.Second)
				failTestSignal <- true
			}()

			// From the global context, create per-replica contexts that can be used to
			// cancel replicas independently of one another. This is useful when we want
			// to simulate replicas "crashing" and "restarting" during consensus.
			replicaCtxs, replicaCtxCancels := make([]context.Context, n), make([]context.CancelFunc, n)
			for i := range replicaCtxs {
				replicaCtxs[i], replicaCtxCancels[i] = context.WithCancel(ctx)
			}

			// Run all of the replicas in independent background goroutines.
			for i := range replicas {
				go replicas[i].Run(replicaCtxs[i])
			}

			completion := func() {
				// cancel the replica contexts
				for i := range replicaCtxs {
					replicaCtxCancels[i]()
				}

				// ensure that all replicas have the same commits
				for i := 0; i < n; i++ {
					if _, ok := killedReplicas[i]; !ok {
						// from the first replica that's alive
						referenceCommits := commits[i]
						for j := i + 1; j < n; j++ {
							for h := process.Height(1); h <= targetHeight; {
								Expect(commits[j][h]).To(Equal(referenceCommits[h]))
								h++
							}
						}
						break
					}
				}
			}

			failTest := func() {
				cancel()
				panic("test failed to complete within the expected timeframe")
			}

			time.Sleep(100 * time.Millisecond)
			isRunning := true
			for isRunning {
				select {
				// if the expected time frame is over, we fail the test
				case _ = <-failTestSignal:
					failTest()
				// else continue watching out for the mqSignal
				case _ = <-mqSignal:
					// this means the target consensus has been reached on every replica
					if len(completionSignal) == 2*f+1 {
						completion()
						isRunning = false
					}

					// kill a replica if there has been a kill signal
					if len(killSignal) > 0 {
						killIndex := <-killSignal
						replicaCtxCancels[killIndex]()
						killedReplicas[killIndex] = true
						continue
					}

					// ignore if there isn't any message
					if len(mq) == 0 {
						continue
					}

					// pop the first message
					m := mq[0]
					mq = mq[1:]

					// ignore if its a nil message
					if m.value == nil {
						continue
					}

					// handle the message
					time.Sleep(5 * time.Millisecond)
					replica := replicas[m.to]
					switch value := m.value.(type) {
					case process.Propose:
						replica.Propose(context.Background(), value)
					case process.Prevote:
						replica.Prevote(context.Background(), value)
					case process.Precommit:
						replica.Precommit(context.Background(), value)
					default:
						panic(fmt.Errorf("non-exhaustive pattern: message.value has type %T", value))
					}
				}
			}
		})
	})

	Context("with 3f+1 replicas online, f replicas behaving maliciously", func() {
		Context("f replicas broadcasting malformed proposals", func() {
			It("should be able to reach consensus", func() {
				// randomness seed
				rSeed := time.Now().UnixNano()
				r := rand.New(rand.NewSource(rSeed))

				// f is the maximum no. of adversaries
				// n is the number of honest replicas online
				// h is the target minimum consensus height
				f := 3
				n := 3*f + 1
				targetHeight := process.Height(12)

				// commits from replicas
				commits := make(map[int]map[process.Height]process.Value)

				// setup private keys for the replicas
				// and their signatories
				privKeys := make([]*id.PrivKey, n)
				signatories := make([]id.Signatory, n)
				for i := range privKeys {
					privKeys[i] = id.NewPrivKey()
					signatories[i] = privKeys[i].Signatory()
					commits[i] = make(map[process.Height]process.Value)
				}

				// every replica sends this signal when they reach the target
				// consensus height
				completionSignal := make(chan bool, n)

				// slice of messages to be broadcasted between replicas.
				// messages from mq are popped and handled whenever mqSignal is signalled
				mq := []Message{}
				mqSignal := make(chan struct{}, n*n)
				mqSignal <- struct{}{}

				// build replicas
				replicas := make([]*replica.Replica, n)
				for i := range replicas {
					replicaIndex := i

					replicas[i] = replica.New(
						replica.DefaultOptions().
							WithTimerOptions(
								timer.DefaultOptions().
									WithTimeout(1*time.Second),
							),
						signatories[i],
						signatories,
						// Proposer
						processutil.MockProposer{
							MockValue: func() process.Value {
								// the first f replicas propose malformed (nil) value
								if replicaIndex < f {
									return process.NilValue
								}
								// the other 2f+1 replicas propose valid values
								v := processutil.RandomGoodValue(r)
								return v
							},
						},
						// Validator
						processutil.MockValidator{
							MockValid: func(value process.Value) bool {
								// a malformed (nil) value is marked invalid
								if value == process.NilValue {
									return false
								}
								// any other value is marked valid
								return true
							},
						},
						// Committer
						processutil.CommitterCallback{
							Callback: func(height process.Height, value process.Value) {
								// add commit to the commits map
								commits[replicaIndex][height] = value

								// signal for completion if this is the target height
								if height == targetHeight {
									completionSignal <- true
								}
							},
						},
						// Catcher
						nil,
						// Broadcaster
						processutil.BroadcasterCallbacks{
							BroadcastProposeCallback: func(propose process.Propose) {
								for j := 0; j < n; j++ {
									mq = append(mq, Message{
										to:    j,
										value: propose,
									})
								}
							},
							BroadcastPrevoteCallback: func(prevote process.Prevote) {
								for j := 0; j < n; j++ {
									mq = append(mq, Message{
										to:    j,
										value: prevote,
									})
								}
							},
							BroadcastPrecommitCallback: func(precommit process.Precommit) {
								for j := 0; j < n; j++ {
									mq = append(mq, Message{
										to:    j,
										value: precommit,
									})
								}
							},
						},
						// Flusher
						func() {
							mqSignal <- struct{}{}
						},
					)
				}

				// Create a global context that can be used to cancel the running of all
				// replicas.
				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()

				// in case the test fails to proceed to completion, we have a signal
				// to mark it as failed
				// this test should take 35 seconds to complete
				failTestSignal := make(chan bool, 1)
				go func() {
					time.Sleep(45 * time.Second)
					failTestSignal <- true
				}()

				// From the global context, create per-replica contexts that can be used to
				// cancel replicas independently of one another. This is useful when we want
				// to simulate replicas "crashing" and "restarting" during consensus.
				replicaCtxs, replicaCtxCancels := make([]context.Context, n), make([]context.CancelFunc, n)
				for i := range replicaCtxs {
					replicaCtxs[i], replicaCtxCancels[i] = context.WithCancel(ctx)
				}

				// Run all of the replicas in independent background goroutines.
				for i := range replicas {
					go replicas[i].Run(replicaCtxs[i])
				}

				completion := func() {
					// cancel the replica contexts
					for i := range replicaCtxs {
						replicaCtxCancels[i]()
					}

					// ensure that all replicas have the same commits
					referenceCommits := commits[0]
					for j := 1; j < n; j++ {
						for h := process.Height(1); h <= targetHeight; {
							Expect(commits[j][h]).To(Equal(referenceCommits[h]))
							h++
						}
					}
				}

				failTest := func() {
					cancel()
					panic("test failed to complete within the expected timeframe")
				}

				time.Sleep(100 * time.Millisecond)
				isRunning := true
				for isRunning {
					select {
					// if the expected time frame is over, we fail the test
					case _ = <-failTestSignal:
						failTest()
						// else continue watching out for the mqSignal
					case _ = <-mqSignal:
						// this means the target consensus has been reached on every replica
						if len(completionSignal) == 3*f+1 {
							completion()
							isRunning = false
						}

						// ignore if there isn't any message
						if len(mq) == 0 {
							continue
						}

						// pop the first message
						m := mq[0]
						mq = mq[1:]

						// ignore if its a nil message
						if m.value == nil {
							continue
						}

						// handle the message
						time.Sleep(5 * time.Millisecond)
						replica := replicas[m.to]
						switch value := m.value.(type) {
						case process.Propose:
							replica.Propose(context.Background(), value)
						case process.Prevote:
							replica.Prevote(context.Background(), value)
						case process.Precommit:
							replica.Precommit(context.Background(), value)
						default:
							panic(fmt.Errorf("non-exhaustive pattern: message.value has type %T", value))
						}
					}
				}
			})
		})

		Context("f replicas running misbehaving proposers and validators", func() {
			It("should be able to reach consensus", func() {
				// randomness seed
				rSeed := time.Now().UnixNano()
				r := rand.New(rand.NewSource(rSeed))

				// f is the maximum no. of adversaries
				// n is the number of honest replicas online
				// h is the target minimum consensus height
				f := 3
				n := 3*f + 1
				targetHeight := process.Height(12)

				// commits from replicas
				commits := make(map[int]map[process.Height]process.Value)

				// setup private keys for the replicas
				// and their signatories
				privKeys := make([]*id.PrivKey, n)
				signatories := make([]id.Signatory, n)
				for i := range privKeys {
					privKeys[i] = id.NewPrivKey()
					signatories[i] = privKeys[i].Signatory()
					commits[i] = make(map[process.Height]process.Value)
				}

				// every replica sends this signal when they reach the target
				// consensus height
				completionSignal := make(chan bool, n)

				// slice of messages to be broadcasted between replicas.
				// messages from mq are popped and handled whenever mqSignal is signalled
				mq := []Message{}
				mqSignal := make(chan struct{}, n*n)
				mqSignal <- struct{}{}

				// build replicas
				replicas := make([]*replica.Replica, n)
				for i := range replicas {
					replicaIndex := i

					replicas[i] = replica.New(
						replica.DefaultOptions().
							WithTimerOptions(
								timer.DefaultOptions().
									WithTimeout(1*time.Second),
							),
						signatories[i],
						signatories,
						// Proposer
						processutil.MockProposer{
							MockValue: func() process.Value {
								// the first f replicas propose malformed (nil) value
								if replicaIndex < f {
									return process.NilValue
								}
								// the other 2f+1 replicas propose valid values
								v := processutil.RandomGoodValue(r)
								return v
							},
						},
						// Validator
						processutil.MockValidator{
							MockValid: func(value process.Value) bool {
								// a misbehaving replica
								if replicaIndex < f {
									if value == process.NilValue {
										return true
									}
									return false
								}

								// an honest replica
								if value == process.NilValue {
									return false
								}
								return true
							},
						},
						// Committer
						processutil.CommitterCallback{
							Callback: func(height process.Height, value process.Value) {
								// add commit to the commits map
								commits[replicaIndex][height] = value

								// signal for completion if this is the target height
								if height == targetHeight {
									completionSignal <- true
								}
							},
						},
						// Catcher
						nil,
						// Broadcaster
						processutil.BroadcasterCallbacks{
							BroadcastProposeCallback: func(propose process.Propose) {
								for j := 0; j < n; j++ {
									mq = append(mq, Message{
										to:    j,
										value: propose,
									})
								}
							},
							BroadcastPrevoteCallback: func(prevote process.Prevote) {
								for j := 0; j < n; j++ {
									mq = append(mq, Message{
										to:    j,
										value: prevote,
									})
								}
							},
							BroadcastPrecommitCallback: func(precommit process.Precommit) {
								for j := 0; j < n; j++ {
									mq = append(mq, Message{
										to:    j,
										value: precommit,
									})
								}
							},
						},
						// Flusher
						func() {
							mqSignal <- struct{}{}
						},
					)
				}

				// Create a global context that can be used to cancel the running of all
				// replicas.
				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()

				// in case the test fails to proceed to completion, we have a signal
				// to mark it as failed
				// this test should take 35 seconds to complete
				failTestSignal := make(chan bool, 1)
				go func() {
					time.Sleep(45 * time.Second)
					failTestSignal <- true
				}()

				// From the global context, create per-replica contexts that can be used to
				// cancel replicas independently of one another. This is useful when we want
				// to simulate replicas "crashing" and "restarting" during consensus.
				replicaCtxs, replicaCtxCancels := make([]context.Context, n), make([]context.CancelFunc, n)
				for i := range replicaCtxs {
					replicaCtxs[i], replicaCtxCancels[i] = context.WithCancel(ctx)
				}

				// Run all of the replicas in independent background goroutines.
				for i := range replicas {
					go replicas[i].Run(replicaCtxs[i])
				}

				completion := func() {
					// cancel the replica contexts
					for i := range replicaCtxs {
						replicaCtxCancels[i]()
					}

					// ensure that all replicas have the same commits
					referenceCommits := commits[f]
					for j := f + 1; j < n; j++ {
						for h := process.Height(1); h <= targetHeight; {
							Expect(commits[j][h]).To(Equal(referenceCommits[h]))
							h++
						}
					}
				}

				failTest := func() {
					cancel()
					panic("test failed to complete within the expected timeframe")
				}

				time.Sleep(100 * time.Millisecond)
				isRunning := true
				for isRunning {
					select {
					// if the expected time frame is over, we fail the test
					case _ = <-failTestSignal:
						failTest()
						// else continue watching out for the mqSignal
					case _ = <-mqSignal:
						// this means the target consensus has been reached on every replica
						if len(completionSignal) == 2*f+1 {
							completion()
							isRunning = false
						}

						// ignore if there isn't any message
						if len(mq) == 0 {
							continue
						}

						// pop the first message
						m := mq[0]
						mq = mq[1:]

						// ignore if its a nil message
						if m.value == nil {
							continue
						}

						// handle the message
						time.Sleep(5 * time.Millisecond)
						replica := replicas[m.to]
						switch value := m.value.(type) {
						case process.Propose:
							replica.Propose(context.Background(), value)
						case process.Prevote:
							replica.Prevote(context.Background(), value)
						case process.Precommit:
							replica.Precommit(context.Background(), value)
						default:
							panic(fmt.Errorf("non-exhaustive pattern: message.value has type %T", value))
						}
					}
				}
			})
		})
	})

	Context("with less than 2f+1 replicas online", func() {
		It("should stall consensus and should not progress", func() {
			// randomness seed
			rSeed := time.Now().UnixNano()
			r := rand.New(rand.NewSource(rSeed))

			// f is the maximum no. of adversaries
			// n is the number of honest replicas online
			// h is the target minimum consensus height
			f := 3
			n := 2 * f

			// setup private keys for the replicas
			// and their signatories
			privKeys := make([]*id.PrivKey, 3*f+1)
			signatories := make([]id.Signatory, 3*f+1)
			for i := range privKeys {
				privKeys[i] = id.NewPrivKey()
				signatories[i] = privKeys[i].Signatory()
			}

			// slice of messages to be broadcasted between replicas.
			// messages from mq are popped and handled whenever mqSignal is signalled
			mq := []Message{}
			mqSignal := make(chan struct{}, n*n)
			mqSignal <- struct{}{}

			// build replicas
			replicas := make([]*replica.Replica, n)
			for i := range replicas {
				replicas[i] = replica.New(
					replica.DefaultOptions(),
					signatories[i],
					signatories,
					// Proposer
					processutil.MockProposer{
						MockValue: func() process.Value {
							v := processutil.RandomGoodValue(r)
							return v
						},
					},
					// Validator
					processutil.MockValidator{
						MockValid: func(process.Value) bool {
							return true
						},
					},
					// Committer
					processutil.CommitterCallback{
						Callback: func(height process.Height, value process.Value) {
							// we don't expect any progress since we are one short of
							// 2f+1 replicas
							panic("consensus should have stalled")
						},
					},
					// Catcher
					nil,
					// Broadcaster
					processutil.BroadcasterCallbacks{
						BroadcastProposeCallback: func(propose process.Propose) {
							for j := 0; j < n; j++ {
								mq = append(mq, Message{
									to:    j,
									value: propose,
								})
							}
						},
						BroadcastPrevoteCallback: func(prevote process.Prevote) {
							for j := 0; j < n; j++ {
								mq = append(mq, Message{
									to:    j,
									value: prevote,
								})
							}
						},
						BroadcastPrecommitCallback: func(precommit process.Precommit) {
							for j := 0; j < n; j++ {
								mq = append(mq, Message{
									to:    j,
									value: precommit,
								})
							}
						},
					},
					// Flusher
					func() {
						mqSignal <- struct{}{}
					},
				)
			}

			// Create a global context that can be used to cancel the running of all
			// replicas.
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			// we expect the consensus to stall, and we wait for the said time
			// before declaring that the expected behaviour was observed
			successTestSignal := make(chan bool, 1)
			go func() {
				time.Sleep(10 * time.Second)
				successTestSignal <- true
			}()

			// From the global context, create per-replica contexts that can be used to
			// cancel replicas independently of one another. This is useful when we want
			// to simulate replicas "crashing" and "restarting" during consensus.
			replicaCtxs, replicaCtxCancels := make([]context.Context, n), make([]context.CancelFunc, n)
			for i := range replicaCtxs {
				replicaCtxs[i], replicaCtxCancels[i] = context.WithCancel(ctx)
			}

			// Run all of the replicas in independent background goroutines.
			for i := range replicas {
				go replicas[i].Run(replicaCtxs[i])
			}

			time.Sleep(100 * time.Millisecond)
			isRunning := true
			for isRunning {
				select {
				case _ = <-successTestSignal:
					isRunning = false
					break
				case _ = <-mqSignal:
					if len(mq) == 0 {
						continue
					}

					// pop the first message
					m := mq[0]
					mq = mq[1:]

					// handle the message
					replica := replicas[m.to]
					switch value := m.value.(type) {
					case process.Propose:
						replica.Propose(context.Background(), value)
					case process.Prevote:
						replica.Prevote(context.Background(), value)
					case process.Precommit:
						replica.Precommit(context.Background(), value)
					default:
						panic(fmt.Errorf("non-exhaustive pattern: message.value has type %T", value))
					}
				}
			}
		})
	})

	Context("with 2f+1 replicas online, and one of them goes offline", func() {
		It("should stall consensus as soon as there are less than 2f+1 online", func() {
			// randomness seed
			rSeed := time.Now().UnixNano()
			r := rand.New(rand.NewSource(rSeed))

			// f is the maximum no. of adversaries
			// n is the number of honest replicas online
			f := 3
			n := 2*f + 1
			intermTargetHeight := process.Height(6)

			// setup private keys for the replicas
			// and their signatories
			privKeys := make([]*id.PrivKey, 3*f+1)
			signatories := make([]id.Signatory, 3*f+1)
			for i := range privKeys {
				privKeys[i] = id.NewPrivKey()
				signatories[i] = privKeys[i].Signatory()
			}

			// a signal to kill one of the replicas
			killSignal := make(chan int, 1)

			// slice of messages to be broadcasted between replicas.
			// messages from mq are popped and handled whenever mqSignal is signalled
			mq := []Message{}
			mqSignal := make(chan struct{}, n*n)
			mqSignal <- struct{}{}

			// build replicas
			replicas := make([]*replica.Replica, n)
			for i := range replicas {
				replicas[i] = replica.New(
					replica.DefaultOptions().
						WithTimerOptions(
							timer.DefaultOptions().
								WithTimeout(1*time.Second),
						),
					signatories[i],
					signatories,
					// Proposer
					processutil.MockProposer{
						MockValue: func() process.Value {
							v := processutil.RandomGoodValue(r)
							return v
						},
					},
					// Validator
					processutil.MockValidator{
						MockValid: func(process.Value) bool {
							return true
						},
					},
					// Committer
					processutil.CommitterCallback{
						Callback: func(height process.Height, value process.Value) {
							// if consensus progresses and any commit above the intermediate
							// target height is found, it is unexpected behaviour
							if height > intermTargetHeight {
								panic("consensus should have stalled")
							}

							// at the intermediate target height, we kill the first replica
							if height == intermTargetHeight {
								killSignal <- 0
								return
							}
						},
					},
					// Catcher
					nil,
					// Broadcaster
					processutil.BroadcasterCallbacks{
						BroadcastProposeCallback: func(propose process.Propose) {
							for j := 0; j < n; j++ {
								mq = append(mq, Message{
									to:    j,
									value: propose,
								})
							}
						},
						BroadcastPrevoteCallback: func(prevote process.Prevote) {
							for j := 0; j < n; j++ {
								mq = append(mq, Message{
									to:    j,
									value: prevote,
								})
							}
						},
						BroadcastPrecommitCallback: func(precommit process.Precommit) {
							for j := 0; j < n; j++ {
								mq = append(mq, Message{
									to:    j,
									value: precommit,
								})
							}
						},
					},
					// Flusher
					func() {
						mqSignal <- struct{}{}
					},
				)
			}

			// Create a global context that can be used to cancel the running of all
			// replicas.
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			// in case the test fails to proceed to completion, we have a signal
			// to mark it as failed
			// this test should take 30 seconds to complete
			successTestSignal := make(chan bool, 1)
			go func() {
				time.Sleep(20 * time.Second)
				successTestSignal <- true
			}()

			// From the global context, create per-replica contexts that can be used to
			// cancel replicas independently of one another. This is useful when we want
			// to simulate replicas "crashing" and "restarting" during consensus.
			replicaCtxs, replicaCtxCancels := make([]context.Context, n), make([]context.CancelFunc, n)
			for i := range replicaCtxs {
				replicaCtxs[i], replicaCtxCancels[i] = context.WithCancel(ctx)
			}

			// Run all of the replicas in independent background goroutines.
			for i := range replicas {
				go replicas[i].Run(replicaCtxs[i])
			}

			time.Sleep(100 * time.Millisecond)
			isRunning := true
			for isRunning {
				select {
				// if the expected time frame is over, we fail the test
				case _ = <-successTestSignal:
					isRunning = false
					break
				// else continue watching out for the mqSignal
				case _ = <-mqSignal:
					// kill a replica if there has been a kill signal
					if len(killSignal) > 0 {
						killIndex := <-killSignal
						replicaCtxCancels[killIndex]()
						continue
					}

					// ignore if there isn't any message
					if len(mq) == 0 {
						continue
					}

					// pop the first message
					m := mq[0]
					mq = mq[1:]

					// ignore if its a nil message
					if m.value == nil {
						continue
					}

					// handle the message
					time.Sleep(5 * time.Millisecond)
					replica := replicas[m.to]
					switch value := m.value.(type) {
					case process.Propose:
						replica.Propose(context.Background(), value)
					case process.Prevote:
						replica.Prevote(context.Background(), value)
					case process.Precommit:
						replica.Precommit(context.Background(), value)
					default:
						panic(fmt.Errorf("non-exhaustive pattern: message.value has type %T", value))
					}
				}
			}
		})
	})
})
