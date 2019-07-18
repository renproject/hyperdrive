package block

import "bytes"

type Hash [32]byte

func (hash Hash) Equal(other Hash) bool {
	return bytes.Equal(hash[:], other[:])
}

type Block interface {
	Hash() Hash
	Equal(Block) bool
}

type Height int64

type Round int64

var (
	InvalidHash   = Hash([32]byte{})
	InvalidBlock  = Block(nil)
	InvalidRound  = Round(-1)
	InvalidHeight = Height(-1)
)

type Blockchain interface {
	InsertBlock(Height, Block)
	Block(Height) Block
}
