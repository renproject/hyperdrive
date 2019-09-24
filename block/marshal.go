package block

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"

	"github.com/renproject/id"
)

// MarshalJSON implements the `json.Marshaler` interface for the Header type.
func (header Header) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Kind        Kind           `json:"kind"`
		ParentHash  id.Hash        `json:"parentHash"`
		BaseHash    id.Hash        `json:"baseHash"`
		Height      Height         `json:"height"`
		Round       Round          `json:"round"`
		Timestamp   Timestamp      `json:"timestamp"`
		Signatories id.Signatories `json:"signatories"`
	}{
		header.kind,
		header.parentHash,
		header.baseHash,
		header.height,
		header.round,
		header.timestamp,
		header.signatories,
	})
}

// UnmarshalJSON implements the `json.Unmarshaler` interface for the Header type.
func (header *Header) UnmarshalJSON(data []byte) error {
	tmp := struct {
		Kind        Kind           `json:"kind"`
		ParentHash  id.Hash        `json:"parentHash"`
		BaseHash    id.Hash        `json:"baseHash"`
		Height      Height         `json:"height"`
		Round       Round          `json:"round"`
		Timestamp   Timestamp      `json:"timestamp"`
		Signatories id.Signatories `json:"signatories"`
	}{}
	if err := json.Unmarshal(data, &tmp); err != nil {
		return err
	}
	header.kind = tmp.Kind
	header.parentHash = tmp.ParentHash
	header.baseHash = tmp.BaseHash
	header.height = tmp.Height
	header.round = tmp.Round
	header.timestamp = tmp.Timestamp
	header.signatories = tmp.Signatories
	return nil
}

// MarshalBinary implements the `encoding.BinaryMarshaler` interface for the
// Header type.
func (header Header) MarshalBinary() ([]byte, error) {
	buf := new(bytes.Buffer)
	if err := binary.Write(buf, binary.LittleEndian, header.kind); err != nil {
		return buf.Bytes(), fmt.Errorf("cannot write header.kind: %v", err)
	}
	if err := binary.Write(buf, binary.LittleEndian, header.parentHash); err != nil {
		return buf.Bytes(), fmt.Errorf("cannot write header.parentHash: %v", err)
	}
	if err := binary.Write(buf, binary.LittleEndian, header.baseHash); err != nil {
		return buf.Bytes(), fmt.Errorf("cannot write header.baseHash: %v", err)
	}
	if err := binary.Write(buf, binary.LittleEndian, header.height); err != nil {
		return buf.Bytes(), fmt.Errorf("cannot write header.height: %v", err)
	}
	if err := binary.Write(buf, binary.LittleEndian, header.round); err != nil {
		return buf.Bytes(), fmt.Errorf("cannot write header.round: %v", err)
	}
	if err := binary.Write(buf, binary.LittleEndian, header.timestamp); err != nil {
		return buf.Bytes(), fmt.Errorf("cannot write header.timestamp: %v", err)
	}
	if err := binary.Write(buf, binary.LittleEndian, uint64(len(header.signatories))); err != nil {
		return buf.Bytes(), fmt.Errorf("cannot write header.signatories len: %v", err)
	}
	for _, sig := range header.signatories {
		if err := binary.Write(buf, binary.LittleEndian, sig); err != nil {
			return buf.Bytes(), fmt.Errorf("cannot write header.signatories data: %v", err)
		}
	}
	return buf.Bytes(), nil
}

// UnmarshalBinary implements the `encoding.BinaryUnmarshaler` interface for the
// Header type.
func (header *Header) UnmarshalBinary(data []byte) error {
	buf := bytes.NewBuffer(data)
	if err := binary.Read(buf, binary.LittleEndian, &header.kind); err != nil {
		return fmt.Errorf("cannot read header.kind: %v", err)
	}
	if err := binary.Read(buf, binary.LittleEndian, &header.parentHash); err != nil {
		return fmt.Errorf("cannot read header.parentHash: %v", err)
	}
	if err := binary.Read(buf, binary.LittleEndian, &header.baseHash); err != nil {
		return fmt.Errorf("cannot read header.baseHash: %v", err)
	}
	if err := binary.Read(buf, binary.LittleEndian, &header.height); err != nil {
		return fmt.Errorf("cannot read header.height: %v", err)
	}
	if err := binary.Read(buf, binary.LittleEndian, &header.round); err != nil {
		return fmt.Errorf("cannot read header.round: %v", err)
	}
	if err := binary.Read(buf, binary.LittleEndian, &header.timestamp); err != nil {
		return fmt.Errorf("cannot read header.timestamp: %v", err)
	}
	var lenSignatories uint64
	if err := binary.Read(buf, binary.LittleEndian, &lenSignatories); err != nil {
		return fmt.Errorf("cannot read header.signatories len: %v", err)
	}
	if lenSignatories > 0 {
		header.signatories = make(id.Signatories, lenSignatories)
		for i := uint64(0); i < lenSignatories; i++ {
			if err := binary.Read(buf, binary.LittleEndian, &header.signatories[i]); err != nil {
				return fmt.Errorf("cannot read header.signatories data: %v", err)
			}
		}
	}
	return nil
}

