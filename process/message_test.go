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

var _ = Describe("Propose", func() {
	Context("when unmarshaling fuzz", func() {
		It("should not panic", func() {
			f := func(fuzz []byte) bool {
				msg := process.Propose{}
				Expect(surge.FromBinary(&msg, fuzz)).ToNot(Succeed())
				return true
			}
			Expect(quick.Check(f, nil)).To(Succeed())
		})
	})

	Context("when marshaling and then unmarshaling", func() {
		r := rand.New(rand.NewSource(time.Now().UnixNano()))

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
				err = surge.FromBinary(&got, data)
				Expect(err).ToNot(HaveOccurred())
				Expect(got.Equal(&expected)).To(BeTrue())
				return true
			}
			Expect(quick.Check(f, nil)).To(Succeed())
		})

		randomMsg := func(r *rand.Rand) interface{} {
			switch r.Int() % 3 {
			case 0:
				return processutil.RandomPropose(r)
			case 1:
				return processutil.RandomPrevote(r)
			case 2:
				return processutil.RandomPrecommit(r)
			default:
				panic("this should not happen")
			}
		}

		It("when enough size is not available (marshaling)", func() {
			loop := func() bool {
				msg := randomMsg(r)
				switch msg := msg.(type) {
				case process.Propose:
					buf := make([]byte, msg.SizeHint())
					sizeAvailable := r.Intn(msg.SizeHint())
					_, _, err := msg.Marshal(buf, sizeAvailable)
					Expect(err).To(HaveOccurred())
				case process.Prevote:
					buf := make([]byte, msg.SizeHint())
					sizeAvailable := r.Intn(msg.SizeHint())
					_, _, err := msg.Marshal(buf, sizeAvailable)
					Expect(err).To(HaveOccurred())
				case process.Precommit:
					buf := make([]byte, msg.SizeHint())
					sizeAvailable := r.Intn(msg.SizeHint())
					_, _, err := msg.Marshal(buf, sizeAvailable)
					Expect(err).To(HaveOccurred())
				}

				return true
			}
			Expect(quick.Check(loop, nil)).To(Succeed())
		})

		It("when enough size is not available (unmarshaling)", func() {
			loop := func() bool {
				msg := randomMsg(r)
				switch msg := msg.(type) {
				case process.Propose:
					sizeHint := msg.SizeHint()
					buf := make([]byte, sizeHint)
					_, _, err := msg.Marshal(buf, sizeHint)
					Expect(err).ToNot(HaveOccurred())
					var unmarshalled process.Propose
					sizeAvailable := r.Intn(sizeHint)
					_, _, err = unmarshalled.Unmarshal(buf, sizeAvailable)
					Expect(err).To(HaveOccurred())
				case process.Prevote:
					sizeHint := msg.SizeHint()
					buf := make([]byte, sizeHint)
					_, _, err := msg.Marshal(buf, sizeHint)
					Expect(err).ToNot(HaveOccurred())
					var unmarshalled process.Prevote
					sizeAvailable := r.Intn(sizeHint)
					_, _, err = unmarshalled.Unmarshal(buf, sizeAvailable)
					Expect(err).To(HaveOccurred())
				case process.Precommit:
					sizeHint := msg.SizeHint()
					buf := make([]byte, sizeHint)
					_, _, err := msg.Marshal(buf, sizeHint)
					Expect(err).ToNot(HaveOccurred())
					var unmarshalled process.Precommit
					sizeAvailable := r.Intn(sizeHint)
					_, _, err = unmarshalled.Unmarshal(buf, sizeAvailable)
					Expect(err).To(HaveOccurred())
				}

				return true
			}
			Expect(quick.Check(loop, nil)).To(Succeed())
		})
	})

	Context("when compute the hash", func() {
		r := rand.New(rand.NewSource(time.Now().UnixNano()))

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

		It("should fail when not enough bytes available (propose)", func() {
			loop := func() bool {
				propose := processutil.RandomPropose(r)
				sizeHint := surge.SizeHint(propose.Height) +
					surge.SizeHint(propose.Round) +
					surge.SizeHint(propose.ValidRound) +
					surge.SizeHint(propose.Value)
				sizeAvailable := r.Intn(sizeHint)
				buf := make([]byte, sizeAvailable)
				_, err := process.NewProposeHashWithBuffer(propose.Height, propose.Round, propose.ValidRound, propose.Value, buf)
				Expect(err).To(HaveOccurred())
				return true
			}
			Expect(quick.Check(loop, nil)).To(Succeed())
		})

		It("should fail when not enough bytes available (prevote)", func() {
			loop := func() bool {
				prevote := processutil.RandomPrevote(r)
				sizeHint := surge.SizeHint(prevote.Height) +
					surge.SizeHint(prevote.Round) +
					surge.SizeHint(prevote.Value)
				sizeAvailable := r.Intn(sizeHint)
				buf := make([]byte, sizeAvailable)
				_, err := process.NewPrevoteHashWithBuffer(prevote.Height, prevote.Round, prevote.Value, buf)
				Expect(err).To(HaveOccurred())
				return true
			}
			Expect(quick.Check(loop, nil)).To(Succeed())
		})

		It("should fail when not enough bytes available (precommit)", func() {
			loop := func() bool {
				precommit := processutil.RandomPrecommit(r)
				sizeHint := surge.SizeHint(precommit.Height) +
					surge.SizeHint(precommit.Round) +
					surge.SizeHint(precommit.Value)
				sizeAvailable := r.Intn(sizeHint)
				buf := make([]byte, sizeAvailable)
				_, err := process.NewPrecommitHashWithBuffer(precommit.Height, precommit.Round, precommit.Value, buf)
				Expect(err).To(HaveOccurred())
				return true
			}
			Expect(quick.Check(loop, nil)).To(Succeed())
		})
	})
})

var _ = Describe("Prevote", func() {
	Context("when unmarshaling fuzz", func() {
		It("should not panic", func() {
			f := func(fuzz []byte) bool {
				msg := process.Prevote{}
				Expect(surge.FromBinary(&msg, fuzz)).ToNot(Succeed())
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
				err = surge.FromBinary(&got, data)
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
				Expect(surge.FromBinary(&msg, fuzz)).ToNot(Succeed())
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
				err = surge.FromBinary(&got, data)
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
