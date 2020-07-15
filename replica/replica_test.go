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
// 3. 3F+1 replicas online, and F replicas slowly go offline (wip)
// 4. 3F+1 replicas online, and F replicas slowly go offline and then come back online
// 5. 3F+1 replicas online, and F replicas are behaving maliciously
//    - Proposal: malformed, missing, fork-attempt, out-of-turn
//    - Prevote: yes-to-malformed, no-to-valid, missing, fork-attempt, out-of-turn
//    - Precommit: yes-to-malformed, no-to-valid, missing, fork-attempt, out-of-turn

var _ = Describe("Replica", func() {
	dumpToFile := func(filename string, scenario Scenario) {
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

	readFromFile := func(filename string) Scenario {
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

	Context("with 3f+1 replicas online", func() {
		It("should be able to reach consensus", func() {
			// randomness seed
			rSeed := time.Now().UnixNano()
			r := rand.New(rand.NewSource(rSeed))

			// do we store message history
			mh := os.Getenv("MESSAGE_HISTORY")
			captureHistory, err := strconv.ParseBool(mh)
			if err != nil {
				panic("error reading environment variable MESSAGE_HISTORY")
			}

			// is this running the test in replay mode
			rm := os.Getenv("REPLAY_MODE")
			replayMode, err := strconv.ParseBool(rm)
			if err != nil {
				panic("error reading environment variable REPLAY_MODE")
			}

			// f is the maximum no. of adversaries
			// n is the number of honest replicas online
			// h is the target minimum consensus height
			f := uint8(3)
			n := 3*f + 1
			targetHeight := process.Height(15)

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

			// read from the dump file if we're replaying a test scenario
			var scenario Scenario
			if replayMode {
				scenario = readFromFile("failure.dump")
				f = scenario.f
				n = scenario.n
				signatories = scenario.signatories
			}

			// every replica sends this signal when they reach the target
			// consensus height
			completionSignal := make(chan bool, n)

			// slice of messages to be broadcasted between replicas.
			// messages from mq are popped and handled whenever mqSignal is signalled
			mq := []Message{}
			var mqMutex = &sync.Mutex{}
			messageHistory := []Message{}
			mqSignal := make(chan struct{}, n*n)
			mqSignal <- struct{}{}

			// build replicas
			replicas := make([]*replica.Replica, n)
			for i := range replicas {
				replicaIndex := uint8(i)

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
							mqMutex.Lock()
							for j := uint8(0); j < n; j++ {
								mq = append(mq, Message{
									to:          j,
									messageType: 1,
									value:       propose,
								})
							}
							mqMutex.Unlock()
						},
						BroadcastPrevoteCallback: func(prevote process.Prevote) {
							mqMutex.Lock()
							for j := uint8(0); j < n; j++ {
								mq = append(mq, Message{
									to:          j,
									messageType: 2,
									value:       prevote,
								})
							}
							mqMutex.Unlock()
						},
						BroadcastPrecommitCallback: func(precommit process.Precommit) {
							mqMutex.Lock()
							for j := uint8(0); j < n; j++ {
								mq = append(mq, Message{
									to:          j,
									messageType: 3,
									value:       precommit,
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

			// From the global context, create per-replica contexts that can be used to
			// cancel replicas independently of one another. This is useful when we want
			// to simulate replicas "crashing" and "restarting" during consensus.
			replicaCtxs, replicaCtxCancels := make([]context.Context, n), make([]context.CancelFunc, n)
			for i := range replicaCtxs {
				replicaCtxs[i], replicaCtxCancels[i] = context.WithCancel(ctx)
			}

			// in case the test fails to proceed to completion, we have a signal
			// to mark it as failed
			// this test should take 35 seconds to complete
			failTestSignal := make(chan bool, 1)
			if !replayMode {
				go func() {
					time.Sleep(20 * time.Second)
					failTestSignal <- true
				}()
			}

			// Run all of the replicas in independent background goroutines.
			for i := range replicas {
				go replicas[i].Run(replicaCtxs[i])
			}

			failTest := func() {
				cancel()
				if captureHistory {
					dumpToFile("failure.dump", Scenario{
						f:           f,
						n:           n,
						signatories: signatories,
						messages:    messageHistory,
					})
				}
				Fail("test failed to complete within the expected timeframe")
			}

			completion := func() {
				// cancel the replica contexts
				for i := range replicaCtxs {
					replicaCtxCancels[i]()
				}

				// ensure that all replicas have the same commits
				referenceCommits := commits[0]
				for j := uint8(0); j < n; j++ {
					for h := process.Height(1); h <= targetHeight; {
						Expect(commits[j][h]).To(Equal(referenceCommits[h]))
						h++
					}
				}
			}

			time.Sleep(100 * time.Millisecond)
			isRunning := true
			for isRunning && !replayMode {
				select {
				case _ = <-failTestSignal:
					failTest()
				case _ = <-mqSignal:
					// this means the target consensus has been reached on every replica
					if len(completionSignal) == int(n) {
						completion()
						isRunning = false
						continue
					}

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

					// ignore if its a nil message
					if m.value == nil {
						continue
					}

					// append the message to message history
					if captureHistory {
						messageHistory = append(messageHistory, m)
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

			if replayMode {
				for _, message := range scenario.messages {
					time.Sleep(5 * time.Millisecond)

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

					if len(completionSignal) == int(n) {
						completion()
					}
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
			f := uint8(3)
			n := 2*f + 1
			targetHeight := process.Height(12)

			// commits from replicas
			commits := make(map[uint8]map[process.Height]process.Value)

			// setup private keys for the replicas
			// and their signatories
			privKeys := make([]*id.PrivKey, 3*f+1)
			signatories := make([]id.Signatory, 3*f+1)
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
					replica.DefaultOptions().WithTimerOptions(timer.DefaultOptions().WithTimeout(1*time.Second)),
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
							mqMutex.Lock()
							for j := uint8(0); j < n; j++ {
								mq = append(mq, Message{
									to:          j,
									messageType: 1,
									value:       propose,
								})
							}
							mqMutex.Unlock()
						},
						BroadcastPrevoteCallback: func(prevote process.Prevote) {
							mqMutex.Lock()
							for j := uint8(0); j < n; j++ {
								mq = append(mq, Message{
									to:          j,
									messageType: 2,
									value:       prevote,
								})
							}
							mqMutex.Unlock()
						},
						BroadcastPrecommitCallback: func(precommit process.Precommit) {
							mqMutex.Lock()
							for j := uint8(0); j < n; j++ {
								mq = append(mq, Message{
									to:          j,
									messageType: 3,
									value:       precommit,
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
			failTestSignal := make(chan bool, 1)
			go func() {
				time.Sleep(40 * time.Second)
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

			failTest := func() {
				cancel()
				Fail("test failed to complete within the expected timeframe")
			}

			completion := func() {
				// cancel the replica contexts
				for i := range replicaCtxs {
					replicaCtxCancels[i]()
				}

				// ensure that all replicas have the same commits
				referenceCommits := commits[0]
				for j := uint8(0); j < n; j++ {
					for h := process.Height(1); h <= targetHeight; {
						Expect(commits[j][h]).To(Equal(referenceCommits[h]))
						h++
					}
				}
			}

			time.Sleep(100 * time.Millisecond)
			isRunning := true
			for isRunning {
				select {
				case _ = <-failTestSignal:
					failTest()
				case _ = <-mqSignal:
					// this means the target consensus has been reached on every replica
					if len(completionSignal) == int(n) {
						completion()
						isRunning = false
						continue
					}

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

	Context("with 3f+1 replicas online, f replicas slowly go offline", func() {
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

			// replicas randomly send this signal that kills them
			// this is to simulate a behaviour where f replicas go offline
			killedReplicas := make(map[uint8]bool)
			killSignal := make(chan uint8, f)

			// lastKilled stores the last height at which a replica was killed
			// killInterval is the number of blocks progressed after which
			// a replica should be killed
			lastKilled := process.Height(0)
			killInterval := process.Height(2)

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
							mqMutex.Lock()
							if height-lastKilled > killInterval {
								if len(killedReplicas) < int(f) {
									killSignal <- replicaIndex
									lastKilled = height
								}
							}
							mqMutex.Unlock()
						},
					},
					// Catcher
					nil,
					// Broadcaster
					processutil.BroadcasterCallbacks{
						BroadcastProposeCallback: func(propose process.Propose) {
							mqMutex.Lock()
							for j := uint8(0); j < n; j++ {
								if _, ok := killedReplicas[j]; !ok {
									mq = append(mq, Message{
										to:    j,
										value: propose,
									})
								}
							}
							mqMutex.Unlock()
						},
						BroadcastPrevoteCallback: func(prevote process.Prevote) {
							mqMutex.Lock()
							for j := uint8(0); j < n; j++ {
								if _, ok := killedReplicas[j]; !ok {
									mq = append(mq, Message{
										to:    j,
										value: prevote,
									})
								}
							}
							mqMutex.Unlock()
						},
						BroadcastPrecommitCallback: func(precommit process.Precommit) {
							mqMutex.Lock()
							for j := uint8(0); j < n; j++ {
								if _, ok := killedReplicas[j]; !ok {
									mq = append(mq, Message{
										to:    j,
										value: precommit,
									})
								}
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
			failTestSignal := make(chan bool, 1)
			go func() {
				time.Sleep(60 * time.Second)
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
				for i := uint8(0); i < n; i++ {
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
						continue
					}

					// kill a replica if there has been a kill signal
					if len(killSignal) > 0 {
						killIndex := <-killSignal
						replicaCtxCancels[killIndex]()
						killedReplicas[killIndex] = true
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
					time.Sleep(10 * time.Millisecond)
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

type Scenario struct {
	f           uint8
	n           uint8
	signatories []id.Signatory
	messages    []Message
}

func (s Scenario) SizeHint() int {
	return surge.SizeHint(s.f) +
		surge.SizeHint(s.n) +
		surge.SizeHint(s.signatories) +
		surge.SizeHint(s.messages)
}

func (s Scenario) Marshal(buf []byte, rem int) ([]byte, int, error) {
	buf, rem, err := surge.Marshal(s.f, buf, rem)
	if err != nil {
		return buf, rem, fmt.Errorf("marshaling f=%v: %v", s.f, err)
	}
	buf, rem, err = surge.Marshal(s.n, buf, rem)
	if err != nil {
		return buf, rem, fmt.Errorf("marshaling n=%v: %v", s.n, err)
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
	buf, rem, err := surge.Unmarshal(&s.f, buf, rem)
	if err != nil {
		return buf, rem, fmt.Errorf("unmarshaling f: %v", err)
	}
	buf, rem, err = surge.Unmarshal(&s.n, buf, rem)
	if err != nil {
		return buf, rem, fmt.Errorf("unmarshaling n: %v", err)
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
