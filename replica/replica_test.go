package replica

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/rand"
	"io/ioutil"
	"reflect"
	"testing/quick"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/renproject/hyperdrive/block"
	"github.com/renproject/hyperdrive/process"
	"github.com/renproject/hyperdrive/schedule"
	"github.com/renproject/hyperdrive/testutil"
	"github.com/renproject/surge"
	"github.com/sirupsen/logrus"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/renproject/hyperdrive/testutil"
)

var _ = Describe("Replica", func() {

	newEcdsaKey := func() *ecdsa.PrivateKey {
		privateKey, err := ecdsa.GenerateKey(crypto.S256(), rand.Reader)
		Expect(err).NotTo(HaveOccurred())
		return privateKey
	}

	Context("shard", func() {
		Context("when comparing two shard", func() {
			It("should be stringified to same text if two shards are equal and vice versa", func() {
				test := func(shard1, shard2 Shard) bool {
					shard := shard1
					Expect(shard.Equal(shard1)).Should(BeTrue())
					Expect(shard1.Equal(shard)).Should(BeTrue())
					Expect(shard.String()).Should(Equal(shard1.String()))

					Expect(shard1.Equal(shard2)).Should(BeFalse())
					Expect(shard1.String()).ShouldNot(Equal(shard2.String()))

					return true
				}

				Expect(quick.Check(test, nil)).Should(Succeed())
			})
		})
	})

	Context("replica", func() {
		Context("when marshaling/unmarshaling message", func() {
			It("should equal itself after binary marshaling and then unmarshaling", func() {
				message := Message{
					Message: RandomMessage(RandomMessageType(true)),
					Shard:   Shard{},
				}

				data, err := surge.ToBinary(message)
				Expect(err).NotTo(HaveOccurred())
				newMessage := Message{}
				Expect(surge.FromBinary(data, &newMessage)).Should(Succeed())

				newData, err := surge.ToBinary(newMessage)
				Expect(err).NotTo(HaveOccurred())
				Expect(bytes.Equal(data, newData)).Should(BeTrue())
			})
		})

		Context("when sending messages to replica", func() {
			It("should only pass message to process when it's a valid message", func() {
				test := func(shard, wrongShard Shard) bool {
					store, _, keys := initStorage(shard)
					pstore := mockProcessStorage{}
					broadcaster, _, _ := newMockBroadcaster()
					scheduler := schedule.RoundRobin(store.LatestBaseBlock(shard).Header().Signatories())
					replica := New(Options{}, pstore, store, mockBlockIterator{}, nil, nil, broadcaster, scheduler, process.CatchAndIgnore(), shard, *newEcdsaKey())

					pMessage := RandomMessage(process.ProposeMessageType)
					numStored := 0
					// Only one proposer is valid, so only one propose should
					// end up stored in the Process state.
					for _, key := range keys {
						Expect(process.Sign(pMessage, *key)).Should(Succeed())
						message := Message{
							Shard:   shard,
							Message: pMessage,
						}
						replica.HandleMessage(message)

						// Expect the message not been inserted into the specific inbox,
						// which indicating the message not passed to the process.
						state := testutil.GetStateFromProcess(replica.p, 2)
						stored := state.Proposals.QueryByHeightRoundSignatory(pMessage.Height(), pMessage.Round(), pMessage.Signatory())
						if reflect.DeepEqual(stored, pMessage) {
							numStored++
						}
					}
					Expect(numStored).To(Equal(1))

					return true
				}

				Expect(quick.Check(test, nil)).Should(Succeed())
			})

			It("should reject message of different shard", func() {
				test := func(shard, wrongShard Shard) bool {
					store, _, _ := initStorage(shard)
					pstore := mockProcessStorage{}
					broadcaster, _, _ := newMockBroadcaster()
					scheduler := schedule.RoundRobin(store.LatestBaseBlock(shard).Header().Signatories())
					replica := New(Options{}, pstore, store, mockBlockIterator{}, nil, nil, broadcaster, scheduler, process.CatchAndIgnore(), shard, *newEcdsaKey())
					logger := logrus.StandardLogger()
					logger.SetOutput(ioutil.Discard)
					replica.options.Logger = logger

					pMessage := RandomSignedMessage(process.ProposeMessageType)
					message := Message{
						Shard:   wrongShard,
						Message: pMessage,
					}
					replica.HandleMessage(message)

					// Expect the message not been inserted into the specific inbox,
					// which indicating the message not passed to the process.
					state := testutil.GetStateFromProcess(replica.p, 2)
					stored := state.Proposals.QueryByHeightRoundSignatory(pMessage.Height(), pMessage.Round(), pMessage.Signatory())
					Expect(stored).Should(BeNil())

					return true
				}

				Expect(quick.Check(test, nil)).Should(Succeed())
			})

			It("should reject message whose signatory is not valid", func() {
				test := func(shard Shard) bool {
					store, _, _ := initStorage(shard)
					pstore := mockProcessStorage{}
					broadcaster, _, _ := newMockBroadcaster()
					scheduler := schedule.RoundRobin(store.LatestBaseBlock(shard).Header().Signatories())
					replica := New(Options{}, pstore, store, mockBlockIterator{}, nil, nil, broadcaster, scheduler, process.CatchAndIgnore(), shard, *newEcdsaKey())

					pMessage := RandomSignedMessage(process.ProposeMessageType)
					message := Message{
						Shard:   shard,
						Message: pMessage,
					}
					replica.HandleMessage(message)

					// Expect the message not been inserted into the specific inbox,
					// which indicating the message not passed to the process.
					state := testutil.GetStateFromProcess(replica.p, 2)
					stored := state.Proposals.QueryByHeightRoundSignatory(pMessage.Height(), pMessage.Round(), pMessage.Signatory())
					Expect(stored).Should(BeNil())

					return true
				}

				Expect(quick.Check(test, nil)).Should(Succeed())
			})
		})
	})

	Context("when sending resync messages to replica", func() {
		Context("when the resync message is greater than the current height", func() {
			It("should resend nothing", func() {
				store, _, _ := initStorage(Shard{})
				pstore := mockProcessStorage{}
				broadcaster, _, castMessages := newMockBroadcaster()
				replica := New(Options{}, pstore, store, mockBlockIterator{}, nil, nil, broadcaster, schedule.RoundRobin(testutil.RandomSignatories()), nil, Shard{}, *newEcdsaKey())
				logger := logrus.StandardLogger()
				logger.SetOutput(ioutil.Discard)
				replica.options.Logger = logger

				replica.HandleMessage(Message{Message: process.NewResync(block.Height(999), block.Round(0)), Shard: Shard{}})
				select {
				case <-time.After(time.Second):
				case <-castMessages:
					Expect(func() { panic("should resent nothing") }).ToNot(Panic())
				}
			})
		})

		Context("when outside the current time", func() {
			FIt("should resend nothing", func() {
				store, _, _ := initStorage(Shard{})
				pstore := mockProcessStorage{}
				broadcaster, _, castMessages := newMockBroadcaster()
				replica := New(Options{}, pstore, store, mockBlockIterator{}, nil, nil, broadcaster, schedule.RoundRobin(testutil.RandomSignatories()), nil, Shard{}, *newEcdsaKey())
				logger := logrus.StandardLogger()
				logger.SetOutput(ioutil.Discard)
				replica.options.Logger = logger

				resync := Message{Message: process.NewResync(block.Height(0), block.Round(0)), Shard: Shard{}}
				time.Sleep(11 * time.Second)
				replica.HandleMessage(resync)
				select {
				case <-time.After(time.Second):
				case <-castMessages:
					Expect(func() { panic("should resent nothing") }).ToNot(Panic())
				}
			})
		})
	})
})
