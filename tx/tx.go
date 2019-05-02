package tx

import (
	"encoding/json"

	"github.com/renproject/hyperdrive/sig"
)

type Transaction interface {
	IsTransaction()
	json.Marshaler
	json.Unmarshaler
	Header() sig.Hash
}

type Transactions []Transaction

type transaction struct {
	Data [32]byte
}

func NewTransaction(data [32]byte) Transaction {
	return &transaction{data}
}

func (transaction) IsTransaction() {}

func (transaction *transaction) MarshalJSON() ([]byte, error) {
	return transaction.Data[:], nil
}

func (transaction *transaction) UnmarshalJSON(data []byte) error {
	copy(transaction.Data[:], data)
	return nil
}

func (transaction *transaction) Header() sig.Hash {
	return transaction.Data
}
