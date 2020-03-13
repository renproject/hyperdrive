package block

import (
	"encoding/json"
	"io"

	"github.com/renproject/id"
	"github.com/renproject/surge"
)

// MarshalJSON is implemented because it is not uncommon that blocks and block
// headers need to be made available through external APIs.
func (header Header) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Kind         Kind           `json:"kind"`
		ParentHash   id.Hash        `json:"parentHash"`
		BaseHash     id.Hash        `json:"baseHash"`
		TxsRef       id.Hash        `json:"txsRef"`
		PlanRef      id.Hash        `json:"planRef"`
		PrevStateRef id.Hash        `json:"prevStateRef"`
		Height       Height         `json:"height"`
		Round        Round          `json:"round"`
		Timestamp    Timestamp      `json:"timestamp"`
		Signatories  id.Signatories `json:"signatories"`
	}{
		header.kind,
		header.parentHash,
		header.baseHash,
		header.txsRef,
		header.planRef,
		header.prevStateRef,
		header.height,
		header.round,
		header.timestamp,
		header.signatories,
	})
}

// UnmarshalJSON is implemented because it is not uncommon that blocks and block
// headers need to be made available through external APIs.
func (header *Header) UnmarshalJSON(data []byte) error {
	tmp := struct {
		Kind         Kind           `json:"kind"`
		ParentHash   id.Hash        `json:"parentHash"`
		BaseHash     id.Hash        `json:"baseHash"`
		TxsRef       id.Hash        `json:"txsRef"`
		PlanRef      id.Hash        `json:"planRef"`
		PrevStateRef id.Hash        `json:"prevStateRef"`
		Height       Height         `json:"height"`
		Round        Round          `json:"round"`
		Timestamp    Timestamp      `json:"timestamp"`
		Signatories  id.Signatories `json:"signatories"`
	}{}
	if err := json.Unmarshal(data, &tmp); err != nil {
		return err
	}
	header.kind = tmp.Kind
	header.parentHash = tmp.ParentHash
	header.baseHash = tmp.BaseHash
	header.txsRef = tmp.TxsRef
	header.planRef = tmp.PlanRef
	header.prevStateRef = tmp.PrevStateRef
	header.height = tmp.Height
	header.round = tmp.Round
	header.timestamp = tmp.Timestamp
	header.signatories = tmp.Signatories
	return nil
}

// SizeHint of how many bytes will be needed to represent a header in binary.
func (header Header) SizeHint() int {
	return surge.SizeHint(header.kind) +
		surge.SizeHint(header.parentHash) +
		surge.SizeHint(header.baseHash) +
		surge.SizeHint(header.txsRef) +
		surge.SizeHint(header.planRef) +
		surge.SizeHint(header.prevStateRef) +
		surge.SizeHint(header.height) +
		surge.SizeHint(header.round) +
		surge.SizeHint(header.timestamp) +
		surge.SizeHint(header.signatories)
}

// Marshal this header into binary.
func (header Header) Marshal(w io.Writer, m int) (int, error) {
	m, err := surge.Marshal(w, header.kind, m)
	if err != nil {
		return m, err
	}
	if m, err = surge.Marshal(w, header.parentHash, m); err != nil {
		return m, err
	}
	if m, err = surge.Marshal(w, header.baseHash, m); err != nil {
		return m, err
	}
	if m, err = surge.Marshal(w, header.txsRef, m); err != nil {
		return m, err
	}
	if m, err = surge.Marshal(w, header.planRef, m); err != nil {
		return m, err
	}
	if m, err = surge.Marshal(w, header.prevStateRef, m); err != nil {
		return m, err
	}
	if m, err = surge.Marshal(w, header.height, m); err != nil {
		return m, err
	}
	if m, err = surge.Marshal(w, header.round, m); err != nil {
		return m, err
	}
	if m, err = surge.Marshal(w, header.timestamp, m); err != nil {
		return m, err
	}
	return surge.Marshal(w, header.signatories, m)
}

// Unmarshal into this header from binary.
func (header *Header) Unmarshal(r io.Reader, m int) (int, error) {
	m, err := surge.Unmarshal(r, &header.kind, m)
	if err != nil {
		return m, err
	}
	if m, err = surge.Unmarshal(r, &header.parentHash, m); err != nil {
		return m, err
	}
	if m, err = surge.Unmarshal(r, &header.baseHash, m); err != nil {
		return m, err
	}
	if m, err = surge.Unmarshal(r, &header.txsRef, m); err != nil {
		return m, err
	}
	if m, err = surge.Unmarshal(r, &header.planRef, m); err != nil {
		return m, err
	}
	if m, err = surge.Unmarshal(r, &header.prevStateRef, m); err != nil {
		return m, err
	}
	if m, err = surge.Unmarshal(r, &header.height, m); err != nil {
		return m, err
	}
	if m, err = surge.Unmarshal(r, &header.round, m); err != nil {
		return m, err
	}
	if m, err = surge.Unmarshal(r, &header.timestamp, m); err != nil {
		return m, err
	}
	return surge.Unmarshal(r, &header.signatories, m)
}

// MarshalJSON is implemented because it is not uncommon that blocks and block
// headers need to be made available through external APIs.
func (block Block) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Hash      id.Hash `json:"hash"`
		Header    Header  `json:"header"`
		Txs       Txs     `json:"txs"`
		Plan      Plan    `json:"plan"`
		PrevState State   `json:"prevState"`
	}{
		block.hash,
		block.header,
		block.txs,
		block.plan,
		block.prevState,
	})
}

// UnmarshalJSON is implemented because it is not uncommon that blocks and block
// headers need to be made available through external APIs.
func (block *Block) UnmarshalJSON(data []byte) error {
	tmp := struct {
		Hash      id.Hash `json:"hash"`
		Header    Header  `json:"header"`
		Txs       Txs     `json:"txs"`
		Plan      Plan    `json:"plan"`
		PrevState State   `json:"prevState"`
	}{}
	if err := json.Unmarshal(data, &tmp); err != nil {
		return err
	}
	block.hash = tmp.Hash
	block.header = tmp.Header
	block.txs = tmp.Txs
	block.plan = tmp.Plan
	block.prevState = tmp.PrevState
	return nil
}

// SizeHint of how many bytes will be needed to represent a header in binary.
func (block Block) SizeHint() int {
	return surge.SizeHint(block.hash) +
		surge.SizeHint(block.header) +
		surge.SizeHint(block.txs) +
		surge.SizeHint(block.plan) +
		surge.SizeHint(block.prevState)
}

// Marshal this block into binary.
func (block Block) Marshal(w io.Writer, m int) (int, error) {
	m, err := surge.Marshal(w, block.hash, m)
	if err != nil {
		return m, err
	}
	if m, err = surge.Marshal(w, block.header, m); err != nil {
		return m, err
	}
	if m, err = surge.Marshal(w, block.txs, m); err != nil {
		return m, err
	}
	if m, err = surge.Marshal(w, block.plan, m); err != nil {
		return m, err
	}
	return surge.Marshal(w, block.prevState, m)
}

// Unmarshal into this block from binary.
func (block *Block) Unmarshal(r io.Reader, m int) (int, error) {
	m, err := surge.Unmarshal(r, &block.hash, m)
	if err != nil {
		return m, err
	}
	if m, err = surge.Unmarshal(r, &block.header, m); err != nil {
		return m, err
	}
	if m, err = surge.Unmarshal(r, &block.txs, m); err != nil {
		return m, err
	}
	if m, err = surge.Unmarshal(r, &block.plan, m); err != nil {
		return m, err
	}
	return surge.Unmarshal(r, &block.prevState, m)
}
