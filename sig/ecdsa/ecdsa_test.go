package ecdsa_test

import (
	"fmt"
	"testing/quick"

	"github.com/ethereum/go-ethereum/crypto"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/renproject/hyperdrive/sig"
	. "github.com/renproject/hyperdrive/sig/ecdsa"
)

var conf = quick.Config{
	MaxCount:      256,
	MaxCountScale: 0,
	Rand:          nil,
	Values:        nil,
}

var _ = Describe("ecdsa SignerVerifier", func() {
	Context("Sanity check for lengths", func() {
		It("Length of Hash should be the length of Keccak256", func() {
			test := func(data []byte) bool {
				hash := crypto.Keccak256(data[:])
				return len(hash) == len(Hash(data[:]))
			}
			Expect(quick.Check(test, &conf)).ShouldNot(HaveOccurred())
		})
	})
	Context("when it generates a private key", func() {
		signerVerifier, err := NewFromRandom()
		It("NewFromRandom should not error", func() {
			Expect(err).ShouldNot(HaveOccurred())
		})
		It("forall hash. Verify(hash, Sign(hash)) == Signatory()", func() {
			test := func(data []byte) bool {
				hash := Hash(data)
				sig, err := signerVerifier.Sign(hash)
				Expect(err).ShouldNot(HaveOccurred())
				signatory, err := signerVerifier.Verify(hash, sig)
				Expect(err).ShouldNot(HaveOccurred())
				return signatory == signerVerifier.Signatory()
			}
			Expect(quick.Check(test, &conf)).ShouldNot(HaveOccurred())
		})
		It("Verify should return an error when hash does not match", func() {
			test := func(h1 sig.Hash, h2 sig.Hash) bool {
				if h1 != h2 {
					sig, err := signerVerifier.Sign(h1)
					fmt.Printf("Sig %v\n", sig)
					Expect(err).ShouldNot(HaveOccurred())
					pubKey, err := signerVerifier.Verify(h2, sig)
					fmt.Printf("What %v\n", pubKey)
					pubKey, _ = signerVerifier.Verify(h1, sig)
					fmt.Printf("What 2! %v\n", pubKey)
					// Expect(err).ShouldNot(HaveOccurred())
					Expect(err).Should(HaveOccurred())
				}
				return true
			}
			Expect(quick.Check(test, &conf)).ShouldNot(HaveOccurred())
		})
	})
})
