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

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/renproject/hyperdrive"
	. "github.com/renproject/hyperdrive/testutil/replica"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/renproject/hyperdrive/block"
	"github.com/renproject/hyperdrive/replica"
	"github.com/renproject/hyperdrive/testutil"
	"github.com/renproject/id"
	"github.com/renproject/phi"
	"github.com/sirupsen/logrus"
)

var AllLoggers = map[int]bool{
	0: true,
	1: true,
	2: true,
	3: true,
	4: true,
	5: true,
	6: true,
}

func init() {
	mrand.Seed(time.Now().Unix())
}

var _ = Describe("Hyperdrive", func() {

	// Test parameters
	testShards := []int{
		1,
	}
	fs := []int{
		2,
	}

	// Test cases
	for _, numShards := range testShards {
		numShards := numShards
		shards := make([]Shard, numShards)
		for i := range shards {
			shards[i] = RandomShard()
		}

		Context(fmt.Sprintf("when there are %v shards", numShards), func() {
			for _, f := range fs {
				f := f

				Context(fmt.Sprintf("when f = %v (network have %v nodes)", f, 3*f+1), func() {
					Context("when all nodes have 100% live time", func() {
						Context("when all nodes start at same time", func() {
							It("should keep producing new blocks", func() {
								ctx, cancel := context.WithCancel(context.Background())
								defer cancel()

								network := NewNetwork(f, shards, nil, 100, 200)
								go network.Run(ctx, nil, false)

								// Expect the network should be handle nodes starting with different delays and
								Eventually(func() bool {
									return network.HealthCheck(nil)
								}, time.Minute).Should(BeTrue())
							})
						})

						Context("when each node has a random delay when starting", func() {
							It("should keep producing new blocks", func() {
								ctx, cancel := context.WithCancel(context.Background())
								defer cancel()

								network := NewNetwork(f, shards, nil, 100, 200)
								go network.Run(ctx, nil, true)

								// Expect the network should be handle nodes starting with different delays and
								Eventually(func() bool {
									return network.HealthCheck(nil)
								}, time.Minute).Should(BeTrue())
							})
						})
					})

					Context("when no more than one third nodes are offline at the beginning", func() {
						It("should keep producing new blocks", func() {
							ctx, cancel := context.WithCancel(context.Background())
							defer cancel()
							network := NewNetwork(f, shards, nil, 100, 200)

							// Start the network with random f nodes offline
							shuffledIndex := mrand.Perm(3*f + 1)
							offlineNodes := map[int]bool{}
							for i := 0; i < f; i++ {
								offlineNodes[shuffledIndex[i]] = true
							}

							go network.Run(ctx, offlineNodes, false)

							// Only check the nodes which are online are progressing after certain amount time
							Eventually(func() bool {
								return network.HealthCheck(shuffledIndex[f:])
							}, time.Duration(f*30)*time.Second).Should(BeTrue())
						})
					})

					Context("when some nodes are having network connection issue", func() {
						Context("when they go back online after certain amount of time", func() {
							It("should keep producing new blocks", func() {
								ctx, cancel := context.WithCancel(context.Background())
								defer cancel()
								network := NewNetwork(f, shards, nil, 100, 200)
								go network.Run(ctx, nil, false)
								time.Sleep(5 * time.Second)

								for i := 0; i < 3; i++ {
									offlineNum := mrand.Intn(f) + 1 // Making sure at least one node is offline
									shuffledIndex := mrand.Perm(3*f + 1)
									offlineNodes := shuffledIndex[:offlineNum]

									// Simulate connection issue for less than 1/3 nodes
									phi.ParForAll(offlineNodes, func(i int) {
										index := offlineNodes[i]
										network.DisableNode(index)
										time.Sleep(10 * time.Second)
										network.EnableNode(index)
									})

									Eventually(func() bool {
										return network.HealthCheck(nil)
									}, 30*time.Second).Should(BeTrue())
								}
							})
						})

						Context("when they fail to reconnect to the network", func() {
							It("should keep producing blocks with the rest of the networks", func() {
								ctx, cancel := context.WithCancel(context.Background())
								defer cancel()
								time.Sleep(5 * time.Second)

								network := NewNetwork(f, shards, nil, 100, 200)
								go network.Run(ctx, nil, false)

								// Random select
								shuffledIndex := mrand.Perm(3*f + 1)
								offlineNodes := shuffledIndex[:f]

								// Simulate connection issue for less than 1/3 nodes
								phi.ParForAll(offlineNodes, func(i int) {
									index := offlineNodes[i]
									log.Printf("node %v is having connectiong issue", index)
									network.DisableNode(index)
								})

								// Give them seconds to catch up
								Eventually(func() bool {
									return network.HealthCheck(shuffledIndex[f:])
								}, time.Minute).Should(BeTrue())
							})
						})
					})

					Context("when nodes are completely offline", func() {
						Context("when they go back online after some time", func() {
							It("should keep producing new blocks", func() {
								ctx, cancel := context.WithCancel(context.Background())
								defer cancel()
								network := NewNetwork(f, shards, nil, 100, 200)
								go network.Run(ctx, nil, false)
								time.Sleep(5 * time.Second)

								shuffledIndex := mrand.Perm(3*f + 1)
								offlineNodes := shuffledIndex[:f]

								// Simulate connection issue for less than 1/3 nodes
								phi.ParForAll(offlineNodes, func(i int) {
									index := offlineNodes[i]
									network.StopNode(index)

									time.Sleep(10 * time.Second)

									go network.StartNode(index)
								})

								Eventually(func() bool {
									return network.HealthCheck(nil)
								}, 30*time.Second).Should(BeTrue())

							})
						})

						Context("when they fail to reconnect to the network", func() {
							It("should keep producing new blocks", func() {
								ctx, cancel := context.WithCancel(context.Background())
								defer cancel()
								network := NewNetwork(f, shards, nil, 100, 200)
								go network.Run(ctx, nil, false)
								time.Sleep(5 * time.Second)

								shuffledIndex := mrand.Perm(3*f + 1)
								offlineNodes := shuffledIndex[:f]

								// Simulate connection issue for less than 1/3 nodes
								phi.ParForAll(offlineNodes, func(i int) {
									index := offlineNodes[i]
									network.StopNode(index)

									time.Sleep(10 * time.Second)
								})

								Eventually(func() bool {
									return network.HealthCheck(shuffledIndex[f:])
								}, 30*time.Second).Should(BeTrue())
							})
						})
					})

					Context("when more than f nodes fail to boot", func() {
						Context("when the failed node never come back", func() {
							It("should not process any blocks", func() {
								ctx, cancel := context.WithCancel(context.Background())
								defer cancel()
								network := NewNetwork(f, shards, nil, 100, 200)

								// Start the network with more than f nodes offline
								shuffledIndex := mrand.Perm(3*f + 1)
								offlineNodes := map[int]bool{}
								for i := 0; i < f+1; i++ {
									offlineNodes[shuffledIndex[i]] = true
								}

								go network.Run(ctx, offlineNodes, false)

								// expect all nodes only have the genesis block
								time.Sleep(30 * time.Second)
								Expect(network.HealthCheck(shuffledIndex[f:])).Should(BeFalse())
							})
						})

						Context("when the failed node come back online", func() {
							It("should start produce blocks", func() {
								ctx, cancel := context.WithCancel(context.Background())
								defer cancel()
								network := NewNetwork(f, shards, nil, 100, 200)

								// Start the network with more than f nodes offline
								shuffledIndex := mrand.Perm(3*f + 1)
								offlineNodes := map[int]bool{}
								for i := 0; i < f+1; i++ {
									offlineNodes[shuffledIndex[i]] = true
								}

								go network.Run(ctx, offlineNodes, false)

								time.Sleep(3 * time.Second)

								// Simulate connection issue for less than 1/3 nodes
								go phi.ParForAll(offlineNodes, func(i int) {
									network.StartNode(i)
								})

								Eventually(func() bool {
									return network.HealthCheck(nil)
								}, 300*time.Second).Should(BeTrue())
							})
						})
					})
				})
			}
		})
	}
})

