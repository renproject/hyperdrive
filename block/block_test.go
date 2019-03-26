package block_test

import (
	"time"

	"github.com/renproject/hyperdrive/sig"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/renproject/hyperdrive/block"
)

var _ = Describe("Block", func() {
	Context("when genesis block is generated", func() {
		It("should create the correct genesis block", func() {
			genesis := Genesis()
			expectedGenesis := Block{
				Time:         time.Unix(0, 0),
				Round:        0,
				Height:       0,
				Header:       sig.Hash{},
				ParentHeader: sig.Hash{},
				Signature:    sig.Signature{},
				Signatory:    sig.Signatory{},
			}
			Expect(genesis).To(Equal(expectedGenesis))
		})
	})
})
