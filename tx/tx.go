package tx

import (
	"bytes"
	"encoding/binary"
)

type Transaction []byte

func (tx Transaction) MarshalBinary() ([]byte, error) {
	buf := new(bytes.Buffer)
	if err := binary.Write(buf, binary.LittleEndian, uint64(len(tx))); err != nil {
		return nil, err
	}
	if err := binary.Write(buf, binary.LittleEndian, tx); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (tx *Transaction) UnmarshalBinary(data []byte) error {
	buf := bytes.NewBuffer(data)

	var n uint64
	if err := binary.Read(buf, binary.LittleEndian, &n); err != nil {
		return err
	}

	_, err := buf.Read(*tx)

	return err
}

type Transactions []Transaction

func (txs Transactions) MarshalBinary() ([]byte, error) {
	buf := new(bytes.Buffer)
	if err := binary.Write(buf, binary.LittleEndian, uint64(len(txs))); err != nil {
		return nil, err
	}
	for _, tx := range txs {
		data, err := tx.MarshalBinary()
		if err != nil {
			return nil, err
		}
		if err := binary.Write(buf, binary.LittleEndian, data); err != nil {
			return nil, err
		}
	}
	return buf.Bytes(), nil
}

func (txs *Transactions) UnmarshalBinary(data []byte) error {
	buf := bytes.NewBuffer(data)

	var n uint64
	if err := binary.Read(buf, binary.LittleEndian, &n); err != nil {
		return err
	}

	*txs = make(Transactions, n)
	for i := range *txs {
		if err := (*txs)[i].UnmarshalBinary(buf.Bytes()); err != nil {
			return err
		}
	}
	return nil
}
