package tx

import (
	"encoding/binary"
	"io"
)

type Transaction []byte

func (tx Transaction) Write(w io.Writer) error {
	if err := binary.Write(w, binary.LittleEndian, uint64(len(tx))); err != nil {
		return err
	}
	_, err := w.Write(tx[:])
	return err
}

func (tx *Transaction) Read(r io.Reader) error {
	var n uint64
	if err := binary.Read(r, binary.LittleEndian, &n); err != nil {
		return err
	}
	*tx = make(Transaction, n)
	_, err := r.Read(*tx)
	return err
}

type Transactions []Transaction

func (txs Transactions) Write(w io.Writer) error {
	if err := binary.Write(w, binary.LittleEndian, uint64(len(txs))); err != nil {
		return err
	}
	for _, sig := range txs {
		if err := sig.Write(w); err != nil {
			return err
		}
	}
	return nil
}

func (txs *Transactions) Read(r io.Reader) error {
	var n uint64
	if err := binary.Read(r, binary.LittleEndian, &n); err != nil {
		return err
	}

	*txs = make(Transactions, n)
	for i := range *txs {
		if err := (*txs)[i].Read(r); err != nil {
			return err
		}
	}
	return nil
}
