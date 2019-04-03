package shard

import (
	"github.com/renproject/hyperdrive/v1/block"
	"github.com/renproject/hyperdrive/v1/sig"
)

type Shard struct {
	// Hash of the Shard, derived from the BlockHash and the sharding PRF.
	Hash sig.Hash
	// BlockHeader and BlockHeight on Ethereum when the Shard was formed.
	BlockHeader sig.Hash
	BlockHeight block.Height

	// Signatories maintaining the Shard. A Signatory is allowed to appear multiple times in the same Shard.
	Signatories sig.Signatories
}

// Leader returns the Signatory that leads the Shard for a given Round.
func (shard Shard) Leader(round block.Round) sig.Signatory {
	return shard.Signatories[int64(round)%int64(len(shard.Signatories))]
}

// Size returns the number of Signatories in the Shard.
func (shard Shard) Size() int {
	return len(shard.Signatories)
}

// ConsensusThreshold returns the number of honest Signatories that are required to reach consensus.
func (shard Shard) ConsensusThreshold() int {
	size := shard.Size()
	if size <= 3 {
		return size
	}
	return size - (size-1)/3
}

// Includes returns whether or not a given Signatory is one of the Signatories maintaining the Shard.
func (shard Shard) Includes(signatory sig.Signatory) bool {
	for i := range shard.Signatories {
		if signatory.Equal(shard.Signatories[i]) {
			return true
		}
	}
	return false
}
