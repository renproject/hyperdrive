package process_test

import (
	"bytes"
	"crypto/ecdsa"
	cRand "crypto/rand"
	"fmt"
	"math/rand"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/renproject/hyperdrive/block"
	"github.com/renproject/hyperdrive/testutil"
	"github.com/renproject/id"
	"github.com/renproject/surge"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/renproject/hyperdrive/process"
	. "github.com/renproject/hyperdrive/testutil"
)

var _ = Describe("Process", func() {

	newEcdsaKey := func() *ecdsa.PrivateKey {
		privateKey, err := ecdsa.GenerateKey(crypto.S256(), cRand.Reader)
		Expect(err).NotTo(HaveOccurred())
		return privateKey
	}

	Context("when marshaling/unmarshaling process", func() {
		It("should equal itself after binary marshaling and then unmarshaling", func() {
			processOrigin := NewProcessOrigin(100)
			processOrigin.State.CurrentHeight = block.Height(100) // make sure it's not proposing block.
			process := processOrigin.ToProcess()

			data, err := surge.ToBinary(process)
			Expect(err).NotTo(HaveOccurred())
			newProcess := processOrigin.ToProcess()
			Expect(surge.FromBinary(data, newProcess)).Should(Succeed())

			// Since state cannot be accessed from the process. We try to compared the
			// marshalling bytes to check if they we get the same process.
			newData, err := surge.ToBinary(newProcess)
			Expect(err).NotTo(HaveOccurred())
			Expect(bytes.Equal(data, newData)).Should(BeTrue())
		})
	})

	Context("when a new process is initialized", func() {
		Context("when the process is the proposer", func() {
			Context("when the valid block is nil", func() {
				It("should propose a block generated proposer and broadcast it", func() {
					// Init a default process to be modified
					processOrigin := NewProcessOrigin(100)
					process := processOrigin.ToProcess()
					process.Start()

					var message Message
					Eventually(processOrigin.BroadcastMessages).Should(Receive(&message))
					_, ok := message.(*Resync)
					Expect(ok).Should(BeTrue())

					// Expect the proposer broadcast a propose message with height 1 and round 0
					Eventually(processOrigin.BroadcastMessages).Should(Receive(&message))
					proposal, ok := message.(*Propose)
					Expect(ok).Should(BeTrue())
					Expect(proposal.Height()).Should(Equal(block.Height(1)))
					Expect(proposal.Round()).Should(BeZero())
				})
			})

			Context("when the valid block is not nil", func() {
				It("should propose the valid block we have and broadcast it", func() {
					// Init a default process to be modified
					processOrigin := NewProcessOrigin(100)
					block := processOrigin.Proposer.BlockProposal(0, 0)
					processOrigin.State.ValidBlock = block
					process := processOrigin.ToProcess()
					process.StartRound(0)

					// Expect the proposer broadcast a propose message with zero height and round
					var message Message
					Eventually(processOrigin.BroadcastMessages).Should(Receive(&message))
					proposal, ok := message.(*Propose)
					Expect(ok).Should(BeTrue())
					proposal.Block().Equal(block)
				})
			})
		})

		Context("when the process is not proposer", func() {
			Context("when we receive a propose from the proposer before the timeout expires", func() {
				Context("when the block is valid", func() {
					It("should broadcast a prevote to the proposal", func() {
						// Initialise a default process.
						processOrigin := NewProcessOrigin(100)

						// Replace the scheduler and start the process.
						privateKey := newEcdsaKey()
						scheduler := NewMockScheduler(id.NewSignatory(privateKey.PublicKey))
						processOrigin.Scheduler = scheduler
						process := processOrigin.ToProcess()

						// Generate a valid proposal.
						message := NewPropose(1, 0, RandomBlock(block.Standard), block.InvalidRound)
						Expect(Sign(message, *privateKey)).NotTo(HaveOccurred())
						process.HandleMessage(message)

						// Expect the proposer broadcasts a propose message with
						// zero height and round.
						var propose Message
						Eventually(processOrigin.BroadcastMessages).Should(Receive(&propose))
						proposal, ok := propose.(*Prevote)
						Expect(ok).Should(BeTrue())
						Expect(proposal.Height()).Should(Equal(block.Height(1)))
						Expect(proposal.Round()).Should(BeZero())
					})
				})

				Context("when the block is invalid", func() {
					It("should broadcast a nil prevote", func() {
						// Initialise a default process.
						processOrigin := NewProcessOrigin(100)

						// Replace the broadcaster and start the process.
						privateKey := newEcdsaKey()
						scheduler := NewMockScheduler(id.NewSignatory(privateKey.PublicKey))
						processOrigin.Scheduler = scheduler
						processOrigin.Validator = NewMockValidator(fmt.Errorf(""))
						process := processOrigin.ToProcess()

						// Generate an invalid proposal.
						message := NewPropose(1, 0, RandomBlock(block.Standard), block.InvalidRound)
						Expect(Sign(message, *privateKey)).NotTo(HaveOccurred())
						process.HandleMessage(message)

						// Ensure we receive a propose message with the zero
						// height and round.
						var propose Message
						Eventually(processOrigin.BroadcastMessages).Should(Receive(&propose))
						proposal, ok := propose.(*Prevote)
						Expect(ok).Should(BeTrue())
						Expect(proposal.Height()).Should(Equal(block.Height(1)))
						Expect(proposal.Round()).Should(BeZero())
					})
				})

				Context("when the valid block is not nil", func() {
					It("should broadcast our prevote from that round", func() {
						// Initialise a default process.
						processOrigin := NewProcessOrigin(100)

						// Replace the broadcaster.
						privateKey := newEcdsaKey()
						scheduler := NewMockScheduler(id.NewSignatory(privateKey.PublicKey))
						processOrigin.Scheduler = scheduler
						processOrigin.Validator = NewMockValidator(fmt.Errorf(""))

						// Insert a prevote for the valid round as this is the
						// message we will be expected to resend later.
						validRound := RandomRound()
						prevote := NewPrevote(1, validRound, RandomBlock(block.Standard).Hash(), nil)
						Expect(Sign(prevote, *processOrigin.PrivateKey)).ShouldNot(HaveOccurred())
						processOrigin.State.Prevotes.Insert(prevote)

						// Start the process.
						process := processOrigin.ToProcess()

						// Generate a valid proposal with a valid round.
						propose := NewPropose(1, 0, RandomBlock(block.Standard), validRound)
						Expect(Sign(propose, *privateKey)).NotTo(HaveOccurred())
						process.HandleMessage(propose)

						// Ensure we broadcast a prevote message for the valid
						// round.
						Eventually(processOrigin.BroadcastMessages).Should(Receive(&prevote))
						Expect(prevote.Height()).Should(Equal(block.Height(1)))
						Expect(prevote.Round()).Should(Equal(validRound))
					})
				})
			})

			Context("when we do not receive a propose during the timeout", func() {
				It("should broadcast a nil prevote", func() {
					By("before reboot")

					// Initialise a default process.
					processOrigin := NewProcessOrigin(100)

					// Replace the broadcaster and start the process.
					scheduler := NewMockScheduler(RandomSignatory())
					processOrigin.Scheduler = scheduler
					process := processOrigin.ToProcess()
					process.Start()

					// Store state for later use.
					stateBytes, err := surge.ToBinary(process)
					Expect(err).ToNot(HaveOccurred())

					// Expect the validator to broadcast a nil prevote message.
					var message Message
					Eventually(processOrigin.BroadcastMessages).Should(Receive(&message))
					_, ok := message.(*Resync)
					Expect(ok).Should(BeTrue())

					// Expect the proposer broadcast a propose message with zero height and round
					Eventually(processOrigin.BroadcastMessages, 2*time.Second).Should(Receive(&message))
					prevote, ok := message.(*Prevote)
					Expect(ok).Should(BeTrue())
					Expect(prevote.Height()).Should(Equal(block.Height(1)))
					Expect(prevote.Round()).Should(BeZero())
					Expect(prevote.BlockHash().Equal(block.InvalidHash)).Should(BeTrue())

					By("after reboot")

					// Initialise a new process using the stored state and
					// ensure it times out and broadcasts a nil prevote.
					newProcessOrigin := NewProcessOrigin(100)

					state := DefaultState(100)
					err = surge.FromBinary(stateBytes, &state)
					Expect(err).ToNot(HaveOccurred())

					newProcessOrigin.UpdateState(state)

					// Replace the broadcaster and start the new process.
					newScheduler := NewMockScheduler(RandomSignatory())
					newProcessOrigin.Scheduler = newScheduler
					newProcess := newProcessOrigin.ToProcess()
					newProcess.Start()

					Eventually(newProcessOrigin.BroadcastMessages).Should(Receive(&message))
					_, ok = message.(*Resync)
					Expect(ok).Should(BeTrue())

					// Expect the validator to broadcast a nil prevote message.
					Eventually(newProcessOrigin.BroadcastMessages, 2*time.Second).Should(Receive(&message))
					prevote, ok = message.(*Prevote)
					Expect(ok).Should(BeTrue())
					Expect(prevote.Height()).Should(Equal(block.Height(1)))
					Expect(prevote.Round()).Should(BeZero())
					Expect(prevote.BlockHash().Equal(block.InvalidHash)).Should(BeTrue())
				})
			})
		})
	})

	Context("when receive 2f + 1 prevote of a proposal at current height and round for the first time", func() {
		Context("when the process is in prevote", func() {
			It("should lock the proposal and round, and broadcast a precommit for it.", func() {
				f := rand.Intn(100) + 1
				processOrigin := NewProcessOrigin(f)

				height, round := block.Height(rand.Int()), block.Round(rand.Int())
				processOrigin.State.CurrentStep = StepPrevote
				processOrigin.State.CurrentHeight = height
				processOrigin.State.CurrentRound = round

				privateKey := newEcdsaKey()
				scheduler := NewMockScheduler(id.NewSignatory(privateKey.PublicKey))
				processOrigin.Scheduler = scheduler
				process := processOrigin.ToProcess()

				// Handle the proposal
				propose := NewPropose(height, round, RandomBlock(block.Standard), block.Round(rand.Intn(int(round))))
				Expect(Sign(propose, *privateKey)).Should(Succeed())
				process.HandleMessage(propose)

				// Send 2F +1 Prevote for this proposal
				for i := 0; i < 2*f+1; i++ {
					prevote := NewPrevote(height, round, propose.BlockHash(), nil)
					pk := newEcdsaKey()
					Expect(Sign(prevote, *pk)).Should(Succeed())
					process.HandleMessage(prevote)
				}

				// Expect the proposer broadcast a precommit message with
				var message Message
				Eventually(processOrigin.BroadcastMessages, 2*time.Second).Should(Receive(&message))
				precommit, ok := message.(*Precommit)
				Expect(ok).Should(BeTrue())
				Expect(precommit.Height()).Should(Equal(height))
				Expect(precommit.Round()).Should(Equal(round))
				Expect(precommit.BlockHash().Equal(propose.BlockHash())).Should(BeTrue())

				// Expect the block is locked in the state
				state := testutil.GetStateFromProcess(process, f)
				Expect(state.LockedBlock.Equal(propose.Block())).Should(BeTrue())
				Expect(state.LockedRound).Should(Equal(round))
				Expect(state.ValidBlock.Equal(propose.Block())).Should(BeTrue())
				Expect(state.ValidRound).Should(Equal(round))
			})
		})

		Context("when the process is in the precommit step", func() {
			Context("when it receives 2*f+1 precommits for any proposal for the first time", func() {
				It("should move to the next round if no consensus is reached within the timeout", func() {
					By("before reboot")

					// Initialise a new process at the precommit step.
					f := rand.Intn(100) + 1
					height, round := block.Height(rand.Int()), block.Round(rand.Int())
					processOrigin := NewProcessOrigin(f)
					processOrigin.State.CurrentStep = StepPrecommit
					processOrigin.State.CurrentHeight = height
					processOrigin.State.CurrentRound = round
					processOrigin.Blockchain.InsertBlockAtHeight(height-1, RandomBlock(block.Standard))

					process := processOrigin.ToProcess()
					process.Start()

					// Handle random precommits.
					for i := 0; i < 2*f+1; i++ {
						precommit := NewPrecommit(height, round, RandomBlock(RandomBlockKind()).Hash())
						privateKey := newEcdsaKey()
						Expect(Sign(precommit, *privateKey)).NotTo(HaveOccurred())
						process.HandleMessage(precommit)
					}

					// Store state for later use.
					stateBytes, err := surge.ToBinary(process)
					Expect(err).ToNot(HaveOccurred())

					var message Message
					Eventually(processOrigin.BroadcastMessages).Should(Receive(&message))
					_, ok := message.(*Resync)
					Expect(ok).Should(BeTrue())

					// Expect the validator to broadcast a propose and move to
					// the next round.
					Eventually(processOrigin.BroadcastMessages, 2*time.Second).Should(Receive(&message))
					propose, ok := message.(*Propose)
					Expect(ok).Should(BeTrue())
					Expect(propose.Height()).Should(Equal(height))
					Expect(propose.Round()).Should(Equal(round + 1))

					By("after reboot")

					// Initialise a new process using the stored state and
					// ensure it times out and broadcasts a nil proposal.
					newProcessOrigin := NewProcessOrigin(f)
					newProcessOrigin.State.CurrentStep = StepPrecommit
					newProcessOrigin.State.CurrentHeight = height
					newProcessOrigin.State.CurrentRound = round
					newProcessOrigin.Blockchain.InsertBlockAtHeight(height-1, RandomBlock(block.Standard))

					state := DefaultState(f)
					err = surge.FromBinary(stateBytes, &state)
					Expect(err).ToNot(HaveOccurred())
					newProcessOrigin.UpdateState(state)

					newProcess := newProcessOrigin.ToProcess()
					newProcess.Start()

					Eventually(newProcessOrigin.BroadcastMessages).Should(Receive(&message))
					_, ok = message.(*Resync)
					Expect(ok).Should(BeTrue())

					// Expect the validator to broadcast a propose and move to
					// the next round.
					Eventually(newProcessOrigin.BroadcastMessages, 2*time.Second).Should(Receive(&message))
					propose, ok = message.(*Propose)
					Expect(ok).Should(BeTrue())
					Expect(propose.Height()).Should(Equal(height))
					Expect(propose.Round()).Should(Equal(round + 1))
				})
			})

			It("should put the proposal in the validBlock", func() {
				f := rand.Intn(100) + 1
				processOrigin := NewProcessOrigin(f)

				height, round := block.Height(rand.Int()), block.Round(rand.Int())
				processOrigin.State.CurrentStep = StepPrecommit
				processOrigin.State.CurrentHeight = height
				processOrigin.State.CurrentRound = round

				privateKey := newEcdsaKey()
				scheduler := NewMockScheduler(id.NewSignatory(privateKey.PublicKey))
				processOrigin.Scheduler = scheduler
				process := processOrigin.ToProcess()

				// Handle the proposal
				propose := NewPropose(height, round, RandomBlock(block.Standard), block.Round(rand.Intn(int(round))))
				Expect(Sign(propose, *privateKey)).Should(Succeed())
				process.HandleMessage(propose)

				// Send 2F +1 Prevote for this proposal
				for i := 0; i < 2*f+1; i++ {
					prevote := NewPrevote(height, round, propose.BlockHash(), nil)
					pk := newEcdsaKey()
					Expect(Sign(prevote, *pk)).Should(Succeed())
					process.HandleMessage(prevote)
				}

				// Expect the block is locked in the state
				state := testutil.GetStateFromProcess(process, f)
				Expect(state.LockedBlock.Equal(processOrigin.State.LockedBlock)).Should(BeTrue())
				Expect(state.LockedRound).Should(Equal(processOrigin.State.LockedRound))
				Expect(state.ValidBlock.Equal(propose.Block())).Should(BeTrue())
				Expect(state.ValidRound).Should(Equal(round))
			})
		})
	})

	Context("when the process is in the prevote step", func() {
		Context("when it receives 2*f+1 prevotes for any proposal for the first time", func() {
			It("should send a nil precommit if no consensus is reached within the timeout", func() {
				By("before reboot")

				// Initialise a new process at the prevote step.
				f := rand.Intn(100) + 1
				height, round := block.Height(rand.Int()), block.Round(rand.Int())
				processOrigin := NewProcessOrigin(f)
				processOrigin.State.CurrentStep = StepPrevote
				processOrigin.State.CurrentHeight = height
				processOrigin.State.CurrentRound = round
				process := processOrigin.ToProcess()
				process.Start()

				// Handle random prevotes.
				for i := 0; i < 2*f+1; i++ {
					prevote := NewPrevote(height, round, RandomBlock(RandomBlockKind()).Hash(), nil)
					privateKey := newEcdsaKey()
					Expect(Sign(prevote, *privateKey)).NotTo(HaveOccurred())
					process.HandleMessage(prevote)
				}

				// Store state for later use.
				stateBytes, err := surge.ToBinary(process)
				Expect(err).ToNot(HaveOccurred())

				var message Message
				Eventually(processOrigin.BroadcastMessages).Should(Receive(&message))
				_, ok := message.(*Resync)
				Expect(ok).Should(BeTrue())

				// Expect the validator to broadcast a nil precommit.
				Eventually(processOrigin.BroadcastMessages, 2*time.Second).Should(Receive(&message))
				precommit, ok := message.(*Precommit)
				Expect(ok).Should(BeTrue())
				Expect(precommit.Height()).Should(Equal(height))
				Expect(precommit.Round()).Should(Equal(round))
				Expect(precommit.BlockHash().Equal(block.InvalidHash)).Should(BeTrue())

				By("after reboot")

				// Initialise a new process using the stored state and
				// ensure it times out and broadcasts a nil precommit.
				newProcessOrigin := NewProcessOrigin(100)
				newProcessOrigin.State.CurrentStep = StepPrevote
				newProcessOrigin.State.CurrentHeight = height
				newProcessOrigin.State.CurrentRound = round

				state := DefaultState(100)
				err = surge.FromBinary(stateBytes, &state)
				Expect(err).ToNot(HaveOccurred())
				newProcessOrigin.UpdateState(state)

				newProcess := newProcessOrigin.ToProcess()
				newProcess.Start()

				Eventually(newProcessOrigin.BroadcastMessages).Should(Receive(&message))
				_, ok = message.(*Resync)
				Expect(ok).Should(BeTrue())

				// Expect the validator to broadcast a nil precommit message.
				Eventually(newProcessOrigin.BroadcastMessages, 2*time.Second).Should(Receive(&message))
				precommit, ok = message.(*Precommit)
				Expect(ok).Should(BeTrue())
				Expect(precommit.Height()).Should(Equal(height))
				Expect(precommit.Round()).Should(Equal(round))
				Expect(precommit.BlockHash().Equal(block.InvalidHash)).Should(BeTrue())
			})
		})

		Context("when it receives 2*f+1 nil prevotes for the current height and round", func() {
			It("should broadcast a nil precommit and move to the precommit step", func() {
				f := rand.Intn(100) + 1
				height, round := block.Height(rand.Int()), block.Round(rand.Int())
				processOrigin := NewProcessOrigin(f)
				processOrigin.State.CurrentStep = StepPrevote
				processOrigin.State.CurrentHeight = height
				processOrigin.State.CurrentRound = round
				process := processOrigin.ToProcess()

				for i := 0; i < 2*f+1; i++ {
					prevote := NewPrevote(height, round, block.InvalidHash, nil)
					privateKey := newEcdsaKey()
					Expect(Sign(prevote, *privateKey)).NotTo(HaveOccurred())
					process.HandleMessage(prevote)
				}

				// Expect the proposer broadcast a precommit message with
				var message Message
				Eventually(processOrigin.BroadcastMessages, 2*time.Second).Should(Receive(&message))
				precommit, ok := message.(*Precommit)
				Expect(ok).Should(BeTrue())
				Expect(precommit.Height()).Should(Equal(height))
				Expect(precommit.Round()).Should(Equal(round))
				Expect(precommit.BlockHash().Equal(block.InvalidHash)).Should(BeTrue())
			})
		})
	})

	Context("when the process receives at least 2*f+1 precommits", func() {
		Context("when starting a timer before executing the OnTimeoutPrecommit function", func() {
			It("should start a round when nothing changes after the timeout", func() {
				for _, step := range []Step{StepPropose, StepPrevote, StepPrecommit} {
					f := rand.Intn(100) + 1
					height, round := block.Height(rand.Int()), block.Round(rand.Int())
					processOrigin := NewProcessOrigin(f)
					processOrigin.State.CurrentStep = step
					processOrigin.State.CurrentHeight = height
					processOrigin.State.CurrentRound = round
					processOrigin.Blockchain.InsertBlockAtHeight(height-1, RandomBlock(block.Standard))
					process := processOrigin.ToProcess()

					for i := 0; i < 2*f+1; i++ {
						precommit := NewPrecommit(height, round, RandomBlock(RandomBlockKind()).Hash())
						privateKey := newEcdsaKey()
						Expect(Sign(precommit, *privateKey)).NotTo(HaveOccurred())
						process.HandleMessage(precommit)
					}

					// Expect the proposer broadcast a propose message with zero height and round
					var message Message
					Eventually(processOrigin.BroadcastMessages, 2*time.Second).Should(Receive(&message))
					proposal, ok := message.(*Propose)
					Expect(ok).Should(BeTrue())
					Expect(proposal.Height()).Should(Equal(height))
					Expect(proposal.Round()).Should(Equal(round + 1))

					state := testutil.GetStateFromProcess(process, f)
					Expect(state.CurrentRound).Should(Equal(round + 1))
					Expect(state.CurrentStep).Should(Equal(StepPropose))
				}
			})
		})
	})

	Context("when receiving f+1 of any message whose round is higher", func() {
		It("should start that round", func() {
			for _, t := range []MessageType{
				// NOTE: You should only ever receive 1 Propose for a height
				// and round, so this test is not meaningful for Propose
				// messages.
				//
				//	ProposeMessageType,
				//
				PrevoteMessageType,
				PrecommitMessageType,
			} {
				messageType := t
				// Init a default process to be modified
				f := rand.Intn(100) + 1
				height, round := RandomHeight(), RandomRound()
				processOrigin := NewProcessOrigin(f)
				processOrigin.State.CurrentHeight = height
				processOrigin.State.CurrentRound = round

				// Replace the broadcaster and start the process
				scheduler := NewMockScheduler(RandomSignatory())
				processOrigin.Scheduler = scheduler
				process := processOrigin.ToProcess()

				// Send f + 1 message with higher round to the process
				newRound := block.Round(rand.Intn(10)+1) + round
				for i := 0; i < f+1; i++ {
					message := RandomMessageWithHeightAndRound(height, newRound, messageType)
					privateKey := newEcdsaKey()
					Expect(Sign(message, *privateKey)).Should(Succeed())
					process.HandleMessage(message)
				}

				// Expect the proposer broadcast a propose message with zero height and round
				var message Message
				Eventually(processOrigin.BroadcastMessages, 2*time.Second).Should(Receive(&message))
				prevote, ok := message.(*Prevote)
				Expect(ok).Should(BeTrue())
				Expect(prevote.Height()).Should(Equal(height))
				Expect(prevote.Round()).Should(Equal(newRound))
				Expect(prevote.BlockHash().Equal(block.InvalidHash)).Should(BeTrue())
			}
		})
	})

	Context("when process in propose state", func() {
		Context("when receive a proposal with a non-zero valid round and the valid round is less than current round", func() {
			Context("when receive at least 2f+1 prevote of the proposal.", func() {
				Context("when the proposal is valid ", func() {
					Context("when lockedRound is less than or equal to the valid round", func() {
						It("should broadcast a prevote to the proposal", func() {
							// Init a default process to be modified
							f := rand.Intn(100) + 1
							height, round := block.Height(rand.Int()), block.Round(rand.Int()+1) // Round needs to be great than 0
							validRound := block.Round(rand.Intn(int(round)))

							processOrigin := NewProcessOrigin(f)
							processOrigin.State.CurrentHeight = height
							processOrigin.State.CurrentRound = round
							processOrigin.State.CurrentStep = StepPropose
							processOrigin.State.LockedRound = block.Round(rand.Intn(int(validRound + 1)))
							process := processOrigin.ToProcess()

							// Send the proposal
							propose := NewPropose(height, round, RandomBlock(RandomBlockKind()), validRound)
							Expect(Sign(propose, *processOrigin.PrivateKey)).Should(Succeed())
							process.HandleMessage(propose)

							// Send 2f + 1 prevotes
							for i := 0; i < 2*f+1; i++ {
								prevote := NewPrevote(height, validRound, propose.BlockHash(), nil)
								privateKey := newEcdsaKey()
								Expect(Sign(prevote, *privateKey)).Should(Succeed())
								process.HandleMessage(prevote)
							}

							// Expect the process broadcast a nil prevote
							var message Message
							Eventually(processOrigin.BroadcastMessages, 2*time.Second).Should(Receive(&message))
							prevote, ok := message.(*Prevote)
							Expect(ok).Should(BeTrue())
							Expect(prevote.Height()).Should(Equal(height))
							Expect(prevote.Round()).Should(Equal(round))
							Expect(prevote.BlockHash().Equal(propose.BlockHash())).Should(BeTrue())

							// Step should be moved to prevote
							state := testutil.GetStateFromProcess(process, f)
							Expect(state.CurrentStep).Should(Equal(StepPrevote))
						})
					})

					Context("when the proposed block is same as the locked block", func() {
						It("should broadcast a prevote to the proposal", func() {
							// Init a default process to be modified
							f := rand.Intn(100) + 1
							height, round := block.Height(rand.Int()), block.Round(rand.Int()+1) // Round needs to be great than 0
							validRound := block.Round(rand.Intn(int(round)))

							processOrigin := NewProcessOrigin(f)
							processOrigin.State.CurrentHeight = height
							processOrigin.State.CurrentRound = round
							processOrigin.State.CurrentStep = StepPropose
							processOrigin.State.LockedRound = validRound + 1 // make sure lockedRound is greater than the valid round
							block := RandomBlock(RandomBlockKind())
							propose := NewPropose(height, round, block, validRound)
							Expect(Sign(propose, *processOrigin.PrivateKey)).Should(Succeed())
							processOrigin.State.LockedBlock = block

							// Send the proposal
							process := processOrigin.ToProcess()
							process.HandleMessage(propose)

							// Send 2f + 1 prevotes
							for i := 0; i < 2*f+1; i++ {
								prevote := NewPrevote(height, validRound, propose.BlockHash(), nil)
								privateKey := newEcdsaKey()
								Expect(Sign(prevote, *privateKey)).Should(Succeed())
								process.HandleMessage(prevote)
							}

							// Expect the process broadcast a nil prevote
							var message Message
							Eventually(processOrigin.BroadcastMessages, 2*time.Second).Should(Receive(&message))
							prevote, ok := message.(*Prevote)
							Expect(ok).Should(BeTrue())
							Expect(prevote.Height()).Should(Equal(height))
							Expect(prevote.Round()).Should(Equal(round))
							Expect(prevote.BlockHash().Equal(propose.BlockHash())).Should(BeTrue())

							// Step should be moved to prevote
							state := testutil.GetStateFromProcess(process, f)
							Expect(state.CurrentStep).Should(Equal(StepPrevote))
						})
					})
				})

				Context("when the proposal is invalid", func() {
					It("should broadcast a nil prevote", func() {
						// Init a default process to be modified
						f := rand.Intn(100) + 1
						height, round := block.Height(rand.Int()), block.Round(rand.Int()+1) // Round needs to be great than 0
						processOrigin := NewProcessOrigin(f)
						processOrigin.State.CurrentHeight = height
						processOrigin.State.CurrentRound = round
						processOrigin.State.CurrentStep = StepPropose
						processOrigin.Validator = NewMockValidator(fmt.Errorf(""))
						process := processOrigin.ToProcess()

						// Send the proposal
						propose := NewPropose(height, round, block.InvalidBlock, block.InvalidRound)
						Expect(Sign(propose, *processOrigin.PrivateKey)).Should(Succeed())
						process.HandleMessage(propose)

						// Expect the process broadcast a nil prevote
						var message Message
						Eventually(processOrigin.BroadcastMessages, 2*time.Second).Should(Receive(&message))
						prevote, ok := message.(*Prevote)
						Expect(ok).Should(BeTrue())
						Expect(prevote.Height()).Should(Equal(height))
						Expect(prevote.Round()).Should(Equal(round))
						Expect(prevote.BlockHash().Equal(block.InvalidHash)).Should(BeTrue())

						// Step should be moved to prevote
						state := testutil.GetStateFromProcess(process, f)
						Expect(state.CurrentStep).Should(Equal(StepPrevote))
					})
				})
			})
		})
	})

	Context("when current block does not exist in the blockchain", func() {
		Context("when receive 2f + 1 precommit of a proposal,", func() {
			It("should finalize the block in blockchain, reset the state, and start from round 0 in height +1 ", func() {
				for _, step := range []Step{StepPropose, StepPrevote, StepPrecommit} {
					// Init a default process to be modified
					f := rand.Intn(100) + 1
					height, round := block.Height(rand.Int()), block.Round(rand.Int()+1) // Round needs to be great than 0
					validRound := block.Round(rand.Intn(int(round + 1)))

					processOrigin := NewProcessOrigin(f)
					processOrigin.State.CurrentHeight = height
					processOrigin.State.CurrentRound = round
					processOrigin.State.CurrentStep = step // step should not matter in this case.
					processOrigin.State.LockedRound = block.Round(rand.Intn(int(validRound + 1)))
					process := processOrigin.ToProcess()

					// Send the proposal
					proposeRound := block.Round(rand.Intn(int(round + 1)))                                  // if proposeRound > currentRound, it will start(proposeRound)
					propose := NewPropose(height, proposeRound, RandomBlock(RandomBlockKind()), validRound) // round and valid round should not matter in this case
					Expect(Sign(propose, *processOrigin.PrivateKey)).Should(Succeed())
					process.HandleMessage(propose)

					// Send 2f + 1 prevotes
					for i := 0; i < 2*f+1; i++ {
						precommit := NewPrecommit(height, proposeRound, propose.BlockHash())
						privateKey := newEcdsaKey()
						Expect(Sign(precommit, *privateKey)).Should(Succeed())
						process.HandleMessage(precommit)
					}

					// Expect process start a new round and start proposing
					var message Message
					Eventually(processOrigin.BroadcastMessages).Should(Receive(&message))

					proposal, ok := message.(*Propose)
					Expect(ok).Should(BeTrue())
					Expect(proposal.Height()).Should(Equal(height + 1))
					Expect(proposal.Round()).Should(BeZero())

					// The proposal should be finalized in the blockchain storage.
					Expect(processOrigin.Blockchain.BlockExistsAtHeight(height)).Should(BeTrue())

					// Step should be reset and new height and 0 round
					state := testutil.GetStateFromProcess(process, f)
					Expect(state.CurrentHeight).Should(Equal(height + 1))
					Expect(state.CurrentRound).Should(BeZero())
					Expect(state.CurrentStep).Should(Equal(StepPropose))
					Expect(state.LockedBlock).Should(Equal(block.InvalidBlock))
					Expect(state.LockedRound).Should(Equal(block.InvalidRound))
					Expect(state.ValidBlock).Should(Equal(block.InvalidBlock))
					Expect(state.ValidRound).Should(Equal(block.InvalidRound))
				}
			})
		})
	})

	Context("when starting the process", func() {
		It("should send a resync message", func() {
			processOrigin := NewProcessOrigin(100)
			process := processOrigin.ToProcess()
			process.Start()

			// Expect the process to broadcast a resync message.
			var message Message
			Eventually(processOrigin.BroadcastMessages, 2*time.Second).Should(Receive(&message))
			_, ok := message.(*Resync)
			Expect(ok).Should(BeTrue())
		})

		Context("when the process has messages from a previous height", func() {
			It("should resend the most recent proposal, prevote, and precommit", func() {
				processOrigin := NewProcessOrigin(100)

				propose := RandomPropose()
				Expect(Sign(propose, *processOrigin.PrivateKey)).ToNot(HaveOccurred())
				prevote := NewPrevote(propose.Height(), propose.Round(), propose.BlockHash(), nil)
				Expect(Sign(prevote, *processOrigin.PrivateKey)).ToNot(HaveOccurred())
				precommit := NewPrecommit(propose.Height(), propose.Round(), propose.BlockHash())
				Expect(Sign(precommit, *processOrigin.PrivateKey)).ToNot(HaveOccurred())

				processOrigin.Blockchain.InsertBlockAtHeight(propose.Height(), propose.Block())
				processOrigin.State.CurrentHeight = propose.Height() + 1
				processOrigin.State.CurrentRound = 0
				processOrigin.State.Proposals.Insert(propose)
				processOrigin.State.Prevotes.Insert(prevote)
				processOrigin.State.Precommits.Insert(precommit)
				process := processOrigin.ToProcess()

				done := make(chan struct{})
				resentProposal := false
				resentPrevote := false
				resentPrecommit := false
				go func() {
					defer close(done)
					for m := range processOrigin.BroadcastMessages {
						switch m.(type) {
						case *Propose:
							Expect(m).To(Equal(propose))
							resentProposal = true
						case *Prevote:
							Expect(m).To(Equal(prevote))
							resentPrevote = true
						case *Precommit:
							Expect(m).To(Equal(precommit))
							resentPrecommit = true
						}
						if resentProposal && resentPrevote && resentPrecommit {
							return
						}
					}
				}()

				go process.Start()
				<-done
			})
		})

		Context("when the process has messages from a current height", func() {
			It("should resend the most recent proposal, prevote, and precommit", func() {
				processOrigin := NewProcessOrigin(100)

				propose := RandomPropose()
				Expect(Sign(propose, *processOrigin.PrivateKey)).ToNot(HaveOccurred())
				prevote := NewPrevote(propose.Height(), propose.Round(), propose.BlockHash(), nil)
				Expect(Sign(prevote, *processOrigin.PrivateKey)).ToNot(HaveOccurred())
				precommit := NewPrecommit(propose.Height(), propose.Round(), propose.BlockHash())
				Expect(Sign(precommit, *processOrigin.PrivateKey)).ToNot(HaveOccurred())

				processOrigin.Blockchain.InsertBlockAtHeight(propose.Height()-1, RandomBlock(block.Standard))
				processOrigin.Blockchain.InsertBlockAtHeight(propose.Height(), propose.Block())
				processOrigin.State.CurrentHeight = propose.Height()
				processOrigin.State.CurrentRound = propose.Round() + 1
				processOrigin.State.Proposals.Insert(propose)
				processOrigin.State.Prevotes.Insert(prevote)
				processOrigin.State.Precommits.Insert(precommit)
				process := processOrigin.ToProcess()

				done := make(chan struct{})
				resentProposal := false
				resentPrevote := false
				resentPrecommit := false
				go func() {
					defer close(done)
					for m := range processOrigin.BroadcastMessages {
						switch m.(type) {
						case *Propose:
							Expect(m).To(Equal(propose))
							resentProposal = true
						case *Prevote:
							Expect(m).To(Equal(prevote))
							resentPrevote = true
						case *Precommit:
							Expect(m).To(Equal(precommit))
							resentPrecommit = true
						}
						if resentProposal && resentPrevote && resentPrecommit {
							return
						}
					}
				}()

				go process.Start()
				<-done
			})
		})
	})

	Context("when the process receives a resync message", func() {
		It("should broadcast latest messages to the sender", func() {
			// Initialise a default process.
			processOrigin := NewProcessOrigin(100)

			// Replace the scheduler.
			privateKey := newEcdsaKey()
			scheduler := NewMockScheduler(id.NewSignatory(privateKey.PublicKey))
			processOrigin.Scheduler = scheduler

			// Insert random messages.
			propose := RandomPropose()
			Expect(Sign(propose, *processOrigin.PrivateKey)).ToNot(HaveOccurred())
			prevote := NewPrevote(propose.Height(), propose.Round(), propose.BlockHash(), nil)
			Expect(Sign(prevote, *processOrigin.PrivateKey)).ToNot(HaveOccurred())
			precommit := NewPrecommit(propose.Height(), propose.Round(), propose.BlockHash())
			Expect(Sign(precommit, *processOrigin.PrivateKey)).ToNot(HaveOccurred())

			processOrigin.Blockchain.InsertBlockAtHeight(propose.Height(), propose.Block())
			processOrigin.State.CurrentHeight = propose.Height() + 1
			processOrigin.State.CurrentRound = 0
			processOrigin.State.Proposals.Insert(propose)
			processOrigin.State.Prevotes.Insert(prevote)
			processOrigin.State.Precommits.Insert(precommit)

			// Start the process.
			process := processOrigin.ToProcess()
			process.Start()

			// Handle a resync message.
			message := NewResync(0, 0)
			Expect(Sign(message, *privateKey)).NotTo(HaveOccurred())
			process.HandleMessage(message)

			// Ensure the process broadcasts latest messages.
			done := make(chan struct{})
			resentProposal := false
			resentPrevote := false
			resentPrecommit := false
			go func() {
				defer close(done)
				for m := range processOrigin.BroadcastMessages {
					switch m.(type) {
					case *Propose:
						Expect(m).To(Equal(propose))
						resentProposal = true
					case *Prevote:
						Expect(m).To(Equal(prevote))
						resentPrevote = true
					case *Precommit:
						Expect(m).To(Equal(precommit))
						resentPrecommit = true
					}
					if resentProposal && resentPrevote && resentPrecommit {
						return
					}
				}
			}()

			go process.Start()
			<-done
		})
	})
})
