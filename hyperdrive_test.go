package hyperdrive_test

import (
	"context"
	"crypto/ecdsa"
	"crypto/rand"
	"fmt"
	"log"
	mrand "math/rand"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/renproject/hyperdrive/block"
	"github.com/renproject/hyperdrive/replica"
	"github.com/renproject/hyperdrive/testutil"
	"github.com/renproject/id"
	"github.com/renproject/phi"
	"github.com/sirupsen/logrus"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/renproject/hyperdrive"
	. "github.com/renproject/hyperdrive/testutil/replica"
)

func init() {
	seed := time.Now().Unix()
	log.Printf("seed = %v", seed)
	mrand.Seed(seed)
}

var _ = Describe("Hyperdrive", func() {

	table := []struct {
		shard int
		f     int
		r     int
	}{
		{1, 2, 0},
		{1, 2, 2},
	}

	for _, entry := range table {
		shards := make([]Shard, entry.shard)
		for i := range shards {
			shards[i] = RandomShard()
		}
		f := entry.f
		r := entry.r

		Context(fmt.Sprintf("when the network have %v signatory nodes (f = %v), %v non-signatory nodes and %v shards", 3*f+1, f, r, len(shards)), func() {
			Context("when all nodes have 100% live time", func() {
				Context("when all nodes start at same time", func() {
					It("should keep producing new blocks", func() {
						options := DefaultOption
						options.maxBootDelay = 0
						network := NewNetwork(f, r, shards, options)

						network.Start()
						defer network.Stop()

						Eventually(func() bool {
							return network.HealthCheck(nil)
						}, time.Minute).Should(BeTrue())
					})
				})

				Context("when each node has a random delay when starting", func() {
					It("should keep producing new blocks", func() {
						network := NewNetwork(f, r, shards, DefaultOption)
						network.Start()
						defer network.Stop()

						// Expect the network should be handle nodes starting with different delays and
						Eventually(func() bool {
							return network.HealthCheck(nil)
						}, time.Minute).Should(BeTrue())
					})
				})
			})

			Context("when f nodes are offline at the beginning", func() {
				It("should keep producing new blocks", func() {
					option := DefaultOption
					shuffledIndices := mrand.Perm(3*f + 1)
					option.disableNodes = shuffledIndices[:f]

					network := NewNetwork(f, r, shards, option)
					network.Start()
					defer network.Stop()

					// Only check the nodes which are online are progressing after certain amount time
					Eventually(func() bool {
						return network.HealthCheck(shuffledIndices[f:])
					}, time.Duration(f*30)*time.Second).Should(BeTrue())
				})
			})

			Context("when some nodes are having network connection issue", func() {
				Context("when they go back online after certain amount of time", func() {
					It("should keep producing new blocks", func() {
						network := NewNetwork(f, r, shards, DefaultOption)
						network.Start()
						defer network.Stop()

						numNodesOffline := mrand.Intn(f) + 1 // Making sure at least one node is offline
						shuffledIndices := mrand.Perm(3*f + 1)

						// Wait for all nodes reach consensus
						Eventually(func() bool {
							return network.HealthCheck(nil)
						}, 30*time.Second).Should(BeTrue())

						// Simulate connection issue for less than 1/3 nodes
						phi.ParForAll(numNodesOffline, func(i int) {
							index := shuffledIndices[i]
							network.BlockNodeConnection(index)
							SleepRandomSeconds(5, 10)
							network.UnblockNodeConnection(index)
						})

						Eventually(func() bool {
							return network.HealthCheck(nil)
						}, 30*time.Second).Should(BeTrue())
					})
				})

				Context("when they fail to reconnect to the network", func() {
					It("should keep producing blocks with the rest of the networks", func() {
						network := NewNetwork(f, r, shards, DefaultOption)
						network.Start()
						defer network.Stop()

						numNodesOffline := mrand.Intn(f) + 1 // Making sure at least one node is offline
						shuffledIndices := mrand.Perm(3*f + 1)

						// Wait for all nodes reach consensus
						Eventually(func() bool {
							return network.HealthCheck(nil)
						}, 30*time.Second).Should(BeTrue())

						// Simulate connection issue for less than 1/3 nodes
						phi.ParForAll(numNodesOffline, func(i int) {
							index := shuffledIndices[i]
							network.BlockNodeConnection(index)
						})

						Eventually(func() bool {
							return network.HealthCheck(shuffledIndices[numNodesOffline:])
						}, 30*time.Second).Should(BeTrue())
					})
				})
			})

			Context("when nodes are completely offline", func() {
				Context("when no more than f nodes crashed", func() {
					Context("when they go back online after some time", func() {
						It("should keep producing new blocks", func() {
							network := NewNetwork(f, r, shards, DefaultOption)
							network.Start()
							defer network.Stop()

							numNodesOffline := mrand.Intn(f) + 1 // Making sure at least one node is offline
							shuffledIndices := mrand.Perm(3*f + 1)

							// Wait for all nodes reach consensus
							Eventually(func() bool {
								return network.HealthCheck(nil)
							}, 30*time.Second).Should(BeTrue())

							// Simulate connection issue for less than 1/3 nodes
							phi.ParForAll(numNodesOffline, func(i int) {
								index := shuffledIndices[i]
								network.StopNode(index)
								SleepRandomSeconds(5, 10)
								network.StartNode(index)
							})

							Eventually(func() bool {
								return network.HealthCheck(nil)
							}, 30*time.Second).Should(BeTrue())
						})
					})

					Context("when they fail to reconnect to the network", func() {
						It("should keep producing new blocks", func() {
							network := NewNetwork(f, r, shards, DefaultOption)
							network.Start()
							defer network.Stop()

							numNodesOffline := mrand.Intn(f) + 1 // Making sure at least one node is offline
							shuffledIndices := mrand.Perm(3*f + 1)

							// Wait for all nodes reach consensus
							Eventually(func() bool {
								return network.HealthCheck(nil)
							}, 30*time.Second).Should(BeTrue())

							// Simulate connection issue for less than 1/3 nodes
							phi.ParForAll(numNodesOffline, func(i int) {
								index := shuffledIndices[i]
								network.StopNode(index)
								SleepRandomSeconds(5, 10)
								network.StartNode(index)
							})

							Eventually(func() bool {
								return network.HealthCheck(shuffledIndices[numNodesOffline:])
							}, 30*time.Second).Should(BeTrue())
						})
					})
				})

				Context("when more than f nodes crash,", func() {
					Context("when they fail to reconnect to the network", func() {
						It("should stop producing new blocks", func() {
							network := NewNetwork(f, r, shards, DefaultOption)
							network.Start()
							defer network.Stop()

							shuffledIndices := mrand.Perm(3*f + 1)
							crashedNodes := shuffledIndices[:f+1]

							// Wait for all nodes reach consensus
							Eventually(func() bool {
								return network.HealthCheck(nil)
							}, 30*time.Second).Should(BeTrue())

							// simulate the nodes crashed.
							phi.ParForAll(crashedNodes, func(i int) {
								index := crashedNodes[i]
								network.StopNode(index)
							})

							// expect the network not progressing
							time.Sleep(30 * time.Second)
							Expect(network.HealthCheck(shuffledIndices[f:])).Should(BeFalse())
						})
					})

					Context("when they successfully reconnect to the network", func() {
						It("should start producing blocks again", func() {
							network := NewNetwork(f, r, shards, DefaultOption)
							network.Start()
							defer network.Stop()

							// Wait for all nodes reach consensus
							Eventually(func() bool {
								return network.HealthCheck(nil)
							}, 30*time.Second).Should(BeTrue())

							// Crash f + 1 random nodes and expect no blocks produced after that
							shuffledIndices := mrand.Perm(3*f + 1)
							crashedNodes := shuffledIndices[:f+1]
							phi.ParForAll(crashedNodes, func(i int) {
								index := crashedNodes[i]
								network.StopNode(index)
							})
							Expect(network.HealthCheck(nil)).Should(BeFalse())

							// Restart the nodes after some time
							phi.ParForAll(crashedNodes, func(i int) {
								SleepRandomSeconds(5, 10)
								index := crashedNodes[i]
								network.StartNode(index)
							})

							Eventually(func() bool {
								return network.HealthCheck(nil)
							}, time.Minute).Should(BeTrue())
						})
					})
				})
			})

			Context("when more than f nodes fail to boot", func() {
				Context("when the failed node never come back", func() {
					It("should not process any blocks", func() {
						// Start the network with more than f nodes offline
						options := DefaultOption
						shuffledIndices := mrand.Perm(3*f + 1)
						options.disableNodes = shuffledIndices[:f+1]

						network := NewNetwork(f, r, shards, options)
						network.Start()
						defer network.Stop()

						// expect all nodes only have the genesis block
						time.Sleep(20 * time.Second)
						Expect(network.HealthCheck(shuffledIndices[f:])).Should(BeFalse())
					})
				})

				Context("when the failed node come back online", func() {
					It("should start produce blocks", func() {
						// Start the network with more than f nodes offline
						options := DefaultOption
						shuffledIndices := mrand.Perm(3*f + 1)
						options.disableNodes = shuffledIndices[:f+1]

						network := NewNetwork(f, r, shards, options)
						network.Start()
						defer network.Stop()

						phi.ParForAll(shuffledIndices[:f+1], func(i int) {
							index := shuffledIndices[i]
							network.StartNode(index)
						})

						Eventually(func() bool {
							return network.HealthCheck(nil)
						}, 30*time.Second).Should(BeTrue())
					})
				})
			})
		})
	}
})

