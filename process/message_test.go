package process_test

import (
	"testing/quick"

	"github.com/renproject/hyperdrive/process"
	"github.com/renproject/id"
	"github.com/renproject/surge"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Propose", func() {
	Context("when unmarshaling fuzz", func() {
		It("should not panic", func() {
			f := func(fuzz []byte) bool {
				msg := process.Propose{}
				Expect(surge.FromBinary(fuzz, &msg)).ToNot(Succeed())
				return true
			}
			Expect(quick.Check(f, nil)).To(Succeed())
		})
	})

	Context("when marshaling and then unmarshaling", func() {
		It("should equal itself", func() {
			f := func(height process.Height, round, validRound process.Round, value process.Value, from id.Signatory, signature id.Signature) bool {
				expected := process.Propose{
					Height:     height,
					Round:      round,
					ValidRound: validRound,
					Value:      value,
					From:       from,
					Signature:  signature,
				}
				data, err := surge.ToBinary(expected)
				Expect(err).ToNot(HaveOccurred())
				got := process.Propose{}
				err = surge.FromBinary(data, &got)
				Expect(err).ToNot(HaveOccurred())
				Expect(got.Equal(&expected)).To(BeTrue())
				return true
			}
			Expect(quick.Check(f, nil)).To(Succeed())
		})
	})

	Context("when compute the hash", func() {
		It("should not be random", func() {
			f := func(height process.Height, round, validRound process.Round, value process.Value) bool {
				expected, err := process.NewProposeHash(height, round, validRound, value)
				Expect(err).ToNot(HaveOccurred())
				got, err := process.NewProposeHash(height, round, validRound, value)
				Expect(err).ToNot(HaveOccurred())
				Expect(got.Equal(&expected)).To(BeTrue())
				return true
			}
			Expect(quick.Check(f, nil)).To(Succeed())
		})

		It("should the expected signatory", func() {
			f := func(height process.Height, round, validRound process.Round, value process.Value) bool {
				privKey := id.NewPrivKey()
				hash, err := process.NewProposeHash(height, round, validRound, value)
				Expect(err).ToNot(HaveOccurred())
				signature, err := privKey.Sign(&hash)
				Expect(err).ToNot(HaveOccurred())
				signatory, err := signature.Signatory(&hash)
				Expect(err).ToNot(HaveOccurred())
				Expect(privKey.Signatory().Equal(&signatory)).To(BeTrue())
				return true
			}
			Expect(quick.Check(f, nil)).To(Succeed())
		})
	})
})

var _ = Describe("Prevote", func() {
	Context("when unmarshaling fuzz", func() {
		It("should not panic", func() {
			f := func(fuzz []byte) bool {
				msg := process.Prevote{}
				Expect(surge.FromBinary(fuzz, &msg)).ToNot(Succeed())
				return true
			}
			Expect(quick.Check(f, nil)).To(Succeed())
		})
	})

	Context("when marshaling and then unmarshaling", func() {
		It("should equal itself", func() {
			f := func(height process.Height, round process.Round, value process.Value, from id.Signatory, signature id.Signature) bool {
				expected := process.Prevote{
					Height:    height,
					Round:     round,
					Value:     value,
					From:      from,
					Signature: signature,
				}
				data, err := surge.ToBinary(expected)
				Expect(err).ToNot(HaveOccurred())
				got := process.Prevote{}
				err = surge.FromBinary(data, &got)
				Expect(err).ToNot(HaveOccurred())
				Expect(got.Equal(&expected)).To(BeTrue())
				return true
			}
			Expect(quick.Check(f, nil)).To(Succeed())
		})
	})

	Context("when compute the hash", func() {
		It("should not be random", func() {
			f := func(height process.Height, round process.Round, value process.Value) bool {
				expected, err := process.NewPrevoteHash(height, round, value)
				Expect(err).ToNot(HaveOccurred())
				got, err := process.NewPrevoteHash(height, round, value)
				Expect(err).ToNot(HaveOccurred())
				Expect(got.Equal(&expected)).To(BeTrue())
				return true
			}
			Expect(quick.Check(f, nil)).To(Succeed())
		})

		It("should the expected signatory", func() {
			f := func(height process.Height, round process.Round, value process.Value) bool {
				privKey := id.NewPrivKey()
				hash, err := process.NewPrevoteHash(height, round, value)
				Expect(err).ToNot(HaveOccurred())
				signature, err := privKey.Sign(&hash)
				Expect(err).ToNot(HaveOccurred())
				signatory, err := signature.Signatory(&hash)
				Expect(err).ToNot(HaveOccurred())
				Expect(privKey.Signatory().Equal(&signatory)).To(BeTrue())
				return true
			}
			Expect(quick.Check(f, nil)).To(Succeed())
		})
	})
})

var _ = Describe("Precommit", func() {
	Context("when unmarshaling fuzz", func() {
		It("should not panic", func() {
			f := func(fuzz []byte) bool {
				msg := process.Precommit{}
				Expect(surge.FromBinary(fuzz, &msg)).ToNot(Succeed())
				return true
			}
			Expect(quick.Check(f, nil)).To(Succeed())
		})
	})

	Context("when marshaling and then unmarshaling", func() {
		It("should equal itself", func() {
			f := func(height process.Height, round process.Round, value process.Value, from id.Signatory, signature id.Signature) bool {
				expected := process.Precommit{
					Height:    height,
					Round:     round,
					Value:     value,
					From:      from,
					Signature: signature,
				}
				data, err := surge.ToBinary(expected)
				Expect(err).ToNot(HaveOccurred())
				got := process.Precommit{}
				err = surge.FromBinary(data, &got)
				Expect(err).ToNot(HaveOccurred())
				Expect(got.Equal(&expected)).To(BeTrue())
				return true
			}
			Expect(quick.Check(f, nil)).To(Succeed())
		})
	})

	Context("when compute the hash", func() {
		It("should not be random", func() {
			f := func(height process.Height, round process.Round, value process.Value) bool {
				expected, err := process.NewPrecommitHash(height, round, value)
				Expect(err).ToNot(HaveOccurred())
				got, err := process.NewPrecommitHash(height, round, value)
				Expect(err).ToNot(HaveOccurred())
				Expect(got.Equal(&expected)).To(BeTrue())
				return true
			}
			Expect(quick.Check(f, nil)).To(Succeed())
		})

		It("should the expected signatory", func() {
			f := func(height process.Height, round process.Round, value process.Value) bool {
				privKey := id.NewPrivKey()
				hash, err := process.NewPrecommitHash(height, round, value)
				Expect(err).ToNot(HaveOccurred())
				signature, err := privKey.Sign(&hash)
				Expect(err).ToNot(HaveOccurred())
				signatory, err := signature.Signatory(&hash)
				Expect(err).ToNot(HaveOccurred())
				Expect(privKey.Signatory().Equal(&signatory)).To(BeTrue())
				return true
			}
			Expect(quick.Check(f, nil)).To(Succeed())
		})
	})
})
