package process_test

import (
	"crypto/ecdsa"
	cRand "crypto/rand"
	"encoding/json"
	"math/rand"
	"reflect"
	"testing/quick"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/renproject/hyperdrive/block"
	"github.com/renproject/id"
	"github.com/renproject/surge"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/renproject/hyperdrive/process"
	. "github.com/renproject/hyperdrive/testutil"
)

var _ = Describe("Messages", func() {

	Context("Propose", func() {
		Context("when initializing", func() {
			It("should return a message with fields equal to those passed during creation", func() {
				test := func() bool {
					height := block.Height(rand.Int63())
					round := block.Round(rand.Int63())
					validRound := block.Round(rand.Int63())
					block := RandomBlock(RandomBlockKind())

					propose := NewPropose(height, round, block, validRound)

					Expect(propose.Type()).Should(Equal(MessageType(ProposeMessageType)))
					Expect(propose.Height()).Should(Equal(height))
					Expect(propose.Round()).Should(Equal(round))
					Expect(propose.ValidRound()).Should(Equal(validRound))
					Expect(propose.Block().Equal(block)).Should(BeTrue())
					return true
				}
				Expect(quick.Check(test, nil)).Should(Succeed())
			})
		})

		Context("when stringifying", func() {
			It("should return equal strings", func() {
				test := func() bool {
					msg := RandomPropose()
					newMsg := msg
					return msg.String() == newMsg.String()
				}

				Expect(quick.Check(test, nil)).Should(Succeed())
			})

			Context("when unequal", func() {
				It("should return unequal strings", func() {
					test := func() bool {
						msg1, msg2 := RandomPropose(), RandomPropose()
						return msg1.String() != msg2.String()
					}

					Expect(quick.Check(test, nil)).Should(Succeed())
				})
			})
		})

		Context("when marshaling random", func() {
			It("should equal itself after binary marshaling and then unmarshaling", func() {
				test := func() bool {
					msg := RandomPropose()
					data, err := surge.ToBinary(msg)
					Expect(err).NotTo(HaveOccurred())

					var newMsg Propose
					Expect(surge.FromBinary(data, &newMsg)).Should(Succeed())
					return msg.String() == newMsg.String()
				}

				Expect(quick.Check(test, nil)).Should(Succeed())
			})
		})

		Context("when signing and verifying", func() {
			It("should verify if a message if has been signed properly", func() {
				test := func() bool {
					propose := RandomPropose()
					Expect(Verify(propose)).ShouldNot(Succeed())

					privateKey, err := ecdsa.GenerateKey(crypto.S256(), cRand.Reader)
					Expect(err).NotTo(HaveOccurred())
					Expect(Sign(propose, *privateKey)).Should(Succeed())
					Expect(Verify(propose)).Should(Succeed())

					return true
				}

				Expect(quick.Check(test, nil)).Should(Succeed())
			})
		})
	})

	Context("Prevote", func() {
		Context("when initializing", func() {
			It("should return a message with fields equal to those passed during creation", func() {
				test := func() bool {
					height := block.Height(rand.Int63())
					round := block.Round(rand.Int63())
					blockHash := RandomHash()

					prevote := NewPrevote(height, round, blockHash, nil)

					Expect(prevote.Type()).Should(Equal(MessageType(PrevoteMessageType)))
					Expect(prevote.Height()).Should(Equal(height))
					Expect(prevote.Round()).Should(Equal(round))
					Expect(prevote.BlockHash().Equal(blockHash)).Should(BeTrue())
					return true
				}
				Expect(quick.Check(test, nil)).Should(Succeed())
			})
		})

		Context("when stringifying", func() {
			It("should return equal strings", func() {
				test := func() bool {
					msg := RandomPrevote()
					newMsg := msg
					return msg.String() == newMsg.String()
				}

				Expect(quick.Check(test, nil)).Should(Succeed())
			})

			Context("when unequal", func() {
				It("should return unequal strings", func() {
					test := func() bool {
						msg1, msg2 := RandomPrevote(), RandomPrevote()
						return msg1.String() != msg2.String()
					}

					Expect(quick.Check(test, nil)).Should(Succeed())
				})
			})
		})

		Context("when marshaling random", func() {
			It("should equal itself after binary marshaling and then unmarshaling", func() {
				test := func() bool {
					msg := RandomPrevote()
					data, err := surge.ToBinary(msg)
					Expect(err).NotTo(HaveOccurred())

					var newMsg Prevote
					Expect(surge.FromBinary(data, &newMsg)).Should(Succeed())
					return msg.String() == newMsg.String()
				}

				Expect(quick.Check(test, nil)).Should(Succeed())
			})
		})

		Context("when signing and verifying", func() {
			It("should verify if a message if has been signed properly", func() {
				test := func() bool {
					prevote := RandomPrevote()
					Expect(Verify(prevote)).ShouldNot(Succeed())

					privateKey, err := ecdsa.GenerateKey(crypto.S256(), cRand.Reader)
					Expect(err).NotTo(HaveOccurred())
					Expect(Sign(prevote, *privateKey)).Should(Succeed())
					Expect(Verify(prevote)).Should(Succeed())

					return true
				}

				Expect(quick.Check(test, nil)).Should(Succeed())
			})
		})
	})

	Context("Precommit", func() {
		Context("when initializing", func() {
			It("should return a message with fields equal to those passed during creation", func() {
				test := func() bool {
					height := block.Height(rand.Int63())
					round := block.Round(rand.Int63())
					blockHash := RandomHash()

					precommit := NewPrecommit(height, round, blockHash)

					Expect(precommit.Type()).Should(Equal(MessageType(PrecommitMessageType)))
					Expect(precommit.Height()).Should(Equal(height))
					Expect(precommit.Round()).Should(Equal(round))
					Expect(precommit.BlockHash().Equal(blockHash)).Should(BeTrue())
					return true
				}
				Expect(quick.Check(test, nil)).Should(Succeed())
			})
		})

		Context("when stringifying", func() {
			It("should return equal strings", func() {
				test := func() bool {
					msg := RandomPrecommit()
					newMsg := msg
					return msg.String() == newMsg.String()
				}

				Expect(quick.Check(test, nil)).Should(Succeed())
			})

			Context("when unequal", func() {
				It("should return unequal strings", func() {
					test := func() bool {
						msg1, msg2 := RandomPrecommit(), RandomPrecommit()
						return msg1.String() != msg2.String()
					}

					Expect(quick.Check(test, nil)).Should(Succeed())
				})
			})
		})

		Context("when marshaling random", func() {
			It("should equal itself after binary marshaling and then unmarshaling", func() {
				test := func() bool {
					msg := RandomPrecommit()
					data, err := surge.ToBinary(msg)
					Expect(err).NotTo(HaveOccurred())

					var newMsg Precommit
					Expect(surge.FromBinary(data, &newMsg)).Should(Succeed())
					return msg.String() == newMsg.String()
				}

				Expect(quick.Check(test, nil)).Should(Succeed())
			})
		})

		Context("when signing and verifying", func() {
			It("should verify if a message if has been signed properly", func() {
				test := func() bool {
					precommit := RandomPrecommit()
					Expect(Verify(precommit)).ShouldNot(Succeed())

					privateKey, err := ecdsa.GenerateKey(crypto.S256(), cRand.Reader)
					Expect(err).NotTo(HaveOccurred())
					Expect(Sign(precommit, *privateKey)).Should(Succeed())
					Expect(Verify(precommit)).Should(Succeed())

					return true
				}

				Expect(quick.Check(test, nil)).Should(Succeed())
			})
		})
	})

	Context("Resync", func() {
		Context("when initializing", func() {
			It("should return a message with fields equal to those passed during creation", func() {
				test := func() bool {
					height := block.Height(rand.Int63())
					round := block.Round(rand.Int63())

					resync := NewResync(height, round)

					Expect(resync.Type()).Should(Equal(MessageType(ResyncMessageType)))
					Expect(resync.Height()).Should(Equal(height))
					Expect(resync.Round()).Should(Equal(round))
					Expect(func() {
						resync.BlockHash()
					}).Should(Panic())

					return true
				}
				Expect(quick.Check(test, nil)).Should(Succeed())
			})
		})

		Context("when stringifying", func() {
			It("should return equal strings", func() {
				test := func() bool {
					msg := RandomResync()
					newMsg := msg
					return msg.String() == newMsg.String()
				}

				Expect(quick.Check(test, nil)).Should(Succeed())
			})

			Context("when unequal", func() {
				It("should return unequal strings", func() {
					test := func() bool {
						msg1, msg2 := RandomResync(), RandomResync()
						return msg1.String() != msg2.String()
					}

					Expect(quick.Check(test, nil)).Should(Succeed())
				})
			})
		})

		Context("when marshaling random", func() {
			It("should equal itself after json marshaling and then unmarshaling", func() {
				test := func() bool {
					msg := RandomResync()
					data, err := json.Marshal(msg)
					Expect(err).NotTo(HaveOccurred())

					var newMsg Resync
					Expect(json.Unmarshal(data, &newMsg)).Should(Succeed())
					return msg.String() == newMsg.String()
				}

				Expect(quick.Check(test, nil)).Should(Succeed())
			})

			It("should equal itself after binary marshaling and then unmarshaling", func() {
				test := func() bool {
					msg := RandomResync()
					data, err := msg.MarshalBinary()
					Expect(err).NotTo(HaveOccurred())

					var newMsg Resync
					Expect(newMsg.UnmarshalBinary(data)).Should(Succeed())
					return msg.String() == newMsg.String()
				}

				Expect(quick.Check(test, nil)).Should(Succeed())
			})
		})

		Context("when signing and verifying", func() {
			It("should verify if a message if has been signed properly", func() {
				test := func() bool {
					resync := RandomResync()
					Expect(Verify(resync)).ShouldNot(Succeed())

					privateKey, err := ecdsa.GenerateKey(crypto.S256(), cRand.Reader)
					Expect(err).NotTo(HaveOccurred())
					Expect(Sign(resync, *privateKey)).Should(Succeed())
					Expect(Verify(resync)).Should(Succeed())

					return true
				}

				Expect(quick.Check(test, nil)).Should(Succeed())
			})
		})
	})

	Context("when initializing a new inbox", func() {
		It("should have the given f and message type", func() {
			test := func() bool {
				messageType := RandomMessageType(false)
				f := rand.Int() + 1
				inbox := RandomInbox(f, messageType)
				Expect(inbox.F()).Should(Equal(f))
				Expect(messageType).Should(Equal(inbox.MessageType()))
				return true
			}
			Expect(quick.Check(test, nil)).Should(Succeed())
		})

		It("should panic when passing a invalid f", func() {
			test := func() bool {
				messageType := RandomMessageType(false)

				// Should panic when passing 0
				Expect(func() {
					_ = RandomInbox(0, messageType)
				}).Should(Panic())

				// Should panic when passing negative number
				Expect(func() {
					_ = RandomInbox(-1*rand.Int(), messageType)
				}).Should(Panic())
				return true
			}
			Expect(quick.Check(test, nil)).Should(Succeed())
		})

		It("should panic when passing a nil message type", func() {
			test := func() bool {
				f := rand.Int() + 1
				Expect(func() {
					_ = RandomInbox(f, NilMessageType)
				}).Should(Panic())
				return true
			}
			Expect(quick.Check(test, nil)).Should(Succeed())
		})
	})

	Context("when marshaling a random inbox", func() {
		It("should equal itself after binary marshaling and then unmarshaling", func() {
			test := func() bool {
				messageType := RandomMessageType(false)
				f := rand.Int() + 1
				inbox := RandomInbox(f, messageType)
				Expect(inbox.F()).Should(Equal(f))
				data, err := surge.ToBinary(inbox)
				Expect(err).NotTo(HaveOccurred())

				newInbox := NewInbox(1, messageType)
				Expect(surge.FromBinary(data, newInbox)).Should(Succeed())
				Expect(inbox.F()).To(Equal(newInbox.F()))
				Expect(inbox.MessageType()).To(Equal(newInbox.MessageType()))

				return reflect.DeepEqual(inbox, newInbox)
			}
			Expect(quick.Check(test, nil)).Should(Succeed())
		})
	})

	Context("when inserting messages into an inbox", func() {
		Context("when we first insert message to an inbox", func() {
			It("should return n=1, firstTime=true, firstTimeExceedingF=false, and firstTimeExceeding2F=false", func() {
				test := func() bool {
					f := rand.Intn(100) + 1
					messageType := RandomMessageType(false)
					inbox := NewInbox(f, messageType)
					n, firstTime, firstTimeExceedingF, firstTimeExceeding2F, _ := inbox.Insert(RandomMessage(messageType))
					Expect(n).Should(Equal(1))
					Expect(firstTime).Should(BeTrue())
					Expect(firstTimeExceedingF).Should(BeFalse())
					Expect(firstTimeExceeding2F).Should(BeFalse())

					return true
				}
				Expect(quick.Check(test, nil)).Should(Succeed())
			})
		})

		Context("when F + 1 messages are inserted", func() {
			It("should return n=F+1, firstTime=false, firstTimeExceedingF=true, and firstTimeExceeding2F=false", func() {
				test := func(height block.Height, round block.Round) bool {
					f := rand.Intn(100) + 1
					messageType := RandomMessageType(false)
					inbox := NewInbox(f, messageType)

					// Expect n, false, false, false when inserting no more than F messages
					for i := 1; i <= f; i++ {
						msg := RandomSingedMessageWithHeightAndRound(height, round, messageType)
						n, firstTime, firstTimeExceedingF, firstTimeExceeding2F, _ := inbox.Insert(msg)
						Expect(n).Should(Equal(i))
						if i == 1 {
							Expect(firstTime).Should(BeTrue())
						} else {
							Expect(firstTime).Should(BeFalse())
						}
						Expect(firstTimeExceedingF).Should(BeFalse())
						Expect(firstTimeExceeding2F).Should(BeFalse())
					}

					// Expect F+1, false, true, false when inserting F+1 message
					msg := RandomSingedMessageWithHeightAndRound(height, round, messageType)
					n, firstTime, firstTimeExceedingF, firstTimeExceeding2F, _ := inbox.Insert(msg)

					Expect(n).Should(Equal(f + 1))
					Expect(firstTime).Should(BeFalse())
					Expect(firstTimeExceedingF).Should(BeTrue())
					Expect(firstTimeExceeding2F).Should(BeFalse())

					return true
				}
				Expect(quick.Check(test, nil)).Should(Succeed())
			})
		})

		Context("when 2F + 1  messages are inserted", func() {
			It("should return n=2F+1, firstTime=false, firstTimeExceedingF=false, and firstTimeExceeding2F=true", func() {
				test := func(height block.Height, round block.Round) bool {
					f := rand.Intn(100) + 1
					messageType := RandomMessageType(false)
					inbox := NewInbox(f, messageType)

					// Expect n, false, false,false when inserting no more than F messages
					for i := 1; i <= 2*f; i++ {
						msg := RandomSingedMessageWithHeightAndRound(height, round, messageType)
						n, _, _, firstTimeExceeding2F, _ := inbox.Insert(msg)
						Expect(n).Should(Equal(i))
						Expect(firstTimeExceeding2F).Should(BeFalse())
					}

					// Expect 2F+1, false, true, false when inserting F+1 message
					msg := RandomSingedMessageWithHeightAndRound(height, round, messageType)
					n, firstTime, firstTimeExceedingF, firstTimeExceeding2F, _ := inbox.Insert(msg)

					Expect(n).Should(Equal(2*f + 1))
					Expect(firstTime).Should(BeFalse())
					Expect(firstTimeExceedingF).Should(BeFalse())
					Expect(firstTimeExceeding2F).Should(BeTrue())

					return true
				}
				Expect(quick.Check(test, nil)).Should(Succeed())
			})
		})

		Context("after 2F + 1  messages are inserted", func() {
			It("should return n=i, firstTime=false, firstTimeExceedingF=false, and firstTimeExceeding2F=false", func() {
				test := func(height block.Height, round block.Round) bool {
					f := rand.Intn(100) + 1
					messageType := RandomMessageType(false)
					inbox := NewInbox(f, messageType)

					// Expect n, false, false,false when inserting no more than F messages
					for i := 1; i <= 2*f+1; i++ {
						msg := RandomSingedMessageWithHeightAndRound(height, round, messageType)
						n, _, _, _, _ := inbox.Insert(msg)
						Expect(n).Should(Equal(i))
					}

					// Expect 3F+1, false, true, false when inserting F+1 message
					for i := 1; i < rand.Intn(100); i++ {
						msg := RandomSingedMessageWithHeightAndRound(height, round, messageType)
						n, firstTime, firstTimeExceedingF, firstTimeExceeding2F, _ := inbox.Insert(msg)

						Expect(n).Should(Equal(2*f + 1 + i))
						Expect(firstTime).Should(BeFalse())
						Expect(firstTimeExceedingF).Should(BeFalse())
						Expect(firstTimeExceeding2F).Should(BeFalse())
					}
					return true
				}
				Expect(quick.Check(test, nil)).Should(Succeed())
			})
		})

		Context("when querying by height, round and block hash", func() {
			It("should return the number of votes", func() {
				test := func(height block.Height, round block.Round) bool {
					f := rand.Intn(100) + 1
					messageType := RandomMessageType(false)
					inbox := NewInbox(f, messageType)

					source := map[block.Height]map[block.Round]map[id.Hash]int{}
					noMessages := rand.Intn(100)
					for i := 0; i < noMessages; i++ {
						msg := RandomSignedMessage(messageType)

						// Inserting the same msg twice should not affect anything
						_, _, _, _, _ = inbox.Insert(msg)
						_, _, _, _, _ = inbox.Insert(msg)

						if _, ok := source[msg.Height()]; !ok {
							source[msg.Height()] = map[block.Round]map[id.Hash]int{}
						}
						if _, ok := source[msg.Height()][msg.Round()]; !ok {
							source[msg.Height()][msg.Round()] = map[id.Hash]int{}
						}
						source[msg.Height()][msg.Round()][msg.BlockHash()]++
					}

					// Expect the query function gives us the same result as the source.
					for height, roundMap := range source {
						for round, hashMap := range roundMap {
							for hash, num := range hashMap {
								Expect(inbox.QueryByHeightRoundBlockHash(height, round, hash)).Should(Equal(num))
							}
						}
					}
					return true
				}
				Expect(quick.Check(test, nil)).Should(Succeed())
			})
		})

		Context("when querying by height, round and signatory", func() {
			It("should return the message if exist", func() {
				test := func(height block.Height, round block.Round) bool {
					f := rand.Intn(100) + 1
					messageType := RandomMessageType(false)
					inbox := NewInbox(f, messageType)

					noMessages := rand.Intn(100)
					for i := 0; i < noMessages; i++ {
						msg := RandomSignedMessage(messageType)

						// It should return nil before inserting into the inbox.
						nilMessage := inbox.QueryByHeightRoundSignatory(msg.Height(), msg.Round(), msg.Signatory())
						Expect(nilMessage).Should(BeNil())

						// Inserting the same msg twice should not affect anything
						_, _, _, _, _ = inbox.Insert(msg)
						_, _, _, _, _ = inbox.Insert(msg)

						// It return the same message we inserted
						storedMsg := inbox.QueryByHeightRoundSignatory(msg.Height(), msg.Round(), msg.Signatory())
						Expect(reflect.DeepEqual(msg, storedMsg)).Should(BeTrue())
					}
					return true
				}
				Expect(quick.Check(test, nil)).Should(Succeed())
			})
		})

		Context("when querying by height, round ", func() {
			It("should return correct number of message of that round", func() {
				test := func(height block.Height, round block.Round) bool {
					f := rand.Intn(100) + 1
					messageType := RandomMessageType(false)
					inbox := NewInbox(f, messageType)

					source := map[block.Height]map[block.Round]int{}
					noMessages := rand.Intn(100)
					for i := 0; i < noMessages; i++ {
						msg := RandomSignedMessage(messageType)

						// Inserting the same msg twice should not affect anything
						_, _, _, _, _ = inbox.Insert(msg)
						_, _, _, _, _ = inbox.Insert(msg)

						if _, ok := source[msg.Height()]; !ok {
							source[msg.Height()] = map[block.Round]int{}
						}
						source[msg.Height()][msg.Round()]++
					}
					// Expect the query function gives us the same result as the source.
					for height, roundMap := range source {
						for round, num := range roundMap {
							Expect(inbox.QueryByHeightRound(height, round)).Should(Equal(num))
						}
					}
					return true
				}
				Expect(quick.Check(test, nil)).Should(Succeed())
			})
		})
	})

	Context("when deleting messages from an inbox", func() {
		It("should return correct number of messages", func() {
			test := func() bool {
				f := rand.Intn(100) + 1
				messageType := RandomMessageType(false)
				inbox := NewInbox(f, messageType)
				message := RandomMessage(messageType)

				inbox.Insert(message)
				Expect(inbox.QueryByHeightRound(message.Height(), message.Round())).Should(Equal(1))

				inbox.Insert(RandomMessage(messageType))
				Expect(inbox.QueryByHeightRound(message.Height(), message.Round())).Should(Equal(1))

				inbox.Insert(RandomMessage(messageType))
				Expect(inbox.QueryByHeightRound(message.Height(), message.Round())).Should(Equal(1))

				inbox.Delete(message.Height())
				Expect(inbox.QueryByHeightRound(message.Height(), message.Round())).Should(Equal(0))

				inbox.Delete(message.Height())
				Expect(inbox.QueryByHeightRound(message.Height(), message.Round())).Should(Equal(0))

				return true
			}
			Expect(quick.Check(test, nil)).Should(Succeed())
		})
	})
})
