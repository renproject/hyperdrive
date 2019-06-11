package replica_test

import (
	"time"

	"github.com/renproject/hyperdrive/block"
	"github.com/renproject/hyperdrive/shard"
	"github.com/renproject/hyperdrive/sig"
	"github.com/renproject/hyperdrive/sig/ecdsa"
	"github.com/renproject/hyperdrive/testutils"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/renproject/hyperdrive/replica"
)

var _ = Describe("Validator", func() {
	Context("when a valid commit is provided", func() {
		It("should return true", func() {
			signer1, err := ecdsa.NewFromRandom()
			Expect(err).ShouldNot(HaveOccurred())

			signer2, err := ecdsa.NewFromRandom()
			Expect(err).ShouldNot(HaveOccurred())

			shard := shard.Shard{
				Hash:        sig.Hash{},
				BlockHeader: sig.Hash{},
				BlockHeight: 0,
				Signatories: sig.Signatories{signer1.Signatory(), signer2.Signatory()},
			}
			validator := NewValidator(signer1, shard)

			parent := testutils.SignBlock(block.Block{Time: time.Now(), Height: 1}, signer1)
			precommit1 := testutils.GenerateSignedPreCommit(*parent, signer1, []sig.SignerVerifier{signer1, signer2})
			precommit2 := testutils.GenerateSignedPreCommit(*parent, signer1, []sig.SignerVerifier{signer1, signer2})
			commit := block.Commit{
				Polka:       precommit1.Polka,
				Signatures:  sig.Signatures{precommit1.Signature, precommit2.Signature},
				Signatories: sig.Signatories{precommit1.Signatory, precommit2.Signatory},
			}
			Expect(validator.ValidateCommit(commit)).To(BeTrue())

			newBlock := testutils.SignBlock(block.Block{Time: time.Now(), Height: 2, ParentHeader: parent.Header}, signer1)
			newPropose := testutils.GenerateSignedPropose(*newBlock, 0, signer1)
			newPropose.LastCommit = &commit
			Expect(validator.ValidatePropose(newPropose)).To(BeTrue())
		})
	})
})
