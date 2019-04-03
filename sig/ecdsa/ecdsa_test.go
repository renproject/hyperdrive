package ecdsa_test

import (
	"testing/quick"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/renproject/hyperdrive/v1/sig"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/renproject/hyperdrive/v1/sig/ecdsa"
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
		It("Verify should always return the Signatory of the signer", func() {
			test := func(data []byte, flag bool) bool {
				var newSV sig.SignerVerifier
				if flag {
					newSV, err = NewFromRandom()
					Expect(err).ShouldNot(HaveOccurred())
				} else {
					privKey, err := crypto.GenerateKey()
					Expect(err).ShouldNot(HaveOccurred())
					newSV = NewFromPrivKey(privKey)
				}
				hash := Hash(data)

				sig, err := newSV.Sign(hash)
				Expect(err).ShouldNot(HaveOccurred())

				signatory, err := signerVerifier.Verify(hash, sig)
				Expect(err).ShouldNot(HaveOccurred())

				return signatory == newSV.Signatory()
			}
			Expect(quick.Check(test, &conf)).ShouldNot(HaveOccurred())
		})
	})
})
