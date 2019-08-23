package testutil

import (
	"fmt"
	"math/rand"
	"reflect"
	"testing/quick"
	"time"

	"github.com/renproject/hyperdrive/block"
	"github.com/renproject/hyperdrive/id"
)

var r *rand.Rand

func init() {
	r = rand.New(rand.NewSource(time.Now().Unix()))
}

// RandomBytesSlice returns a random bytes slice.
func RandomBytesSlice() []byte {
	t := reflect.TypeOf([]byte{})
	value, ok := quick.Value(t, r)
	if !ok {
		panic(fmt.Sprintf("cannot generate random value of type %v", t.Name()))
	}
	return value.Interface().([]byte)
}

// RandomBlockKind returns a random valid block kind.
func RandomBlockKind() block.Kind {
	return block.Kind(rand.Intn(3) + 1)
}

// BlockHeaderJSON is almost a copy of the block.Header struct except all fields are exposed.
// This is for the convenience of initializing and marshaling.
type BlockHeaderJSON struct {
	Kind        block.Kind      `json:"kind"`
	ParentHash  id.Hash         `json:"parentHash"`
	BaseHash    id.Hash         `json:"baseHash"`
	Height      block.Height    `json:"height"`
	Round       block.Round     `json:"round"`
	Timestamp   block.Timestamp `json:"timestamp"`
	Signatories id.Signatories  `json:"signatories"`
}

// ToBlockHeader converts the BlockHeaderJSON object to a block.Header.
func (header BlockHeaderJSON) ToBlockHeader() block.Header {
	return block.NewHeader(
		header.Kind,
		header.ParentHash,
		header.BaseHash,
		header.Height,
		header.Round,
		header.Timestamp,
		header.Signatories,
	)
}

// RandomBlockHeaderJSON returns a valid BlockHeaderJSON of the given kind block.
func RandomBlockHeaderJSON(kind block.Kind) BlockHeaderJSON {
	parentHash := RandomHash()
	baseHash := RandomHash()
	height := block.Height(rand.Int63())
	round := block.Round(rand.Int63())
	timestamp := block.Timestamp(rand.Intn(int(time.Now().Unix())))
	var signatories id.Signatories
	switch kind {
	case block.Standard:
		signatories = nil
	case block.Rebase, block.Base:
		for len(signatories) == 0 {
			signatories = RandomSignatories()
		}
	}
	return BlockHeaderJSON{
		Kind:        kind,
		ParentHash:  parentHash,
		BaseHash:    baseHash,
		Height:      height,
		Round:       round,
		Timestamp:   timestamp,
		Signatories: signatories,
	}
}

// RandomBlockHeader generates a random block.Header of the given kind which
// guarantee to be valid.
func RandomBlockHeader(kind block.Kind) block.Header {
	return RandomBlockHeaderJSON(kind).ToBlockHeader()
}

// BlockHeaderJSON is almost a copy of the block.Header struct except all fields are exposed.
// This is for the convenience of initializing and marshaling.
type BlockJSON struct {
	Hash      id.Hash      `json:"hash"`
	Header    block.Header `json:"header"`
	Data      block.Data   `json:"data"`
	PrevState block.State  `json:"prevState"`
}

func RandomBlock(kind block.Kind) block.Block {
	header := RandomBlockHeader(kind)
	return block.New(header, RandomBytesSlice(), RandomBytesSlice())
}
