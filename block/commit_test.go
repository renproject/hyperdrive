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

var _ = Describe("Commit Builder", func() {
	Context("when given a set of semi-random PreCommit", func() {
		It("should always return the most relevant Commit", func() {
			test := func(mock mockSignedPreCommit, testDrop bool) bool {

				builder := make(CommitBuilder)

				for _, signedPreCommit := range mock.votes {
					builder.Insert(signedPreCommit)
				}

				if testDrop {
					builder.Drop(mock.expectedCommit.Polka.Height)
				}

				commit, found := builder.Commit(mock.consensusThreshold)

				Expect(commit.String()).To(Equal(mock.expectedCommit.String()),
					"input SignedPreCommit: %v\nthreshold: %v\nexpectedFound: %v\nbuilder map: %v\nfound: %v",
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

type mockSignedPreCommit struct {
	votes              []SignedPreCommit
	expectedCommit     Commit
	expectedFound      bool
	consensusThreshold int64
}

//FIXME This test needs to be fixed like polka_test
func (mockSignedPreCommit) Generate(rand *rand.Rand, size int) reflect.Value {
	headersSeed := make([]sig.Hash, (rand.Intn(10) + 1))
	for i := range headersSeed {
		val, err := quick.Value(reflect.TypeOf(sig.Hash{}), rand)
		if !err {
			panic("reflect failed")
		}
		headersSeed[i] = val.Interface().(sig.Hash)
	}

	numCommits := rand.Uint32() % 50
	consensusThreshold := rand.Uint32() % 50

	signedPreCommit := make([]SignedPreCommit, numCommits)

	heighestCommit := Commit{}
	found := false

	scratch := make(map[Height]map[sig.Hash]uint32)

	for i := uint32(0); i < numCommits; i++ {

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
			scratch[height] = make(map[sig.Hash]uint32)
			scratch[height][hash] = 1
		}

		blk := Genesis()
		blk.Height = height
		blk.Header = hash
		polka := Polka{
			Block:  &blk,
			Round:  0,
			Height: height,
		}
		preCommit := PreCommit{
			Polka: polka,
		}

		newSV, err := ecdsa.NewFromRandom()
		if err != nil {
			panic("ecdsa failed")
		}
		signed, err := preCommit.Sign(newSV)
		if err != nil {
			panic("sign failed")
		}

		signedPreCommit[i] = signed

		if scratch[height][hash] >= consensusThreshold &&
			(heighestCommit.Polka.Height < height || !found) {
			heighestCommit = Commit{
				Polka: signed.PreCommit.Polka,
			}
			found = true
		}
	}

	return reflect.ValueOf(mockSignedPreCommit{
		votes:              signedPreCommit,
		expectedCommit:     heighestCommit,
		expectedFound:      found,
		consensusThreshold: int64(consensusThreshold),
	})
}
