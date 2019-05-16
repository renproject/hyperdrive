package tx_test

import (
	"fmt"
	"math/rand"

	"github.com/renproject/hyperdrive/testutils"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/renproject/hyperdrive/tx"
)

var _ = Describe("txPool", func() {
	table := []struct {
		cap int
	}{
		{1},
		{100},
		{5000},
	}

	for _, entry := range table {
		entry := entry
		txPool := FIFOPool(entry.cap)
		removeIndex := rand.Intn(entry.cap)
		removeTx := Transaction{}

		Context(fmt.Sprintf("when a new FIFOPool is created with cap = %d", entry.cap), func() {
			It(fmt.Sprintf("should enqueue %d transactions without errors", entry.cap), func() {
				for i := 0; i < entry.cap; i++ {
					tx := testutils.RandomTransaction()
					if i == removeIndex {
						removeTx = tx
					}
					Expect(txPool.Enqueue(tx)).ShouldNot(HaveOccurred())
				}
			})

			Context("when max cap has reached", func() {
				It("should error on enqueuing a new transaction", func() {
					Expect(txPool.Enqueue(testutils.RandomTransaction())).Should(HaveOccurred())
				})
			})

			It("should be able to remove existing transactions without errors", func() {
				Expect(txPool.Remove(removeTx)).To(BeTrue())
				Expect(txPool.Remove(testutils.RandomTransaction())).To(BeFalse())
			})

			It("should not error on enqueuing a new transaction", func() {
				Expect(txPool.Enqueue(testutils.RandomTransaction())).ShouldNot(HaveOccurred())
			})

			It(fmt.Sprintf("should be able to dequeue %d transactions without errors", entry.cap), func() {
				for i := 0; i < entry.cap; i++ {
					tx, ok := txPool.Dequeue()
					Expect(ok).Should(BeTrue())
					Expect(tx).NotTo(BeNil())
				}
			})

			Context("when queue is empty", func() {
				It("should return nil Transaction on dequeuing", func() {
					tx, ok := txPool.Dequeue()
					Expect(ok).Should(BeFalse())
					Expect(tx).To(BeNil())
				})
			})
		})
	}
})
