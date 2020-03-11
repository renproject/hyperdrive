package process_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/renproject/hyperdrive/process"
	. "github.com/renproject/hyperdrive/testutil"
)

var _ = Describe("Marshaling", func() {
	Context("when marshaling the same propose multiple times", func() {
		It("should return the same bytes", func() {
			propose := RandomMessage(ProposeMessageType)
			proposeBytes, err := propose.MarshalBinary()
			Expect(err).ToNot(HaveOccurred())
			for i := 0; i < 100; i++ {
				tmpProposeBytes, err := propose.MarshalBinary()
				Expect(err).ToNot(HaveOccurred())
				Expect(tmpProposeBytes).Should(Equal(proposeBytes))
			}
		})
	})

	Context("when marshaling the same prevote multiple times", func() {
		It("should return the same bytes", func() {
			prevote := RandomMessage(PrevoteMessageType)
			prevoteBytes, err := prevote.MarshalBinary()
			Expect(err).ToNot(HaveOccurred())
			for i := 0; i < 100; i++ {
				tmpPrevoteBytes, err := prevote.MarshalBinary()
				Expect(err).ToNot(HaveOccurred())
				Expect(tmpPrevoteBytes).Should(Equal(prevoteBytes))
			}
		})
	})

	Context("when marshaling the same prevote multiple times", func() {
		It("should return the same bytes", func() {
			precommit := RandomMessage(PrecommitMessageType)
			precommitBytes, err := precommit.MarshalBinary()
			Expect(err).ToNot(HaveOccurred())
			for i := 0; i < 100; i++ {
				tmpPrecommitBytes, err := precommit.MarshalBinary()
				Expect(err).ToNot(HaveOccurred())
				Expect(tmpPrecommitBytes).Should(Equal(precommitBytes))
			}
		})
	})

	Context("when marshaling the same resync multiple times", func() {
		It("should return the same bytes", func() {
			resync := RandomMessage(ResyncMessageType)
			resyncBytes, err := resync.MarshalBinary()
			Expect(err).ToNot(HaveOccurred())
			for i := 0; i < 100; i++ {
				tmpResyncBytes, err := resync.MarshalBinary()
				Expect(err).ToNot(HaveOccurred())
				Expect(tmpResyncBytes).Should(Equal(resyncBytes))
			}
		})
	})
})
