package state_test

import (
	"bytes"
	"math/rand"

	"github.com/renproject/hyperdrive/block"
	"github.com/renproject/hyperdrive/sig"
	"github.com/renproject/hyperdrive/state"
	"github.com/renproject/hyperdrive/testutils"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Actions", func() {
	Context("when using Propose", func() {
		It("should implement the Action interface", func() {
			state.Propose{}.IsAction()
		})

		It("should marshal using Read/Write pattern", func() {
			signedBlock, signer, err := testutils.GenerateSignedBlock()
			Expect(err).ShouldNot(HaveOccurred())
			propose := state.Propose{
				SignedPropose: testutils.GenerateSignedPropose(signedBlock, block.Round(rand.Int()), signer),
				LastCommit: state.Commit{
					Commit: block.Commit{
						Polka:       testutils.GeneratePolkaWithSignatures(signedBlock, []sig.SignerVerifier{signer}),
						Signatories: testutils.RandomSignatories(2),
						Signatures:  testutils.RandomSignatures(2),
					},
				},
			}

			writer := new(bytes.Buffer)
			Expect(propose.Write(writer)).ShouldNot(HaveOccurred())

			proposeClone := state.Propose{}
			reader := bytes.NewReader(writer.Bytes())
			Expect(proposeClone.Read(reader)).ShouldNot(HaveOccurred())

			Expect(proposeClone.String()).To(Equal(propose.String()))
		})
	})

	Context("when using SignedPreVote", func() {
		It("should implement the Action interface", func() {
			state.SignedPreVote{}.IsAction()
		})

		It("should marshal using Read/Write pattern", func() {
			signedBlock, signer, err := testutils.GenerateSignedBlock()
			Expect(err).ShouldNot(HaveOccurred())
			prevote := state.SignedPreVote{
				SignedPreVote: testutils.GenerateSignedPreVote(signedBlock, signer),
			}

			writer := new(bytes.Buffer)
			Expect(prevote.Write(writer)).ShouldNot(HaveOccurred())

			prevoteClone := state.SignedPreVote{}
			reader := bytes.NewReader(writer.Bytes())
			Expect(prevoteClone.Read(reader)).ShouldNot(HaveOccurred())

			Expect(prevoteClone.String()).To(Equal(prevote.String()))
		})
	})

	Context("when using SignedPreCommit", func() {
		It("should implement the Action interface", func() {
			state.SignedPreCommit{}.IsAction()
		})

		It("should marshal using Read/Write pattern", func() {
			signedBlock, signer, err := testutils.GenerateSignedBlock()
			Expect(err).ShouldNot(HaveOccurred())
			precommit := state.SignedPreCommit{
				testutils.GenerateSignedPreCommit(signedBlock, signer, []sig.SignerVerifier{signer}),
			}

			writer := new(bytes.Buffer)
			Expect(precommit.Write(writer)).ShouldNot(HaveOccurred())

			precommitClone := state.SignedPreCommit{}
			reader := bytes.NewReader(writer.Bytes())
			Expect(precommitClone.Read(reader)).ShouldNot(HaveOccurred())

			Expect(precommitClone.String()).To(Equal(precommit.String()))
		})
	})

	Context("when using Commit", func() {
		It("should implement the Action interface", func() {
			state.Commit{}.IsAction()
		})
	})
})
