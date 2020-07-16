package replica_test

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/renproject/hyperdrive/process"
	"github.com/renproject/hyperdrive/process/processutil"
	"github.com/renproject/hyperdrive/replica"
	"github.com/renproject/hyperdrive/timer"
	"github.com/renproject/id"
	"github.com/renproject/surge"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

// TODO: Implement the other scenarios.
//
// 1. 3F+1 replicas online (done)
// 2. 2F+1 replicas online (done)
// 3. 3F+1 replicas online, and F replicas slowly go offline (done)
// 5. 3F+1 replicas online, and F replicas are behaving maliciously
//    - Proposal: malformed, missing, fork-attempt, out-of-turn
//    - Prevote: yes-to-malformed, no-to-valid, missing, fork-attempt, out-of-turn
//    - Precommit: yes-to-malformed, no-to-valid, missing, fork-attempt, out-of-turn

var _ = Describe("Replica", func() {
	setup := func(
		seed int64,
		f, n, completion uint8,
		targetHeight process.Height,
		mq *[]Message,
		commits *map[uint8]map[process.Height]process.Value,
	) (
		*rand.Rand,
		Scenario,
		bool,
		*sync.Mutex,
		chan struct{},
		chan bool,
		[]*replica.Replica,
		[]context.Context,
		[]context.CancelFunc,
		context.CancelFunc,
	) {
		// create private keys for the signatories participating in consensus
		// rounds, and get their signatories
		privKeys := make([]*id.PrivKey, 3*f+1)
		signatories := make([]id.Signatory, 3*f+1)
		for i := range privKeys {
			privKeys[i] = id.NewPrivKey()
			signatories[i] = privKeys[i].Signatory()
			(*commits)[uint8(i)] = make(map[process.Height]process.Value)
		}

		// construct the scenario for this test case
		scenario := Scenario{seed, f, n, completion, signatories, []Message{}}

		// whether we are running the test in replay mode
		replayMode := readBoolEnvVar("REPLAY_MODE")

		// fetch from the dump file in that case
		if replayMode {
			scenario = readFromFile("failure.dump")
			seed = scenario.seed
			f = scenario.f
			n = scenario.n
			completion = scenario.completion
			signatories = scenario.signatories
		}

		// random number generator
		r := rand.New(rand.NewSource(seed))

		// signals to denote processing of messages
		mqSignal := make(chan struct{}, n*n)
		mqSignal <- struct{}{}

		// signal to denote completion of consensus target
		completionSignal := make(chan bool, completion)

		// mutex to synchronize access to the mq slice for all the replicas
		var mqMutex = &sync.Mutex{}

		// instantiate the replicas
		replicas := make([]*replica.Replica, n)
		for i := range replicas {
			replicaIndex := uint8(i)
			replicas[i] = replica.New(
				replica.DefaultOptions().
					WithTimerOptions(
						timer.DefaultOptions().
							WithTimeout(500*time.Millisecond),
					),
				signatories[i],
				signatories,
				// Proposer
				processutil.MockProposer{
					MockValue: func() process.Value {
						mqMutex.Lock()
						v := processutil.RandomGoodValue(r)
						mqMutex.Unlock()
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
						// add to the map of commits
						mqMutex.Lock()
						(*commits)[replicaIndex][height] = value
						mqMutex.Unlock()

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
						for j := uint8(0); j < n; j++ {
							mqMutex.Lock()
							*mq = append(*mq, Message{
								to:          j,
								messageType: 1,
								value:       propose,
							})
							mqMutex.Unlock()
						}
					},
					BroadcastPrevoteCallback: func(prevote process.Prevote) {
						for j := uint8(0); j < n; j++ {
							mqMutex.Lock()
							*mq = append(*mq, Message{
								to:          j,
								messageType: 2,
								value:       prevote,
							})
							mqMutex.Unlock()
						}
					},
					BroadcastPrecommitCallback: func(precommit process.Precommit) {
						for j := uint8(0); j < n; j++ {
							mqMutex.Lock()
							*mq = append(*mq, Message{
								to:          j,
								messageType: 3,
								value:       precommit,
							})
							mqMutex.Unlock()
						}
					},
				},
				// Flusher
				func() {
					mqSignal <- struct{}{}
				},
			)
		}

		// global context within which all replicas will run
		ctx, cancel := context.WithCancel(context.Background())

		// individual replica contexts and functions to kill/cancel them
		replicaCtxs, replicaCtxCancels := make([]context.Context, n), make([]context.CancelFunc, n)
		for i := range replicaCtxs {
			replicaCtxs[i], replicaCtxCancels[i] = context.WithCancel(ctx)
		}

		return r, scenario, replayMode, mqMutex, mqSignal, completionSignal, replicas, replicaCtxs, replicaCtxCancels, cancel
	}

	play := func(
		scenario *Scenario,
		timeout time.Duration,
		mq *[]Message,
		mqMutex *sync.Mutex,
		mqSignal chan struct{},
		completionSignal chan bool,
		replicas []*replica.Replica,
		killedReplicas *map[uint8]bool,
		successFn func(),
		failureFn func(*Scenario),
		inspectFn func(*Scenario),
	) {
		isRunning := true
		for timeoutSignal := time.After(timeout); isRunning; {
			select {
			case <-timeoutSignal:
				failureFn(scenario)
			case <-mqSignal:
				// the consensus target has been achieved on every replica
				if len(completionSignal) == int(scenario.completion) {
					mqMutex.Lock()
					successFn()
					mqMutex.Unlock()

					isRunning = false
					continue
				}

				// synchronously get the length of mq
				mqMutex.Lock()
				mqLen := len(*mq)
				mqMutex.Unlock()

				// ignore if it was a spurious signal
				if mqLen == 0 {
					continue
				}

				// pop the first message off the slice
				mqMutex.Lock()
				m := (*mq)[0]
				*mq = (*mq)[1:]
				mqMutex.Unlock()

				// ignore if it is a nil message
				if m.value == nil {
					continue
				}

				// append the message to message history
				scenario.messages = append(scenario.messages, m)

				// is the recipient replica killed
				mqMutex.Lock()
				_, isKilled := (*killedReplicas)[uint8(m.to)]
				mqMutex.Unlock()

				// handle the message
				if !isKilled {
					time.Sleep(1 * time.Millisecond)
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

					mqMutex.Lock()
					inspectFn(scenario)
					mqMutex.Unlock()
				} else {
					mqSignal <- struct{}{}
				}
			}
		}
	}

	replay := func(
		scenario *Scenario,
		mqSignal chan struct{},
		completionSignal chan bool,
		replicas []*replica.Replica,
		killedReplicas *map[uint8]bool,
		successFn func(),
		inspectFn func(*Scenario),
	) {
		// dummy goroutine to keep consuming the mqSignal
		go func() {
			for range mqSignal {
			}
		}()

		// handle every message in the messages history
		for _, message := range scenario.messages {
			if _, isKilled := (*killedReplicas)[uint8(message.to)]; !isKilled {
				time.Sleep(1 * time.Millisecond)
				recipient := replicas[message.to]
				switch value := message.value.(type) {
				case process.Propose:
					recipient.Propose(context.Background(), value)
				case process.Prevote:
					recipient.Prevote(context.Background(), value)
				case process.Precommit:
					recipient.Precommit(context.Background(), value)
				default:
					panic(fmt.Errorf("non-exhaustive pattern: message.value has type %T", value))
				}

				inspectFn(scenario)
			}

			// exit replay if the consensus target has been achieved
			if len(completionSignal) == int(scenario.completion) {
				successFn()
				break
			}
		}
	}

	Context("with sufficient replicas online for consensus to progress", func() {
		Context("with 3f+1 replicas online", func() {
			It("should be able to reach consensus", func() {
				// randomness seed
				seed := time.Now().UnixNano()
				// maximum number of adversaries that the consensus network can tolerate
				f := uint8(3)
				// number of replicas online
				n := 3*f + 1
				// number of replicas to signal completion
				completion := n
				// target height of consensus to mark the test as succeeded
				targetHeight := process.Height(30)
				// dynamic slice to hold the messages being sent between replicas
				mq := []Message{}
				// commits from replicas
				commits := make(map[uint8]map[process.Height]process.Value)
				// map to keep a record of which replica was killed
				killedReplicas := make(map[uint8]bool)

				// setup the test scenario
				_, scenario, replayMode, mqMutex, mqSignal, completionSignal, replicas, replicaCtxs, replicaCtxCancels, cancel := setup(seed, f, n, completion, targetHeight, &mq, &commits)

				// Run all of the replicas in independent background goroutines
				for i := range replicas {
					go replicas[i].Run(replicaCtxs[i])
				}

				// callback function called on test failure
				failureFn := func(scenario *Scenario) {
					cancel()
					dumpToFile("failure.dump", *scenario)
					Fail("test failed to complete within the expected timeframe")
				}

				// callback function called on test success
				successFn := func() {
					// cancel the replica contexts
					for i := range replicaCtxs {
						replicaCtxCancels[i]()
					}

					// fetch the first replica's commits
					referenceCommits := commits[0]

					// ensure that all replicas have the same commits
					for j := uint8(0); j < n; j++ {
						for h := process.Height(1); h <= targetHeight; {
							Expect(commits[j][h]).To(Equal(referenceCommits[h]))
							h++
						}
					}
				}

				// callback function called on every message processed
				inspectFn := func(scenario *Scenario) {}

				if !replayMode {
					timeout := 15 * time.Second

					play(&scenario, timeout, &mq, mqMutex, mqSignal, completionSignal, replicas, &killedReplicas, successFn, failureFn, inspectFn)
				}

				if replayMode {
					replay(&scenario, mqSignal, completionSignal, replicas, &killedReplicas, successFn, inspectFn)
				}
			})
		})

		Context("with 2f+1 replicas online", func() {
			It("should be able to reach consensus", func() {
				// randomness seed
				seed := time.Now().UnixNano()
				// maximum number of adversaries that the consensus network can tolerate
				f := uint8(3)
				// number of replicas online
				n := 2*f + 1
				// number of replicas that should signal completion
				completion := n
				// target height of consensus to mark the test as succeeded
				targetHeight := process.Height(30)
				// dynamic slice to hold the messages being sent between replicas
				mq := []Message{}
				// commits from replicas
				commits := make(map[uint8]map[process.Height]process.Value)
				// map to keep a record of which replica was killed
				killedReplicas := make(map[uint8]bool)

				// setup the test scenario
				_, scenario, replayMode, mqMutex, mqSignal, completionSignal, replicas, replicaCtxs, replicaCtxCancels, cancel := setup(seed, f, n, completion, targetHeight, &mq, &commits)

				// Run all of the replicas in independent background goroutines.
				for i := range replicas {
					go replicas[i].Run(replicaCtxs[i])
				}

				// callback function called on test failure
				failureFn := func(scenario *Scenario) {
					cancel()
					dumpToFile("failure.dump", *scenario)
					Fail("test failed to complete within the expected timeframe")
				}

				// callback function called on test success
				successFn := func() {
					// cancel the replica contexts
					for i := range replicaCtxs {
						replicaCtxCancels[i]()
					}

					// fetch the first replica's commits
					referenceCommits := commits[0]

					// ensure that all replicas have the same commits
					for j := uint8(0); j < n; j++ {
						for h := process.Height(1); h <= targetHeight; {
							Expect(commits[j][h]).To(Equal(referenceCommits[h]))
							h++
						}
					}
				}

				// callback function called on every message processed
				inspectFn := func(scenario *Scenario) {}

				if !replayMode {
					timeout := 35 * time.Second

					play(&scenario, timeout, &mq, mqMutex, mqSignal, completionSignal, replicas, &killedReplicas, successFn, failureFn, inspectFn)
				}

				if replayMode {
					replay(&scenario, mqSignal, completionSignal, replicas, &killedReplicas, successFn, inspectFn)
				}
			})
		})
	})

	Context("with 3f+1 replicas online, f replicas slowly go offline", func() {
		It("should be able to reach consensus", func() {
			// randomness seed
			seed := time.Now().UnixNano()
			// f is the maximum no. of adversaries
			f := uint8(3)
			// n is the number of honest replicas online
			n := 3*f + 1
			// completion is the number of replicas that should signal completion
			completion := 2*f + 1
			// target height of consensus to mark the test as succeeded
			targetHeight := process.Height(30)
			// dynamic slice to hold the messages being sent between replicas
			mq := []Message{}
			// commits from replicas
			commits := make(map[uint8]map[process.Height]process.Value)
			// map to keep a record of which replica was killed
			killedReplicas := make(map[uint8]bool)

			// setup the test scenario
			r, scenario, replayMode, mqMutex, mqSignal, completionSignal, replicas, replicaCtxs, replicaCtxCancels, cancel := setup(seed, f, n, completion, targetHeight, &mq, &commits)

			// Run all of the replicas in independent background goroutines
			for i := range replicas {
				go replicas[i].Run(replicaCtxs[i])
			}

			// callback function called on test failure
			failureFn := func(scenario *Scenario) {
				cancel()
				dumpToFile("failure.dump", *scenario)
				Fail("test failed to complete within the expected timeframe")
			}

			// callback function called on test success
			successFn := func() {
				// cancel the replica contexts
				for i := range replicaCtxs {
					replicaCtxCancels[i]()
				}

				// find the first replica that was alive till the consensus target
				id := uint8(0)
				for _, ok := killedReplicas[id]; ok; _, ok = killedReplicas[id] {
					id++
				}
				referenceCommits := commits[id]

				// ensure for all alive replicas
				for j := id + 1; j < n; j++ {
					// ignore if the replica was killed midway
					if _, ok := killedReplicas[j]; ok {
						continue
					}

					// commits are the same across all heights
					for h := process.Height(1); h <= targetHeight; {
						Expect(commits[j][h]).To(Equal(referenceCommits[h]))
						h++
					}
				}
			}

			// callback function called on every message processed
			inspectFn := func(scenario *Scenario) {
				// we wish to kill at the most f replicas
				if len(killedReplicas) < int(f) {
					if r.Float64() < 0.01 {
						// get a random replica to kill
						id := r.Intn(int(n))

						// if its not yet killed
						if _, isKilled := killedReplicas[uint8(id)]; !isKilled {
							// kill the replica
							killedReplicas[uint8(id)] = true
							replicaCtxCancels[id]()
						}
					}
				}
			}

			if !replayMode {
				timeout := 30 * time.Second

				play(&scenario, timeout, &mq, mqMutex, mqSignal, completionSignal, replicas, &killedReplicas, successFn, failureFn, inspectFn)
			}

			if replayMode {
				replay(&scenario, mqSignal, completionSignal, replicas, &killedReplicas, successFn, inspectFn)
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
				f := uint8(3)
				n := 3*f + 1
				targetHeight := process.Height(12)

				// commits from replicas
				commits := make(map[uint8]map[process.Height]process.Value)

				// setup private keys for the replicas
				// and their signatories
				privKeys := make([]*id.PrivKey, n)
				signatories := make([]id.Signatory, n)
				for i := range privKeys {
					privKeys[i] = id.NewPrivKey()
					signatories[i] = privKeys[i].Signatory()
					commits[uint8(i)] = make(map[process.Height]process.Value)
				}

				// every replica sends this signal when they reach the target
				// consensus height
				completionSignal := make(chan bool, n)

				// slice of messages to be broadcasted between replicas.
				// messages from mq are popped and handled whenever mqSignal is signalled
				mq := []Message{}
				mqMutex := &sync.Mutex{}
				mqSignal := make(chan struct{}, n*n)
				mqSignal <- struct{}{}

				// build replicas
				replicas := make([]*replica.Replica, n)
				for i := range replicas {
					replicaIndex := uint8(i)

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
								mqMutex.Lock()
								for j := uint8(0); j < n; j++ {
									mq = append(mq, Message{
										to:    j,
										value: propose,
									})
								}
								mqMutex.Unlock()
							},
							BroadcastPrevoteCallback: func(prevote process.Prevote) {
								mqMutex.Lock()
								for j := uint8(0); j < n; j++ {
									mq = append(mq, Message{
										to:    j,
										value: prevote,
									})
								}
								mqMutex.Unlock()
							},
							BroadcastPrecommitCallback: func(precommit process.Precommit) {
								mqMutex.Lock()
								for j := uint8(0); j < n; j++ {
									mq = append(mq, Message{
										to:    j,
										value: precommit,
									})
								}
								mqMutex.Unlock()
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
					for j := uint8(1); j < n; j++ {
						for h := process.Height(1); h <= targetHeight; {
							Expect(commits[j][h]).To(Equal(referenceCommits[h]))
							h++
						}
					}
				}

				failTest := func() {
					cancel()
					Fail("test failed to complete within the expected timeframe")
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
						if len(completionSignal) == int(3*f+1) {
							completion()
							isRunning = false
						}

						mqMutex.Lock()
						mqLen := len(mq)
						mqMutex.Unlock()

						// ignore if there isn't any message
						if mqLen == 0 {
							continue
						}

						// pop the first message
						mqMutex.Lock()
						m := mq[0]
						mq = mq[1:]
						mqMutex.Unlock()

						// ignore if its a nil message
						if m.value == nil {
							continue
						}

						// handle the message
						time.Sleep(1 * time.Millisecond)
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
				f := uint8(3)
				n := 3*f + 1
				targetHeight := process.Height(12)

				// commits from replicas
				commits := make(map[uint8]map[process.Height]process.Value)

				// setup private keys for the replicas
				// and their signatories
				privKeys := make([]*id.PrivKey, n)
				signatories := make([]id.Signatory, n)
				for i := range privKeys {
					privKeys[i] = id.NewPrivKey()
					signatories[i] = privKeys[i].Signatory()
					commits[uint8(i)] = make(map[process.Height]process.Value)
				}

				// every replica sends this signal when they reach the target
				// consensus height
				completionSignal := make(chan bool, n)

				// slice of messages to be broadcasted between replicas.
				// messages from mq are popped and handled whenever mqSignal is signalled
				mq := []Message{}
				mqMutex := &sync.Mutex{}
				mqSignal := make(chan struct{}, n*n)
				mqSignal <- struct{}{}

				// build replicas
				replicas := make([]*replica.Replica, n)
				for i := range replicas {
					replicaIndex := uint8(i)

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
								mqMutex.Lock()
								for j := uint8(0); j < n; j++ {
									mq = append(mq, Message{
										to:    j,
										value: propose,
									})
								}
								mqMutex.Unlock()
							},
							BroadcastPrevoteCallback: func(prevote process.Prevote) {
								mqMutex.Lock()
								for j := uint8(0); j < n; j++ {
									mq = append(mq, Message{
										to:    j,
										value: prevote,
									})
								}
								mqMutex.Unlock()
							},
							BroadcastPrecommitCallback: func(precommit process.Precommit) {
								mqMutex.Lock()
								for j := uint8(0); j < n; j++ {
									mq = append(mq, Message{
										to:    j,
										value: precommit,
									})
								}
								mqMutex.Unlock()
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
					Fail("test failed to complete within the expected timeframe")
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
						if len(completionSignal) == int(2*f+1) {
							completion()
							isRunning = false
						}

						mqMutex.Lock()
						mqLen := len(mq)
						mqMutex.Unlock()

						// ignore if there isn't any message
						if mqLen == 0 {
							continue
						}

						// pop the first message
						mqMutex.Lock()
						m := mq[0]
						mq = mq[1:]
						mqMutex.Unlock()

						// ignore if its a nil message
						if m.value == nil {
							continue
						}

						// handle the message
						time.Sleep(1 * time.Millisecond)
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
			f := uint8(3)
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
			mqMutex := &sync.Mutex{}
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
							Fail("consensus should have stalled")
						},
					},
					// Catcher
					nil,
					// Broadcaster
					processutil.BroadcasterCallbacks{
						BroadcastProposeCallback: func(propose process.Propose) {
							mqMutex.Lock()
							for j := uint8(0); j < n; j++ {
								mq = append(mq, Message{
									to:    j,
									value: propose,
								})
							}
							mqMutex.Unlock()
						},
						BroadcastPrevoteCallback: func(prevote process.Prevote) {
							mqMutex.Lock()
							for j := uint8(0); j < n; j++ {
								mq = append(mq, Message{
									to:    j,
									value: prevote,
								})
							}
							mqMutex.Unlock()
						},
						BroadcastPrecommitCallback: func(precommit process.Precommit) {
							mqMutex.Lock()
							for j := uint8(0); j < n; j++ {
								mq = append(mq, Message{
									to:    j,
									value: precommit,
								})
							}
							mqMutex.Unlock()
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
					mqMutex.Lock()
					mqLen := len(mq)
					mqMutex.Unlock()

					if mqLen == 0 {
						continue
					}

					// pop the first message
					mqMutex.Lock()
					m := mq[0]
					mq = mq[1:]
					mqMutex.Unlock()

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
			f := uint8(3)
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
			mqMutex := &sync.Mutex{}
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
								Fail("consensus should have stalled")
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
							mqMutex.Lock()
							for j := uint8(0); j < n; j++ {
								mq = append(mq, Message{
									to:    j,
									value: propose,
								})
							}
							mqMutex.Unlock()
						},
						BroadcastPrevoteCallback: func(prevote process.Prevote) {
							mqMutex.Lock()
							for j := uint8(0); j < n; j++ {
								mq = append(mq, Message{
									to:    j,
									value: prevote,
								})
							}
							mqMutex.Unlock()
						},
						BroadcastPrecommitCallback: func(precommit process.Precommit) {
							mqMutex.Lock()
							for j := uint8(0); j < n; j++ {
								mq = append(mq, Message{
									to:    j,
									value: precommit,
								})
							}
							mqMutex.Unlock()
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

					mqMutex.Lock()
					mqLen := len(mq)
					mqMutex.Unlock()

					// ignore if there isn't any message
					if mqLen == 0 {
						continue
					}

					// pop the first message
					mqMutex.Lock()
					m := mq[0]
					mq = mq[1:]
					mqMutex.Unlock()

					// ignore if its a nil message
					if m.value == nil {
						continue
					}

					// handle the message
					time.Sleep(1 * time.Millisecond)
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

type Scenario struct {
	seed        int64
	f           uint8
	n           uint8
	completion  uint8
	signatories []id.Signatory
	messages    []Message
}

func (s Scenario) SizeHint() int {
	return surge.SizeHint(s.seed) +
		surge.SizeHint(s.f) +
		surge.SizeHint(s.n) +
		surge.SizeHint(s.completion) +
		surge.SizeHint(s.signatories) +
		surge.SizeHint(s.messages)
}

func (s Scenario) Marshal(buf []byte, rem int) ([]byte, int, error) {
	buf, rem, err := surge.Marshal(s.seed, buf, rem)
	if err != nil {
		return buf, rem, fmt.Errorf("marshaling seed=%v: %v", s.seed, err)
	}
	buf, rem, err = surge.Marshal(s.f, buf, rem)
	if err != nil {
		return buf, rem, fmt.Errorf("marshaling f=%v: %v", s.f, err)
	}
	buf, rem, err = surge.Marshal(s.n, buf, rem)
	if err != nil {
		return buf, rem, fmt.Errorf("marshaling n=%v: %v", s.n, err)
	}
	buf, rem, err = surge.Marshal(s.completion, buf, rem)
	if err != nil {
		return buf, rem, fmt.Errorf("marshaling completion=%v: %v", s.completion, err)
	}
	buf, rem, err = surge.Marshal(s.signatories, buf, rem)
	if err != nil {
		return buf, rem, fmt.Errorf("marshaling signatories=%v: %v", s.signatories, err)
	}
	buf, rem, err = surge.Marshal(s.messages, buf, rem)
	if err != nil {
		return buf, rem, fmt.Errorf("marshaling messages=%v: %v", s.messages, err)
	}

	return buf, rem, nil
}

func (s *Scenario) Unmarshal(buf []byte, rem int) ([]byte, int, error) {
	buf, rem, err := surge.Unmarshal(&s.seed, buf, rem)
	if err != nil {
		return buf, rem, fmt.Errorf("unmarshaling seed: %v", err)
	}
	buf, rem, err = surge.Unmarshal(&s.f, buf, rem)
	if err != nil {
		return buf, rem, fmt.Errorf("unmarshaling f: %v", err)
	}
	buf, rem, err = surge.Unmarshal(&s.n, buf, rem)
	if err != nil {
		return buf, rem, fmt.Errorf("unmarshaling n: %v", err)
	}
	buf, rem, err = surge.Unmarshal(&s.completion, buf, rem)
	if err != nil {
		return buf, rem, fmt.Errorf("unmarshaling completion: %v", err)
	}
	buf, rem, err = surge.Unmarshal(&s.signatories, buf, rem)
	if err != nil {
		return buf, rem, fmt.Errorf("unmarshaling signatories: %v", err)
	}
	buf, rem, err = surge.Unmarshal(&s.messages, buf, rem)
	if err != nil {
		return buf, rem, fmt.Errorf("unmarshaling messages: %v", err)
	}

	return buf, rem, nil
}

type Message struct {
	to          uint8
	messageType uint8
	value       surge.Marshaler
}

func (m Message) SizeHint() int {
	switch value := m.value.(type) {
	case process.Propose:
		return surge.SizeHint(m.to) +
			surge.SizeHint(m.messageType) +
			surge.SizeHint(value)
	case process.Prevote:
		return surge.SizeHint(m.to) +
			surge.SizeHint(m.messageType) +
			surge.SizeHint(value)
	case process.Precommit:
		return surge.SizeHint(m.to) +
			surge.SizeHint(m.messageType) +
			surge.SizeHint(value)
	default:
		panic(fmt.Errorf("non-exhaustive pattern: message.value has type %T", value))
	}
}

func (m Message) Marshal(buf []byte, rem int) ([]byte, int, error) {
	buf, rem, err := surge.Marshal(m.to, buf, rem)
	if err != nil {
		return buf, rem, fmt.Errorf("marshaling to=%v: %v", m.to, err)
	}
	buf, rem, err = surge.Marshal(m.messageType, buf, rem)
	if err != nil {
		return buf, rem, fmt.Errorf("marshaling messageType=%v: %v", m.messageType, err)
	}

	switch value := m.value.(type) {
	case process.Propose:
		buf, rem, err = surge.Marshal(value, buf, rem)
		if err != nil {
			return buf, rem, fmt.Errorf("marshaling value=%v: %v", value, err)
		}
	case process.Prevote:
		buf, rem, err = surge.Marshal(value, buf, rem)
		if err != nil {
			return buf, rem, fmt.Errorf("marshaling value=%v: %v", value, err)
		}
	case process.Precommit:
		buf, rem, err = surge.Marshal(value, buf, rem)
		if err != nil {
			return buf, rem, fmt.Errorf("marshaling value=%v: %v", value, err)
		}
	default:
		panic(fmt.Errorf("non-exhaustive pattern: message.value has type %T", value))
	}

	return buf, rem, nil
}

func (m *Message) Unmarshal(buf []byte, rem int) ([]byte, int, error) {
	buf, rem, err := surge.Unmarshal(&m.to, buf, rem)
	if err != nil {
		return buf, rem, fmt.Errorf("unmarshaling to: %v", err)
	}
	buf, rem, err = surge.Unmarshal(&m.messageType, buf, rem)
	if err != nil {
		return buf, rem, fmt.Errorf("unmarshaling to: %v", err)
	}

	switch m.messageType {
	case 1:
		message := process.Propose{}
		buf, rem, err = message.Unmarshal(buf, rem)
		if err != nil {
			return buf, rem, fmt.Errorf("unmarshaling value: %v", err)
		}
		m.value = message
	case 2:
		message := process.Prevote{}
		buf, rem, err = message.Unmarshal(buf, rem)
		if err != nil {
			return buf, rem, fmt.Errorf("unmarshaling value: %v", err)
		}
		m.value = message
	case 3:
		message := process.Precommit{}
		buf, rem, err = message.Unmarshal(buf, rem)
		if err != nil {
			return buf, rem, fmt.Errorf("unmarshaling value: %v", err)
		}
		m.value = message
	default:
		panic(fmt.Errorf("non-exhaustive pattern: messageType has type %T", m.messageType))
	}

	return buf, rem, nil
}

func readBoolEnvVar(name string) bool {
	envVar := os.Getenv(name)
	boolEnvVar, err := strconv.ParseBool(envVar)
	if err != nil {
		panic(fmt.Errorf("error parsing env var %v to boolean", name))
	}
	return boolEnvVar
}

func dumpToFile(filename string, scenario Scenario) {
	file, err := os.Create(filename)
	if err != nil {
		fmt.Printf("unable to create dump file: %v\n", err)
	}
	defer file.Close()

	fmt.Printf("dumping msg history to file %s\n", filename)

	data := make([]byte, surge.SizeHint(scenario))
	_, _, err = surge.Marshal(scenario, data, surge.SizeHint(scenario))
	if err != nil {
		fmt.Printf("unable to marshal scenario: %v\n", err)
	}

	_, err = file.Write(data)
	if err != nil {
		fmt.Printf("unable to write data to file: %v\n", err)
	}
}

func readFromFile(filename string) Scenario {
	file, err := os.Open(filename)
	if err != nil {
		fmt.Printf("unable to open dump file: %v\n", err)
	}
	defer file.Close()

	fmt.Printf("reading msg history from file %s\n", filename)

	data := make([]byte, surge.MaxBytes)
	_, err = file.Read(data)
	if err != nil {
		fmt.Printf("unable to read data from file: %v\n", err)
	}

	var scenario Scenario
	_, _, err = surge.Unmarshal(&scenario, data, surge.MaxBytes)
	if err != nil {
		fmt.Printf("unable to unmarshal scenario: %v\n", err)
	}

	return scenario
}
