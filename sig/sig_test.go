package sig_test

import (
	"crypto/rand"

	"github.com/renproject/hyperdrive/sig"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Signatures", func() {

	Context("when testing equality of hashes", func() {
		It("should return true for empty hashes", func() {
			lhs := sig.Hash{}
			rhs := sig.Hash{}
			Expect(lhs.Equal(rhs)).To(BeTrue())
			Expect(lhs.String()).To(Equal(rhs.String()))
		})

		It("should return true for the same hashes", func() {
			lhs := sig.Hash{}
			n, err := rand.Read(lhs[:])
			Expect(n).To(Equal(32))
			Expect(err).ToNot(HaveOccurred())

			rhs := sig.Hash{}
			copy(rhs[:], lhs[:])

			Expect(lhs.Equal(rhs)).To(BeTrue())
			Expect(lhs.String()).To(Equal(rhs.String()))
		})

		It("should return false for two random hashes", func() {
			lhs := sig.Hash{}
			n, err := rand.Read(lhs[:])
			Expect(n).To(Equal(32))
			Expect(err).ToNot(HaveOccurred())

			rhs := sig.Hash{}
			n, err = rand.Read(rhs[:])
			Expect(n).To(Equal(32))
			Expect(err).ToNot(HaveOccurred())

			Expect(lhs.Equal(rhs)).To(BeFalse())
			Expect(lhs.String()).ToNot(Equal(rhs.String()))
		})
	})

	Context("when testing equality of signatures", func() {
		It("should return true for empty signatures", func() {
			lhs := sig.Signature{}
			rhs := sig.Signature{}
			Expect(lhs.Equal(rhs)).To(BeTrue())
		})

		It("should return true for the same signatures", func() {
			lhs := sig.Signature{}
			n, err := rand.Read(lhs[:])
			Expect(n).To(Equal(65))
			Expect(err).ToNot(HaveOccurred())

			rhs := sig.Signature{}
			copy(rhs[:], lhs[:])

			Expect(lhs.Equal(rhs)).To(BeTrue())
		})

		It("should return false for two random signatures", func() {
			lhs := sig.Signature{}
			n, err := rand.Read(lhs[:])
			Expect(n).To(Equal(65))
			Expect(err).ToNot(HaveOccurred())

			rhs := sig.Signature{}
			n, err = rand.Read(rhs[:])
			Expect(n).To(Equal(65))
			Expect(err).ToNot(HaveOccurred())

			Expect(lhs.Equal(rhs)).To(BeFalse())
		})

		It("should return false for signature slices with different lengths", func() {
			lhs := sig.Signatures{sig.Signature{}}
			rhs := sig.Signatures{}
			Expect(lhs.Equal(rhs)).To(BeFalse())
		})

		It("should return false for random signature slices", func() {
			lhs := sig.Signatures{}
			for i := 0; i < 4; i++ {
				s := sig.Signature{}
				n, err := rand.Read(s[:])
				Expect(n).To(Equal(65))
				Expect(err).ToNot(HaveOccurred())
				lhs = append(lhs, s)
			}

			rhs := sig.Signatures{}
			for i := 0; i < 4; i++ {
				s := sig.Signature{}
				n, err := rand.Read(s[:])
				Expect(n).To(Equal(65))
				Expect(err).ToNot(HaveOccurred())
				rhs = append(rhs, s)
			}

			Expect(lhs.Equal(rhs)).To(BeFalse())
		})

		It("should return true for the same signature slices", func() {
			lhs := sig.Signatures{}
			for i := 0; i < 4; i++ {
				s := sig.Signature{}
				n, err := rand.Read(s[:])
				Expect(n).To(Equal(65))
				Expect(err).ToNot(HaveOccurred())
				lhs = append(lhs, s)
			}

			rhs := sig.Signatures{}
			for i := 0; i < 4; i++ {
				s := sig.Signature{}
				copy(s[:], lhs[i][:])
				rhs = append(rhs, s)
			}

			Expect(lhs.Equal(rhs)).To(BeTrue())
		})
	})

	Context("when testing equality of signatories", func() {
		It("should return true for empty signatories", func() {
			lhs := sig.Signatory{}
			rhs := sig.Signatory{}
			Expect(lhs.Equal(rhs)).To(BeTrue())
			Expect(lhs.String()).To(Equal(rhs.String()))
		})

		It("should return true for the same signatories", func() {
			lhs := sig.Signatory{}
			n, err := rand.Read(lhs[:])
			Expect(n).To(Equal(20))
			Expect(err).ToNot(HaveOccurred())

			rhs := sig.Signatory{}
			copy(rhs[:], lhs[:])

			Expect(lhs.Equal(rhs)).To(BeTrue())
			Expect(lhs.String()).To(Equal(rhs.String()))
		})

		It("should return false for two random signatories", func() {
			lhs := sig.Signatory{}
			n, err := rand.Read(lhs[:])
			Expect(n).To(Equal(20))
			Expect(err).ToNot(HaveOccurred())

			rhs := sig.Signatory{}
			n, err = rand.Read(rhs[:])
			Expect(n).To(Equal(20))
			Expect(err).ToNot(HaveOccurred())

			Expect(lhs.Equal(rhs)).To(BeFalse())
			Expect(lhs.String()).ToNot(Equal(rhs.String()))
		})

		It("should return false for signatory slices of different lengths", func() {
			lhs := sig.Signatories{sig.Signatory{}}
			rhs := sig.Signatories{}
			Expect(lhs.Equal(rhs)).To(BeFalse())
		})

		It("should return false for random signatory slices", func() {
			lhs := sig.Signatories{}
			for i := 0; i < 4; i++ {
				s := sig.Signatory{}
				n, err := rand.Read(s[:])
				Expect(n).To(Equal(20))
				Expect(err).ToNot(HaveOccurred())
				lhs = append(lhs, s)
			}

			rhs := sig.Signatories{}
			for i := 0; i < 4; i++ {
				s := sig.Signatory{}
				n, err := rand.Read(s[:])
				Expect(n).To(Equal(20))
				Expect(err).ToNot(HaveOccurred())
				rhs = append(rhs, s)
			}

			Expect(lhs.Equal(rhs)).To(BeFalse())
		})

		It("should return true for the same signatory slices", func() {
			lhs := sig.Signatories{}
			for i := 0; i < 4; i++ {
				s := sig.Signatory{}
				n, err := rand.Read(s[:])
				Expect(n).To(Equal(20))
				Expect(err).ToNot(HaveOccurred())
				lhs = append(lhs, s)
			}

			rhs := sig.Signatories{}
			for i := 0; i < 4; i++ {
				s := sig.Signatory{}
				copy(s[:], lhs[i][:])
				rhs = append(rhs, s)
			}

			Expect(lhs.Equal(rhs)).To(BeTrue())
		})
	})
})
