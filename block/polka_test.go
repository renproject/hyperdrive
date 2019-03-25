package block_test

import (
	"math/rand"
	"reflect"
	"testing/quick"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/renproject/hyperdrive/block"
	"github.com/renproject/hyperdrive/sig"
	"github.com/renproject/hyperdrive/sig/ecdsa"
)

var conf = quick.Config{
	MaxCount:      256,
	MaxCountScale: 0,
	Rand:          nil,
	Values:        nil,
}

var _ = Describe("Polka Builder", func() {
	Context("when given a set of semi-random PreVote", func() {
		It("should always return the most relevant Polka", func() {
			test := func(mock mockPreVotes, testDrop bool) bool {

				builder := make(PolkaBuilder)

				for _, signedPreVote := range mock.votes {
					builder.Insert(signedPreVote)
				}

				if testDrop {
					builder.Drop(mock.expectedPolka.Height)
				}

				polka, found := builder.Polka(mock.consensusThreshold)

				Expect(polka.String()).To(Equal(mock.expectedPolka.String()),
					"input PreVotes: %v\nthreshold: %v\nexpectedFound: %v\nbuilder map: %v\nfound: %v",
					mock.votes,
					mock.consensusThreshold,
					mock.expectedFound,
					builder,
					found)
				Expect(found).To(Equal(mock.expectedFound))

				return true
			}
			Expect(quick.Check(test, &conf)).ShouldNot(HaveOccurred())
		})
	})
})

type mockPreVotes struct {
	votes              []SignedPreVote
	expectedPolka      Polka
	expectedFound      bool
	consensusThreshold int
}

func (mockPreVotes) Generate(rand *rand.Rand, size int) reflect.Value {
	headersSeed := make([]sig.Hash, (rand.Intn(10) + 1))
	for i := range headersSeed {
		val, err := quick.Value(reflect.TypeOf(sig.Hash{}), rand)
		if !err {
			panic("reflect failed")
		}
		headersSeed[i] = val.Interface().(sig.Hash)
	}

	numPreVotes := rand.Int() % 50
	consensusThreshold := rand.Int() % 50

	signedPreVotes := make([]SignedPreVote, numPreVotes)

	heighestPolka := Polka{}
	found := false

	scratch := make(map[Height]map[sig.Hash]int)

	for i := 0; i < numPreVotes; i++ {

		randHashIndex := rand.Intn(len(headersSeed))
		hash := headersSeed[randHashIndex]
		height := Height(randHashIndex)

		if val, ok := scratch[height]; ok {
			if _, ok := val[hash]; !ok {
				val[hash] = 1
			} else {
				val[hash]++
			}
		} else {
			scratch[height] = make(map[sig.Hash]int)
			scratch[height][hash] = 1
		}

		blk := Genesis()
		blk.Height = height
		blk.Header = hash
		preVote := NewPreVote(&blk, 0, height)

		newSV, err := ecdsa.NewFromRandom()
		if err != nil {
			panic("ecdsa failed")
		}
		signed, err := preVote.Sign(newSV)
		if err != nil {
			panic("sign failed")
		}

		signedPreVotes[i] = signed

		if scratch[height][hash] >= consensusThreshold &&
			(heighestPolka.Height < height || !found) {
			heighestPolka = Polka{
				Block:  &blk,
				Round:  0,
				Height: height,
			}
			found = true
		}
	}

	return reflect.ValueOf(mockPreVotes{
		votes:              signedPreVotes,
		expectedPolka:      heighestPolka,
		expectedFound:      found,
		consensusThreshold: consensusThreshold,
	})
}