type Network struct {
	mu    *sync.RWMutex
	Nodes []*Node

	F           int
	Shards      replica.Shards
	Context     context.Context
	Cancels     []context.CancelFunc
	Signatories id.Signatories
	DebugNodes  map[int]bool
	Broadcaster *MockBroadcaster
}

func NewNetwork(f int, shards replica.Shards, debugNodes map[int]bool, minDelay, maxDelay int) Network {
	if f <= 0 {
		panic("f must be positive")
	}
	total := 3*f + 1

	// Generate keys for all the nodes
	keys := make([]*ecdsa.PrivateKey, total)
	sigs := make([]id.Signatory, total)
	for i := range keys {
		pk, err := ecdsa.GenerateKey(crypto.S256(), rand.Reader)
		if err != nil {
			panic(err)
		}
		keys[i] = pk
		sig := id.NewSignatory(keys[i].PublicKey)
		sigs[i] = sig
	}

	// Initialize all nodes
	genesisBlock := testutil.GenesisBlock(sigs)
	broadcaster := NewMockBroadcaster(keys, minDelay, maxDelay)
	nodes := make([]*Node, total)
	for i := range nodes {
		logger := logrus.New()
		if debugNodes[i] {
			logger.SetLevel(logrus.DebugLevel)
		}
		store := NewMockPersistentStorage(shards)
		store.Init(genesisBlock)
		nodes[i] = NewNode(logger.WithField("node", i), shards, keys[i], broadcaster, store)
	}

	return Network{
		mu:    new(sync.RWMutex),
		Nodes: nodes,

		F:           f,
		Shards:      shards,
		Cancels:     make([]context.CancelFunc, 3*f+1),
		Signatories: sigs,
		DebugNodes:  debugNodes,
		Broadcaster: broadcaster,
	}
}

