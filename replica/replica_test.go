package replica_test

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"testing"
	"time"

	"github.com/renproject/hyperdrive/process"
	"github.com/renproject/hyperdrive/process/processutil"
	"github.com/renproject/hyperdrive/replica"
	"github.com/renproject/hyperdrive/timer"
	"github.com/renproject/id"
)

// TODO: Implement the other scenarios.
//
// 1. 3F+1 replicas online
// 2. 2F+1 replicas online
// 3. 3F+1 replicas online, and F replicas slowly go offline
// 4. 3F+1 replicas online, and F replicas slowly go offline and then come back online
// 5. 3F+1 replicas online, and F replicas are behaving maliciously
//    - Proposal: malformed, missing, fork-attempt, out-of-turn
//    - Prevote: yes-to-malformed, no-to-valid, missing, fork-attempt, out-of-turn
//    - Precommit: yes-to-malformed, no-to-valid, missing, fork-attempt, out-of-turn

func TestReplica(t *testing.T) {
	n := 4
	messageDropRate := 0.0
	debugMode := true
	messageShuffling := false

	// Setup some random private keys that will be used by the replicas to
	// identify themselves.
	privKeys := make([]*id.PrivKey, n)
	for i := range privKeys {
		privKeys[i] = id.NewPrivKey()
	}

	// Get the public signatory identities associated with each private key.
	signatories := make([]id.Signatory, n)
	for i := range signatories {
		signatories[i] = privKeys[i].Signatory()
	}

	type message struct {
		to    int         // index
		value interface{} // value being sent
	}
	mqHistory := []message{}
	mq := []message{}
	mqSignal := make(chan struct{}, n*n)
	mqSignal <- struct{}{}

	// Store the random seed in a local variable so that we can save/restore it
	// when re-running tests multiple times.
	rSeed := time.Now().UnixNano()
	r := rand.New(rand.NewSource(rSeed))

	// Build replicas.
	replicas := make([]*replica.Replica, n)
	for i := range replicas {
		replicas[i] = replica.New(
			replica.DefaultOptions().
				WithTimerOptions(
					timer.DefaultOptions().
						WithTimeout(100*time.Second),
				),
			signatories[i],
			signatories,

			// Proposer
			processutil.MockProposer{
				MockValue: func() process.Value {
					v := processutil.RandomValue(r)
					fmt.Printf("%v is proposing %v\n", signatories[i], v)
					return v
				},
			},
			// Validator
			processutil.MockValidator{
				MockValid: func() bool {
					return true
				},
			},
			// Committer
			processutil.CommitterCallback{
				Callback: func(height process.Height, value process.Value) {
					fmt.Printf("%v committed %v at %v\n", signatories[i], value, height)
				},
			},
			// Catcher
			nil,
			// Broadcaster
			processutil.BroadcasterCallbacks{
				BroadcastProposeCallback: func(propose process.Propose) {
					for i := 0; i < n; i++ {
						mq = append(mq, message{
							to:    i,
							value: propose,
						})
					}
				},
				BroadcastPrevoteCallback: func(prevote process.Prevote) {
					for i := 0; i < n; i++ {
						mq = append(mq, message{
							to:    i,
							value: prevote,
						})
					}
				},
				BroadcastPrecommitCallback: func(precommit process.Precommit) {
					for i := 0; i < n; i++ {
						mq = append(mq, message{
							to:    i,
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

	time.Sleep(100 * time.Millisecond)
	for range mqSignal {
		if len(mq) == 0 {
			println("message queue is empty")
			continue // Spurious signal was received.
		}

		// Shuffle the message queue to simulate "out of order" message
		// delivery.
		if messageShuffling {
			rand.Shuffle(len(mq), func(i, j int) {
				m := mq[i]
				mq[i] = mq[j]
				mq[j] = m
			})
		}

		// Pop the first message from the queue.
		m := mq[0]
		mq = mq[1:]
		mqHistory = append(mqHistory, m)

		// Simulate random message drop rate of 5%.
		if r.Float64() < messageDropRate {
			mqSignal <- struct{}{}
			continue
		}

		data, err := json.MarshalIndent(m.value, "  ", "  ")
		if err != nil {
			panic(err)
		}
		fmt.Printf("%v is receiving %T %v\n\n", signatories[m.to], m.value, string(data))

		if debugMode {
			// fmt.Printf("Press ENTER to step:")
			// // in := bufio.NewScanner(os.Stdin)
			// // in.Scan()
			// var in string
			// fmt.Scanln(&in)
			time.Sleep(100 * time.Millisecond)
		}

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