type networkOptions struct {
	minNetworkDelay int   // minimum network latency when sending messages in milliseconds
	maxNetworkDelay int   // maximum network latency when sending messages in milliseconds
	minBootDelay    int   // minimum delay when booting the node in seconds
	maxBootDelay    int   // maximum delay when booting the node in seconds
	debugLogger     []int // indexes of the nodes which we want to enable the debug logger, nil for disable all
	disableNodes    []int // indexes of the nodes which we want to disable at the starting of the network, nil for enable all

}

var DefaultOption = networkOptions{
	minNetworkDelay: 100,
	maxNetworkDelay: 500,
	minBootDelay:    0,
	maxBootDelay:    3,
	debugLogger:     nil,
	disableNodes:    nil,
}

type Network struct {
	f       int
	r       int
	shards  replica.Shards
	options networkOptions

	nodesMu *sync.RWMutex
	nodes   []*Node

	context      context.Context
	cancel       context.CancelFunc
	nodesCancels []context.CancelFunc
	keys         []*ecdsa.PrivateKey
	Broadcaster  *MockBroadcaster
}

func NewNetwork(f, r int, shards replica.Shards, options networkOptions) Network {
	if f <= 0 {
		panic("f must be positive")
	}
	total := (3*f + 1) + r

	// Generate keys for all the nodes
	keys := make([]*ecdsa.PrivateKey, total)
	sigs := make([]id.Signatory, total)
	for i := range keys {
		pk, err := ecdsa.GenerateKey(crypto.S256(), rand.Reader)
		if err != nil {
			panic(err)
		}
		sig := id.NewSignatory(pk.PublicKey)
		keys[i], sigs[i] = pk, sig
	}

	// Initialize all nodes
	genesisBlock := testutil.GenesisBlock(sigs[:3*f+1])
	broadcaster := NewMockBroadcaster(keys, options.minNetworkDelay, options.maxNetworkDelay)
	nodes := make([]*Node, total)
	for i := range nodes {
		logger := logrus.New()
		if Contain(options.debugLogger, i) {
			logger.SetLevel(logrus.DebugLevel)
		}
		store := NewMockPersistentStorage(shards)
		store.Init(genesisBlock)
		nodes[i] = NewNode(logger.WithField("node", i), shards, keys[i], broadcaster, store, i < 3*f+1)
	}
	ctx, cancel := context.WithCancel(context.Background())

	return Network{
		f:            f,
		r:            r,
		shards:       shards,
		options:      options,
		nodesMu:      new(sync.RWMutex),
		nodes:        nodes,
		context:      ctx,
		cancel:       cancel,
		keys:         keys,
		nodesCancels: make([]context.CancelFunc, len(nodes)),
		Broadcaster:  broadcaster,
	}
}

