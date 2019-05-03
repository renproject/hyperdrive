package tx

import (
	"errors"
	"sync"
)

var ErrPoolCapacityExceeded = errors.New("pool capacity exceeded")

type Pool interface {
	Enqueue(Transaction) error
	Dequeue() (Transaction, bool)
}

type fifoPool struct {
	cap int

	txsMu *sync.Mutex
	txs   []Transaction
}

// FIFOPool is a First-In, First-Out transaction pool that is thread safe.
func FIFOPool(cap int) Pool {
	return &fifoPool{
		cap: cap,

		txsMu: new(sync.Mutex),
		txs:   []Transaction{},
	}
}

func (pool *fifoPool) Enqueue(tx Transaction) error {
	pool.txsMu.Lock()
	defer pool.txsMu.Unlock()

	if len(pool.txs) >= pool.cap {
		return ErrPoolCapacityExceeded
	}

	pool.txs = append(pool.txs, tx)
	return nil
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
		pool.txs = []Transaction{}
		return tx, true
	}
	return nil, false
}
