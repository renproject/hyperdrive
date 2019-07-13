package tx_test

import (
	"bytes"
	"math/rand"

	"github.com/renproject/hyperdrive/testutils"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/renproject/hyperdrive/tx"
)

var _ = Describe("Transactions", func() {
	Context("when using Transaction", func() {
		It("should marshal using Read/Write pattern", func() {
			tx := testutils.RandomTransaction()

			writer := new(bytes.Buffer)
			Expect(tx.Write(writer)).ShouldNot(HaveOccurred())

			txClone := Transaction{}
			reader := bytes.NewReader(writer.Bytes())
			Expect(txClone.Read(reader)).ShouldNot(HaveOccurred())

			Expect(txClone).To(Equal(tx))
		})
	})

	Context("when using Transactions", func() {
		It("should marshal using Read/Write pattern", func() {
			txs := testutils.RandomTransactions(rand.Intn(10) + 1)

			writer := new(bytes.Buffer)
			Expect(txs.Write(writer)).ShouldNot(HaveOccurred())

			txsClone := Transactions{}
			reader := bytes.NewReader(writer.Bytes())
			Expect(txsClone.Read(reader)).ShouldNot(HaveOccurred())

			Expect(txsClone).To(Equal(txs))
		})
	})
})
