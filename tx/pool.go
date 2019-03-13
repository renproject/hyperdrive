package tx

type Pool interface {
	Enqueue(Transaction)
	Dequeue() (Transaction, bool)
}

type fifoPool struct {
}

func (pool *fifoPool) Enqueue(tx Transaction) {
}

func (pool *fifoPool) Dequeue() (Transaction, bool) {
	return Transaction{}, false
}

type priorityPool struct {
}

func (pool *priorityPool) Enqueue(tx Transaction) {
}

func (pool *priorityPool) Dequeue() (Transaction, bool) {
	return Transaction{}, false
}
