package block

import (
	"encoding/json"
	"io"

	"github.com/renproject/id"
	"github.com/renproject/surge"
)

// MarshalJSON implements the `json.Marshaler` interface for the `Header` type.
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

// UnmarshalJSON implements the `json.Unmarshaler` interface for the `Header`
// type.
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

func (header Header) Marshal(w io.Writer, m int) (int, error) {
	if m <= 0 {
		return m, surge.ErrMaxBytesExceeded
	}

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

func (header *Header) Unmarshal(r io.Reader, m int) (int, error) {
	if m <= 0 {
		return m, surge.ErrMaxBytesExceeded
	}

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

// MarshalBinary implements the `encoding.BinaryMarshaler` interface for the
// `Header` type.
func (header Header) MarshalBinary() ([]byte, error) {
	return surge.ToBinary(header)
	// buf := new(bytes.Buffer)
	// if err := binary.Write(buf, binary.LittleEndian, header.kind); err != nil {
	// 	return buf.Bytes(), fmt.Errorf("cannot write header.kind: %v", err)
	// }
	// if err := binary.Write(buf, binary.LittleEndian, header.parentHash); err != nil {
	// 	return buf.Bytes(), fmt.Errorf("cannot write header.parentHash: %v", err)
	// }
	// if err := binary.Write(buf, binary.LittleEndian, header.baseHash); err != nil {
	// 	return buf.Bytes(), fmt.Errorf("cannot write header.baseHash: %v", err)
	// }
	// if err := binary.Write(buf, binary.LittleEndian, header.txsRef); err != nil {
	// 	return buf.Bytes(), fmt.Errorf("cannot write header.txsRef: %v", err)
	// }
	// if err := binary.Write(buf, binary.LittleEndian, header.planRef); err != nil {
	// 	return buf.Bytes(), fmt.Errorf("cannot write header.planRef: %v", err)
	// }
	// if err := binary.Write(buf, binary.LittleEndian, header.prevStateRef); err != nil {
	// 	return buf.Bytes(), fmt.Errorf("cannot write header.prevStateRef: %v", err)
	// }
	// if err := binary.Write(buf, binary.LittleEndian, header.height); err != nil {
	// 	return buf.Bytes(), fmt.Errorf("cannot write header.height: %v", err)
	// }
	// if err := binary.Write(buf, binary.LittleEndian, header.round); err != nil {
	// 	return buf.Bytes(), fmt.Errorf("cannot write header.round: %v", err)
	// }
	// if err := binary.Write(buf, binary.LittleEndian, header.timestamp); err != nil {
	// 	return buf.Bytes(), fmt.Errorf("cannot write header.timestamp: %v", err)
	// }
	// if err := binary.Write(buf, binary.LittleEndian, uint64(len(header.signatories))); err != nil {
	// 	return buf.Bytes(), fmt.Errorf("cannot write header.signatories len: %v", err)
	// }
	// for _, sig := range header.signatories {
	// 	if err := binary.Write(buf, binary.LittleEndian, sig); err != nil {
	// 		return buf.Bytes(), fmt.Errorf("cannot write header.signatories data: %v", err)
	// 	}
	// }
	// return buf.Bytes(), nil
}

// UnmarshalBinary implements the `encoding.BinaryUnmarshaler` interface for the
// `Header` type.
func (header *Header) UnmarshalBinary(data []byte) error {
	return surge.FromBinary(data, header)
	// buf := bytes.NewBuffer(data)
	// if err := binary.Read(buf, binary.LittleEndian, &header.kind); err != nil {
	// 	return fmt.Errorf("cannot read header.kind: %v", err)
	// }
	// if err := binary.Read(buf, binary.LittleEndian, &header.parentHash); err != nil {
	// 	return fmt.Errorf("cannot read header.parentHash: %v", err)
	// }
	// if err := binary.Read(buf, binary.LittleEndian, &header.baseHash); err != nil {
	// 	return fmt.Errorf("cannot read header.baseHash: %v", err)
	// }
	// if err := binary.Read(buf, binary.LittleEndian, &header.txsRef); err != nil {
	// 	return fmt.Errorf("cannot read header.txsRef: %v", err)
	// }
	// if err := binary.Read(buf, binary.LittleEndian, &header.planRef); err != nil {
	// 	return fmt.Errorf("cannot read header.planRef: %v", err)
	// }
	// if err := binary.Read(buf, binary.LittleEndian, &header.prevStateRef); err != nil {
	// 	return fmt.Errorf("cannot read header.prevStateRef: %v", err)
	// }
	// if err := binary.Read(buf, binary.LittleEndian, &header.height); err != nil {
	// 	return fmt.Errorf("cannot read header.height: %v", err)
	// }
	// if err := binary.Read(buf, binary.LittleEndian, &header.round); err != nil {
	// 	return fmt.Errorf("cannot read header.round: %v", err)
	// }
	// if err := binary.Read(buf, binary.LittleEndian, &header.timestamp); err != nil {
	// 	return fmt.Errorf("cannot read header.timestamp: %v", err)
	// }
	// var lenSignatories uint64
	// if err := binary.Read(buf, binary.LittleEndian, &lenSignatories); err != nil {
	// 	return fmt.Errorf("cannot read header.signatories len: %v", err)
	// }
	// if lenSignatories > 0 {
	// 	header.signatories = make(id.Signatories, lenSignatories)
	// 	for i := uint64(0); i < lenSignatories; i++ {
	// 		if err := binary.Read(buf, binary.LittleEndian, &header.signatories[i]); err != nil {
	// 			return fmt.Errorf("cannot read header.signatories data: %v", err)
	// 		}
	// 	}
	// }
	// return nil
}

// MarshalJSON implements the `json.Marshaler` interface for the `Block` type.
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

// UnmarshalJSON implements the `json.Unmarshaler` interface for the `Block`
// type.
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

func (block Block) SizeHint() int {
	return surge.SizeHint(block.hash) +
		surge.SizeHint(block.header) +
		surge.SizeHint(block.txs) +
		surge.SizeHint(block.plan) +
		surge.SizeHint(block.prevState)
}

func (block Block) Marshal(w io.Writer, m int) (int, error) {
	if m <= 0 {
		return m, surge.ErrMaxBytesExceeded
	}

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

func (block *Block) Unmarshal(r io.Reader, m int) (int, error) {
	if m <= 0 {
		return m, surge.ErrMaxBytesExceeded
	}

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

// MarshalBinary implements the `encoding.BinaryMarshaler` interface for the
// `Block` type.
func (block Block) MarshalBinary() ([]byte, error) {
	return surge.ToBinary(block)
	// buf := new(bytes.Buffer)
	// if err := binary.Write(buf, binary.LittleEndian, block.hash); err != nil {
	// 	return buf.Bytes(), fmt.Errorf("cannot write block.hash: %v", err)
	// }
	// headerData, err := block.header.MarshalBinary()
	// if err != nil {
	// 	return buf.Bytes(), fmt.Errorf("cannot marshal block.header: %v", err)
	// }
	// if err := binary.Write(buf, binary.LittleEndian, uint64(len(headerData))); err != nil {
	// 	return buf.Bytes(), fmt.Errorf("cannot write block.header len: %v", err)
	// }
	// if err := binary.Write(buf, binary.LittleEndian, headerData); err != nil {
	// 	return buf.Bytes(), fmt.Errorf("cannot write block.header data: %v", err)
	// }
	// if err := binary.Write(buf, binary.LittleEndian, uint64(len(block.txs))); err != nil {
	// 	return buf.Bytes(), fmt.Errorf("cannot write block.txs len: %v", err)
	// }
	// if err := binary.Write(buf, binary.LittleEndian, block.txs); err != nil {
	// 	return buf.Bytes(), fmt.Errorf("cannot write block.txs data: %v", err)
	// }
	// if err := binary.Write(buf, binary.LittleEndian, uint64(len(block.plan))); err != nil {
	// 	return buf.Bytes(), fmt.Errorf("cannot write block.plan len: %v", err)
	// }
	// if err := binary.Write(buf, binary.LittleEndian, block.plan); err != nil {
	// 	return buf.Bytes(), fmt.Errorf("cannot write block.plan data: %v", err)
	// }
	// if err := binary.Write(buf, binary.LittleEndian, uint64(len(block.prevState))); err != nil {
	// 	return buf.Bytes(), fmt.Errorf("cannot write block.prevState len: %v", err)
	// }
	// if err := binary.Write(buf, binary.LittleEndian, block.prevState); err != nil {
	// 	return buf.Bytes(), fmt.Errorf("cannot write block.prevState data: %v", err)
	// }
	// return buf.Bytes(), nil
}

// UnmarshalBinary implements the `encoding.BinaryUnmarshaler` interface for the
// `Block` type.
func (block *Block) UnmarshalBinary(data []byte) error {
	return surge.FromBinary(data, block)
	// buf := bytes.NewBuffer(data)
	// if err := binary.Read(buf, binary.LittleEndian, &block.hash); err != nil {
	// 	return fmt.Errorf("cannot read block.hash: %v", err)
	// }
	// var numBytes uint64
	// if err := binary.Read(buf, binary.LittleEndian, &numBytes); err != nil {
	// 	return fmt.Errorf("cannot read block.header len: %v", err)
	// }
	// headerBytes := make([]byte, numBytes)
	// if _, err := buf.Read(headerBytes); err != nil {
	// 	return fmt.Errorf("cannot read block.header data: %v", err)
	// }
	// if err := block.header.UnmarshalBinary(headerBytes); err != nil {
	// 	return fmt.Errorf("cannot unmarshal block.header: %v", err)
	// }
	// if err := binary.Read(buf, binary.LittleEndian, &numBytes); err != nil {
	// 	return fmt.Errorf("cannot read block.txs len: %v", err)
	// }
	// if numBytes > 0 {
	// 	txsBytes := make([]byte, numBytes)
	// 	if _, err := buf.Read(txsBytes); err != nil {
	// 		return fmt.Errorf("cannot read block.txs data: %v", err)
	// 	}
	// 	block.txs = txsBytes
	// }
	// if err := binary.Read(buf, binary.LittleEndian, &numBytes); err != nil {
	// 	return fmt.Errorf("cannot read block.plan len: %v", err)
	// }
	// if numBytes > 0 {
	// 	planBytes := make([]byte, numBytes)
	// 	if _, err := buf.Read(planBytes); err != nil {
	// 		return fmt.Errorf("cannot read block.plan data: %v", err)
	// 	}
	// 	block.plan = planBytes
	// }
	// if err := binary.Read(buf, binary.LittleEndian, &numBytes); err != nil {
	// 	return fmt.Errorf("cannot read block.prevState len: %v", err)
	// }
	// if numBytes > 0 {
	// 	prevStateBytes := make([]byte, numBytes)
	// 	if _, err := buf.Read(prevStateBytes); err != nil {
	// 		return fmt.Errorf("cannot read block.prevState data: %v", err)
	// 	}
	// 	block.prevState = prevStateBytes
	// }
	// return nil
}
