package tx

type Transaction interface {
	IsTransaction()
	Marshal() ([]byte, error)
	Unmarshal(data []byte) error
}

type Transactions []Transaction

type transaction struct {
	Data [32]byte
}

func NewTransaction(data [32]byte) Transaction {
	return &transaction{data}
}

func (transaction) IsTransaction() {}

func (transaction *transaction) Marshal() ([]byte, error) {
	return transaction.Data[:], nil
}

func (transaction *transaction) Unmarshal(data []byte) error {
	copy(transaction.Data[:], data)
	return nil
}
