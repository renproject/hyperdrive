package hyperdrive

import (
	"crypto/ecdsa"

	"github.com/renproject/hyperdrive/block"
	"github.com/renproject/hyperdrive/process"
	"github.com/renproject/hyperdrive/replica"
	"github.com/renproject/id"
)

// Re-export types.
type (
	Hashes         = id.Hashes
	Hash           = id.Hash
	Signatures     = id.Signatures
	Signature      = id.Signature
	Signatories    = id.Signatories
	Signatory      = id.Signatory
	Blocks         = block.Blocks
	Block          = block.Block
	Height         = block.Height
	Round          = block.Round
	Timestamp      = block.Timestamp
	BlockData      = block.Data
	BlockState     = block.State
	Messages       = replica.Messages
	Message        = replica.Message
	Shards         = replica.Shards
	Shard          = replica.Shard
	Options        = replica.Options
	Replicas       = replica.Replicas
	Replica        = replica.Replica
	ProcessStorage = replica.ProcessStorage
	BlockStorage   = replica.BlockStorage
	BlockIterator  = replica.BlockIterator
	Validator      = replica.Validator
	Observer       = replica.Observer
	Broadcaster    = replica.Broadcaster
	Blockchain     = process.Blockchain
	Process        = process.Process
	ProcessState   = process.State
)

// Re-export variables.
var (
	NewSignatory = id.NewSignatory

	StandardBlockKind = block.Standard
	RebaseBlockKind   = block.Rebase
	BaseBlockKind     = block.Base
	NewBlock          = block.New
	NewBlockHeader    = block.NewHeader
)

// Hyperdrive manages multiple `Replicas` from different
// `Shards`.
type Hyperdrive interface {
	Rebase(sigs Signatories)
	HandleMessage(message Message)
}

type hyperdrive struct {
	replicas map[Shard]Replica
}

// New Hyperdrive.
func New(options Options, pStorage ProcessStorage, blockStorage BlockStorage, blockIterator BlockIterator, validator Validator, observer Observer, broadcaster Broadcaster, shards Shards, privKey ecdsa.PrivateKey) Hyperdrive {
	replicas := make(map[Shard]Replica, len(shards))
	for _, shard := range shards {
		replicas[shard] = replica.New(options, pStorage, blockStorage, blockIterator, validator, observer, broadcaster, shard, privKey)
	}
	return &hyperdrive{
		replicas: replicas,
	}
}

func (hyper *hyperdrive) Rebase(sigs Signatories) {
	for shard, replica := range hyper.replicas {
		replica.Rebase(sigs)
		hyper.replicas[shard] = replica
	}
}

func (hyper *hyperdrive) HandleMessage(message Message) {
	replica, ok := hyper.replicas[message.Shard]
	if !ok {
		return
	}
	defer func() {
		hyper.replicas[message.Shard] = replica
	}()
	replica.HandleMessage(message)
}
