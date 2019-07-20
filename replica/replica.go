package replica

import (
	"crypto/ecdsa"
	"time"

	"github.com/renproject/hyperdrive/block"
	"github.com/renproject/hyperdrive/message"
	"github.com/renproject/hyperdrive/process"
	"github.com/sirupsen/logrus"
)

// Shard uniquely identifies the Shard being maintained by the Replica.
type Shard [32]byte

// A Message sent/received by a Replica is composed of a Shard, and the
// underlying `message.Message` data.
type Message struct {
	Shard   Shard           `json:"shard"`
	Message message.Message `json:"message"`
}

// ProcessStorage saves and restores `process.State` to persistent memory. This
// guarantess that in the event of an unexpected shutdown, the Replica will only
// drop the `message.Message` that was currently being handling.
type ProcessStorage interface {
	SaveProcess(shard Shard, p process.Process)
	RestoreProcess(shard Shard, p *process.Process)
}

// Options define a set of properties that can be used to parameterise the
// Replica and its behaviour.
type Options struct {
	// Logging
	Logger logrus.FieldLogger

	// Timeout options for proposing, prevoting, and precommiting
	BackOffExp  float64
	BackOffBase time.Duration
	BackOffMax  time.Duration
}

type Replica struct {
	options      Options
	shard        Shard
	p            process.Process
	pStorage     ProcessStorage
	blockStorage BlockStorage
}

func New(options Options, shard Shard, blockRebaser BlockRebaser, blockStorage BlockStorage, pStorage ProcessStorage, broadcaster process.Broadcaster, privKey ecdsa.PrivateKey) Replica {
	latestBase := blockStorage.LatestBaseBlock()
	p := process.New(
		block.NewSignatory(privKey.PublicKey),
		blockStorage,
		process.DefaultState(),
		blockRebaser,
		blockRebaser,
		NewRoundRobinScheduler(latestBase.Header().Signatories()),
		NewSignerBroadcaster(broadcaster, privKey),
		NewBackOffTimer(options.BackOffExp, options.BackOffBase, options.BackOffMax),
		blockRebaser,
	)
	pStorage.RestoreProcess(shard, &p)
	return Replica{
		options:  options,
		shard:    shard,
		p:        p,
		pStorage: pStorage,
	}
}

func (replica *Replica) HandleMessage(m message.Message) {
	if err := message.Verify(m); err != nil {
		replica.options.Logger.Warnf("unverified message: %v", err)
		return
	}
	replica.p.HandleMessage(m)
	replica.pStorage.SaveProcess(replica.shard, replica.p)
}