func (network Network) Start() {
	phi.ParForAll(network.nodes, func(i int) {
		if Contain(network.options.disableNodes, i) {
			return
		}
		SleepRandomSeconds(network.options.minBootDelay, network.options.maxBootDelay)

		network.nodesMu.RLock()
		defer network.nodesMu.RUnlock()
		network.startNode(i)
	})
}

func (network Network) Stop() {
	for _, cancel := range network.nodesCancels {
		if cancel != nil {
			cancel()
		}
	}
	network.cancel()
	time.Sleep(time.Second)
}

func (network Network) Signatories() id.Signatories {
	sigs := make(id.Signatories, len(network.nodes))
	for i := range sigs {
		sigs[i] = id.NewSignatory(network.keys[i].PublicKey)
	}
	return sigs
}

func (network *Network) StartNode(i int) {
	logger := logrus.New()
	if Contain(network.options.debugLogger, i) {
		logger.SetLevel(logrus.DebugLevel)
	}
	store := network.nodes[i].storage

	network.nodesMu.Lock()
	defer network.nodesMu.Unlock()

	network.nodes[i] = NewNode(logger.WithField("node", i), network.shards, network.nodes[i].key, network.Broadcaster, store, i >= network.r)
	network.startNode(i)
}

func (network *Network) startNode(i int) {
	// Creating cancel for this node and enable its network connection
	node := network.nodes[i]
	innerCtx, cancel := context.WithCancel(network.context)
	network.nodesCancels[i] = cancel
	network.Broadcaster.EnablePeer(node.Signatory())

	// Start the node
	node.logger.Infof("💡starting hyperdrive...")
	node.hyperdrive.Start()

	// Start reading messages from the broadcaster
	messages := network.Broadcaster.Messages(node.Signatory())
	hyperdrive, logger := node.hyperdrive, node.logger

	go func() {
		defer logger.Info("❌ shutting down hyperdrive...")

		for {
			select {
			case message := <-messages:
				hyperdrive.HandleMessage(message)
				select {
				case <-innerCtx.Done():
					return
				default:
				}
			case <-innerCtx.Done():
				return
			}
		}
	}()
}

