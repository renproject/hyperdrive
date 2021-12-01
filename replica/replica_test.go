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

var _ = Describe("Replica", func() {
	setup := func(
		seed int64,
		f, n, completion uint8,
		targetHeight process.Height,
		mq *[]Message,
		commits *map[uint8]map[process.Height]process.Value,
		proposerFn func() process.Value,
		validationFn func(process.Value) bool,
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
				replica.DefaultOptions(),
				signatories[i],
				signatories,
				// timer
				timer.NewLinearTimer(
					timer.DefaultOptions().WithTimeout(500*time.Millisecond),
					// on timeout propose
					func(timeout timer.Timeout) {
						mqMutex.Lock()
						*mq = append(*mq, Message{
							to:          replicaIndex,
							messageType: 4,
							value:       timeout,
						})
						mqMutex.Unlock()
					},
					// on timeout prevote
					func(timeout timer.Timeout) {
						mqMutex.Lock()
						*mq = append(*mq, Message{
							to:          replicaIndex,
							messageType: 5,
							value:       timeout,
						})
						mqMutex.Unlock()
					},
					// on timeout precommit
					func(timeout timer.Timeout) {
						mqMutex.Lock()
						*mq = append(*mq, Message{
							to:          replicaIndex,
							messageType: 6,
							value:       timeout,
						})
						mqMutex.Unlock()
					},
				),
				// Proposer
				processutil.MockProposer{
					MockValue: func() process.Value {
						// if this is a malicious replica, but less than f total malicious
						if proposerFn != nil && replicaIndex < f {
							return proposerFn()
						}

						// honest behaviour
						mqMutex.Lock()
						v := processutil.RandomGoodValue(r)
						mqMutex.Unlock()
						return v
					},
				},
				// Validator
				processutil.MockValidator{
					MockValid: func(_ process.Height, _ process.Round, value process.Value) bool {
						// if this is a malicious replica, but less than f total malicious
						if validationFn != nil && replicaIndex < f {
							return validationFn(value)
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
					Callback: func(height process.Height, value process.Value) (uint64, process.Scheduler) {
						// add to the map of commits
						mqMutex.Lock()
						(*commits)[replicaIndex][height] = value
						mqMutex.Unlock()

						// signal for completion if this is the target height
						if height == targetHeight {
							completionSignal <- true
						}
						return 0, nil
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
				isRunning = false
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
					mqSignal <- struct{}{}
					continue
				}

				// pop the first message off the slice
				mqMutex.Lock()
				m := (*mq)[0]
				*mq = (*mq)[1:]
				mqMutex.Unlock()

				// ignore if it is a nil message
				if m.value == nil {
					mqSignal <- struct{}{}
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
					case timer.Timeout:
						switch m.messageType {
						case 4:
							replica.TimeoutPropose(context.Background(), value)
						case 5:
							replica.TimeoutPrevote(context.Background(), value)
						case 6:
							replica.TimeoutPrecommit(context.Background(), value)
						default:
							panic(fmt.Errorf("non-exhaustive pattern: timeout message.messageType is %v", m.messageType))
						}
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
		mqMutex *sync.Mutex,
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

				mqMutex.Lock()
				inspectFn(scenario)
				mqMutex.Unlock()
			}

			// exit replay if the consensus target has been achieved
			if len(completionSignal) == int(scenario.completion) {
				mqMutex.Lock()
				successFn()
				mqMutex.Unlock()
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
				_, scenario, replayMode, mqMutex, mqSignal, completionSignal, replicas, replicaCtxs, replicaCtxCancels, cancel := setup(seed, f, n, completion, targetHeight, &mq, &commits, nil, nil)

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
					replay(&scenario, mqMutex, mqSignal, completionSignal, replicas, &killedReplicas, successFn, inspectFn)
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
				_, scenario, replayMode, mqMutex, mqSignal, completionSignal, replicas, replicaCtxs, replicaCtxCancels, cancel := setup(seed, f, n, completion, targetHeight, &mq, &commits, nil, nil)

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
					replay(&scenario, mqMutex, mqSignal, completionSignal, replicas, &killedReplicas, successFn, inspectFn)
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
			r, scenario, replayMode, mqMutex, mqSignal, completionSignal, replicas, replicaCtxs, replicaCtxCancels, cancel := setup(seed, f, n, completion, targetHeight, &mq, &commits, nil, nil)

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
				replay(&scenario, mqMutex, mqSignal, completionSignal, replicas, &killedReplicas, successFn, inspectFn)
			}
		})
	})

	Context("with 3f+1 replicas online, f replicas behaving maliciously", func() {
		Context("f replicas running misbehaving proposers and validators", func() {
			It("should be able to reach consensus", func() {
				// randomness seed
				seed := time.Now().UnixNano()
				// maximum number of adversaries that the consensus network can tolerate
				f := uint8(3)
				// number of replicas online
				n := 3*f + 1
				// number of replicas to signal completion
				completion := 2*f + 1
				// h is the target minimum consensus height
				targetHeight := process.Height(30)
				// dynamic slice to hold the messages being sent between replicas
				mq := []Message{}
				// commits from replicas
				commits := make(map[uint8]map[process.Height]process.Value)
				// map to keep a record of which replica was killed
				killedReplicas := make(map[uint8]bool)

				// malicious proposing behaviour
				proposerFn := func() process.Value {
					return process.NilValue
				}
				// malicious validation behaviour
				validationFn := func(value process.Value) bool {
					if value == process.NilValue {
						return true
					}
					return false
				}
				// setup the test scenario
				_, scenario, replayMode, mqMutex, mqSignal, completionSignal, replicas, replicaCtxs, replicaCtxCancels, cancel := setup(seed, f, n, completion, targetHeight, &mq, &commits, proposerFn, validationFn)

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
					referenceCommits := commits[f]

					// ensure that all replicas have the same commits
					for j := f + 1; j < n; j++ {
						for h := process.Height(1); h <= targetHeight; {
							Expect(commits[j][h]).To(Equal(referenceCommits[h]))
							h++
						}
					}
				}

				// callback function called on every message processed
				inspectFn := func(scenario *Scenario) {}

				if !replayMode {
					timeout := 45 * time.Second

					play(&scenario, timeout, &mq, mqMutex, mqSignal, completionSignal, replicas, &killedReplicas, successFn, failureFn, inspectFn)
				}

				if replayMode {
					replay(&scenario, mqMutex, mqSignal, completionSignal, replicas, &killedReplicas, successFn, inspectFn)
				}
			})
		})
	})

	Context("with less than 2f+1 replicas online", func() {
		It("should stall consensus and should not progress", func() {
			// randomness seed
			seed := time.Now().UnixNano()
			// f is the maximum no. of adversaries
			f := uint8(3)
			// n is the number of honest replicas online
			n := 2 * f
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
			_, scenario, replayMode, mqMutex, mqSignal, completionSignal, replicas, replicaCtxs, replicaCtxCancels, cancel := setup(seed, f, n, completion, targetHeight, &mq, &commits, nil, nil)

			// Run all of the replicas in independent background goroutines
			for i := range replicas {
				go replicas[i].Run(replicaCtxs[i])
			}

			// callback function called on test failure
			failureFn := func(scenario *Scenario) {
				// cancel the replica contexts
				for i := range replicaCtxs {
					replicaCtxCancels[i]()
				}

				// ensure that none of the replicas have reached consensus for any height
				for j := uint8(0); j < n; j++ {
					for h := process.Height(1); h <= targetHeight; {
						Expect(commits[j][h]).To(Equal(process.NilValue))
						h++
					}
				}
			}

			// callback function called on test success
			successFn := func() {
				cancel()
				Fail("test was not expected to reach target consensus")
			}

			// callback function called on every message processed
			inspectFn := func(scenario *Scenario) {}

			if !replayMode {
				timeout := 10 * time.Second

				play(&scenario, timeout, &mq, mqMutex, mqSignal, completionSignal, replicas, &killedReplicas, successFn, failureFn, inspectFn)
			}

			if replayMode {
				replay(&scenario, mqMutex, mqSignal, completionSignal, replicas, &killedReplicas, successFn, inspectFn)
			}
		})
	})

	Context("with 2f+1 replicas online, and one of them goes offline", func() {
		It("should stall consensus as soon as there are less than 2f+1 online", func() {
			// randomness seed
			seed := time.Now().UnixNano()
			// f is the maximum no. of adversaries
			f := uint8(3)
			// n is the number of honest replicas online
			n := 2*f + 1
			// number of replicas that should send completion signal
			completion := 2 * f
			// consensus height at which one of the replicas drops out
			intermTargetHeight := process.Height(0)
			// consensus height at which one of the replicas drops out
			targetHeight := process.Height(30)
			// dynamic slice to hold the messages being sent between replicas
			mq := []Message{}
			// commits from replicas
			commits := make(map[uint8]map[process.Height]process.Value)
			// map to keep a record of which replica was killed
			killedReplicas := make(map[uint8]bool)

			// setup the test scenario
			r, scenario, replayMode, mqMutex, mqSignal, completionSignal, replicas, replicaCtxs, replicaCtxCancels, cancel := setup(seed, f, n, completion, targetHeight, &mq, &commits, nil, nil)

			// Run all of the replicas in independent background goroutines
			for i := range replicas {
				go replicas[i].Run(replicaCtxs[i])
			}

			// callback function called on test failure
			failureFn := func(scenario *Scenario) {
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

					// commits are non-nil and the same until intermediate target height
					for h := process.Height(1); h < intermTargetHeight; {
						Expect(commits[j][h]).To(Equal(referenceCommits[h]))
						Expect(commits[j][h]).ToNot(Equal(process.NilValue))
						h++
					}

					// commits are nil values after intermediate target height
					for h := process.Height(intermTargetHeight + 1); h <= targetHeight; {
						Expect(commits[j][h]).To(Equal(process.NilValue))
						h++
					}
				}
			}

			// callback function called on test success
			successFn := func() {
				cancel()
			}

			// callback function called on every message processed
			inspectFn := func(scenario *Scenario) {
				// we wish to kill only one replica
				if len(killedReplicas) < 1 {
					if r.Float64() < 0.003 {
						// get a random replica to kill
						id := r.Intn(int(n))

						// if its not yet killed
						if _, isKilled := killedReplicas[uint8(id)]; !isKilled {
							var err error
							intermTargetHeight, _, _, err = replicas[id].State(context.Background())
							if err != nil {
								panic(err)
							}

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
				replay(&scenario, mqMutex, mqSignal, completionSignal, replicas, &killedReplicas, successFn, inspectFn)
			}
		})
	})
})

// Scenario describes a test scenario with test configuration and message history
type Scenario struct {
	seed        int64
	f           uint8
	n           uint8
	completion  uint8
	signatories []id.Signatory
	messages    []Message
}

// SizeHint implements surge SizeHinter for Scenario
func (s Scenario) SizeHint() int {
	return surge.SizeHint(s.seed) +
		surge.SizeHint(s.f) +
		surge.SizeHint(s.n) +
		surge.SizeHint(s.completion) +
		surge.SizeHint(s.signatories) +
		surge.SizeHint(s.messages)
}

// Marshal implements surge Marshaler for Scenario
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

// Unmarshal implements surge Unmarshaler for Scenario
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

// Message describes a message sent between two replicas in the consensus
// mechanism
type Message struct {
	to          uint8
	messageType uint8
	value       surge.Marshaler
}

// SizeHint implements surge SizeHinter for Message
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
	case timer.Timeout:
		return surge.SizeHint(m.to) +
			surge.SizeHint(m.messageType) +
			surge.SizeHint(value)
	default:
		panic(fmt.Errorf("non-exhaustive pattern: message.value has type %T", value))
	}
}

// Marshal implements surge Marshaler for Message
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
	case timer.Timeout:
		buf, rem, err = surge.Marshal(value, buf, rem)
		if err != nil {
			return buf, rem, fmt.Errorf("marshaling value=%v: %v", value, err)
		}
	default:
		panic(fmt.Errorf("non-exhaustive pattern: message.value has type %T", value))
	}

	return buf, rem, nil
}

// Unmarshal implements surge Unmarshaler for Message
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
	case 4:
		message := timer.Timeout{}
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

// readBoolEnvVar reads an environment variable `name` and returns its boolean value
func readBoolEnvVar(name string) bool {
	envVar := os.Getenv(name)
	boolEnvVar, err := strconv.ParseBool(envVar)
	if err != nil {
		panic(fmt.Errorf("error parsing env var %v to boolean", name))
	}
	return boolEnvVar
}

// dumpToFile dumps a serialised test scenario data to a dump file
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

// readFromFile reads and deserialises a dump file to return the test scenario
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
