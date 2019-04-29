package tx

import (
	"sync"
)

type Pool interface {
	Enqueue(Transaction)
	Dequeue() (Transaction, bool)
}

type fifoPool struct {
	txsMu *sync.Mutex
	txs   Transactions
}

// FIFOPool is a First-In, First-Out transaction pool that is thread safe.
func FIFOPool() Pool {
	return &fifoPool{
		txsMu: new(sync.Mutex),
		txs:   Transactions{},
	}
}

func (pool *fifoPool) Enqueue(tx Transaction) {
	pool.txsMu.Lock()
	defer pool.txsMu.Unlock()

	pool.txs = append(pool.txs, tx)
}

func (pool *fifoPool) Dequeue() (Transaction, bool) {
	pool.txsMu.Lock()
	defer pool.txsMu.Unlock()

	if len(pool.txs) > 0 {
		tx := pool.txs[0]
		if len(pool.txs) > 1 {
			pool.txs = pool.txs[1:]
			return tx, true
		}
		pool.txs = Transactions{}
		return tx, true
	}
	return nil, false
}