func (network *Network) Run(ctx context.Context, disableNodes map[int]bool, delayAtBeginning bool) {
	network.Context = ctx

	phi.ParForAll(network.Nodes, func(i int) {
		if disableNodes != nil && disableNodes[i] {
			return
		}

		// Add random delay before running the nodes.
		if delayAtBeginning {
			delay := time.Duration(mrand.Intn(5))
			time.Sleep(delay * time.Second)
		}

		network.startNode(i)
	})
}

func (network *Network) StartNode(i int) {
	logger := logrus.New()
	if network.DebugNodes[i] {
		logger.SetLevel(logrus.DebugLevel)
	}
	store := network.Nodes[i].Storage

	network.mu.Lock()
	network.Nodes[i] = NewNode(logger.WithField("node", i), network.Shards, network.Nodes[i].Key, network.Broadcaster, store)
	network.mu.Unlock()

	network.startNode(i)
}

func (network *Network) startNode(i int) {
	time.Sleep(time.Second) // waiting for previous test clean up

	// Get the node.
	network.mu.RLock()
	node := network.Nodes[i]
	network.mu.RUnlock()

	// Creating cancel for this node and enable its network connection
	innerCtx, cancel := context.WithCancel(network.Context)
	network.Cancels[i] = cancel
	network.Broadcaster.EnablePeer(node.Sig)

	node.Logger.Infof("💡 starting hyperdrive...")
	node.Hyperdrive.Start()
	defer node.Logger.Info("❌ shutting down hyperdrive...")

	messages := network.Broadcaster.Messages(node.Sig)
	for {
		select {
		case message := <-messages:
			node.Hyperdrive.HandleMessage(message)
			select {
			case <-innerCtx.Done():
				return
			default:
			}
		case <-innerCtx.Done():
			return
		}
	}
}

func (network *Network) StopNode(i int) {
	network.mu.RLock()
	defer network.mu.RUnlock()

	if network.Cancels[i] == nil {
		return
	}
	network.Cancels[i]()
	sig := network.Nodes[i].Sig
	network.Broadcaster.DisablePeer(sig)
}

// EnableNode enable the network connection to the node
func (network *Network) EnableNode(i int) {
	network.mu.RLock()
	defer network.mu.RUnlock()

	network.Nodes[i].Logger.Infof("💡 enable network connection... ")
	sig := network.Nodes[i].Sig
	network.Broadcaster.EnablePeer(sig)
}

func (network *Network) HealthCheck(indexes []int) bool {
	network.mu.RLock()
	defer network.mu.RUnlock()
	nodes := network.Nodes
	if indexes != nil {
		nodes = make([]*Node, 0, len(indexes))
		for _, index := range indexes {
			nodes = append(nodes, network.Nodes[index])
		}
	}

	// Check the block height of each nodes
	currentBlockHeights := make([]block.Height, len(nodes))
	for _, shard := range network.Shards {
		for i, node := range nodes {
			block := node.Storage.LatestBlock(shard)
			currentBlockHeights[i] = block.Header().Height()
		}
	}

	time.Sleep(4 * time.Second)

	for _, shard := range network.Shards {
		for i, node := range nodes {
			block := node.Storage.LatestBlock(shard)
			if block.Header().Height() <= currentBlockHeights[i] {
				log.Printf("⚠️ node %v didn't progress ,old height = %v, new height = %v", i, currentBlockHeights[i], block.Header().Height())
				return false
			}
		}
	}
	return true
}

// DisableNode block network connection to the node
func (network *Network) DisableNode(i int) {
	network.mu.RLock()
	defer network.mu.RUnlock()

	network.Nodes[i].Logger.Infof("⚠️ block network connection...")
	sig := network.Nodes[i].Sig
	network.Broadcaster.DisablePeer(sig)
}

type Node struct {
	Logger      logrus.FieldLogger
	Key         *ecdsa.PrivateKey
	Sig         id.Signatory
	Storage     *MockPersistentStorage
	Iterator    replica.BlockIterator
	Validator   replica.Validator
	Observer    replica.Observer
	Broadcaster *MockBroadcaster
	Shards      Shards
	Hyperdrive  Hyperdrive
}

func NewNode(logger logrus.FieldLogger, shards Shards, pk *ecdsa.PrivateKey, broadcaster *MockBroadcaster, store *MockPersistentStorage) *Node {
	sig := id.NewSignatory(pk.PublicKey)
	option := Options{
		Logger:      logger,
		BackOffExp:  1,
		BackOffBase: 3 * time.Second,
		BackOffMax:  3 * time.Second,
	}
	iter := NewMockBlockIterator(store)
	validator := NewMockValidator(store)
	observer := NewMockObserver(store)
	hd := New(option, store, store, iter, validator, observer, broadcaster, shards, *pk)

	return &Node{
		Logger:      logger,
		Key:         pk,
		Sig:         sig,
		Storage:     store,
		Iterator:    iter,
		Validator:   validator,
		Observer:    observer,
		Broadcaster: broadcaster,
		Shards:      shards,
		Hyperdrive:  hd,
	}
}
