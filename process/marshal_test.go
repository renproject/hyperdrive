package process_test

import (
	"github.com/renproject/surge"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/renproject/hyperdrive/process"
	. "github.com/renproject/hyperdrive/testutil"
)

var _ = Describe("Marshaling", func() {
	Context("when marshaling the same propose multiple times", func() {
		It("should return the same bytes", func() {
			propose := RandomMessage(ProposeMessageType)
			proposeBytes, err := surge.ToBinary(propose)
			Expect(err).ToNot(HaveOccurred())
			for i := 0; i < 100; i++ {
				tmpProposeBytes, err := surge.ToBinary(propose)
				Expect(err).ToNot(HaveOccurred())
				Expect(tmpProposeBytes).Should(Equal(proposeBytes))
			}
		})
	})

	Context("when marshaling the same prevote multiple times", func() {
		It("should return the same bytes", func() {
			prevote := RandomMessage(PrevoteMessageType)
			prevoteBytes, err := surge.ToBinary(prevote)
			Expect(err).ToNot(HaveOccurred())
			for i := 0; i < 100; i++ {
				tmpPrevoteBytes, err := surge.ToBinary(prevote)
				Expect(err).ToNot(HaveOccurred())
				Expect(tmpPrevoteBytes).Should(Equal(prevoteBytes))
			}
		})
	})

	Context("when marshaling the same prevote multiple times", func() {
		It("should return the same bytes", func() {
			precommit := RandomMessage(PrecommitMessageType)
			precommitBytes, err := surge.ToBinary(precommit)
			Expect(err).ToNot(HaveOccurred())
			for i := 0; i < 100; i++ {
				tmpPrecommitBytes, err := surge.ToBinary(precommit)
				Expect(err).ToNot(HaveOccurred())
				Expect(tmpPrecommitBytes).Should(Equal(precommitBytes))
			}
		})
	})

	Context("when marshaling the same resync multiple times", func() {
		It("should return the same bytes", func() {
			resync := RandomMessage(ResyncMessageType)
			resyncBytes, err := surge.ToBinary(resync)
			Expect(err).ToNot(HaveOccurred())
			for i := 0; i < 100; i++ {
				tmpResyncBytes, err := surge.ToBinary(resync)
				Expect(err).ToNot(HaveOccurred())
				Expect(tmpResyncBytes).Should(Equal(resyncBytes))
			}
		})
	})
})
