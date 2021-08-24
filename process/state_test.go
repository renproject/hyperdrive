package process_test

import (
	"math/rand"
	"testing/quick"
	"time"

	"github.com/renproject/hyperdrive/process"
	"github.com/renproject/hyperdrive/process/processutil"
	"github.com/renproject/id"
	"github.com/renproject/surge"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("State", func() {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))

	Context("when unmarshaling fuzz", func() {
		It("should not panic", func() {
			f := func(fuzz []byte) bool {
				msg := process.State{}
				Expect(surge.FromBinary(&msg, fuzz)).ToNot(Succeed())
				return true
			}
			Expect(quick.Check(f, nil)).To(Succeed())
		})
	})

	Context("when marshaling and then unmarshaling", func() {
		It("should equal itself", func() {
			f := func(currentHeight process.Height, currentRound process.Round, currentStep process.Step, lockedRound process.Round, lockedValue process.Value, validRound process.Round, validValue process.Value, proposeLogs map[process.Round]process.Propose, prevoteLogs map[process.Round]map[id.Signatory]process.Prevote, precommitLogs map[process.Round]map[id.Signatory]process.Precommit, onceFlags map[process.Round]process.OnceFlag) bool {
				expected := process.State{
					CurrentHeight: currentHeight,
					CurrentRound:  currentRound,
					CurrentStep:   currentStep,
					LockedRound:   lockedRound,
					LockedValue:   lockedValue,
					ValidRound:    validRound,
					ValidValue:    validValue,

					ProposeLogs:   proposeLogs,
					PrevoteLogs:   prevoteLogs,
					PrecommitLogs: precommitLogs,
					OnceFlags:     onceFlags,
				}
				data, err := surge.ToBinary(expected)
				Expect(err).ToNot(HaveOccurred())
				got := process.State{}
				err = surge.FromBinary(&got, data)
				Expect(err).ToNot(HaveOccurred())
				Expect(got.Equal(&expected)).To(BeTrue())
				return true
			}
			Expect(quick.Check(f, nil)).To(Succeed())
		})

		It("should return an error when not enough bytes (marshaling)", func() {
			loop := func() bool {
				expected := processutil.RandomState(r)
				sizeAvailable := r.Intn(expected.SizeHint())
				buf := make([]byte, sizeAvailable)
				_, _, err := expected.Marshal(buf, sizeAvailable)
				Expect(err).To(HaveOccurred())

				return true
			}
			Expect(quick.Check(loop, nil)).To(Succeed())
		})

		It("should return an error when not enough bytes (unmarshaling)", func() {
			loop := func() bool {
				expected := processutil.RandomState(r)
				sizeHint := expected.SizeHint()
				buf := make([]byte, sizeHint)
				_, _, err := expected.Marshal(buf, sizeHint)
				Expect(err).ToNot(HaveOccurred())

				var unmarshalled process.State
				sizeAvailable := r.Intn(sizeHint)
				_, _, err = unmarshalled.Unmarshal(buf, sizeAvailable)
				Expect(err).To(HaveOccurred())

				return true
			}
			Expect(quick.Check(loop, nil)).To(Succeed())
		})
	})

	Context("when initialising the default state", func() {
		It("should have height=1", func() {
			Expect(process.DefaultState().CurrentHeight).To(Equal(process.Height(1)))
		})

		It("should have round=0", func() {
			Expect(process.DefaultState().CurrentRound).To(Equal(process.Round(0)))
		})

		It("should have step=proposing", func() {
			Expect(process.DefaultState().CurrentStep).To(Equal(process.Proposing))
		})

		It("should have locked round=invalid", func() {
			Expect(process.DefaultState().LockedRound).To(Equal(process.InvalidRound))
		})

		It("should have locked value=nil", func() {
			Expect(process.DefaultState().LockedValue).To(Equal(process.NilValue))
		})

		It("should have valid round=invalid", func() {
			Expect(process.DefaultState().ValidRound).To(Equal(process.InvalidRound))
		})

		It("should have valid value=nil", func() {
			Expect(process.DefaultState().ValidValue).To(Equal(process.NilValue))
		})
	})

	Context("when cloned", func() {
		It("should clone correctly", func() {
			loop := func() bool {
				original := processutil.RandomState(r)
				duplicate := original.Clone()
				Expect(duplicate.Equal(&original)).To(BeTrue())

				return true
			}
			Expect(quick.Check(loop, nil)).To(Succeed())
		})
	})
})
