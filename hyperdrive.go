package hyperdrive

import (
	"github.com/renproject/hyperdrive/block"
	"github.com/renproject/hyperdrive/id"
	"github.com/renproject/hyperdrive/replica"
)

type (
	Hashes            = id.Hashes
	Hash              = id.Hash
	Signatures        = id.Signatures
	Signature         = id.Signature
	Signatories       = id.Signatories
	Signatory         = id.Signatory
	Blocks            = block.Blocks
	Block             = block.Block
	Messages          = replica.Messages
	Message           = replica.Message
	Shards            = replica.Shards
	Shard             = replica.Shard
	Replicas          = replica.Replicas
	Replica           = replica.Replica
	ProcessStorage    = replica.ProcessStorage
	BlockStorage      = replica.BlockStorage
	BlockDataIterator = replica.BlockDataIterator
	Validator         = replica.Validator
	Observer          = replica.Observer
	Broadcaster       = replica.Broadcaster
)

// Hyperdrive manages multiple `replica.Replicas` from different
// `replica.Shards`.
type Hyperdrive interface {
	Rebase(sigs id.Signatories, shard replica.Shard)
	HandleMessage(message replica.Message)
}

type hyperdrive struct {
	replicas map[replica.Shard]replica.Replica
}

// New Hyperdrive.
func New() Hyperdrive {
	return &hyperdrive{
		replicas: map[replica.Shard]replica.Replica{},
	}
}

func (hyper *hyperdrive) Rebase(sigs id.Signatories, shard replica.Shard) {
}

func (hyper *hyperdrive) HandleMessage(message replica.Message) {
}
