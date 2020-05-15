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

var _ = Describe("Process", func() {
	Context("when unmarshaling fuzz", func() {
		It("should not panic", func() {
			f := func(fuzz []byte) bool {
				msg := process.Process{}
				Expect(surge.FromBinary(fuzz, &msg)).ToNot(Succeed())
				return true
			}
			Expect(quick.Check(f, nil)).To(Succeed())
		})
	})

	Context("when marshaling and then unmarshaling", func() {
		It("should equal itself", func() {
			r := rand.New(rand.NewSource(time.Now().UnixNano()))
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
				err = surge.FromBinary(data, &got)
				Expect(err).ToNot(HaveOccurred())
				Expect(got.State.Equal(&expected.State)).To(BeTrue())
				return true
			}
			Expect(quick.Check(f, nil)).To(Succeed())
		})
	})
})
