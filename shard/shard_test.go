package shard_test

import (
	"fmt"

	"github.com/renproject/hyperdrive/block"
	"github.com/renproject/hyperdrive/testutils"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/renproject/hyperdrive/shard"
)

var _ = Describe("Shard", func() {
	table := []struct {
		cap int
	}{
		{1},
		{100},
		{5000},
	}

	for _, entry := range table {
		entry := entry

		signatories := testutils.RandomSignatories(entry.cap)
		expectedConsensusThreshold := entry.cap - entry.cap/3

		shard := Shard{
			Hash:        testutils.RandomHash(),
			BlockHeader: testutils.RandomHash(),
			BlockHeight: block.Height(entry.cap),
			Signatories: signatories,
		}

		Context(fmt.Sprintf("when a new Shard is created with %d signatories", entry.cap), func() {
			It(fmt.Sprintf("should return size of shard = %d", entry.cap), func() {
				Expect(shard.Size()).Should(Equal(entry.cap))
			})

			It("should return the correct leader for a given round", func() {
				for i := 0; i < 2*entry.cap; i++ {
					Expect(shard.Leader(block.Round(i))).Should(Equal(signatories[int64(i)%int64(len(shard.Signatories))]))
				}
			})

			It(fmt.Sprintf("should return %d as the consensus threashold for shard of size %d", expectedConsensusThreshold, entry.cap), func() {
				Expect(shard.ConsensusThreshold()).Should(Equal(expectedConsensusThreshold))
			})

			It("should return true for all valid signatories and false otherwise", func() {
				for i := 0; i < entry.cap; i++ {
					Expect(shard.Includes(signatories[i])).Should(BeTrue())
				}
				Expect(shard.Includes(testutils.RandomSignatory())).Should(BeFalse())
			})

		})
	}
})