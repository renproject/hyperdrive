package testutils

import (
	"crypto/rand"

	"github.com/renproject/hyperdrive/tx"
)

func RandomTransaction() tx.Transaction {
	bytes := [32]byte{}
	rand.Read(bytes[:])
	return tx.NewTransaction(bytes)
}
