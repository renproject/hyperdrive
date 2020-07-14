package process_test

import (
	"bytes"
	"math/rand"
	"testing/quick"
	"time"

	"github.com/renproject/hyperdrive/process"
	"github.com/renproject/hyperdrive/process/processutil"
	"github.com/renproject/hyperdrive/scheduler"
	"github.com/renproject/hyperdrive/timer"
	"github.com/renproject/id"
	"github.com/renproject/surge"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Process", func() {

	Context("when unmarshaling fuzz", func() {
		It("should not panic", func() {
			f := func(fuzz []byte) bool {
				msg := process.Process{}
				Expect(surge.FromBinary(&msg, fuzz)).ToNot(Succeed())
				return true
			}
			Expect(quick.Check(f, nil)).To(Succeed())
		})
	})

	Context("when marshaling and then unmarshaling", func() {
		r := rand.New(rand.NewSource(time.Now().UnixNano()))

		It("should equal itself", func() {
			f := func(proposeLogs map[process.Round]process.Propose, prevoteLogs map[process.Round]map[id.Signatory]process.Prevote, precommitLogs map[process.Round]map[id.Signatory]process.Precommit, onceFlags map[process.Round]process.OnceFlag) bool {
				expected := process.Process{
					State: processutil.RandomState(r),
				}
				expected.ProposeLogs = proposeLogs
				expected.PrevoteLogs = prevoteLogs
				expected.PrecommitLogs = precommitLogs
				expected.OnceFlags = onceFlags
				data, err := surge.ToBinary(expected)
				Expect(err).ToNot(HaveOccurred())
				got := process.Process{}
				err = surge.FromBinary(&got, data)
				Expect(err).ToNot(HaveOccurred())
				Expect(got.State.Equal(&expected.State)).To(BeTrue())
				return true
			}
			Expect(quick.Check(f, nil)).To(Succeed())
		})

		Context("when not enough byte size available", func() {
			It("should return an error while marshaling", func() {
				f := func(proposeLogs map[process.Round]process.Propose, prevoteLogs map[process.Round]map[id.Signatory]process.Prevote, precommitLogs map[process.Round]map[id.Signatory]process.Precommit, onceFlags map[process.Round]process.OnceFlag) bool {
					expected := process.Process{
						State: processutil.RandomState(r),
					}
					expected.ProposeLogs = proposeLogs
					expected.PrevoteLogs = prevoteLogs
					expected.PrecommitLogs = precommitLogs
					expected.OnceFlags = onceFlags

					sizeAvailable := rand.Intn(expected.SizeHint())

					buf := bytes.NewBuffer([]byte{})
					_, _, err := expected.Marshal(buf.Bytes(), sizeAvailable)
					Expect(err).To(HaveOccurred())

					return true
				}
				Expect(quick.Check(f, nil)).To(Succeed())
			})
		})
	})

	// L11:
	//	Function StartRound(round)
	//		currentRound ← round
	//		currentStep ← propose
	//		if proposer(currentHeight, currentRound) = p then
	//			if validValue != nil then
	//				proposal ← validValue
	//			else
	//				proposal ← getValue()
	//			broadcast〈PROPOSAL, currentHeight, currentRound, proposal, validRound〉
	//		else
	//			schedule OnTimeoutPropose(currentHeight, currentRound) to be executed after timeoutPropose(currentRound)
	Context("when starting a round", func() {
		r := rand.New(rand.NewSource(time.Now().UnixNano()))

		It("should start the zeroeth round on start", func() {
			f := func() bool {
				p := process.New(id.NewPrivKey().Signatory(), 33, nil, nil, nil, nil, nil, nil, nil)
				p.Start()
				Expect(p.CurrentRound).To(Equal(process.Round(0)))
				Expect(p.CurrentHeight).To(Equal(process.Height(1)))
				Expect(p.CurrentStep).To(Equal(process.Proposing))
				return true
			}
			Expect(quick.Check(f, nil)).To(Succeed())
		})

		It("should set the current round to that round and set the current step to proposing", func() {
			f := func() bool {
				round := processutil.RandomRound(r)
				p := process.New(id.NewPrivKey().Signatory(), 33, nil, nil, nil, nil, nil, nil, nil)
				p.StartRound(round)
				Expect(p.CurrentRound).To(Equal(round))
				Expect(p.CurrentStep).To(Equal(process.Proposing))
				return true
			}
			Expect(quick.Check(f, nil)).To(Succeed())
		})

		Context("when we are the proposer", func() {
			Context("when our valid value is non-nil", func() {
				It("should propose the valid value", func() {
					f := func() bool {
						round := processutil.RandomRound(r)
						for round == process.InvalidRound {
							round = processutil.RandomRound(r)
						}
						whoami := id.NewPrivKey().Signatory()
						scheduler := scheduler.NewRoundRobin([]id.Signatory{whoami})
						value := processutil.RandomValue(r)
						broadcaster := processutil.BroadcasterCallbacks{
							BroadcastProposeCallback: func(proposal process.Propose) {
								Expect(proposal.From.Equal(&whoami)).To(BeTrue())
								Expect(proposal.Value).To(Equal(value))
							},
						}
						p := process.New(whoami, 33, nil, scheduler, nil, nil, broadcaster, nil, nil)
						p.State.ValidValue = value
						p.StartRound(round)
						return true
					}
					Expect(quick.Check(f, nil)).To(Succeed())
				})
			})

			Context("when our valid value is nil", func() {
				It("should propose a new value", func() {
					f := func() bool {
						round := processutil.RandomRound(r)
						for round == process.InvalidRound {
							round = processutil.RandomRound(r)
						}
						whoami := id.NewPrivKey().Signatory()
						scheduler := scheduler.NewRoundRobin([]id.Signatory{whoami})
						value := processutil.RandomValue(r)
						proposer := processutil.MockProposer{MockValue: func() process.Value { return value }}
						broadcaster := processutil.BroadcasterCallbacks{
							BroadcastProposeCallback: func(proposal process.Propose) {
								Expect(proposal.From.Equal(&whoami)).To(BeTrue())
								Expect(proposal.Value).To(Equal(value))
							},
						}
						p := process.New(whoami, 33, nil, scheduler, proposer, nil, broadcaster, nil, nil)
						p.StartRound(round)
						return true
					}
					Expect(quick.Check(f, nil)).To(Succeed())
				})
			})
		})

		Context("when we are not the proposer", func() {
			It("should schedule a propose timeout", func() {
				f := func() bool {
					round := processutil.RandomRound(r)
					for round == process.InvalidRound {
						round = processutil.RandomRound(r)
					}
					whoami := id.NewPrivKey().Signatory()
					scheduler := scheduler.NewRoundRobin([]id.Signatory{id.NewPrivKey().Signatory()})

					timerOptions := timer.
						DefaultOptions().
						WithTimeout(10 * time.Millisecond).
						WithTimeoutScaling(0)
					onProposeTimeoutChan := make(chan timer.Timeout, 2)
					timer := timer.NewLinearTimer(timerOptions, onProposeTimeoutChan, nil, nil)

					p := process.New(whoami, 33, timer, scheduler, nil, nil, nil, nil, nil)
					p.StartRound(round)

					timeout := <-onProposeTimeoutChan
					Expect(timeout.Height).To(Equal(process.Height(1)))
					Expect(timeout.Round).To(Equal(round))
					return true
				}
				Expect(quick.Check(f, nil)).To(Succeed())
			})
		})
	})

	// L57:
	//	Function OnTimeoutPropose(height, round)
	//		if height = currentHeight ∧ round = currentRound ∧ currentStep = propose then
	//			broadcast〈PREVOTE, currentHeight, currentRound, nil〉
	//			currentStep ← prevote
	Context("when timing out on a propose", func() {
		r := rand.New(rand.NewSource(time.Now().UnixNano()))

		Context("when the timeout is for the current height", func() {
			Context("when the timeout is for the current round", func() {
				Context("when we are in the proposing step", func() {
					It("should prevote nil and move to the prevoting step", func() {
						f := func() bool {
							round := processutil.RandomRound(r)
							for round == process.InvalidRound {
								round = processutil.RandomRound(r)
							}

							whoami := id.NewPrivKey().Signatory()
							broadcaster := processutil.BroadcasterCallbacks{
								BroadcastPrevoteCallback: func(prevote process.Prevote) {
									Expect(prevote.From.Equal(&whoami)).To(BeTrue())
									Expect(prevote.Value).To(Equal(process.NilValue))
								},
							}

							timerOptions := timer.
								DefaultOptions().
								WithTimeout(10 * time.Millisecond).
								WithTimeoutScaling(0)
							onProposeTimeoutChan := make(chan timer.Timeout, 2)
							timer := timer.NewLinearTimer(timerOptions, onProposeTimeoutChan, nil, nil)

							p := process.New(whoami, 33, timer, nil, nil, nil, broadcaster, nil, nil)
							p.OnTimeoutPropose(process.Height(1), round)
							return true
						}
						Expect(quick.Check(f, nil)).To(Succeed())
					})
				})

				Context("when we are not the proposing step", func() {
					It("should do nothing", func() {
						f := func() bool {
							round := processutil.RandomRound(r)
							for round == process.InvalidRound {
								round = processutil.RandomRound(r)
							}

							whoami := id.NewPrivKey().Signatory()
							broadcaster := processutil.BroadcasterCallbacks{
								BroadcastPrevoteCallback: func(prevote process.Prevote) {
									// We expect the prevote message to never be broadcasted
									Expect(false).To(BeTrue())
								},
							}

							timerOptions := timer.
								DefaultOptions().
								WithTimeout(10 * time.Millisecond).
								WithTimeoutScaling(0)
							onProposeTimeoutChan := make(chan timer.Timeout, 2)
							timer := timer.NewLinearTimer(timerOptions, onProposeTimeoutChan, nil, nil)

							p := process.New(whoami, 33, timer, nil, nil, nil, broadcaster, nil, nil)
							p.State.CurrentStep = process.Prevoting
							p.OnTimeoutPropose(process.Height(1), round)
							return true
						}
						Expect(quick.Check(f, nil)).To(Succeed())
					})
				})
			})

			Context("when the timeout is not in the current round", func() {
				It("should do nothing", func() {
					f := func() bool {
						round := processutil.RandomRound(r)
						for round == process.InvalidRound {
							round = processutil.RandomRound(r)
						}
						whoami := id.NewPrivKey().Signatory()
						broadcaster := processutil.BroadcasterCallbacks{
							BroadcastPrevoteCallback: func(prevote process.Prevote) {
								// We expect the prevote message to never be broadcasted
								Expect(false).To(BeTrue())
							},
						}
						timerOptions := timer.
							DefaultOptions().
							WithTimeout(10 * time.Millisecond).
							WithTimeoutScaling(0)
						onProposeTimeoutChan := make(chan timer.Timeout, 2)
						timer := timer.NewLinearTimer(timerOptions, onProposeTimeoutChan, nil, nil)
						p := process.New(whoami, 33, timer, nil, nil, nil, broadcaster, nil, nil)

						// set the current round
						p.State.CurrentRound = round

						// timeout on some other round
						someOtherRound := processutil.RandomRound(r)
						for someOtherRound == round {
							someOtherRound = processutil.RandomRound(r)
						}
						p.OnTimeoutPropose(process.Height(1), someOtherRound)

						return true
					}
					Expect(quick.Check(f, nil)).To(Succeed())
				})
			})
		})

		Context("when the timeout is not in the current height", func() {
			It("should do nothing", func() {
				f := func() bool {
					round := processutil.RandomRound(r)
					for round == process.InvalidRound {
						round = processutil.RandomRound(r)
					}
					whoami := id.NewPrivKey().Signatory()
					broadcaster := processutil.BroadcasterCallbacks{
						BroadcastPrevoteCallback: func(prevote process.Prevote) {
							// We expect the prevote message to never be broadcasted
							Expect(false).To(BeTrue())
						},
					}
					timerOptions := timer.
						DefaultOptions().
						WithTimeout(10 * time.Millisecond).
						WithTimeoutScaling(0)
					onProposeTimeoutChan := make(chan timer.Timeout, 2)
					timer := timer.NewLinearTimer(timerOptions, onProposeTimeoutChan, nil, nil)
					p := process.New(whoami, 33, timer, nil, nil, nil, broadcaster, nil, nil)

					// when a new process starts, it starts at height == 1
					// timeout for some other height not equal to 1
					height := processutil.RandomHeight(r)
					for height == process.Height(1) {
						height = processutil.RandomHeight(r)
					}
					p.OnTimeoutPropose(height, round)

					return true
				}
				Expect(quick.Check(f, nil)).To(Succeed())
			})
		})
	})

	// L61:
	//	Function OnTimeoutPrevote(height, round)
	//		if height = currentHeight ∧ round = currentRound ∧ currentStep = prevote then
	//			broadcast〈PRECOMMIT, currentHeight, currentRound, nil
	//			currentStep ← precommitting
	Context("when timing out on a prevote", func() {
		r := rand.New(rand.NewSource(time.Now().UnixNano()))

		Context("when the timeout is for the current height", func() {
			Context("when the timeout is for the current round", func() {
				Context("when the current step is prevoting", func() {
					It("should precommit nil and move to the precommitting step", func() {
						f := func() bool {
							round := processutil.RandomRound(r)
							for round == process.InvalidRound {
								round = processutil.RandomRound(r)
							}
							whoami := id.NewPrivKey().Signatory()
							broadcaster := processutil.BroadcasterCallbacks{
								BroadcastPrecommitCallback: func(precommit process.Precommit) {
									Expect(precommit.From.Equal(&whoami)).To(BeTrue())
									Expect(precommit.Value).To(Equal(process.NilValue))
								},
							}
							timerOptions := timer.
								DefaultOptions().
								WithTimeout(10 * time.Millisecond).
								WithTimeoutScaling(0)
							onPrevoteTimeoutChan := make(chan timer.Timeout, 2)
							timer := timer.NewLinearTimer(timerOptions, nil, onPrevoteTimeoutChan, nil)

							p := process.New(whoami, 33, timer, nil, nil, nil, broadcaster, nil, nil)
							p.State.CurrentStep = process.Prevoting
							p.State.CurrentRound = round
							p.OnTimeoutPrevote(process.Height(1), round)

							return true
						}
						Expect(quick.Check(f, nil)).To(Succeed())
					})
				})

				Context("when the current step is not prevoting", func() {
					It("should do nothing", func() {
						f := func() bool {
							round := processutil.RandomRound(r)
							for round == process.InvalidRound {
								round = processutil.RandomRound(r)
							}
							whoami := id.NewPrivKey().Signatory()
							broadcaster := processutil.BroadcasterCallbacks{
								BroadcastPrecommitCallback: func(precommit process.Precommit) {
									// We expect the process to not broadcast any precommit
									Expect(false).To(BeTrue())
								},
							}
							timerOptions := timer.
								DefaultOptions().
								WithTimeout(10 * time.Millisecond).
								WithTimeoutScaling(0)
							onPrevoteTimeoutChan := make(chan timer.Timeout, 2)
							timer := timer.NewLinearTimer(timerOptions, nil, onPrevoteTimeoutChan, nil)

							p := process.New(whoami, 33, timer, nil, nil, nil, broadcaster, nil, nil)
							someOtherStep := processutil.RandomStep(r)
							for someOtherStep == process.Prevoting {
								someOtherStep = processutil.RandomStep(r)
							}
							p.State.CurrentStep = someOtherStep
							p.State.CurrentRound = round
							p.OnTimeoutPrevote(process.Height(1), round)

							return true
						}
						Expect(quick.Check(f, nil)).To(Succeed())
					})
				})
			})

			Context("when the timeout is not for the current round", func() {
				It("should do nothing", func() {
					f := func() bool {
						round := processutil.RandomRound(r)
						for round == process.InvalidRound {
							round = processutil.RandomRound(r)
						}
						whoami := id.NewPrivKey().Signatory()
						broadcaster := processutil.BroadcasterCallbacks{
							BroadcastPrecommitCallback: func(precommit process.Precommit) {
								// We expect the process to not broadcast any precommit
								Expect(false).To(BeTrue())
							},
						}
						timerOptions := timer.
							DefaultOptions().
							WithTimeout(10 * time.Millisecond).
							WithTimeoutScaling(0)
						onPrevoteTimeoutChan := make(chan timer.Timeout, 2)
						timer := timer.NewLinearTimer(timerOptions, nil, onPrevoteTimeoutChan, nil)

						p := process.New(whoami, 33, timer, nil, nil, nil, broadcaster, nil, nil)
						p.State.CurrentStep = process.Prevoting
						p.State.CurrentRound = round
						someOtherRound := processutil.RandomRound(r)
						for someOtherRound == round {
							someOtherRound = processutil.RandomRound(r)
						}
						p.OnTimeoutPrevote(process.Height(1), someOtherRound)

						return true
					}
					Expect(quick.Check(f, nil)).To(Succeed())
				})
			})
		})

		Context("when the timeout is not for the current height", func() {
			It("should do nothing", func() {
				f := func() bool {
					round := processutil.RandomRound(r)
					for round == process.InvalidRound {
						round = processutil.RandomRound(r)
					}
					whoami := id.NewPrivKey().Signatory()
					broadcaster := processutil.BroadcasterCallbacks{
						BroadcastPrecommitCallback: func(precommit process.Precommit) {
							// We expect the process to not broadcast any precommit
							Expect(false).To(BeTrue())
						},
					}
					timerOptions := timer.
						DefaultOptions().
						WithTimeout(10 * time.Millisecond).
						WithTimeoutScaling(0)
					onPrevoteTimeoutChan := make(chan timer.Timeout, 2)
					timer := timer.NewLinearTimer(timerOptions, nil, onPrevoteTimeoutChan, nil)

					p := process.New(whoami, 33, timer, nil, nil, nil, broadcaster, nil, nil)
					p.State.CurrentStep = process.Prevoting
					p.State.CurrentRound = round
					someOtherHeight := processutil.RandomHeight(r)
					for someOtherHeight == process.Height(1) {
						someOtherHeight = processutil.RandomHeight(r)
					}
					p.OnTimeoutPrevote(someOtherHeight, round)

					return true
				}
				Expect(quick.Check(f, nil)).To(Succeed())
			})
		})
	})

	// L65:
	//	Function OnTimeoutPrecommit(height, round)
	//		if height = currentHeight ∧ round = currentRound then
	//			StartRound(currentRound + 1)
	Context("when timing out on a precommit", func() {
		r := rand.New(rand.NewSource(time.Now().UnixNano()))

		Context("when the timeout is for the current height", func() {
			Context("when the timeout is for the current round", func() {
				It("should start a new round by incrementing currentRound", func() {
					f := func() bool {
						round := processutil.RandomRound(r)
						for round == process.InvalidRound {
							round = processutil.RandomRound(r)
						}
						whoami := id.NewPrivKey().Signatory()
						p := process.New(whoami, 33, nil, nil, nil, nil, nil, nil, nil)
						p.State.CurrentStep = processutil.RandomStep(r)
						p.State.CurrentRound = round

						p.OnTimeoutPrecommit(process.Height(1), round)
						Expect(p.State.CurrentRound).To(Equal(round + 1))
						return true
					}
					Expect(quick.Check(f, nil)).To(Succeed())
				})
			})

			Context("when the timeout is not for the current round", func() {
				It("should do nothing", func() {
					f := func() bool {
						round := processutil.RandomRound(r)
						for round == process.InvalidRound {
							round = processutil.RandomRound(r)
						}
						whoami := id.NewPrivKey().Signatory()
						p := process.New(whoami, 33, nil, nil, nil, nil, nil, nil, nil)
						p.State.CurrentRound = round
						p.State.CurrentStep = processutil.RandomStep(r)
						someOtherRound := processutil.RandomRound(r)
						for someOtherRound == round {
							someOtherRound = processutil.RandomRound(r)
						}

						oldState := p.State
						p.OnTimeoutPrecommit(process.Height(1), someOtherRound)
						Expect(p.State).To(Equal(oldState))
						return true
					}
					Expect(quick.Check(f, nil)).To(Succeed())
				})
			})
		})

		Context("when the timeout is not for the current height", func() {
			It("should do nothing", func() {
				f := func() bool {
					round := processutil.RandomRound(r)
					for round == process.InvalidRound {
						round = processutil.RandomRound(r)
					}
					whoami := id.NewPrivKey().Signatory()
					p := process.New(whoami, 33, nil, nil, nil, nil, nil, nil, nil)
					p.State.CurrentStep = processutil.RandomStep(r)
					p.State.CurrentRound = round
					someOtherHeight := processutil.RandomHeight(r)
					for someOtherHeight == process.Height(1) {
						someOtherHeight = processutil.RandomHeight(r)
					}

					oldState := p.State
					p.OnTimeoutPrecommit(someOtherHeight, round)
					Expect(p.State).To(Equal(oldState))
					return true
				}
				Expect(quick.Check(f, nil)).To(Succeed())
			})
		})
	})

	// L22:
	//  upon〈PROPOSAL, currentHeight, currentRound, v, −1〉from proposer(currentHeight, currentRound)
	//  while currentStep = propose do
	//      if valid(v) ∧ (lockedRound = −1 ∨ lockedValue = v) then
	//          broadcast〈PREVOTE, currentHeight, currentRound, id(v)
	//      else
	//          broadcast〈PREVOTE, currentHeight, currentRound, nil
	//      currentStep ← prevote
	Context("when receiving a propose", func() {
		r := rand.New(rand.NewSource(time.Now().UnixNano()))

		Context("when the message sender is the correct proposer for this height and round", func() {
			Context("when we are in the propose step", func() {
				Context("when the propose message is valid", func() {
					Context("when the locked round is equal to -1", func() {
						It("should prevote the value and move to the precommitting step", func() {
							f := func() bool {
								round := processutil.RandomRound(r)
								for round == process.InvalidRound {
									round = processutil.RandomRound(r)
								}
								whoami := id.NewPrivKey().Signatory()
								acknowledge := false
								proposedValue := processutil.RandomValue(r)
								broadcaster := processutil.BroadcasterCallbacks{
									BroadcastPrevoteCallback: func(prevote process.Prevote) {
										Expect(prevote.Value.Equal(&proposedValue)).To(BeTrue())
										Expect(prevote.From).To(Equal(whoami))
										acknowledge = true
									},
								}
								scheduledProposer := id.NewPrivKey().Signatory()
								scheduler := scheduler.NewRoundRobin([]id.Signatory{scheduledProposer})
								validator := processutil.MockValidator{MockValid: func(process.Value) bool { return true }}

								p := process.New(whoami, 33, nil, scheduler, nil, validator, broadcaster, nil, nil)
								p.StartRound(round)

								p.State.CurrentStep = process.Proposing
								p.State.LockedRound = process.InvalidRound

								p.Propose(process.Propose{
									Height:     process.Height(1),
									Round:      round,
									ValidRound: process.InvalidRound,
									Value:      proposedValue,
									From:       scheduledProposer,
								})
								Expect(acknowledge).To(BeTrue())
								Expect(p.State.CurrentStep).To(Equal(process.Prevoting))
								return true
							}
							Expect(quick.Check(f, nil)).To(Succeed())
						})
					})

					Context("when the locked value is the propose value", func() {
						It("should prevote the value and move to the precommitting step", func() {
							f := func() bool {
								round := processutil.RandomRound(r)
								for round == process.InvalidRound {
									round = processutil.RandomRound(r)
								}
								whoami := id.NewPrivKey().Signatory()
								acknowledge := false
								proposedValue := processutil.RandomValue(r)
								broadcaster := processutil.BroadcasterCallbacks{
									BroadcastPrevoteCallback: func(prevote process.Prevote) {
										Expect(prevote.Value.Equal(&proposedValue)).To(BeTrue())
										Expect(prevote.From).To(Equal(whoami))
										acknowledge = true
									},
								}
								scheduledProposer := id.NewPrivKey().Signatory()
								scheduler := scheduler.NewRoundRobin([]id.Signatory{scheduledProposer})
								validator := processutil.MockValidator{MockValid: func(process.Value) bool { return true }}

								p := process.New(whoami, 33, nil, scheduler, nil, validator, broadcaster, nil, nil)
								p.StartRound(round)

								someValidRound := processutil.RandomRound(r)
								for someValidRound == process.InvalidRound {
									someValidRound = processutil.RandomRound(r)
								}
								p.State.CurrentStep = process.Proposing
								p.State.LockedRound = someValidRound
								p.State.LockedValue = proposedValue

								p.Propose(process.Propose{
									Height:     process.Height(1),
									Round:      round,
									ValidRound: process.InvalidRound,
									Value:      proposedValue,
									From:       scheduledProposer,
								})
								Expect(acknowledge).To(BeTrue())
								Expect(p.State.CurrentStep).To(Equal(process.Prevoting))
								return true
							}
							Expect(quick.Check(f, nil)).To(Succeed())
						})
					})

					Context("when the locked round is not -1 and the locked value is not the propose value", func() {
						It("should prevote nil and move to the precommitting step", func() {
							f := func() bool {
								round := processutil.RandomRound(r)
								for round == process.InvalidRound {
									round = processutil.RandomRound(r)
								}
								whoami := id.NewPrivKey().Signatory()
								acknowledge := false
								broadcaster := processutil.BroadcasterCallbacks{
									BroadcastPrevoteCallback: func(prevote process.Prevote) {
										Expect(prevote.Value.Equal(&process.NilValue)).To(BeTrue())
										Expect(prevote.From).To(Equal(whoami))
										acknowledge = true
									},
								}
								scheduledProposer := id.NewPrivKey().Signatory()
								scheduler := scheduler.NewRoundRobin([]id.Signatory{scheduledProposer})
								validator := processutil.MockValidator{MockValid: func(process.Value) bool { return true }}

								p := process.New(whoami, 33, nil, scheduler, nil, validator, broadcaster, nil, nil)
								p.StartRound(round)
								someValidRound := processutil.RandomRound(r)
								for someValidRound == process.InvalidRound {
									someValidRound = processutil.RandomRound(r)
								}
								p.State.CurrentStep = process.Proposing
								p.State.LockedRound = someValidRound

								p.Propose(process.Propose{
									Height:     process.Height(1),
									Round:      round,
									ValidRound: process.InvalidRound,
									Value:      processutil.RandomValue(r),
									From:       scheduledProposer,
								})
								Expect(acknowledge).To(BeTrue())
								Expect(p.State.CurrentStep).To(Equal(process.Prevoting))
								return true
							}
							Expect(quick.Check(f, nil)).To(Succeed())
						})
					})
				})

				Context("when the propose message is invalid", func() {
					It("should prevote nil and move to the precommitting step", func() {
						f := func() bool {
							round := processutil.RandomRound(r)
							for round == process.InvalidRound {
								round = processutil.RandomRound(r)
							}
							whoami := id.NewPrivKey().Signatory()
							acknowledge := false
							broadcaster := processutil.BroadcasterCallbacks{
								BroadcastPrevoteCallback: func(prevote process.Prevote) {
									Expect(prevote.Value.Equal(&process.NilValue)).To(BeTrue())
									Expect(prevote.From).To(Equal(whoami))
									acknowledge = true
								},
							}
							scheduledProposer := id.NewPrivKey().Signatory()
							scheduler := scheduler.NewRoundRobin([]id.Signatory{scheduledProposer})
							validator := processutil.MockValidator{MockValid: func(process.Value) bool { return false }}

							p := process.New(whoami, 33, nil, scheduler, nil, validator, broadcaster, nil, nil)
							p.StartRound(round)
							p.State.CurrentStep = process.Proposing
							p.Propose(process.Propose{
								Height:     process.Height(1),
								Round:      round,
								ValidRound: process.InvalidRound,
								Value:      processutil.RandomValue(r),
								From:       scheduledProposer,
							})
							Expect(acknowledge).To(BeTrue())
							Expect(p.State.CurrentStep).To(Equal(process.Prevoting))
							return true
						}
						Expect(quick.Check(f, nil)).To(Succeed())
					})
				})
			})

			Context("when we are not in the propose step", func() {
				It("should do nothing", func() {
					f := func() bool {
						round := processutil.RandomRound(r)
						for round == process.InvalidRound {
							round = processutil.RandomRound(r)
						}
						whoami := id.NewPrivKey().Signatory()
						broadcaster := processutil.BroadcasterCallbacks{
							BroadcastPrevoteCallback: func(prevote process.Prevote) {
								// We don't expect any prevote broadcast
								Expect(false).To(BeTrue())
							},
						}
						scheduledProposer := id.NewPrivKey().Signatory()
						scheduler := scheduler.NewRoundRobin([]id.Signatory{scheduledProposer})
						validator := processutil.MockValidator{MockValid: func(process.Value) bool { return true }}

						p := process.New(whoami, 33, nil, scheduler, nil, validator, broadcaster, nil, nil)
						p.StartRound(round)
						someOtherStep := processutil.RandomStep(r)
						for someOtherStep == process.Proposing {
							someOtherStep = processutil.RandomStep(r)
						}
						p.State.CurrentStep = someOtherStep
						p.Propose(process.Propose{
							Height:     process.Height(1),
							Round:      round,
							ValidRound: process.InvalidRound,
							Value:      processutil.RandomValue(r),
							From:       scheduledProposer,
						})
						return true
					}
					Expect(quick.Check(f, nil)).To(Succeed())
				})
			})
		})

		Context("when the message sender is not the correct proposer for this height and round", func() {
			It("should do nothing", func() {
				f := func() bool {
					round := processutil.RandomRound(r)
					for round == process.InvalidRound {
						round = processutil.RandomRound(r)
					}
					whoami := id.NewPrivKey().Signatory()
					broadcaster := processutil.BroadcasterCallbacks{
						BroadcastPrevoteCallback: func(prevote process.Prevote) {
							// We don't expect any prevote broadcast
							Expect(false).To(BeTrue())
						},
					}
					scheduler := scheduler.NewRoundRobin([]id.Signatory{id.NewPrivKey().Signatory()})

					p := process.New(whoami, 33, nil, scheduler, nil, nil, broadcaster, nil, nil)
					p.StartRound(round)
					p.State.CurrentStep = process.Proposing
					p.Propose(process.Propose{
						Height:     process.Height(1),
						Round:      round,
						ValidRound: process.InvalidRound,
						Value:      processutil.RandomValue(r),
						From:       id.NewPrivKey().Signatory(),
					})
					return true
				}
				Expect(quick.Check(f, nil)).To(Succeed())
			})
		})

		Context("when the height of the propose message is not the current height of the process", func() {
			It("should do nothing", func() {
				f := func() bool {
					round := processutil.RandomRound(r)
					for round == process.InvalidRound {
						round = processutil.RandomRound(r)
					}
					height := process.Height(r.Int63())
					whoami := id.NewPrivKey().Signatory()
					broadcaster := processutil.BroadcasterCallbacks{
						BroadcastPrevoteCallback: func(prevote process.Prevote) {
							// We don't expect any prevote broadcast
							Expect(false).To(BeTrue())
						},
					}
					scheduler := scheduler.NewRoundRobin([]id.Signatory{id.NewPrivKey().Signatory()})

					p := process.New(whoami, 33, nil, scheduler, nil, nil, broadcaster, nil, nil)
					p.StartRound(round)
					p.State.CurrentStep = process.Proposing
					prevState := p.State

					// propose msg for a different height
					p.Propose(process.Propose{
						Height:     height,
						Round:      round,
						ValidRound: process.InvalidRound,
						Value:      processutil.RandomValue(r),
						From:       id.NewPrivKey().Signatory(),
					})

					Expect(p.State).To(Equal(prevState))
					return true
				}
				Expect(quick.Check(f, nil)).To(Succeed())
			})
		})
	})

	// L28:
	//
	//  upon〈PROPOSAL, currentHeight, currentRound, v, vr〉from proposer(currentHeight, currentRound) AND 2f+ 1〈PREVOTE, currentHeight, vr, id(v)〉
	//  while currentStep = propose ∧ (vr ≥ 0 ∧ vr < currentRound) do
	//      if valid(v) ∧ (lockedRound ≤ vr ∨ lockedValue = v) then
	//          broadcast〈PREVOTE, currentHeight, currentRound, id(v)〉
	//      else
	//          broadcast〈PREVOTE, currentHeight, currentRound, nil〉
	//      currentStep ← prevote
	Context("when receiving a propose and 2f+1 prevotes", func() {
		r := rand.New(rand.NewSource(time.Now().UnixNano()))

		randomValidPrevoteMsg := func(
			r *rand.Rand,
			from id.Signatory,
			height process.Height,
			round process.Round,
			value process.Value,
		) process.Prevote {
			msg := processutil.RandomPrevote(r)

			msg.From = from
			msg.Height = height
			msg.Round = round
			msg.Value = value

			return msg
		}

		Context("when the message sender is the correct proposer for the given height and round", func() {
			Context("when we are in the proposing step", func() {
				Context("when the proposed valid round is valid", func() {
					Context("when the proposed valid round is less than the current round", func() {
						Context("when the proposed value is valid", func() {
							Context("when the proposed valid round is greater than our locked round, or the proposed value is our locked value", func() {
								It("should prevote for the proposed value (when valid round is greater than locked round)", func() {
									loop := func() bool {
										// ensure that valid round is greater than or equal to the locked round
										lockedRound := process.Round(5 + rand.Intn(20))
										validRound := lockedRound + process.Round(rand.Intn(2))
										// the current round should be greater than the valid round
										currentRound := validRound + 1

										// identity of this process
										whoami := id.NewPrivKey().Signatory()
										// random f
										f := 10 + rand.Intn(40)
										// identity of the valid proposer for current round
										scheduledProposer := id.NewPrivKey().Signatory()
										// this scheduler will always schedule the above proposer to propose
										mockScheduler := scheduler.NewRoundRobin([]id.Signatory{scheduledProposer})
										// mock validator that considers any proposal as valid
										mockValidator := processutil.MockValidator{MockValid: func(process.Value) bool { return true }}
										// this will be the proposed value
										proposedValue := processutil.RandomValue(r)
										for proposedValue == process.NilValue {
											proposedValue = processutil.RandomValue(r)
										}
										// broadcaster callbacks, only a prevote is expected
										acknowledge := false
										broadcaster := processutil.BroadcasterCallbacks{
											BroadcastProposeCallback: func(msg process.Propose) {
												// this should never get called
												Expect(true).ToNot(BeTrue())
											},
											BroadcastPrevoteCallback: func(msg process.Prevote) {
												Expect(msg.Value).To(Equal(proposedValue))
												Expect(msg.From.Equal(&whoami)).To(BeTrue())
												acknowledge = true
											},
											BroadcastPrecommitCallback: func(msg process.Precommit) {
												// this should never get called
												Expect(true).ToNot(BeTrue())
											},
										}
										// create process and start this round
										p := process.New(whoami, f, nil, mockScheduler, nil, mockValidator, broadcaster, nil, nil)
										p.StartRound(currentRound)
										p.State.LockedRound = lockedRound

										// send 2*f + 1 prevotes for the valid round
										for t := 0; t < 2*f+1; t++ {
											prevoteMsg := randomValidPrevoteMsg(r, id.NewPrivKey().Signatory(), process.Height(1), validRound, proposedValue)
											p.Prevote(prevoteMsg)
										}

										// construct the proposal message
										p.Propose(process.Propose{
											From:       scheduledProposer,
											Value:      proposedValue,
											Height:     process.Height(1),
											Round:      currentRound,
											ValidRound: validRound,
										})

										Expect(p.State.CurrentStep).To(Equal(process.Prevoting))
										Expect(acknowledge).To(BeTrue())
										return true
									}
									Expect(quick.Check(loop, nil)).To(Succeed())
								})

								It("should prevote for the proposed value (when proposed value is also the locked value)", func() {
									loop := func() bool {
										// the valid round is less than the locked round
										// but the proposed value should be equal to the locked value
										lockedRound := process.Round(5 + rand.Intn(20))
										validRound := lockedRound - 1
										// current round is ensured to be just greater than the set valid round
										currentRound := validRound + 1

										// identity of this process
										whoami := id.NewPrivKey().Signatory()
										// random f
										f := 10 + rand.Intn(40)
										// identity of the valid proposer for current round
										scheduledProposer := id.NewPrivKey().Signatory()
										// this scheduler will always schedule the above proposer to propose
										mockScheduler := scheduler.NewRoundRobin([]id.Signatory{scheduledProposer})
										// mock validator that considers any proposal as valid
										mockValidator := processutil.MockValidator{MockValid: func(process.Value) bool { return true }}
										// this will be the proposed value
										proposedValue := processutil.RandomValue(r)
										for proposedValue == process.NilValue {
											proposedValue = processutil.RandomValue(r)
										}
										// broadcaster callbacks, only a prevote is expected
										acknowledge := false
										broadcaster := processutil.BroadcasterCallbacks{
											BroadcastProposeCallback: func(msg process.Propose) {
												// this should never get called
												Expect(true).ToNot(BeTrue())
											},
											BroadcastPrevoteCallback: func(msg process.Prevote) {
												Expect(msg.Value).To(Equal(proposedValue))
												Expect(msg.From.Equal(&whoami)).To(BeTrue())
												acknowledge = true
											},
											BroadcastPrecommitCallback: func(msg process.Precommit) {
												// this should never get called
												Expect(true).ToNot(BeTrue())
											},
										}
										// create process and start this round
										p := process.New(whoami, f, nil, mockScheduler, nil, mockValidator, broadcaster, nil, nil)
										p.StartRound(currentRound)
										p.State.LockedRound = lockedRound
										p.State.LockedValue = proposedValue

										// send 2*f + 1 prevotes for the valid round
										for t := 0; t < 2*f+1; t++ {
											prevoteMsg := randomValidPrevoteMsg(r, id.NewPrivKey().Signatory(), process.Height(1), validRound, proposedValue)
											p.Prevote(prevoteMsg)
										}

										// construct the proposal message
										p.Propose(process.Propose{
											From:       scheduledProposer,
											Value:      proposedValue,
											Height:     process.Height(1),
											Round:      currentRound,
											ValidRound: validRound,
										})

										Expect(p.State.CurrentStep).To(Equal(process.Prevoting))
										Expect(acknowledge).To(BeTrue())
										return true
									}
									Expect(quick.Check(loop, nil)).To(Succeed())
								})
							})

							Context("when the proposed valid round is less than our locked round, and the proposed value is not our locked value", func() {
								It("should prevote nil", func() {
									loop := func() bool {
										// ensure that the valid round (of prevotes and proposal)
										// is less than the process' locked round
										lockedRound := process.Round(5 + rand.Intn(20))
										validRound := lockedRound - 1
										currentRound := lockedRound

										// identity of this process
										whoami := id.NewPrivKey().Signatory()
										// random f
										f := 10 + rand.Intn(40)
										// identity of the valid proposer for current round
										scheduledProposer := id.NewPrivKey().Signatory()
										// this scheduler will always schedule the above proposer to propose
										mockScheduler := scheduler.NewRoundRobin([]id.Signatory{scheduledProposer})
										// mock validator that considers any proposal as valid
										mockValidator := processutil.MockValidator{MockValid: func(process.Value) bool { return true }}
										// this will be the proposed value
										proposedValue := processutil.RandomValue(r)
										for proposedValue == process.NilValue {
											proposedValue = processutil.RandomValue(r)
										}
										lockedValue := processutil.RandomValue(r)
										for lockedValue == proposedValue {
											lockedValue = processutil.RandomValue(r)
										}
										// broadcaster callbacks, only a prevote is expected
										acknowledge := false
										broadcaster := processutil.BroadcasterCallbacks{
											BroadcastProposeCallback: func(msg process.Propose) {
												// this should never get called
												Expect(true).ToNot(BeTrue())
											},
											BroadcastPrevoteCallback: func(msg process.Prevote) {
												// nil prevote
												Expect(msg.Value).To(Equal(process.NilValue))
												Expect(msg.From.Equal(&whoami)).To(BeTrue())
												acknowledge = true
											},
											BroadcastPrecommitCallback: func(msg process.Precommit) {
												// this should never get called
												Expect(true).ToNot(BeTrue())
											},
										}
										// create process and start this round
										p := process.New(whoami, f, nil, mockScheduler, nil, mockValidator, broadcaster, nil, nil)
										p.StartRound(currentRound)
										p.State.LockedRound = lockedRound
										p.State.LockedValue = lockedValue

										// send 2*f + 1 prevotes for the valid round
										for t := 0; t < 2*f+1; t++ {
											prevoteMsg := randomValidPrevoteMsg(r, id.NewPrivKey().Signatory(), process.Height(1), validRound, proposedValue)
											p.Prevote(prevoteMsg)
										}

										// construct the proposal message
										p.Propose(process.Propose{
											From:       scheduledProposer,
											Value:      proposedValue,
											Height:     process.Height(1),
											Round:      currentRound,
											ValidRound: validRound,
										})

										Expect(p.State.CurrentStep).To(Equal(process.Prevoting))
										Expect(acknowledge).To(BeTrue())
										return true
									}
									Expect(quick.Check(loop, nil)).To(Succeed())
								})
							})
						})

						Context("when the proposed value is not valid", func() {
							It("should prevote nil", func() {
								loop := func() bool {
									// ensure that all conditions WRT locked round/locked value are satisfied
									lockedRound := process.Round(5 + rand.Intn(20))
									validRound := lockedRound + 1
									currentRound := validRound + 1

									// identity of this process
									whoami := id.NewPrivKey().Signatory()
									// random f
									f := 10 + rand.Intn(40)
									// identity of the valid proposer for current round
									scheduledProposer := id.NewPrivKey().Signatory()
									// this scheduler will always schedule the above proposer to propose
									mockScheduler := scheduler.NewRoundRobin([]id.Signatory{scheduledProposer})
									// mock validator that considers any proposal as invalid
									mockValidator := processutil.MockValidator{MockValid: func(process.Value) bool { return false }}
									// this will be the proposed value
									proposedValue := processutil.RandomValue(r)
									for proposedValue == process.NilValue {
										proposedValue = processutil.RandomValue(r)
									}
									// broadcaster callbacks, only a prevote is expected
									acknowledge := false
									broadcaster := processutil.BroadcasterCallbacks{
										BroadcastProposeCallback: func(msg process.Propose) {
											// this should never get called
											Expect(true).ToNot(BeTrue())
										},
										BroadcastPrevoteCallback: func(msg process.Prevote) {
											// nil prevote
											Expect(msg.Value).To(Equal(process.NilValue))
											Expect(msg.From.Equal(&whoami)).To(BeTrue())
											acknowledge = true
										},
										BroadcastPrecommitCallback: func(msg process.Precommit) {
											// this should never get called
											Expect(true).ToNot(BeTrue())
										},
									}
									// create process and start this round
									p := process.New(whoami, f, nil, mockScheduler, nil, mockValidator, broadcaster, nil, nil)
									p.StartRound(currentRound)
									p.State.LockedRound = lockedRound
									p.State.LockedValue = proposedValue

									// send 2*f + 1 prevotes for the valid round
									for t := 0; t < 2*f+1; t++ {
										prevoteMsg := randomValidPrevoteMsg(r, id.NewPrivKey().Signatory(), process.Height(1), validRound, proposedValue)
										p.Prevote(prevoteMsg)
									}

									// construct the proposal message
									p.Propose(process.Propose{
										From:       scheduledProposer,
										Value:      proposedValue,
										Height:     process.Height(1),
										Round:      currentRound,
										ValidRound: validRound,
									})

									Expect(p.State.CurrentStep).To(Equal(process.Prevoting))
									Expect(acknowledge).To(BeTrue())
									return true
								}
								Expect(quick.Check(loop, nil)).To(Succeed())
							})
						})
					})

					Context("when the proposed valid round is not less than the current round", func() {
						It("should do nothing", func() {
							loop := func() bool {
								// ensure that all conditions WRT locked round/locked value are satisfied
								// but set the valid round equal to the current round
								lockedRound := process.Round(5 + rand.Intn(20))
								currentRound := lockedRound + 1
								validRound := currentRound

								// identity of this process
								whoami := id.NewPrivKey().Signatory()
								// random f
								f := 10 + rand.Intn(40)
								// identity of the valid proposer for current round
								scheduledProposer := id.NewPrivKey().Signatory()
								// this scheduler will always schedule the above proposer to propose
								mockScheduler := scheduler.NewRoundRobin([]id.Signatory{scheduledProposer})
								// mock validator that considers any proposal as valid
								mockValidator := processutil.MockValidator{MockValid: func(process.Value) bool { return true }}
								// this will be the proposed value
								proposedValue := processutil.RandomValue(r)
								for proposedValue == process.NilValue {
									proposedValue = processutil.RandomValue(r)
								}
								// broadcaster callbacks, only a prevote is expected
								broadcaster := processutil.BroadcasterCallbacks{
									BroadcastProposeCallback: func(msg process.Propose) {
										// this should never get called
										Expect(true).ToNot(BeTrue())
									},
									BroadcastPrevoteCallback: func(msg process.Prevote) {
										// this should never get called
										Expect(true).ToNot(BeTrue())
									},
									BroadcastPrecommitCallback: func(msg process.Precommit) {
										// this should never get called
										Expect(true).ToNot(BeTrue())
									},
								}
								// create process and start this round
								p := process.New(whoami, f, nil, mockScheduler, nil, mockValidator, broadcaster, nil, nil)
								p.StartRound(currentRound)
								p.State.LockedRound = lockedRound
								p.State.LockedValue = proposedValue

								// send 2*f + 1 prevotes for the valid round
								for t := 0; t < 2*f+1; t++ {
									prevoteMsg := randomValidPrevoteMsg(r, id.NewPrivKey().Signatory(), process.Height(1), validRound, proposedValue)
									p.Prevote(prevoteMsg)
								}

								// construct the proposal message
								p.Propose(process.Propose{
									From:       scheduledProposer,
									Value:      proposedValue,
									Height:     process.Height(1),
									Round:      currentRound,
									ValidRound: validRound,
								})

								// note that in this case, the process simply ignores this
								Expect(p.State.CurrentStep).To(Equal(process.Proposing))
								return true
							}
							Expect(quick.Check(loop, nil)).To(Succeed())
						})
					})
				})
			})
		})

		Context("when the message sender is not the correct proposer for the given height and round", func() {
			It("should do nothing", func() {
				loop := func() bool {
					round := processutil.RandomRound(r)
					for round == process.InvalidRound {
						round = processutil.RandomRound(r)
					}
					whoami := id.NewPrivKey().Signatory()
					f := 10 + rand.Intn(40)
					scheduledProposer := id.NewPrivKey().Signatory()
					mockScheduler := scheduler.NewRoundRobin([]id.Signatory{scheduledProposer})
					mockValidator := processutil.MockValidator{MockValid: func(process.Value) bool { return true }}
					value := processutil.RandomValue(r)
					for value == process.NilValue {
						value = processutil.RandomValue(r)
					}
					broadcaster := processutil.BroadcasterCallbacks{
						BroadcastPrevoteCallback: func(msg process.Prevote) {
							// this should never get called
							Expect(true).ToNot(BeTrue())
						},
					}
					p := process.New(whoami, f, nil, mockScheduler, nil, mockValidator, broadcaster, nil, nil)
					p.StartRound(round)

					for t := 0; t < 2*f+1; t++ {
						prevoteMsg := randomValidPrevoteMsg(r, id.NewPrivKey().Signatory(), process.Height(1), round, value)
						p.Prevote(prevoteMsg)
					}
					prevState := p.State

					p.Propose(process.Propose{
						From:   id.NewPrivKey().Signatory(),
						Value:  value,
						Height: process.Height(1),
						Round:  round,
					})

					Expect(p.State).To(Equal(prevState))
					return true
				}
				Expect(quick.Check(loop, nil)).To(Succeed())
			})
		})
	})

	// L34:
	//
	//  upon 2f+ 1〈PREVOTE, currentHeight, currentRound, ∗〉
	//  while currentStep = prevote for the first time do
	//      scheduleOnTimeoutPrevote(currentHeight, currentRound) to be executed after timeoutPrevote(currentRound)
	Context("when receiving 2f+1 prevotes", func() {
		r := rand.New(rand.NewSource(time.Now().UnixNano()))

		randomValidPrevoteMsg := func(
			r *rand.Rand,
			from id.Signatory,
			height process.Height,
			round process.Round,
		) process.Prevote {
			msg := processutil.RandomPrevote(r)

			msg.From = from
			msg.Height = height
			msg.Round = round

			return msg
		}

		Context("when we are in step prevote", func() {
			It("should schedule a prevote timeout for the current height and round", func() {
				loop := func() bool {
					// current round
					currentRound := processutil.RandomRound(r)
					for currentRound == process.InvalidRound {
						currentRound = processutil.RandomRound(r)
					}

					// own process components
					whoami := id.NewPrivKey().Signatory()
					f := 10 + (r.Int() % 5)
					timerOptions := timer.
						DefaultOptions().
						WithTimeout(1 * time.Millisecond).
						WithTimeoutScaling(0)
					onPrevoteTimeoutChan := make(chan timer.Timeout, 2)
					timer := timer.NewLinearTimer(timerOptions, nil, onPrevoteTimeoutChan, nil)

					// instantiate a new process
					// and start round
					p := process.New(whoami, f, timer, nil, nil, nil, nil, nil, nil)
					p.StartRound(currentRound)
					// set the current step to be prevoting
					p.State.CurrentStep = process.Prevoting

					// no change up to 2*f prevotes
					for t := 0; t < 2*f; t++ {
						prevoteMsg := randomValidPrevoteMsg(r, id.NewPrivKey().Signatory(), process.Height(1), currentRound)
						p.Prevote(prevoteMsg)

						time.Sleep(5 * time.Millisecond)
						select {
						case _ = <-onPrevoteTimeoutChan:
							// this should never happen
							Expect(true).ToNot(BeTrue())
						default:
							Expect(true).To(BeTrue())
						}
					}

					// send the 2f+1'th prevote
					prevoteMsg := randomValidPrevoteMsg(r, id.NewPrivKey().Signatory(), process.Height(1), currentRound)
					p.Prevote(prevoteMsg)

					time.Sleep(5 * time.Millisecond)
					select {
					case timeout := <-onPrevoteTimeoutChan:
						Expect(timeout.Round).To(Equal(currentRound))
						Expect(timeout.Height).To(Equal(process.Height(1)))
					default:
						// we do expect to receive a scheduled timeout
						Expect(true).ToNot(BeTrue())
					}

					// should not schedule a timeout again (that the once flags work)
					prevoteMsg = randomValidPrevoteMsg(r, id.NewPrivKey().Signatory(), process.Height(1), currentRound)
					p.Prevote(prevoteMsg)

					time.Sleep(5 * time.Millisecond)
					select {
					case _ = <-onPrevoteTimeoutChan:
						// this should never happen
						Expect(true).ToNot(BeTrue())
					default:
						Expect(true).To(BeTrue())
					}

					return true
				}
				Expect(quick.Check(loop, nil)).To(Succeed())
			})
		})

		Context("when we are not in the step prevote", func() {
			It("should do nothing", func() {
				loop := func() bool {
					// current round
					currentRound := processutil.RandomRound(r)
					for currentRound == process.InvalidRound {
						currentRound = processutil.RandomRound(r)
					}

					// own process components
					whoami := id.NewPrivKey().Signatory()
					f := 10 + (r.Int() % 5)
					timerOptions := timer.
						DefaultOptions().
						WithTimeout(1 * time.Millisecond).
						WithTimeoutScaling(0)
					onPrevoteTimeoutChan := make(chan timer.Timeout, 2)
					timer := timer.NewLinearTimer(timerOptions, nil, onPrevoteTimeoutChan, nil)

					// instantiate a new process
					// and start round
					p := process.New(whoami, f, timer, nil, nil, nil, nil, nil, nil)
					p.StartRound(currentRound)
					// set the current step to not be prevoting
					someOtherStep := processutil.RandomStep(r)
					for someOtherStep == process.Prevoting {
						someOtherStep = processutil.RandomStep(r)
					}
					p.State.CurrentStep = someOtherStep

					// no change up to 2*f prevotes
					for t := 0; t < 2*f; t++ {
						prevoteMsg := randomValidPrevoteMsg(r, id.NewPrivKey().Signatory(), process.Height(1), currentRound)
						p.Prevote(prevoteMsg)

						time.Sleep(5 * time.Millisecond)
						select {
						case _ = <-onPrevoteTimeoutChan:
							// this should never happen
							Expect(true).ToNot(BeTrue())
						default:
							Expect(true).To(BeTrue())
						}
					}

					// send the 2f+1'th prevote
					prevoteMsg := randomValidPrevoteMsg(r, id.NewPrivKey().Signatory(), process.Height(1), currentRound)
					p.Prevote(prevoteMsg)

					time.Sleep(5 * time.Millisecond)
					select {
					case _ = <-onPrevoteTimeoutChan:
						// this should never happen
						Expect(true).ToNot(BeTrue())
					default:
						// we dont expect to receive any timeout
						Expect(true).To(BeTrue())
					}

					return true
				}
				Expect(quick.Check(loop, nil)).To(Succeed())
			})
		})
	})

	// L36:
	//
	//  upon〈PROPOSAL, currentHeight, currentRound, v, ∗〉from proposer(currentHeight, currentRound) AND 2f+ 1〈PREVOTE, currentHeight, currentRound, id(v)〉
	//  while valid(v) ∧ currentStep ≥ prevote for the first time do
	//      if currentStep = prevote then
	//          lockedValue ← v
	//          lockedRound ← currentRound
	//          broadcast〈PRECOMMIT, currentHeight, currentRound, id(v))〉
	//          currentStep ← precommit
	//      validValue ← v
	//      validRound ← currentRound
	Context("when receiving a propose and 2f+1 prevotes, for any locked round in the propose message", func() {
		r := rand.New(rand.NewSource(time.Now().UnixNano()))

		randomValidPrevoteMsg := func(
			r *rand.Rand,
			height process.Height,
			round process.Round,
			value process.Value,
		) process.Prevote {
			msg := processutil.RandomPrevote(r)

			msg.From = id.NewPrivKey().Signatory()
			msg.Height = height
			msg.Round = round
			msg.Value = value

			return msg
		}

		Context("when the proposed value is valid", func() {
			Context("when we are at least in the prevoting step", func() {
				Context("when we are in the prevoting step", func() {
					It("should set the locked value, locked round, precommit the value, step to precommit", func() {
						loop := func() bool {
							currentHeight := process.Height(r.Int63())
							currentRound := process.Round(r.Int63())
							proposedValue := processutil.RandomValue(r)
							for proposedValue == process.NilValue {
								proposedValue = processutil.RandomValue(r)
							}
							whoami := id.NewPrivKey().Signatory()
							f := 5 + (r.Int() % 10)

							acknowledge := false
							broadcaster := processutil.BroadcasterCallbacks{
								BroadcastProposeCallback: func(msg process.Propose) {
									// we dont expect any propose
									Expect(true).ToNot(BeTrue())
								},
								BroadcastPrevoteCallback: func(msg process.Prevote) {
									// we dont expect any prevotes
									Expect(true).ToNot(BeTrue())
								},
								BroadcastPrecommitCallback: func(msg process.Precommit) {
									Expect(msg.From.Equal(&whoami)).To(BeTrue())
									Expect(msg.Value).To(Equal(proposedValue))
									acknowledge = true
								},
							}
							scheduledProposer := id.NewPrivKey().Signatory()
							scheduler := scheduler.NewRoundRobin([]id.Signatory{scheduledProposer})

							// instantiate a new process and its state
							p := process.New(whoami, f, nil, scheduler, nil, nil, broadcaster, nil, nil)
							p.State.CurrentHeight = currentHeight
							p.StartRound(currentRound)
							p.State.CurrentStep = process.Prevoting

							// feed 2f+1 prevotes
							for t := 0; t < 2*f+1; t++ {
								msg := randomValidPrevoteMsg(r, currentHeight, currentRound, proposedValue)
								p.Prevote(msg)
							}
							Expect(p.State.LockedValue).ToNot(Equal(proposedValue))
							Expect(p.State.LockedRound).ToNot(Equal(currentRound))
							Expect(p.State.CurrentStep).To(Equal(process.Prevoting))
							Expect(acknowledge).ToNot(BeTrue())

							// feed the propose message with a different value, no change
							invalidMsg := process.Propose{
								From:   id.NewPrivKey().Signatory(),
								Height: currentHeight,
								Round:  currentRound,
								Value:  processutil.RandomValue(r),
							}
							p.Propose(invalidMsg)
							Expect(p.State.LockedValue).ToNot(Equal(proposedValue))
							Expect(p.State.LockedRound).ToNot(Equal(currentRound))
							Expect(p.State.CurrentStep).To(Equal(process.Prevoting))
							Expect(acknowledge).ToNot(BeTrue())

							// feed the propose message with the correct value, expected behaviour
							validMsg := process.Propose{
								From:   scheduledProposer,
								Height: currentHeight,
								Round:  currentRound,
								Value:  proposedValue,
							}
							p.Propose(validMsg)
							Expect(p.State.LockedValue).To(Equal(proposedValue))
							Expect(p.State.LockedRound).To(Equal(currentRound))
							Expect(p.State.ValidValue).To(Equal(proposedValue))
							Expect(p.State.ValidRound).To(Equal(currentRound))
							Expect(p.State.CurrentStep).To(Equal(process.Precommitting))
							Expect(acknowledge).To(BeTrue())

							// no changes further (once flags should prevent any changes)
							validMsg = process.Propose{
								From:   scheduledProposer,
								Height: currentHeight,
								Round:  currentRound,
								Value:  proposedValue,
							}
							p.Propose(validMsg)
							Expect(p.State.LockedValue).To(Equal(proposedValue))
							Expect(p.State.LockedRound).To(Equal(currentRound))
							Expect(p.State.ValidValue).To(Equal(proposedValue))
							Expect(p.State.ValidRound).To(Equal(currentRound))
							Expect(p.State.CurrentStep).To(Equal(process.Precommitting))

							// no changes further (once flags should prevent any changes)
							msg := randomValidPrevoteMsg(r, currentHeight, currentRound, proposedValue)
							p.Prevote(msg)
							Expect(p.State.LockedValue).To(Equal(proposedValue))
							Expect(p.State.LockedRound).To(Equal(currentRound))
							Expect(p.State.ValidValue).To(Equal(proposedValue))
							Expect(p.State.ValidRound).To(Equal(currentRound))
							Expect(p.State.CurrentStep).To(Equal(process.Precommitting))

							return true
						}
						Expect(quick.Check(loop, nil)).To(Succeed())
					})
				})

				Context("when we are in the precommitting step", func() {
					It("should set the valid value and the valid round", func() {
						loop := func() bool {
							currentHeight := process.Height(r.Int63())
							currentRound := process.Round(r.Int63())
							proposedValue := processutil.RandomValue(r)
							for proposedValue == process.NilValue {
								proposedValue = processutil.RandomValue(r)
							}
							whoami := id.NewPrivKey().Signatory()
							f := 5 + (r.Int() % 10)

							broadcaster := processutil.BroadcasterCallbacks{
								BroadcastProposeCallback: func(msg process.Propose) {
									// we dont expect any propose
									Expect(true).ToNot(BeTrue())
								},
								BroadcastPrevoteCallback: func(msg process.Prevote) {
									// we dont expect any prevotes
									Expect(true).ToNot(BeTrue())
								},
								BroadcastPrecommitCallback: func(msg process.Precommit) {
									// we dont expect any prevotes
									Expect(true).ToNot(BeTrue())
								},
							}
							scheduledProposer := id.NewPrivKey().Signatory()
							scheduler := scheduler.NewRoundRobin([]id.Signatory{scheduledProposer})

							// instantiate a new process and its state
							p := process.New(whoami, f, nil, scheduler, nil, nil, broadcaster, nil, nil)
							p.State.CurrentHeight = currentHeight
							p.StartRound(currentRound)
							p.State.CurrentStep = process.Precommitting

							// feed 2f+1 prevotes
							for t := 0; t < 2*f+1; t++ {
								msg := randomValidPrevoteMsg(r, currentHeight, currentRound, proposedValue)
								p.Prevote(msg)
							}
							Expect(p.State.ValidValue).ToNot(Equal(proposedValue))
							Expect(p.State.ValidRound).ToNot(Equal(currentRound))
							Expect(p.State.CurrentStep).To(Equal(process.Precommitting))

							// feed the propose message with a different value, no change
							invalidMsg := process.Propose{
								From:   id.NewPrivKey().Signatory(),
								Height: currentHeight,
								Round:  currentRound,
								Value:  processutil.RandomValue(r),
							}
							p.Propose(invalidMsg)
							Expect(p.State.ValidValue).ToNot(Equal(proposedValue))
							Expect(p.State.ValidRound).ToNot(Equal(currentRound))
							Expect(p.State.CurrentStep).To(Equal(process.Precommitting))

							// feed the propose message with the correct value, expected behaviour
							validMsg := process.Propose{
								From:   scheduledProposer,
								Height: currentHeight,
								Round:  currentRound,
								Value:  proposedValue,
							}
							p.Propose(validMsg)
							Expect(p.State.ValidValue).To(Equal(proposedValue))
							Expect(p.State.ValidRound).To(Equal(currentRound))
							Expect(p.State.CurrentStep).To(Equal(process.Precommitting))

							// no changes further (once flags should prevent any changes)
							validMsg = process.Propose{
								From:   scheduledProposer,
								Height: currentHeight,
								Round:  currentRound,
								Value:  proposedValue,
							}
							p.Propose(validMsg)
							Expect(p.State.ValidValue).To(Equal(proposedValue))
							Expect(p.State.ValidRound).To(Equal(currentRound))
							Expect(p.State.CurrentStep).To(Equal(process.Precommitting))

							return true
						}
						Expect(quick.Check(loop, nil)).To(Succeed())
					})
				})
			})

			Context("when we are in the proposing step", func() {
				It("should do nothing", func() {
					loop := func() bool {
						currentHeight := process.Height(r.Int63())
						currentRound := process.Round(r.Int63())
						proposedValue := processutil.RandomValue(r)
						for proposedValue == process.NilValue {
							proposedValue = processutil.RandomValue(r)
						}
						whoami := id.NewPrivKey().Signatory()
						f := 5 + (r.Int() % 10)

						broadcaster := processutil.BroadcasterCallbacks{
							BroadcastProposeCallback: func(msg process.Propose) {
								// we dont expect any propose
								Expect(true).ToNot(BeTrue())
							},
							BroadcastPrevoteCallback: func(msg process.Prevote) {
								// we dont expect any prevotes
								Expect(true).ToNot(BeTrue())
							},
							BroadcastPrecommitCallback: func(msg process.Precommit) {
								// we dont expect any prevotes
								Expect(true).ToNot(BeTrue())
							},
						}
						scheduledProposer := id.NewPrivKey().Signatory()
						scheduler := scheduler.NewRoundRobin([]id.Signatory{scheduledProposer})

						// instantiate a new process and its state
						p := process.New(whoami, f, nil, scheduler, nil, nil, broadcaster, nil, nil)
						p.State.CurrentHeight = currentHeight
						p.StartRound(currentRound)
						p.State.CurrentStep = process.Proposing

						// feed 2f+1 prevotes
						for t := 0; t < 2*f+1; t++ {
							msg := randomValidPrevoteMsg(r, currentHeight, currentRound, proposedValue)
							p.Prevote(msg)
						}
						Expect(p.State.ValidValue).ToNot(Equal(proposedValue))
						Expect(p.State.ValidRound).ToNot(Equal(currentRound))

						// feed the propose message with a different value, no change
						invalidMsg := process.Propose{
							From:   id.NewPrivKey().Signatory(),
							Height: currentHeight,
							Round:  currentRound,
							Value:  processutil.RandomValue(r),
						}
						p.Propose(invalidMsg)
						Expect(p.State.ValidValue).ToNot(Equal(proposedValue))
						Expect(p.State.ValidRound).ToNot(Equal(currentRound))

						// feed the propose message with the correct value, still do nothing
						validMsg := process.Propose{
							From:   scheduledProposer,
							Height: currentHeight,
							Round:  currentRound,
							Value:  proposedValue,
						}
						p.Propose(validMsg)
						Expect(p.State.ValidValue).ToNot(Equal(proposedValue))
						Expect(p.State.ValidRound).ToNot(Equal(currentRound))

						return true
					}
					Expect(quick.Check(loop, nil)).To(Succeed())
				})
			})
		})

		Context("when the proposed value is not valid", func() {
			It("should do nothing", func() {
				loop := func() bool {
					currentHeight := process.Height(r.Int63())
					currentRound := process.Round(r.Int63())
					proposedValue := processutil.RandomValue(r)
					for proposedValue == process.NilValue {
						proposedValue = processutil.RandomValue(r)
					}
					whoami := id.NewPrivKey().Signatory()
					f := 5 + (r.Int() % 10)

					broadcaster := processutil.BroadcasterCallbacks{
						BroadcastProposeCallback: func(msg process.Propose) {
							// we dont expect any propose
							Expect(true).ToNot(BeTrue())
						},
						BroadcastPrevoteCallback: func(msg process.Prevote) {
							// the process will prevote nil (for the invalid propose)
							Expect(msg.Value).To(Equal(process.NilValue))
						},
						BroadcastPrecommitCallback: func(msg process.Precommit) {
							// we dont expect any prevotes
							Expect(true).ToNot(BeTrue())
						},
					}
					scheduledProposer := id.NewPrivKey().Signatory()
					scheduler := scheduler.NewRoundRobin([]id.Signatory{scheduledProposer})
					validator := processutil.MockValidator{MockValid: func(process.Value) bool { return false }}

					// instantiate a new process and its state
					p := process.New(whoami, f, nil, scheduler, nil, validator, broadcaster, nil, nil)
					p.State.CurrentHeight = currentHeight
					p.StartRound(currentRound)
					p.State.CurrentStep = process.Prevoting

					// feed 2f+1 prevotes
					for t := 0; t < 2*f+1; t++ {
						msg := randomValidPrevoteMsg(r, currentHeight, currentRound, proposedValue)
						p.Prevote(msg)
					}
					Expect(p.State.ValidValue).ToNot(Equal(proposedValue))
					Expect(p.State.ValidRound).ToNot(Equal(currentRound))

					// feed the propose message with the correct value, still do nothing
					validMsg := process.Propose{
						From:   scheduledProposer,
						Height: currentHeight,
						Round:  currentRound,
						Value:  proposedValue,
					}
					p.Propose(validMsg)
					Expect(p.State.ValidValue).ToNot(Equal(proposedValue))
					Expect(p.State.ValidRound).ToNot(Equal(currentRound))
					Expect(p.State.CurrentStep).To(Equal(process.Prevoting))

					return true
				}
				Expect(quick.Check(loop, nil)).To(Succeed())
			})
		})
	})

	// L44:
	//
	//  upon 2f+ 1〈PREVOTE, currentHeight, currentRound, nil〉
	//  while currentStep = prevote do
	//      broadcast〈PRECOMMIT, currentHeight, currentRound, nil〉
	//      currentStep ← precommit
	Context("when receiving 2f+1 nil prevotes", func() {
		r := rand.New(rand.NewSource(time.Now().UnixNano()))

		nilPrevoteMsg := func(
			r *rand.Rand,
			from id.Signatory,
			height process.Height,
			round process.Round,
		) process.Prevote {
			msg := processutil.RandomPrevote(r)

			msg.From = from
			msg.Height = height
			msg.Round = round
			msg.Value = process.NilValue

			return msg
		}

		Context("when we are in the prevote step", func() {
			It("should precommit nil and move to the precommitting step", func() {
				loop := func() bool {
					currentRound := processutil.RandomRound(r)
					for currentRound == process.InvalidRound {
						currentRound = processutil.RandomRound(r)
					}
					whoami := id.NewPrivKey().Signatory()
					acknowledge := false
					broadcaster := processutil.BroadcasterCallbacks{
						BroadcastProposeCallback: func(msg process.Propose) {
							// we expect to never receive propose broadcast
							Expect(true).ToNot(BeTrue())
						},
						BroadcastPrevoteCallback: func(msg process.Prevote) {
							// we expect to never receive prevote broadcast
							Expect(true).ToNot(BeTrue())
						},
						BroadcastPrecommitCallback: func(msg process.Precommit) {
							// the process precommits nil
							Expect(msg.From.Equal(&whoami)).To(BeTrue())
							Expect(msg.Value).To(Equal(process.NilValue))
							acknowledge = true
						},
					}
					f := 5 + (r.Int() % 10)
					p := process.New(whoami, f, nil, nil, nil, nil, broadcaster, nil, nil)
					p.StartRound(currentRound)

					// the process is in the Prevoting step
					p.State.CurrentStep = process.Prevoting

					// receive 2f nil prevotes, expect nothing to happen
					for t := 0; t < 2*f; t++ {
						msg := nilPrevoteMsg(r, id.NewPrivKey().Signatory(), process.Height(1), currentRound)
						p.Prevote(msg)
					}
					Expect(acknowledge).ToNot(BeTrue())

					// receive the 2f+1'th nil prevote, expect to broadcast a nil precommit
					// and step to precommitting
					msg := nilPrevoteMsg(r, id.NewPrivKey().Signatory(), process.Height(1), currentRound)
					p.Prevote(msg)

					Expect(p.State.CurrentStep).To(Equal(process.Precommitting))
					Expect(acknowledge).To(BeTrue())
					return true
				}
				Expect(quick.Check(loop, nil)).To(Succeed())
			})
		})

		Context("when we are not in the prevote step", func() {
			It("should do nothing", func() {
				loop := func() bool {
					currentRound := processutil.RandomRound(r)
					for currentRound == process.InvalidRound {
						currentRound = processutil.RandomRound(r)
					}
					whoami := id.NewPrivKey().Signatory()
					acknowledge := false
					broadcaster := processutil.BroadcasterCallbacks{
						BroadcastProposeCallback: func(msg process.Propose) {
							// we expect to never receive propose broadcast
							Expect(true).ToNot(BeTrue())
						},
						BroadcastPrevoteCallback: func(msg process.Prevote) {
							// we expect to never receive prevote broadcast
							Expect(true).ToNot(BeTrue())
						},
						BroadcastPrecommitCallback: func(msg process.Precommit) {
							// we expect to never receive prevote broadcast
							Expect(true).ToNot(BeTrue())
						},
					}
					f := 5 + (r.Int() % 10)
					p := process.New(whoami, f, nil, nil, nil, nil, broadcaster, nil, nil)
					p.StartRound(currentRound)

					// the process is NOT in the Prevoting step
					someOtherStep := processutil.RandomStep(r)
					for someOtherStep == process.Prevoting {
						someOtherStep = processutil.RandomStep(r)
					}
					p.State.CurrentStep = someOtherStep

					// receive 2f nil prevotes, expect nothing to happen
					for t := 0; t < 2*f; t++ {
						msg := nilPrevoteMsg(r, id.NewPrivKey().Signatory(), process.Height(1), currentRound)
						p.Prevote(msg)
					}
					Expect(acknowledge).ToNot(BeTrue())

					// receive the 2f+1'th nil prevote, still expect nothing to happen
					msg := nilPrevoteMsg(r, id.NewPrivKey().Signatory(), process.Height(1), currentRound)
					p.Prevote(msg)

					Expect(p.State.CurrentStep).To(Equal(someOtherStep))
					Expect(acknowledge).ToNot(BeTrue())
					return true
				}
				Expect(quick.Check(loop, nil)).To(Succeed())
			})
		})
	})

	// L47:
	//
	//  upon 2f+ 1〈PRECOMMIT, currentHeight, currentRound, ∗〉for the first time do
	//      scheduleOnTimeoutPrecommit(currentHeight, currentRound) to be executed after timeoutPrecommit(currentRound)
	Context("when receiving 2f+1 precommits", func() {
		r := rand.New(rand.NewSource(time.Now().UnixNano()))

		randomValidPrecommitMsg := func(
			r *rand.Rand,
			height process.Height,
			round process.Round,
		) process.Precommit {
			msg := processutil.RandomPrecommit(r)

			msg.Height = height
			msg.Round = round
			msg.From = id.NewPrivKey().Signatory()

			return msg
		}

		It("should schedule a precommit timeout for the current height and round", func() {
			loop := func() bool {
				// current round
				currentRound := processutil.RandomRound(r)
				for currentRound == process.InvalidRound {
					currentRound = processutil.RandomRound(r)
				}
				// parameters for the process
				f := 10 + (r.Int() % 5)
				timerOptions := timer.
					DefaultOptions().
					WithTimeout(1 * time.Millisecond).
					WithTimeoutScaling(0)
				onPrecommitTimeoutChan := make(chan timer.Timeout, 2)
				timer := timer.NewLinearTimer(timerOptions, nil, nil, onPrecommitTimeoutChan)

				// intantiate the process
				p := process.New(id.NewPrivKey().Signatory(), f, timer, nil, nil, nil, nil, nil, nil)
				p.StartRound(currentRound)

				// set the process to be at any random step
				p.State.CurrentStep = processutil.RandomStep(r)

				// send 2f precommit msgs on which we shouldn't expect any change
				for t := 0; t < 2*f; t++ {
					msg := randomValidPrecommitMsg(r, process.Height(1), currentRound)
					p.Precommit(msg)

					time.Sleep(5 * time.Millisecond)
					select {
					case _ = <-onPrecommitTimeoutChan:
						// this should never happen
						Expect(true).ToNot(BeTrue())
					default:
						Expect(true).To(BeTrue())
					}
				}

				// send the 2f+1'th precommit
				msg := randomValidPrecommitMsg(r, process.Height(1), currentRound)
				p.Precommit(msg)

				time.Sleep(5 * time.Millisecond)
				select {
				case timeout := <-onPrecommitTimeoutChan:
					Expect(timeout.Height).To(Equal(process.Height(1)))
					Expect(timeout.Round).To(Equal(currentRound))
				default:
					Expect(true).ToNot(BeTrue())
				}

				// sending further messages should not schedule another timeout
				// (once flags should prevent this)
				msg = randomValidPrecommitMsg(r, process.Height(1), currentRound)
				p.Precommit(msg)

				time.Sleep(5 * time.Millisecond)
				select {
				case _ = <-onPrecommitTimeoutChan:
					// this should never happen
					Expect(true).ToNot(BeTrue())
				default:
					Expect(true).To(BeTrue())
				}

				return true
			}
			Expect(quick.Check(loop, nil)).To(Succeed())
		})

		It("should not schedule a precommit timeout if some other height", func() {
			loop := func() bool {
				// current round
				currentRound := processutil.RandomRound(r)
				for currentRound == process.InvalidRound {
					currentRound = processutil.RandomRound(r)
				}
				// parameters for the process
				f := 10 + (r.Int() % 5)
				timerOptions := timer.
					DefaultOptions().
					WithTimeout(1 * time.Millisecond).
					WithTimeoutScaling(0)
				onPrecommitTimeoutChan := make(chan timer.Timeout, 2)
				timer := timer.NewLinearTimer(timerOptions, nil, nil, onPrecommitTimeoutChan)

				// intantiate the process
				p := process.New(id.NewPrivKey().Signatory(), f, timer, nil, nil, nil, nil, nil, nil)
				p.StartRound(currentRound)

				// set the process to be at any random step
				p.State.CurrentStep = processutil.RandomStep(r)

				// send 2f precommit msgs on which we shouldn't expect any change
				for t := 0; t < 2*f; t++ {
					msg := randomValidPrecommitMsg(r, processutil.RandomHeight(r), currentRound)
					p.Precommit(msg)

					time.Sleep(5 * time.Millisecond)
					select {
					case _ = <-onPrecommitTimeoutChan:
						// this should never happen
						Expect(true).ToNot(BeTrue())
					default:
						Expect(true).To(BeTrue())
					}
				}

				// send the 2f+1'th precommit
				msg := randomValidPrecommitMsg(r, processutil.RandomHeight(r), currentRound)
				p.Precommit(msg)

				time.Sleep(5 * time.Millisecond)
				select {
				case _ = <-onPrecommitTimeoutChan:
					// this hsould neveer happen
					Expect(true).ToNot(BeTrue())
				default:
					Expect(true).To(BeTrue())
				}

				return true
			}
			Expect(quick.Check(loop, nil)).To(Succeed())
		})

		It("should not schedule a precommit timeout if some other round", func() {
			loop := func() bool {
				// current round
				currentRound := processutil.RandomRound(r)
				for currentRound == process.InvalidRound {
					currentRound = processutil.RandomRound(r)
				}
				// parameters for the process
				f := 10 + (r.Int() % 5)
				timerOptions := timer.
					DefaultOptions().
					WithTimeout(1 * time.Millisecond).
					WithTimeoutScaling(0)
				onPrecommitTimeoutChan := make(chan timer.Timeout, 2)
				timer := timer.NewLinearTimer(timerOptions, nil, nil, onPrecommitTimeoutChan)

				// intantiate the process
				p := process.New(id.NewPrivKey().Signatory(), f, timer, nil, nil, nil, nil, nil, nil)
				p.StartRound(currentRound)

				// set the process to be at any random step
				p.State.CurrentStep = processutil.RandomStep(r)

				// send 2f precommit msgs on which we shouldn't expect any change
				for t := 0; t < 2*f; t++ {
					msg := randomValidPrecommitMsg(r, process.Height(1), processutil.RandomRound(r))
					p.Precommit(msg)

					time.Sleep(5 * time.Millisecond)
					select {
					case _ = <-onPrecommitTimeoutChan:
						// this should never happen
						Expect(true).ToNot(BeTrue())
					default:
						Expect(true).To(BeTrue())
					}
				}

				// send the 2f+1'th precommit
				msg := randomValidPrecommitMsg(r, process.Height(1), processutil.RandomRound(r))
				p.Precommit(msg)

				time.Sleep(5 * time.Millisecond)
				select {
				case _ = <-onPrecommitTimeoutChan:
					// this should never happen
					Expect(true).ToNot(BeTrue())
				default:
					Expect(true).To(BeTrue())
				}

				return true
			}
			Expect(quick.Check(loop, nil)).To(Succeed())
		})
	})

	// L49:
	//
	//  upon〈PROPOSAL, currentHeight, r, v, ∗〉from proposer(currentHeight, r) AND 2f+ 1〈PRECOMMIT, currentHeight, r, id(v)〉
	//  while decision[currentHeight] = nil do
	//      if valid(v) then
	//          decision[currentHeight] = v
	//          currentHeight ← currentHeight + 1
	//          reset
	//          StartRound(0)
	Context("when receiving a propose and 2f+1 precommits", func() {
		r := rand.New(rand.NewSource(time.Now().UnixNano()))

		randomValidPrecommitMsg := func(
			r *rand.Rand,
			height process.Height,
			round process.Round,
			value process.Value,
		) process.Precommit {
			msg := processutil.RandomPrecommit(r)

			msg.Height = height
			msg.Round = round
			msg.From = id.NewPrivKey().Signatory()
			msg.Value = value

			return msg
		}

		Context("when we have not finalised the given height", func() {
			Context("when the received propose value is valid", func() {
				// Should finalise the given height (by committing it)
				// Should increment the current height
				// Should start a new consensus round at round = 0
				//
				// A process without a scheduler/validator simply assumes that the
				// propose message is valid and from the scheduled proposer
				It("should finalise the given height", func() {
					loop := func() bool {
						currentHeight := process.Height(r.Int63())
						currentRound := process.Round(r.Int63())
						proposedValue := processutil.RandomValue(r)
						for proposedValue == process.NilValue {
							proposedValue = processutil.RandomValue(r)
						}
						whoami := id.NewPrivKey().Signatory()
						f := 5 + (r.Int() % 10)
						acknowledge := false
						committer := processutil.CommitterCallback{
							Callback: func(height process.Height, value process.Value) {
								Expect(height).To(Equal(currentHeight))
								Expect(value).To(Equal(proposedValue))
								acknowledge = true
							},
						}
						scheduledProposer := id.NewPrivKey().Signatory()
						scheduler := scheduler.NewRoundRobin([]id.Signatory{scheduledProposer})
						validator := processutil.MockValidator{MockValid: func(process.Value) bool { return true }}

						// instantiate a new process at the current round and height
						// and at any valid step
						p := process.New(whoami, f, nil, scheduler, nil, validator, nil, committer, nil)
						p.StartRound(currentRound)
						p.State.CurrentHeight = currentHeight
						p.State.CurrentStep = process.Step(r.Int() % 3)

						// feed the process with 2f+1 precommit messages
						// we expect nothing to happen
						for t := 0; t < 2*f+1; t++ {
							msg := randomValidPrecommitMsg(r, currentHeight, currentRound, proposedValue)
							p.Precommit(msg)
						}
						Expect(p.State.CurrentHeight).To(Equal(currentHeight))
						Expect(p.State.CurrentRound).To(Equal(currentRound))
						Expect(acknowledge).ToNot(BeTrue())

						// feed the process with a propose message
						msg := process.Propose{
							Height:     currentHeight,
							Round:      currentRound,
							ValidRound: processutil.RandomRound(r),
							Value:      proposedValue,
							From:       scheduledProposer,
						}
						p.Propose(msg)

						defaultState := process.DefaultState()
						Expect(p.State.CurrentHeight).To(Equal(currentHeight + 1))
						Expect(p.CurrentRound).To(Equal(process.Round(0)))
						Expect(p.State.CurrentStep).To(Equal(defaultState.CurrentStep))
						Expect(p.State.LockedRound).To(Equal(defaultState.LockedRound))
						Expect(p.State.LockedValue).To(Equal(defaultState.LockedValue))
						Expect(p.State.ValidValue).To(Equal(defaultState.ValidValue))
						Expect(p.State.ValidRound).To(Equal(defaultState.ValidRound))
						Expect(acknowledge).To(BeTrue())
						return true
					}
					Expect(quick.Check(loop, nil)).To(Succeed())
				})

				It("should finalise the given height (without scheduler or validator)", func() {
					loop := func() bool {
						currentHeight := process.Height(r.Int63())
						currentRound := process.Round(r.Int63())
						proposedValue := processutil.RandomValue(r)
						for proposedValue == process.NilValue {
							proposedValue = processutil.RandomValue(r)
						}
						whoami := id.NewPrivKey().Signatory()
						f := 5 + (r.Int() % 10)
						acknowledge := false
						committer := processutil.CommitterCallback{
							Callback: func(height process.Height, value process.Value) {
								Expect(height).To(Equal(currentHeight))
								Expect(value).To(Equal(proposedValue))
								acknowledge = true
							},
						}

						// instantiate a new process at the current round and height
						// and at any valid step
						p := process.New(whoami, f, nil, nil, nil, nil, nil, committer, nil)
						p.StartRound(currentRound)
						p.State.CurrentHeight = currentHeight
						p.State.CurrentStep = process.Step(r.Int() % 3)

						// feed the process with 2f+1 precommit messages
						// we expect nothing to happen
						for t := 0; t < 2*f+1; t++ {
							msg := randomValidPrecommitMsg(r, currentHeight, currentRound, proposedValue)
							p.Precommit(msg)
						}
						Expect(p.State.CurrentHeight).To(Equal(currentHeight))
						Expect(p.State.CurrentRound).To(Equal(currentRound))
						Expect(acknowledge).ToNot(BeTrue())

						// feed the process with a propose message
						msg := process.Propose{
							Height:     currentHeight,
							Round:      currentRound,
							ValidRound: processutil.RandomRound(r),
							Value:      proposedValue,
							From:       id.NewPrivKey().Signatory(),
						}
						p.Propose(msg)

						defaultState := process.DefaultState()
						Expect(p.State.CurrentHeight).To(Equal(currentHeight + 1))
						Expect(p.CurrentRound).To(Equal(process.Round(0)))
						Expect(p.State.CurrentStep).To(Equal(defaultState.CurrentStep))
						Expect(p.State.LockedRound).To(Equal(defaultState.LockedRound))
						Expect(p.State.LockedValue).To(Equal(defaultState.LockedValue))
						Expect(p.State.ValidValue).To(Equal(defaultState.ValidValue))
						Expect(p.State.ValidRound).To(Equal(defaultState.ValidRound))
						Expect(acknowledge).To(BeTrue())
						return true
					}
					Expect(quick.Check(loop, nil)).To(Succeed())
				})
			})

			Context("when the 2f+1 precommits are not all towards the same value", func() {
				It("should do nothing", func() {
					loop := func() bool {
						currentHeight := process.Height(r.Int63())
						currentRound := process.Round(r.Int63())
						proposedValue := processutil.RandomValue(r)
						for proposedValue == process.NilValue {
							proposedValue = processutil.RandomValue(r)
						}
						whoami := id.NewPrivKey().Signatory()
						f := 5 + (r.Int() % 10)
						committer := processutil.CommitterCallback{
							Callback: func(height process.Height, value process.Value) {
								// the process should never broadcast a commit
								Expect(true).ToNot(BeTrue())
							},
						}
						broadcaster := processutil.BroadcasterCallbacks{
							// the process should not broadcast any message
							BroadcastProposeCallback: func(msg process.Propose) {
								Expect(true).ToNot(BeTrue())
							},
							BroadcastPrecommitCallback: func(msg process.Precommit) {
								Expect(true).ToNot(BeTrue())
							},
						}

						// instantiate a new process at the current round and height
						// and at any valid step
						p := process.New(whoami, f, nil, nil, nil, nil, broadcaster, committer, nil)
						p.StartRound(currentRound)
						p.State.CurrentHeight = currentHeight
						p.State.CurrentStep = process.Step(r.Int() % 3)

						// feed the process with 2f+1 precommit messages
						// we expect nothing to happen
						for t := 0; t < 2*f+1; t++ {
							// precommit either the proposed value, or a random value
							if r.Int()%3 == 0 {
								msg := randomValidPrecommitMsg(r, currentHeight, currentRound, proposedValue)
								p.Precommit(msg)
							} else {
								msg := randomValidPrecommitMsg(r, currentHeight, currentRound, processutil.RandomValue(r))
								p.Precommit(msg)
							}
						}
						Expect(p.State.CurrentHeight).To(Equal(currentHeight))
						Expect(p.State.CurrentRound).To(Equal(currentRound))

						// feed the process with a propose message
						msg := process.Propose{
							Height:     currentHeight,
							Round:      currentRound,
							ValidRound: processutil.RandomRound(r),
							Value:      proposedValue,
							From:       id.NewPrivKey().Signatory(),
						}
						p.Propose(msg)

						Expect(p.State.CurrentHeight).To(Equal(currentHeight))
						Expect(p.CurrentRound).To(Equal(currentRound))
						return true
					}
					Expect(quick.Check(loop, nil)).To(Succeed())
				})
			})

			Context("when the received propose is not valid", func() {
				It("should do nothing (invalid proposer)", func() {
					loop := func() bool {
						currentHeight := process.Height(r.Int63())
						currentRound := process.Round(r.Int63())
						proposedValue := processutil.RandomValue(r)
						for proposedValue == process.NilValue {
							proposedValue = processutil.RandomValue(r)
						}
						whoami := id.NewPrivKey().Signatory()
						f := 5 + (r.Int() % 10)
						committer := processutil.CommitterCallback{
							Callback: func(height process.Height, value process.Value) {
								// the process should never broadcast the commit
								Expect(true).ToNot(BeTrue())
							},
						}
						scheduler := scheduler.NewRoundRobin([]id.Signatory{id.NewPrivKey().Signatory()})

						// instantiate a new process at the current round and height
						// and at any valid step
						p := process.New(whoami, f, nil, scheduler, nil, nil, nil, committer, nil)
						p.StartRound(currentRound)
						p.State.CurrentHeight = currentHeight
						p.State.CurrentStep = process.Step(r.Int() % 3)

						// feed the process with 2f+1 precommit messages
						// we expect nothing to happen
						for t := 0; t < 2*f+1; t++ {
							msg := randomValidPrecommitMsg(r, currentHeight, currentRound, proposedValue)
							p.Precommit(msg)
						}
						Expect(p.State.CurrentHeight).To(Equal(currentHeight))
						Expect(p.State.CurrentRound).To(Equal(currentRound))

						// feed the process with a propose message
						msg := process.Propose{
							Height:     currentHeight,
							Round:      currentRound,
							ValidRound: processutil.RandomRound(r),
							Value:      proposedValue,
							From:       id.NewPrivKey().Signatory(),
						}
						p.Propose(msg)

						Expect(p.State.CurrentHeight).To(Equal(currentHeight))
						Expect(p.CurrentRound).To(Equal(currentRound))
						return true
					}
					Expect(quick.Check(loop, nil)).To(Succeed())
				})

				It("should do nothing (invalid value)", func() {
					loop := func() bool {
						currentHeight := process.Height(r.Int63())
						currentRound := process.Round(r.Int63())
						proposedValue := processutil.RandomValue(r)
						for proposedValue == process.NilValue {
							proposedValue = processutil.RandomValue(r)
						}
						whoami := id.NewPrivKey().Signatory()
						f := 5 + (r.Int() % 10)
						committer := processutil.CommitterCallback{
							Callback: func(height process.Height, value process.Value) {
								// the process should never broadcast the commit
								Expect(true).ToNot(BeTrue())
							},
						}
						validator := processutil.MockValidator{MockValid: func(process.Value) bool { return false }}

						// instantiate a new process at the current round and height
						// and at any valid step
						p := process.New(whoami, f, nil, nil, nil, validator, nil, committer, nil)
						p.StartRound(currentRound)
						p.State.CurrentHeight = currentHeight
						p.State.CurrentStep = process.Step(r.Int() % 3)

						// feed the process with 2f+1 precommit messages
						// we expect nothing to happen
						for t := 0; t < 2*f+1; t++ {
							msg := randomValidPrecommitMsg(r, currentHeight, currentRound, proposedValue)
							p.Precommit(msg)
						}
						Expect(p.State.CurrentHeight).To(Equal(currentHeight))
						Expect(p.State.CurrentRound).To(Equal(currentRound))

						// feed the process with a propose message
						msg := process.Propose{
							Height:     currentHeight,
							Round:      currentRound,
							ValidRound: processutil.RandomRound(r),
							Value:      proposedValue,
							From:       id.NewPrivKey().Signatory(),
						}
						p.Propose(msg)

						Expect(p.State.CurrentHeight).To(Equal(currentHeight))
						Expect(p.CurrentRound).To(Equal(currentRound))
						return true
					}
					Expect(quick.Check(loop, nil)).To(Succeed())
				})
			})
		})
	})

	// L55:
	//
	//  upon f+ 1〈∗, currentHeight, r, ∗, ∗〉with r > currentRound do
	//      StartRound(r)
	Context("when receiving f+1 messages from a future round", func() {
		r := rand.New(rand.NewSource(time.Now().UnixNano()))

		randomValidPrevote := func(
			r *rand.Rand,
			height process.Height,
			round process.Round,
		) process.Prevote {
			msg := processutil.RandomPrevote(r)
			msg.From = id.NewPrivKey().Signatory()
			msg.Height = height
			msg.Round = round
			return msg
		}

		randomValidPrecommit := func(
			r *rand.Rand,
			height process.Height,
			round process.Round,
		) process.Precommit {
			msg := processutil.RandomPrecommit(r)
			msg.From = id.NewPrivKey().Signatory()
			msg.Height = height
			msg.Round = round
			return msg
		}

		It("should start a new round set as the given future round", func() {
			loop := func() bool {
				currentHeight := process.Height(r.Int63())
				currentRound := process.Round(r.Int63())
				whoami := id.NewPrivKey().Signatory()
				f := 5 + (r.Int() % 10)

				// instantiate a new process
				p := process.New(whoami, f, nil, nil, nil, nil, nil, nil, nil)
				p.StartRound(currentRound)
				p.State.CurrentHeight = currentHeight

				// feed with f+1 random messages
				futureRound := currentRound + 1 + process.Round(r.Int()%10)
				for t := 0; t < f+1; t++ {
					switch r.Int() % 2 {
					case 0:
						msg := randomValidPrevote(r, currentHeight, futureRound)
						p.Prevote(msg)
					case 1:
						msg := randomValidPrecommit(r, currentHeight, futureRound)
						p.Precommit(msg)
					default:
						panic("this should never happen")
					}
				}

				// the process should have moved to the future round
				Expect(p.State.CurrentRound).To(Equal(futureRound))

				return true
			}
			Expect(quick.Check(loop, nil)).To(Succeed())
		})

		It("(only single propose is considered) should start a new round set as the given future round", func() {
			loop := func() bool {
				currentHeight := process.Height(r.Int63())
				currentRound := process.Round(r.Int63())
				whoami := id.NewPrivKey().Signatory()
				f := 5 + (r.Int() % 10)

				// instantiate a new process
				p := process.New(whoami, f, nil, nil, nil, nil, nil, nil, nil)
				p.StartRound(currentRound)
				p.State.CurrentHeight = currentHeight

				// feed with f-1 random messages
				futureRound := currentRound + 1 + process.Round(r.Int()%10)
				for t := 0; t < f-1; t++ {
					switch r.Int() % 2 {
					case 0:
						msg := randomValidPrevote(r, currentHeight, futureRound)
						p.Prevote(msg)
					case 1:
						msg := randomValidPrecommit(r, currentHeight, futureRound)
						p.Precommit(msg)
					default:
						panic("this should never happen")
					}
				}

				// nothing should have happened
				Expect(p.State.CurrentRound).To(Equal(currentRound))

				// feed with a few propose messages
				tMax := 1 + r.Int()%10
				for t := 0; t < tMax; t++ {
					msg := process.Propose{
						From:   id.NewPrivKey().Signatory(),
						Value:  processutil.RandomValue(r),
						Height: currentHeight,
						Round:  futureRound,
					}
					p.Propose(msg)
				}

				// nothing should have happened
				Expect(p.State.CurrentRound).To(Equal(currentRound))

				// send one more random prevote/precommit
				switch r.Int() % 2 {
				case 0:
					msg := randomValidPrevote(r, currentHeight, futureRound)
					p.Prevote(msg)
				case 1:
					msg := randomValidPrecommit(r, currentHeight, futureRound)
					p.Precommit(msg)
				default:
					panic("this should never happen")
				}

				// the process should have moved to the future round
				Expect(p.State.CurrentRound).To(Equal(futureRound))

				return true
			}
			Expect(quick.Check(loop, nil)).To(Succeed())
		})

		It("[only prevote] should start a new round set as the given future round", func() {
			loop := func() bool {
				currentHeight := process.Height(r.Int63())
				currentRound := process.Round(r.Int63())
				whoami := id.NewPrivKey().Signatory()
				f := 5 + (r.Int() % 10)

				// instantiate a new process
				p := process.New(whoami, f, nil, nil, nil, nil, nil, nil, nil)
				p.StartRound(currentRound)
				p.State.CurrentHeight = currentHeight

				// feed with f+1 random messages
				futureRound := currentRound + 1 + process.Round(r.Int()%10)
				for t := 0; t < f+1; t++ {
					msg := randomValidPrevote(r, currentHeight, futureRound)
					p.Prevote(msg)
				}

				// the process should have moved to the future round
				Expect(p.State.CurrentRound).To(Equal(futureRound))

				return true
			}
			Expect(quick.Check(loop, nil)).To(Succeed())
		})

		It("[only precommit] should start a new round set as the given future round", func() {
			loop := func() bool {
				currentHeight := process.Height(r.Int63())
				currentRound := process.Round(r.Int63())
				whoami := id.NewPrivKey().Signatory()
				f := 5 + (r.Int() % 10)

				// instantiate a new process
				p := process.New(whoami, f, nil, nil, nil, nil, nil, nil, nil)
				p.StartRound(currentRound)
				p.State.CurrentHeight = currentHeight

				// feed with f+1 random messages
				futureRound := currentRound + 1 + process.Round(r.Int()%10)
				for t := 0; t < f+1; t++ {
					msg := randomValidPrecommit(r, currentHeight, futureRound)
					p.Precommit(msg)
				}

				// the process should have moved to the future round
				Expect(p.State.CurrentRound).To(Equal(futureRound))

				return true
			}
			Expect(quick.Check(loop, nil)).To(Succeed())
		})

		It("should do nothing if the round is not a future round", func() {
			loop := func() bool {
				currentHeight := process.Height(r.Int63())
				currentRound := process.Round(r.Int63())
				whoami := id.NewPrivKey().Signatory()
				f := 5 + (r.Int() % 10)

				// instantiate a new process
				p := process.New(whoami, f, nil, nil, nil, nil, nil, nil, nil)
				p.StartRound(currentRound)
				p.State.CurrentHeight = currentHeight

				// feed with f+1 random messages
				pastOrCurrentRound := currentRound - process.Round(r.Int()%3)
				for t := 0; t < f+1; t++ {
					switch r.Int() % 2 {
					case 0:
						msg := randomValidPrevote(r, currentHeight, pastOrCurrentRound)
						p.Prevote(msg)
					case 1:
						msg := randomValidPrecommit(r, currentHeight, pastOrCurrentRound)
						p.Precommit(msg)
					default:
						panic("this should never happen")
					}
				}

				// nothing should happen
				Expect(p.State.CurrentRound).To(Equal(currentRound))

				return true
			}
			Expect(quick.Check(loop, nil)).To(Succeed())
		})

		It("should do nothing if the height is not the current height", func() {
			loop := func() bool {
				currentHeight := process.Height(r.Int63())
				currentRound := process.Round(r.Int63())
				whoami := id.NewPrivKey().Signatory()
				f := 5 + (r.Int() % 10)

				// instantiate a new process
				p := process.New(whoami, f, nil, nil, nil, nil, nil, nil, nil)
				p.StartRound(currentRound)
				p.State.CurrentHeight = currentHeight

				// feed with f+1 random messages
				futureRound := currentRound + 1 + process.Round(r.Int()%10)
				someOtherHeight := processutil.RandomHeight(r)
				for someOtherHeight == currentHeight {
					someOtherHeight = processutil.RandomHeight(r)
				}
				for t := 0; t < f+1; t++ {
					switch r.Int() % 2 {
					case 0:
						msg := randomValidPrevote(r, someOtherHeight, futureRound)
						p.Prevote(msg)
					case 1:
						msg := randomValidPrecommit(r, someOtherHeight, futureRound)
						p.Precommit(msg)
					default:
						panic("this should never happen")
					}
				}

				// nothing should happen
				Expect(p.State.CurrentRound).To(Equal(currentRound))

				return true
			}
			Expect(quick.Check(loop, nil)).To(Succeed())
		})
	})

	// Catcher
	Context("when receiving two different messages from the same process", func() {
		r := rand.New(rand.NewSource(time.Now().UnixNano()))

		Context("when receiving two different proposes", func() {
			It("should catch the double propose", func() {
				loop := func() bool {
					round := processutil.RandomRound(r)
					for round == process.InvalidRound {
						round = processutil.RandomRound(r)
					}

					whoami := id.NewPrivKey().Signatory()
					doubleSender := id.NewPrivKey().Signatory()
					acknowledge := false
					catcher := processutil.CatcherCallbacks{
						CatchDoubleProposeCallback: func(propose1 process.Propose, propose2 process.Propose) {
							Expect(propose1.From.Equal(&doubleSender)).To(BeTrue())
							Expect(propose2.From.Equal(&doubleSender)).To(BeTrue())
							acknowledge = true
						},
						CatchDoublePrevoteCallback: func(prevote1 process.Prevote, prevote2 process.Prevote) {
							// this should never happen
							Expect(true).ToNot(BeTrue())
						},
						CatchDoublePrecommitCallback: func(precommit1 process.Precommit, precommit2 process.Precommit) {
							// this should never happen
							Expect(true).ToNot(BeTrue())
						},
						CatchOutOfTurnProposeCallback: func(propose process.Propose) {
							// this should never happen
							Expect(true).ToNot(BeTrue())
						},
					}
					scheduler := scheduler.NewRoundRobin([]id.Signatory{doubleSender})
					p := process.New(whoami, 33, nil, scheduler, nil, nil, nil, nil, catcher)
					p.StartRound(round)

					// receive the first propose msg
					firstValue := processutil.RandomValue(r)
					propose1 := process.Propose{
						From:   doubleSender,
						Height: process.Height(1),
						Round:  round,
						Value:  firstValue,
					}
					p.Propose(propose1)
					Expect(acknowledge).ToNot(BeTrue())

					// if the same propose is sent twice, it is not caught as a double
					p.Propose(process.Propose{
						From:   doubleSender,
						Height: process.Height(1),
						Round:  round,
						Value:  firstValue,
					})
					Expect(acknowledge).ToNot(BeTrue())

					// on second propose msg (different), catch as double
					secondValue := processutil.RandomValue(r)
					for secondValue == firstValue {
						secondValue = processutil.RandomValue(r)
					}
					propose2 := process.Propose{
						From:   doubleSender,
						Height: process.Height(1),
						Round:  round,
						Value:  secondValue,
					}
					p.Propose(propose2)
					Expect(acknowledge).To(BeTrue())

					return true
				}
				Expect(quick.Check(loop, nil)).To(Succeed())
			})
		})

		Context("when receiving two different prevotes", func() {
			It("should catch the double prevote", func() {
				loop := func() bool {
					round := processutil.RandomRound(r)
					for round == process.InvalidRound {
						round = processutil.RandomRound(r)
					}

					whoami := id.NewPrivKey().Signatory()
					doubleSender := id.NewPrivKey().Signatory()
					acknowledge := false
					catcher := processutil.CatcherCallbacks{
						CatchDoubleProposeCallback: func(propose1 process.Propose, propose2 process.Propose) {
							// this should never happen
							Expect(true).ToNot(BeTrue())
						},
						CatchDoublePrevoteCallback: func(prevote1 process.Prevote, prevote2 process.Prevote) {
							Expect(prevote1.From.Equal(&doubleSender)).To(BeTrue())
							Expect(prevote2.From.Equal(&doubleSender)).To(BeTrue())
							acknowledge = true
						},
						CatchDoublePrecommitCallback: func(precommit1 process.Precommit, precommit2 process.Precommit) {
							// this should never happen
							Expect(true).ToNot(BeTrue())
						},
						CatchOutOfTurnProposeCallback: func(propose process.Propose) {
							// this should never happen
							Expect(true).ToNot(BeTrue())
						},
					}
					p := process.New(whoami, 33, nil, nil, nil, nil, nil, nil, catcher)
					p.StartRound(round)
					p.State.CurrentStep = process.Prevoting

					// receive the first prevote msg
					firstValue := processutil.RandomValue(r)
					prevote1 := process.Prevote{
						From:   doubleSender,
						Height: process.Height(1),
						Round:  round,
						Value:  firstValue,
					}
					p.Prevote(prevote1)
					Expect(acknowledge).ToNot(BeTrue())

					// receive another prevote msg, from a random process
					// expect no double catches since they are two different sources
					prevote2 := process.Prevote{
						From:   id.NewPrivKey().Signatory(),
						Height: process.Height(1),
						Round:  round,
						Value:  firstValue,
					}
					p.Prevote(prevote2)
					Expect(acknowledge).ToNot(BeTrue())

					// on second prevote msg, but the same prevote
					// don't catch as double (as its the same prevote)
					p.Prevote(process.Prevote{
						From:   doubleSender,
						Height: process.Height(1),
						Round:  round,
						Value:  firstValue,
					})
					Expect(acknowledge).ToNot(BeTrue())

					// on second prevote msg, catch as double if the second value
					// is different than the first
					secondValue := processutil.RandomValue(r)
					for secondValue == firstValue {
						secondValue = processutil.RandomValue(r)
					}
					prevote3 := process.Prevote{
						From:   doubleSender,
						Height: process.Height(1),
						Round:  round,
						Value:  secondValue,
					}
					p.Prevote(prevote3)
					Expect(acknowledge).To(BeTrue())

					return true
				}
				Expect(quick.Check(loop, nil)).To(Succeed())
			})
		})

		Context("when receiving two different precommits", func() {
			It("should catch the double precommit", func() {
				loop := func() bool {
					round := processutil.RandomRound(r)
					for round == process.InvalidRound {
						round = processutil.RandomRound(r)
					}

					whoami := id.NewPrivKey().Signatory()
					doubleSender := id.NewPrivKey().Signatory()
					acknowledge := false
					catcher := processutil.CatcherCallbacks{
						CatchDoubleProposeCallback: func(propose1 process.Propose, propose2 process.Propose) {
							// this should never happen
							Expect(true).ToNot(BeTrue())
						},
						CatchDoublePrevoteCallback: func(prevote1 process.Prevote, prevote2 process.Prevote) {
							// this should never happen
							Expect(true).ToNot(BeTrue())
						},
						CatchDoublePrecommitCallback: func(precommit1 process.Precommit, precommit2 process.Precommit) {
							Expect(precommit1.From.Equal(&doubleSender)).To(BeTrue())
							Expect(precommit2.From.Equal(&doubleSender)).To(BeTrue())
							acknowledge = true
						},
						CatchOutOfTurnProposeCallback: func(propose process.Propose) {
							// this should never happen
							Expect(true).ToNot(BeTrue())
						},
					}
					p := process.New(whoami, 33, nil, nil, nil, nil, nil, nil, catcher)
					p.StartRound(round)
					p.State.CurrentStep = process.Precommitting

					// receive the first precommit msg
					firstValue := processutil.RandomValue(r)
					precommit1 := process.Precommit{
						From:   doubleSender,
						Height: process.Height(1),
						Round:  round,
						Value:  firstValue,
					}
					p.Precommit(precommit1)
					Expect(acknowledge).ToNot(BeTrue())

					// receive another precommit msg, from a random process
					// expect no double catches since they are two different sources
					precommit2 := process.Precommit{
						From:   id.NewPrivKey().Signatory(),
						Height: process.Height(1),
						Round:  round,
						Value:  firstValue,
					}
					p.Precommit(precommit2)
					Expect(acknowledge).ToNot(BeTrue())

					// on second precommit msg, but the same precommit
					// don't catch as double (as its the same precommit)
					p.Precommit(process.Precommit{
						From:   doubleSender,
						Height: process.Height(1),
						Round:  round,
						Value:  firstValue,
					})
					Expect(acknowledge).ToNot(BeTrue())

					// on second precommit msg, catch as double if the second value
					// is different than the first
					secondValue := processutil.RandomValue(r)
					for secondValue == firstValue {
						secondValue = processutil.RandomValue(r)
					}
					precommit3 := process.Precommit{
						From:   doubleSender,
						Height: process.Height(1),
						Round:  round,
						Value:  secondValue,
					}
					p.Precommit(precommit3)
					Expect(acknowledge).To(BeTrue())

					return true
				}
				Expect(quick.Check(loop, nil)).To(Succeed())
			})
		})

		Context("when receiving out of turn propose", func() {
			It("should catch the out of turn propose", func() {
				loop := func() bool {
					round := processutil.RandomRound(r)
					for round == process.InvalidRound {
						round = processutil.RandomRound(r)
					}

					whoami := id.NewPrivKey().Signatory()
					scheduledSender := id.NewPrivKey().Signatory()
					outOfTurnSender := id.NewPrivKey().Signatory()
					acknowledge := false
					catcher := processutil.CatcherCallbacks{
						CatchDoubleProposeCallback: func(propose1 process.Propose, propose2 process.Propose) {
							// this should never happen
							Expect(true).ToNot(BeTrue())
						},
						CatchDoublePrevoteCallback: func(prevote1 process.Prevote, prevote2 process.Prevote) {
							// this should never happen
							Expect(true).ToNot(BeTrue())
						},
						CatchDoublePrecommitCallback: func(precommit1 process.Precommit, precommit2 process.Precommit) {
							// this should never happen
							Expect(true).ToNot(BeTrue())
						},
						CatchOutOfTurnProposeCallback: func(propose process.Propose) {
							Expect(propose.From.Equal(&outOfTurnSender)).To(BeTrue())
							acknowledge = true
						},
					}
					scheduler := scheduler.NewRoundRobin([]id.Signatory{scheduledSender})
					p := process.New(whoami, 33, nil, scheduler, nil, nil, nil, nil, catcher)
					p.StartRound(round)

					// receive propose msg from the scheduled sender
					propose1 := process.Propose{
						From:   scheduledSender,
						Height: process.Height(1),
						Round:  round,
						Value:  processutil.RandomValue(r),
					}
					p.Propose(propose1)
					Expect(acknowledge).ToNot(BeTrue())

					// receive propose from out of turn sender, must catch
					propose2 := process.Propose{
						From:   outOfTurnSender,
						Height: process.Height(1),
						Round:  round,
						Value:  processutil.RandomValue(r),
					}
					p.Propose(propose2)
					Expect(acknowledge).To(BeTrue())

					return true
				}
				Expect(quick.Check(loop, nil)).To(Succeed())
			})
		})
	})
})
