package txpool

import "github.com/renproject/hyperdrive/tx"

type TxPool interface {
	Enqueue(tx.Transaction)
	Dequeue() (tx.Transaction, bool)
}

type fifoTxPool struct {
}

func (txPool *fifoTxPool) Enqueue(tx tx.Transaction) {
}

func (txPool *fifoTxPool) Dequeue() (tx.Transaction, bool) {
	return tx.Transaction{}, false
}

type priorityTxPool struct {
}

func (txPool *priorityTxPool) Enqueue(tx tx.Transaction) {
}

func (txPool *priorityTxPool) Dequeue() (tx.Transaction, bool) {
	return tx.Transaction{}, false
}