// MarshalJSON implements the `json.Marshaler` interface for the Block type.
func (block Block) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Hash      id.Hash `json:"hash"`
		Header    Header  `json:"header"`
		Data      Data    `json:"data"`
		PrevState State   `json:"prevState"`
	}{
		block.hash,
		block.header,
		block.data,
		block.prevState,
	})
}

// UnmarshalJSON implements the `json.Unmarshaler` interface for the Block type.
func (block *Block) UnmarshalJSON(data []byte) error {
	tmp := struct {
		Hash      id.Hash `json:"hash"`
		Header    Header  `json:"header"`
		Data      Data    `json:"data"`
		PrevState State   `json:"prevState"`
	}{}
	if err := json.Unmarshal(data, &tmp); err != nil {
		return err
	}
	block.hash = tmp.Hash
	block.header = tmp.Header
	block.data = tmp.Data
	block.prevState = tmp.PrevState
	return nil
}

// MarshalBinary implements the `encoding.BinaryMarshaler` interface for the
// Block type.
func (block Block) MarshalBinary() ([]byte, error) {
	buf := new(bytes.Buffer)
	if err := binary.Write(buf, binary.LittleEndian, block.hash); err != nil {
		return buf.Bytes(), fmt.Errorf("cannot write block.hash: %v", err)
	}
	headerData, err := block.header.MarshalBinary()
	if err != nil {
		return buf.Bytes(), fmt.Errorf("cannot marshal block.header: %v", err)
	}
	if err := binary.Write(buf, binary.LittleEndian, uint64(len(headerData))); err != nil {
		return buf.Bytes(), fmt.Errorf("cannot write block.header len: %v", err)
	}
	if err := binary.Write(buf, binary.LittleEndian, headerData); err != nil {
		return buf.Bytes(), fmt.Errorf("cannot write block.header data: %v", err)
	}
	if err := binary.Write(buf, binary.LittleEndian, uint64(len(block.data))); err != nil {
		return buf.Bytes(), fmt.Errorf("cannot write block.data len: %v", err)
	}
	if err := binary.Write(buf, binary.LittleEndian, block.data); err != nil {
		return buf.Bytes(), fmt.Errorf("cannot write block.data data: %v", err)
	}
	if err := binary.Write(buf, binary.LittleEndian, uint64(len(block.prevState))); err != nil {
		return buf.Bytes(), fmt.Errorf("cannot write block.prevState len: %v", err)
	}
	if err := binary.Write(buf, binary.LittleEndian, block.prevState); err != nil {
		return buf.Bytes(), fmt.Errorf("cannot write block.prevState data: %v", err)
	}
	return buf.Bytes(), nil
}

// UnmarshalBinary implements the `encoding.BinaryUnmarshaler` interface for the
// Block type.
func (block *Block) UnmarshalBinary(data []byte) error {
	buf := bytes.NewBuffer(data)
	if err := binary.Read(buf, binary.LittleEndian, &block.hash); err != nil {
		return fmt.Errorf("cannot read block.hash: %v", err)
	}
	var numBytes uint64
	if err := binary.Read(buf, binary.LittleEndian, &numBytes); err != nil {
		return fmt.Errorf("cannot read block.header len: %v", err)
	}
	headerBytes := make([]byte, numBytes)
	if _, err := buf.Read(headerBytes); err != nil {
		return fmt.Errorf("cannot read block.header data: %v", err)
	}
	if err := block.header.UnmarshalBinary(headerBytes); err != nil {
		return fmt.Errorf("cannot unmarshal block.header: %v", err)
	}
	if err := binary.Read(buf, binary.LittleEndian, &numBytes); err != nil {
		return fmt.Errorf("cannot read block.data len: %v", err)
	}
	if numBytes > 0 {
		dataBytes := make([]byte, numBytes)
		if _, err := buf.Read(dataBytes); err != nil {
			return fmt.Errorf("cannot read block.data data: %v", err)
		}
		block.data = dataBytes
	}
	if err := binary.Read(buf, binary.LittleEndian, &numBytes); err != nil {
		return fmt.Errorf("cannot read block.prevState len: %v", err)
	}
	if numBytes > 0 {
		prevStateBytes := make([]byte, numBytes)
		if _, err := buf.Read(prevStateBytes); err != nil {
			return fmt.Errorf("cannot read block.prevState data: %v", err)
		}
		block.prevState = prevStateBytes
	}
	return nil
}
