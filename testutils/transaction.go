package testutils

import (
	"crypto/rand"
	"fmt"

	"github.com/renproject/hyperdrive/tx"
)

func RandomTransaction() tx.Transaction {
	bytes := [32]byte{}
	_, err := rand.Read(bytes[:])
	if err != nil {
		panic(fmt.Sprintf("error generating random bytes: %v", err))
	}
	return bytes[:]
}
