package replica

import (
	"time"

	"github.com/renproject/hyperdrive/process"
	"github.com/renproject/hyperdrive/process/block"
	"github.com/renproject/hyperdrive/process/message"
	"github.com/sirupsen/logrus"
)

type BlockchainStorage interface {
	block.Blockchain

	LatestBlock() block.Block
	LatestBaseBlock() block.Block
}

type ProcessStateStorage interface {
	SaveProcessState(shard Shard, p process.Process)
	RestoreProcessState(shard Shard, p *process.Process)
}

type Shard [32]byte

type Options struct {
	// Logging
	Logger logrus.FieldLogger

	// Timeout options for proposing, prevoting, and precommiting
	BackOffExp  float64
	BackOffBase time.Duration
	BackOffMax  time.Duration
}

type Replica struct {
	shard    Shard
	p        process.Process
	pStorage ProcessStateStorage
}

func New(options Options, shard Shard, signatory block.Signatory, blockchain block.Blockchain, pStorage ProcessStateStorage) Replica {
	p := process.New(
		signatory,
		blockchain,
		process.DefaultState(),
		nil,
		nil,
		RoundRobinScheduler(nil),
		nil,
		BackOffTimer(options.BackOffExp, options.BackOffBase, options.BackOffMax),
		nil,
	)
	pStorage.RestoreProcessState(shard, &p)
	return Replica{
		shard:    shard,
		p:        p,
		pStorage: pStorage,
	}
}

func (replica *Replica) HandleMessage(m message.Message) {
	replica.p.HandleMessage(m)
	replica.pStorage.SaveProcessState(replica.shard, replica.p)
}
