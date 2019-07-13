package testutils

import (
	"crypto/rand"
	"fmt"

	"github.com/renproject/hyperdrive/tx"
)

func RandomTransactions(n int) tx.Transactions {
	txs := make(tx.Transactions, n)
	for i := 0; i < n; i++ {
		txs[i] = RandomTransaction()
	}
	return txs
}

func RandomTransaction() tx.Transaction {
	bytes := [32]byte{}
	_, err := rand.Read(bytes[:])
	if err != nil {
		panic(fmt.Sprintf("error generating random bytes: %v", err))
	}
	return tx.Transaction(bytes[:])
}
