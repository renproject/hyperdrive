package hyperdrive_test

import (
	"context"
	"crypto/ecdsa"
	"crypto/rand"
	"fmt"
	"log"
	mrand "math/rand"
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

var FirstLoggerOnly = map[int]bool{3: true}

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
						FIt("should keep producing new blocks", func() {
							ctx, cancel := context.WithCancel(context.Background())
							defer cancel()
							network := NewNetwork(f, shards, FirstLoggerOnly, 100, 200)
							go network.Run(ctx)

							// Expect all nodes reach block height 30 after 30 seconds
							time.Sleep(30 * time.Second)
							for _, shard := range shards {
								for _, node := range network.Nodes {
									block := node.Storage.LatestBlock(shard)
									Expect(block.Header().Height()).Should(BeNumerically(">=", 30))
								}
							}
						})
					})

					Context("when less than one third nodes are offline at the beginning", func() {
						It("should keep producing new blocks", func() {
							for offlineNum := 1; offlineNum <= f; offlineNum++ {
								log.Printf("when there is %v node offline in the network", offlineNum)
								ctx, cancel := context.WithCancel(context.Background())
								defer cancel()
								network := NewNetwork(f, shards, nil, 100, 200)

								// Start the network with certain number of nodes offline
								shuffledIndex := mrand.Perm(3*f + 1)
								offlineNodes := shuffledIndex[:offlineNum]
								for _, index := range offlineNodes {
									log.Print("shutting down node ", index)
									network.DropNode(index)
								}

								go network.Run(ctx)

								// Expect all nodes reach block height 30 after 30 seconds
								time.Sleep(time.Duration(offlineNum*30) * time.Second)
								for _, shard := range shards {
									// Only check the nodes which are not online
									onlineNodes := shuffledIndex[offlineNum:]
									for _, index := range onlineNodes {
										node := network.Nodes[index]
										block := node.Storage.LatestBlock(shard)
										Expect(block.Header().Height()).Should(BeNumerically(">=", 10))
									}
								}
							}
						})
					})

					// Context("when less than one third nodes are offline at the beginning", func() {
					// 	It("should keep producing new blocks", func() {
					// 		ctx, cancel := context.WithTimeout(context.Background(), time.Hour)
					// 		defer cancel()
					// 		network := NewNetwork(f, shards, nil, 100, 200)
					//
					// 		// Start the network with [0,f] number of nodes offline
					// 		offlineNum := mrand.Intn(f) + 1 // Making sure at least one node is offline
					// 		shuffledIndex := mrand.Perm(3*f + 1)
					// 		offlineNodes := shuffledIndex[:offlineNum]
					// 		for _, index := range offlineNodes {
					// 			log.Print("shutting down node ", index)
					// 			network.DropNode(index)
					// 		}
					//
					// 		go network.Run(ctx)
					//
					// 		// Expect all nodes reach block height 30 after 30 seconds
					// 		time.Sleep(time.Minute)
					// 		for _, shard := range shards {
					// 			// Only check the nodes which are not online
					// 			onlineNodes := shuffledIndex[offlineNum:]
					// 			for _, index := range onlineNodes {
					// 				node := network.Nodes[index]
					// 				block := node.Storage.LatestBlock(shard)
					// 				Expect(block.Header().Height()).Should(BeNumerically(">=", 10))
					// 			}
					// 		}
					// 	})
					// })

				})
			}
		})
	}
})

type Network struct {
	F            int
	Shards       replica.Shards
	GenesisBlock block.Block
	Nodes        []*Node
	Broadcaster  *MockBroadcaster
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

	// Generate the genesis block
	parentHash, baseHash := testutil.RandomHash(), testutil.RandomHash()
	header := block.NewHeader(block.Base, parentHash, baseHash, block.Height(0), block.Round(0), block.Timestamp(time.Now().Unix()), sigs)
	genesisBlock := block.New(header, nil, nil)

	broadcaster := NewMockBroadcaster(keys, minDelay, maxDelay)
	nodes := make([]*Node, total)
	for i := range nodes {
		logger := logrus.New()
		if debugNodes == nil || debugNodes[i] {
			logger.SetLevel(logrus.DebugLevel)
		}
		nodes[i] = NewNode(logger.WithField("node", i), shards, keys[i], broadcaster, genesisBlock)
	}

	return Network{
		F:            f,
		Shards:       shards,
		GenesisBlock: genesisBlock,
		Nodes:        nodes,
		Broadcaster:  broadcaster,
	}
}

func (network Network) Run(ctx context.Context) {
	phi.ParForAll(network.Nodes, func(i int) {
		node := network.Nodes[i]

		// // Add random delay before running the nodes.
		// delay := time.Duration(mrand.Intn(10))
		// log.Printf("starting node %v in %v seconds", i, delay)
		// time.Sleep(delay * time.Second)
		if i == 3 {
			time.Sleep(10 * time.Second)
		}

		log.Printf("starting node %v", i)
		network.Broadcaster.EnablePeer(node.Sig)
		node.Hyperdrive.Run(ctx)

		messages := network.Broadcaster.Messages(node.Sig)
		for {
			select {
			case message := <-messages:
				node.Hyperdrive.HandleMessage(message)
			case <-ctx.Done():
				return
			}
		}
	})
}

// DropNode re-enbale the connection of the node with given index in the network
func (network Network) StartNode(i int) {
	sig := network.Nodes[i].Sig
	network.Broadcaster.EnablePeer(sig)
}

// DropNode shuts down the node with given index in the network
func (network Network) DropNode(i int) {
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

func NewNode(logger logrus.FieldLogger, shards Shards, pk *ecdsa.PrivateKey, broadcaster *MockBroadcaster, gb block.Block) *Node {
	sig := id.NewSignatory(pk.PublicKey)
	store := NewMockPersistentStorage(shards)
	store.Init(gb)
	option := Options{
		Logger:      logger,
		BackOffExp:  1,
		BackOffBase: 5 * time.Second,
		BackOffMax:  5 * time.Second,
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
