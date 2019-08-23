package id_test

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"encoding/json"
	"testing/quick"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/renproject/hyperdrive/id"
	. "github.com/renproject/hyperdrive/testutil"
)

var _ = Describe("ID", func() {

	Context("Hash", func() {
		Context("when two hashes are equal", func() {
			It("should be stringified to the same string", func() {
				test := func(hash Hash) bool {
					var newHash Hash
					copy(newHash[:], hash[:])

					Expect(hash.Equal(newHash)).Should(BeTrue())
					Expect(newHash.Equal(hash)).Should(BeTrue())
					return hash.String() == newHash.String()
				}

				Expect(quick.Check(test, nil)).Should(Succeed())
			})
		})

		Context("when two hashes are different", func() {
			It("should be stringified to different strings", func() {
				test := func() bool {
					hash1, hash2 := RandomHash(), RandomHash()

					Expect(hash1.Equal(hash2)).Should(BeFalse())
					Expect(hash2.Equal(hash1)).Should(BeFalse())
					return true
				}

				Expect(quick.Check(test, nil)).Should(Succeed())
			})
		})

		Context("when marshaling/unmarshaling", func() {
			It("should equal itself after marshaling and then unmarshaling", func() {
				test := func(hash Hash) bool {
					data, err := json.Marshal(hash)
					Expect(err).NotTo(HaveOccurred())

					var newHash Hash
					Expect(json.Unmarshal(data, &newHash)).Should(Succeed())

					return hash.Equal(newHash)
				}

				Expect(quick.Check(test, nil)).Should(Succeed())
			})

			It("return error when trying to unmarshal incorrect data", func() {
				test := func(data []byte, array30 [30]byte) bool {
					if len(data) == HashLength {
						return true
					}
					var hash Hash
					Expect(json.Unmarshal(data, &hash)).ShouldNot(Succeed())

					data30, err := json.Marshal(array30[:])
					Expect(err).NotTo(HaveOccurred())
					Expect(json.Unmarshal(data30, &hash)).ShouldNot(Succeed())

					return true
				}

				Expect(quick.Check(test, nil)).Should(Succeed())
			})
		})
	})

	Context("Hashes", func() {
		Context("when two lists of hashes are equal", func() {
			It("should be stringified to the same string", func() {
				test := func(hashes Hashes) bool {
					newHashes := make(Hashes, len(hashes))
					for i := range hashes {
						copy(newHashes[i][:], hashes[i][:])
					}

					Expect(hashes.Equal(newHashes)).Should(BeTrue())
					Expect(newHashes).ShouldNot(BeNil())
					Expect(newHashes.Equal(hashes)).Should(BeTrue())
					return true
				}

				Expect(quick.Check(test, nil)).Should(Succeed())
			})
		})

		Context("when two lists of hashes are different", func() {
			It("should return false when comparing them", func() {
				test := func() bool {
					hashes1, hashes2 := RandomHashes(), RandomHashes()
					for len(hashes1) == 0 && len(hashes2) == 0 {
						hashes1, hashes2 = RandomHashes(), RandomHashes()
					}
					Expect(hashes1.Equal(hashes2)).Should(BeFalse())
					Expect(hashes2.Equal(hashes1)).Should(BeFalse())
					return true
				}

				Expect(quick.Check(test, nil)).Should(Succeed())
			})
		})

		Context("when marshaling/unmarshaling", func() {
			It("should equal itself after marshaling and then unmarshaling", func() {
				test := func(hashes Hashes) bool {
					data, err := json.Marshal(hashes)
					Expect(err).NotTo(HaveOccurred())

					var newHashes Hashes
					Expect(json.Unmarshal(data, &newHashes)).Should(Succeed())

					return hashes.Equal(newHashes)
				}

				Expect(quick.Check(test, nil)).Should(Succeed())
			})
		})
	})

	Context("Signature", func() {
		Context("when two signatures are equal", func() {
			It("should be stringified to the same string", func() {
				test := func(sig Signature) bool {
					var newSig Signature
					copy(newSig[:], sig[:])

					Expect(sig.Equal(newSig)).Should(BeTrue())
					Expect(newSig.Equal(sig)).Should(BeTrue())
					return sig.String() == newSig.String()
				}

				Expect(quick.Check(test, nil)).Should(Succeed())
			})
		})

		Context("when two signatures are different", func() {
			It("should return false when comparing them", func() {
				test := func() bool {
					sigs1, sigs2 := RandomSignature(), RandomSignature()
					Expect(sigs1.Equal(sigs2)).Should(BeFalse())
					Expect(sigs2.Equal(sigs1)).Should(BeFalse())
					return true
				}

				Expect(quick.Check(test, nil)).Should(Succeed())
			})
		})

		Context("when marshaling/unmarshaling", func() {
			It("should equal itself after marshaling and then unmarshaling", func() {
				test := func(sig Signature) bool {
					data, err := json.Marshal(sig)
					Expect(err).NotTo(HaveOccurred())

					var newSig Signature
					Expect(json.Unmarshal(data, &newSig)).Should(Succeed())

					return sig.Equal(newSig)
				}

				Expect(quick.Check(test, nil)).Should(Succeed())
			})

			It("return error when trying to unmarshal incorrect data", func() {
				test := func(data []byte, array30 [30]byte) bool {
					if len(data) == SignatureLength {
						return true
					}
					var sig Signature
					Expect(json.Unmarshal(data, &sig)).ShouldNot(Succeed())

					data30, err := json.Marshal(array30[:])
					Expect(err).NotTo(HaveOccurred())
					Expect(json.Unmarshal(data30, &sig)).ShouldNot(Succeed())
					return true
				}

				Expect(quick.Check(test, nil)).Should(Succeed())
			})
		})
	})

	Context("Signatures", func() {
		Context("when two lists of signatures are equal", func() {
			It("should be stringified to the same string and have the same hash", func() {
				test := func(sigs Signatures) bool {
					newSigs := make(Signatures, len(sigs))
					for i := range sigs {
						copy(newSigs[i][:], sigs[i][:])
					}

					Expect(sigs.Equal(newSigs)).Should(BeTrue())
					Expect(newSigs).ShouldNot(BeNil())
					Expect(newSigs.Equal(sigs)).Should(BeTrue())
					Expect(sigs.Hash().Equal(newSigs.Hash())).Should(BeTrue())
					return sigs.String() == newSigs.String()
				}

				Expect(quick.Check(test, nil)).Should(Succeed())
			})
		})

		Context("when two lists of signatures are different", func() {
			It("should return false when comparing them", func() {
				test := func() bool {
					sigs1, sigs2 := RandomSignatures(), RandomSignatures()
					for len(sigs1) == 0 && len(sigs2) == 0 {
						sigs1, sigs2 = RandomSignatures(), RandomSignatures()
					}
					Expect(sigs1.Equal(sigs2)).Should(BeFalse())
					Expect(sigs2.Equal(sigs1)).Should(BeFalse())
					return true
				}

				Expect(quick.Check(test, nil)).Should(Succeed())
			})
		})

		Context("when marshaling/unmarshaling", func() {
			It("should equal itself after marshaling and then unmarshaling", func() {
				test := func(sigs Signatures) bool {
					data, err := json.Marshal(sigs)
					Expect(err).NotTo(HaveOccurred())

					var newSigs Signatures
					Expect(json.Unmarshal(data, &newSigs)).Should(Succeed())

					return sigs.Equal(newSigs)
				}

				Expect(quick.Check(test, nil)).Should(Succeed())
			})
		})
	})

	Context("Signatory", func() {
		Context("when two signatories are equal", func() {
			It("should be stringified to the same string", func() {
				test := func(sig Signatory) bool {
					var newSig Signatory
					copy(newSig[:], sig[:])

					Expect(sig.Equal(newSig)).Should(BeTrue())
					Expect(newSig.Equal(sig)).Should(BeTrue())
					return sig.String() == newSig.String()
				}

				Expect(quick.Check(test, nil)).Should(Succeed())
			})
		})

		Context("when two signatories are different", func() {
			It("should return false when comparing them", func() {
				test := func() bool {
					hashes1, hashes2 := RandomSignatory(), RandomSignatory()
					Expect(hashes1.Equal(hashes2)).Should(BeFalse())
					Expect(hashes2.Equal(hashes1)).Should(BeFalse())
					return true
				}

				Expect(quick.Check(test, nil)).Should(Succeed())
			})
		})

		Context("when marshaling/unmarshaling", func() {
			It("should equal itself after marshaling and then unmarshaling", func() {
				test := func(sig Signatory) bool {
					data, err := json.Marshal(sig)
					Expect(err).NotTo(HaveOccurred())

					var newSig Signatory
					Expect(json.Unmarshal(data, &newSig)).Should(Succeed())

					return sig.Equal(newSig)
				}

				Expect(quick.Check(test, nil)).Should(Succeed())
			})

			It("return error when trying to unmarshal incorrect data", func() {
				test := func(data []byte, array30 [30]byte) bool {
					if len(data) == SignatoryLength {
						return true
					}
					var sig Signatory
					Expect(json.Unmarshal(data, &sig)).ShouldNot(Succeed())

					data30, err := json.Marshal(array30[:])
					Expect(err).NotTo(HaveOccurred())
					Expect(json.Unmarshal(data30, &sig)).ShouldNot(Succeed())

					return true
				}

				Expect(quick.Check(test, nil)).Should(Succeed())
			})
		})
	})

	Context("Signatories", func() {
		Context("when two lists of signatories are equal", func() {
			It("should be stringified to the same string and have the same hash", func() {
				test := func(sigs Signatories) bool {
					newSigs := make(Signatories, len(sigs))
					for i := range sigs {
						copy(newSigs[i][:], sigs[i][:])
					}

					Expect(sigs.Equal(newSigs)).Should(BeTrue())
					Expect(newSigs).ShouldNot(BeNil())
					Expect(newSigs.Equal(sigs)).Should(BeTrue())
					Expect(sigs.Hash().Equal(newSigs.Hash())).Should(BeTrue())
					return sigs.String() == newSigs.String()
				}

				Expect(quick.Check(test, nil)).Should(Succeed())
			})
		})

		Context("when two lists of signatories are different", func() {
			It("should return false when comparing them", func() {
				test := func() bool {
					sigs1, sigs2 := RandomSignatories(), RandomSignatories()
					for len(sigs1) == 0 && len(sigs2) == 0 {
						sigs1, sigs2 = RandomSignatories(), RandomSignatories()
					}
					Expect(sigs1.Equal(sigs2)).Should(BeFalse())
					Expect(sigs2.Equal(sigs1)).Should(BeFalse())
					return true
				}

				Expect(quick.Check(test, nil)).Should(Succeed())
			})
		})

		Context("when creating signatories", func() {
			Context("when using pubkeys that are equal", func() {
				It("should return signatories that are equal", func() {
					test := func() bool {
						privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
						Expect(err).NotTo(HaveOccurred())

						pubKey1 := privateKey.PublicKey
						pubKey2 := pubKey1
						sig1 := NewSignatory(pubKey1)
						sig2 := NewSignatory(pubKey2)

						return sig1.Equal(sig2)
					}

					Expect(quick.Check(test, nil)).Should(Succeed())
				})
			})

			Context("when using pubkeys that are unequal", func() {
				It("should return signatories that are unequal", func() {
					test := func() bool {
						privateKey1, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
						Expect(err).NotTo(HaveOccurred())
						privateKey2, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
						Expect(err).NotTo(HaveOccurred())

						sig1 := NewSignatory(privateKey1.PublicKey)
						sig2 := NewSignatory(privateKey2.PublicKey)

						return !sig1.Equal(sig2)
					}

					Expect(quick.Check(test, nil)).Should(Succeed())
				})
			})
		})

		Context("when marshaling/unmarshaling", func() {
			It("should equal itself after marshaling and then unmarshaling", func() {
				test := func(sigs Signatories) bool {
					data, err := json.Marshal(sigs)
					Expect(err).NotTo(HaveOccurred())

					var newSigs Signatories
					Expect(json.Unmarshal(data, &newSigs)).Should(Succeed())

					return sigs.Equal(newSigs)
				}

				Expect(quick.Check(test, nil)).Should(Succeed())
			})
		})
	})
})
