package sig_test

import (
	"encoding/base64"
	"fmt"
	"math/rand"

	"github.com/renproject/hyperdrive/testutils"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/renproject/hyperdrive/sig"
)

var _ = Describe("Sig", func() {
	Context("when a random Hash is generated", func() {
		It("should equal itself", func() {
			hash := testutils.RandomHash()
			Expect(hash.Equal(hash)).Should(BeTrue())
		})

		It("should not equal another hash", func() {
			hash := testutils.RandomHash()
			otherHash := testutils.RandomHash()
			Expect(hash.Equal(otherHash)).Should(BeFalse())
		})

		It("should generate base64 string representation of itself", func() {
			hash := testutils.RandomHash()
			expectedHashStr := base64.StdEncoding.EncodeToString(hash[:])
			Expect(hash.String()).Should(Equal(expectedHashStr))
		})
	})

	Context("when a random Signature is generated", func() {
		It("should equal itself", func() {
			signature := testutils.RandomSignature()
			Expect(signature.Equal(signature)).Should(BeTrue())
		})

		It("should not equal another signature", func() {
			signature := testutils.RandomSignature()
			otherSignature := testutils.RandomSignature()
			Expect(signature.Equal(otherSignature)).Should(BeFalse())
		})
	})

	Context("when a random Signatory is generated", func() {
		It("should equal itself", func() {
			signatory := testutils.RandomSignatory()
			Expect(signatory.Equal(signatory)).Should(BeTrue())
		})

		It("should not equal another signatory", func() {
			signatory := testutils.RandomSignatory()
			otherSignatory := testutils.RandomSignatory()
			Expect(signatory.Equal(otherSignatory)).Should(BeFalse())
		})

		It("should generate base64 string representation of itself", func() {
			signatory := testutils.RandomSignatory()
			expectedSigStr := base64.StdEncoding.EncodeToString(signatory[:])
			Expect(signatory.String()).Should(Equal(expectedSigStr))
		})
	})

	table := []struct {
		cap int
	}{
		{1},
		{100},
		{5000},
	}

	for _, entry := range table {
		entry := entry

		Context(fmt.Sprintf("when %d signatures are created", entry.cap), func() {
			It("should equal a differently ordered similar set of signatures", func() {
				shuffled := testutils.RandomSignatures(entry.cap)

				signatures := Signatures{}
				for _, sig := range shuffled {
					signatures = append(signatures, sig)
				}

				rand.Shuffle(len(shuffled), func(i, j int) {
					shuffled[i], shuffled[j] = shuffled[j], shuffled[i]
				})

				Expect(shuffled.Equal(signatures)).Should(BeTrue())
			})

			It("should not equal a differently ordered smaller subset of signatures", func() {
				shuffled := testutils.RandomSignatures(entry.cap)

				signatures := Signatures{}
				for _, sig := range shuffled {
					signatures = append(signatures, sig)
				}

				rand.Shuffle(len(shuffled), func(i, j int) {
					shuffled[i], shuffled[j] = shuffled[j], shuffled[i]
				})
				shuffled = shuffled[:len(shuffled)-1]

				Expect(shuffled.Equal(signatures)).Should(BeFalse())
			})
		})

		Context(fmt.Sprintf("when %d signatories are created", entry.cap), func() {
			It("should equal a differently ordered similar set of signatories", func() {
				shuffled := testutils.RandomSignatories(entry.cap)

				signatories := Signatories{}
				for _, sig := range shuffled {
					signatories = append(signatories, sig)
				}

				rand.Shuffle(len(shuffled), func(i, j int) {
					shuffled[i], shuffled[j] = shuffled[j], shuffled[i]
				})

				Expect(shuffled.Equal(signatories)).Should(BeTrue())
			})

			It("should not equal a differently ordered smaller subset of signatories", func() {
				shuffled := testutils.RandomSignatories(entry.cap)

				signatories := Signatories{}
				for _, sig := range shuffled {
					signatories = append(signatories, sig)
				}

				rand.Shuffle(len(shuffled), func(i, j int) {
					shuffled[i], shuffled[j] = shuffled[j], shuffled[i]
				})
				shuffled = shuffled[:len(shuffled)-1]

				Expect(shuffled.Equal(signatories)).Should(BeFalse())
			})
		})
	}
})
