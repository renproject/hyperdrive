package block_test

import (
	"math"
	"math/rand"
	"reflect"
	"testing/quick"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/renproject/hyperdrive/block"
	"github.com/renproject/hyperdrive/sig"
)

var conf = quick.Config{
	MaxCount: 256,
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

				Expect(polka.Equal(mock.expectedPolka) &&
					(found == mock.expectedFound)).Should(BeTrue(),
					"input SignedPreVotes: %+v\n\nfound: %+v\n\nexpectedFound: %+v\n\nbuilder map: %+v\n\nthreshold: %+v\n\npolka: %+v\n\nexpectedPolka: %+v\n\npolka signatures: %v\n\npolka signatories:%v\n\nexpectedPolka signatures: %v\n\n expectedPolka signatories: %v\n",
					mock.votes,
					found,
					mock.expectedFound,
					builder,
					mock.consensusThreshold,
					polka,
					mock.expectedPolka,
					polka.Signatures,
					polka.Signatories,
					mock.expectedPolka.Signatures,
					mock.expectedPolka.Signatories,
				)

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
	consensusThreshold int64
	scratch            map[sig.Hash]*tuple
}

type tuple struct {
	num   uint32
	polka *Polka
}

func (mockPreVotes) Generate(rand *rand.Rand, size int) reflect.Value {

	// pick how many blocks you want to generate
	blocks := make([]Block, (rand.Intn(40) + 1))

	// Decide how many PreVotes you want to send
	numPreVotes := rand.Uint32() % 50

	// Decide on the max height (must be less than the number of
	// blocks)
	//
	// essentially the height range is 0 to maxHeight
	maxHeight := rand.Int() % len(blocks)
	if maxHeight == 0 {
		maxHeight++
	}

	for i := range blocks {
		// Now we will put at least on block for each height in the
		// blocks array and some duplicate heights.
		blocks[i] = GenerateBlock(rand, Height(i%maxHeight))
	}

	// Store for SignedPreVote
	signedPreVotes := make([]SignedPreVote, numPreVotes)

	var heighestPolka *Polka
	heighestPolka = &Polka{}
	found := false

	scratch := make(map[sig.Hash]*tuple)

	// This logic is to both have multiple blocks at the same height,
	// but at the same time prevent any one height from having more
	// than maxPreVotesAtGivenHeight SignedPreVote. This is so I can have a
	// sensible consensusThreshold
	maxPreVotesAtGivenHeight := uint32(math.Ceil(float64(numPreVotes) / float64(maxHeight)))
	preVotesAtGivenHeight := make([]uint32, maxHeight)

	// An approximation for 2/3 of the votes at a given height needed
	// to consider a block valid
	consensusThreshold := uint32(math.Ceil(float64(maxPreVotesAtGivenHeight*2) / float64(3)))

	for i := uint32(0); i < numPreVotes; i++ {

		randBlockIndex := rand.Intn(len(blocks))
		block := blocks[randBlockIndex]
		for preVotesAtGivenHeight[block.Height] >= maxPreVotesAtGivenHeight {
			randBlockIndex = (randBlockIndex + 1) % len(blocks)
			block = blocks[randBlockIndex]
		}
		preVotesAtGivenHeight[block.Height]++

		signed := GenerateSignedPreVote(&block)
		signedPreVotes[i] = signed

		if _, ok := scratch[block.Header]; !ok {
			sigs := make(sig.Signatures, 1)
			sigs[0] = signed.Signature
			signa := make(sig.Signatories, 1)
			signa[0] = signed.Signatory
			scratch[block.Header] = &tuple{
				num: 1,
				polka: &Polka{
					Block:       &block,
					Round:       block.Round,
					Height:      block.Height,
					Signatures:  sigs,
					Signatories: signa,
				},
			}
		} else {
			tuple := scratch[block.Header]
			tuple.num++
			tuple.polka.Signatures =
				append(tuple.polka.Signatures,
					signed.Signature)
			tuple.polka.Signatories =
				append(tuple.polka.Signatories,
					signed.Signatory)
		}

		if scratch[block.Header].num >= consensusThreshold &&
			(heighestPolka.Height < block.Height || !found) {
			heighestPolka = scratch[block.Header].polka
			found = true
		}
	}

	return reflect.ValueOf(mockPreVotes{
		votes:              signedPreVotes,
		expectedPolka:      *heighestPolka,
		expectedFound:      found,
		consensusThreshold: int64(consensusThreshold),
		scratch:            scratch,
	})
}
