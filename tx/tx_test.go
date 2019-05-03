package tx_test

// import (
// 	"github.com/renproject/hyperdrive/testutils"

// 	. "github.com/onsi/ginkgo"
// 	. "github.com/onsi/gomega"
// 	. "github.com/renproject/hyperdrive/tx"
// )

// var _ = Describe("tx", func() {

// 	Context("when a transaction is created", func() {
// 		It("should marshal/unmarshal correctly", func() {
// 			tx := testutils.RandomTransaction()
// 			bytes, err := tx.Marshal()
// 			Expect(err).ShouldNot(HaveOccurred())

// 			unmarshalledTx := NewTransaction([32]byte{})
// 			Expect(unmarshalledTx.Unmarshal(bytes)).ShouldNot(HaveOccurred())
// 			Expect(unmarshalledTx).To(Equal(tx))
// 		})
// 	})

// })