func (network *Network) StopNode(i int) {
	if network.nodesCancels[i] == nil {
		return
	}
	network.nodesCancels[i]()
	network.Broadcaster.DisablePeer(network.Signatories()[i])
}

func (network *Network) UnblockNodeConnection(i int) {
	network.nodes[i].logger.Infof("💡 enable network connection... ")
	network.Broadcaster.EnablePeer(network.Signatories()[i])
}

func (network *Network) BlockNodeConnection(i int) {
	network.nodes[i].logger.Infof("⚠️ block network connection...")
	network.Broadcaster.DisablePeer(network.Signatories()[i])
}

// Check the nodes of given indexes are working together producing new blocks.
func (network *Network) HealthCheck(indexes []int) bool {
	network.nodesMu.RLock()
	defer network.nodesMu.RUnlock()

	nodes := network.nodes
	if indexes != nil {
		nodes = make([]*Node, 0, len(indexes))
		for _, index := range indexes {
			nodes = append(nodes, network.nodes[index])
		}
	}

	// Check the block height of each nodes
	currentBlockHeights := make([]block.Height, len(nodes))
	for _, shard := range network.shards {
		for i, node := range nodes {
			if node.observer.IsSignatory(shard) {
				block := node.storage.LatestBlock(shard)
				currentBlockHeights[i] = block.Header().Height()
			}
		}
	}

	time.Sleep(5 * time.Second)

	for _, shard := range network.shards {
		for i, node := range nodes {
			if node.observer.IsSignatory(shard) {
				block := node.storage.LatestBlock(shard)
				if block.Header().Height() <= currentBlockHeights[i] {
					log.Printf("⚠️ node %v didn't progress ,old height = %v, new height = %v", i, currentBlockHeights[i], block.Header().Height())
					return false
				}
			}
		}
	}
	return true
}

type Node struct {
	logger     logrus.FieldLogger
	key        *ecdsa.PrivateKey
	storage    *MockPersistentStorage
	iter       replica.BlockIterator
	validator  replica.Validator
	observer   replica.Observer
	hyperdrive Hyperdrive
}

func (node Node) Signatory() id.Signatory {
	return id.NewSignatory(node.key.PublicKey)
}

func NewNode(logger logrus.FieldLogger, shards Shards, pk *ecdsa.PrivateKey, broadcaster *MockBroadcaster, store *MockPersistentStorage, isSignatory bool) *Node {
	option := Options{
		Logger:      logger,
		BackOffExp:  1,
		BackOffBase: 3 * time.Second,
		BackOffMax:  3 * time.Second,
	}
	iter := NewMockBlockIterator(store)
	validator := NewMockValidator(store)
	observer := NewMockObserver(store, isSignatory)
	hd := New(option, store, store, iter, validator, observer, broadcaster, shards, *pk)

	return &Node{
		logger:     logger,
		key:        pk,
		storage:    store,
		iter:       iter,
		validator:  validator,
		observer:   observer,
		hyperdrive: hd,
	}
}
