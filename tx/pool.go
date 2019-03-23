package tx

type Pool interface {
	Enqueue(Transaction)
	Dequeue() (Transaction, bool)
}

type fifoPool struct {
}

func FIFOPool() Pool {
	return &fifoPool{}
}

func (pool *fifoPool) Enqueue(tx Transaction) {
	// TODO: Implement.
}

func (pool *fifoPool) Dequeue() (Transaction, bool) {
	// TODO: Implement.
	return Transaction{}, false
}
