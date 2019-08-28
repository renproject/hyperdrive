package process_test

import (
	"bytes"
	"crypto/ecdsa"
	cRand "crypto/rand"
	"encoding/json"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/renproject/hyperdrive/process"
	. "github.com/renproject/hyperdrive/testutil"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/renproject/hyperdrive/block"
	"github.com/renproject/hyperdrive/id"
)

var _ = Describe("Process", func() {

	Context("when marshaling/unmarshaling process", func() {
		It("should equal itself after marshaling and then unmarshaling", func() {
			processOrigin := NewProcessOrigin(100)
			processOrigin.State.CurrentHeight = block.Height(100) // make sure it's not proposing block.
			process := processOrigin.ToProcess()

			data, err := json.Marshal(process)
			Expect(err).NotTo(HaveOccurred())
			var newProcess Process
			Expect(json.Unmarshal(data, &newProcess)).Should(Succeed())

			// Since state cannot be accessed from the process. We try to compared the
			// marshalling bytes to check if they we get the same process.
			newData, err := json.Marshal(newProcess)
			Expect(err).NotTo(HaveOccurred())
			Expect(bytes.Equal(data, newData)).Should(BeTrue())
		})
	})

	Context("when the process is honest", func() {
		Context("when the process is at propose step", func() {
			Context("when the process is the proposer", func() {
				It("should generate a valid block and broadcast to everyone.", func() {
					// Init a default process to be modified
					processOrigin := NewProcessOrigin(100)
					_ = processOrigin.ToProcess()

					// Expect the proposer broadcast a propose message with zero height and round
					var message Message
					Eventually(processOrigin.BroadcastMessages).Should(Receive(&message))
					proposal, ok := message.(*Propose)
					Expect(ok).Should(BeTrue())
					Expect(proposal.Height()).Should(BeZero())
					Expect(proposal.Round()).Should(BeZero())
				})
			})

			Context("when the process is not proposer", func() {
				Context("when receive a propose from the proposer before the timeout expire", func() {
					Context("when the block is valid", func() {
						It("should broadcast a prevote to the proposal", func() {
							// Init a default process to be modified
							processOrigin := NewProcessOrigin(100)

							// Replace the broadcaster and start the process
							privateKey, err := ecdsa.GenerateKey(crypto.S256(), cRand.Reader)
							Expect(err).NotTo(HaveOccurred())
							scheduler := NewMockScheduler(id.NewSignatory(privateKey.PublicKey))
							processOrigin.Scheduler = scheduler
							process := processOrigin.ToProcess()

							// Generate a valid proposal
							message := NewPropose(0, 0, RandomBlock(block.Standard), block.InvalidRound)
							Expect(Sign(message, *privateKey)).NotTo(HaveOccurred())
							process.HandleMessage(message)

							// Expect the proposer broadcast a propose message with zero height and round
							var propose Message
							Eventually(processOrigin.BroadcastMessages).Should(Receive(&propose))
							proposal, ok := propose.(*Prevote)
							Expect(ok).Should(BeTrue())
							Expect(proposal.Height()).Should(BeZero())
							Expect(proposal.Round()).Should(BeZero())
						})
					})

					Context("when the block is invalid", func() {
						It("should broadcast a nil prevote", func() {
							// Init a default process to be modified
							processOrigin := NewProcessOrigin(100)

							// Replace the broadcaster and start the process
							privateKey, err := ecdsa.GenerateKey(crypto.S256(), cRand.Reader)
							Expect(err).NotTo(HaveOccurred())
							scheduler := NewMockScheduler(id.NewSignatory(privateKey.PublicKey))
							processOrigin.Scheduler = scheduler
							processOrigin.Validator = NewMockValidator(false)
							process := processOrigin.ToProcess()

							// Generate a invalid proposal
							message := NewPropose(0, 0, RandomBlock(block.Standard), block.InvalidRound)
							Expect(Sign(message, *privateKey)).NotTo(HaveOccurred())
							process.HandleMessage(message)

							// Expect the proposer broadcast a propose message with zero height and round
							var propose Message
							Eventually(processOrigin.BroadcastMessages).Should(Receive(&propose))
							proposal, ok := propose.(*Prevote)
							Expect(ok).Should(BeTrue())
							Expect(proposal.Height()).Should(BeZero())
							Expect(proposal.Round()).Should(BeZero())
						})
					})
				})

				Context("when not receive anything during the timeout", func() {
					It("should broadcast a nil prevote", func() {
						// Init a default process to be modified
						processOrigin := NewProcessOrigin(100)

						// Replace the broadcaster and start the process
						scheduler := NewMockScheduler(RandomSignatory())
						processOrigin.Scheduler = scheduler
						_ = processOrigin.ToProcess()

						// Expect the proposer broadcast a propose message with zero height and round
						var message Message
						Eventually(processOrigin.BroadcastMessages, 2*time.Second).Should(Receive(&message))
						prevote, ok := message.(*Prevote)
						Expect(ok).Should(BeTrue())
						Expect(prevote.Height()).Should(BeZero())
						Expect(prevote.Round()).Should(BeZero())
						Expect(prevote.BlockHash().Equal(block.InvalidHash)).Should(BeTrue())
					})
				})
			})
		})
	})

	Context("when the proposer fail to propose a block", func() {
		It("should time out and vote a precommit for a nil block if the process is not the proposer", func() {

		})

		It("should  if the process is the proposer", func() {

		})
	})
})
