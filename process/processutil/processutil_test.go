package processutil_test

import (
	"math/rand"
	"time"

	"github.com/renproject/hyperdrive/process"
	"github.com/renproject/hyperdrive/process/processutil"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Process utilities", func() {
	Context("when generating random proposes", func() {
		It("should not generate the same propose multiple times", func() {
			r := rand.New(rand.NewSource(time.Now().UnixNano()))
			proposes := make([]process.Propose, 100)
			for i := range proposes {
				proposes[i] = processutil.RandomPropose(r)
			}
			all := true
			for i := range proposes {
				if !proposes[0].Equal(&proposes[i]) {
					all = false
					break
				}
			}
			Expect(all).To(BeFalse())
		})
	})

	Context("when generating random prevotes", func() {
		It("should not generate the same prevote multiple times", func() {
			r := rand.New(rand.NewSource(time.Now().UnixNano()))
			prevotes := make([]process.Prevote, 100)
			for i := range prevotes {
				prevotes[i] = processutil.RandomPrevote(r)
			}
			all := true
			for i := range prevotes {
				if !prevotes[0].Equal(&prevotes[i]) {
					all = false
					break
				}
			}
			Expect(all).To(BeFalse())
		})
	})

	Context("when generating random precommits", func() {
		It("should not generate the same precommit multiple times", func() {
			r := rand.New(rand.NewSource(time.Now().UnixNano()))
			precommits := make([]process.Precommit, 100)
			for i := range precommits {
				precommits[i] = processutil.RandomPrecommit(r)
			}
			all := true
			for i := range precommits {
				if !precommits[0].Equal(&precommits[i]) {
					all = false
					break
				}
			}
			Expect(all).To(BeFalse())
		})
	})

	Context("when generating random states", func() {
		It("should not generate the same state multiple times", func() {
			r := rand.New(rand.NewSource(time.Now().UnixNano()))
			states := make([]process.State, 100)
			for i := range states {
				states[i] = processutil.RandomState(r)
			}
			all := true
			for i := range states {
				if !states[0].Equal(&states[i]) {
					all = false
					break
				}
			}
			Expect(all).To(BeFalse())
		})
	})
})
