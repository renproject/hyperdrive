package block_test

import (
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

				Expect(polka.Signatures.Equal(mock.expectedPolka.Signatures)).Should(BeTrue(), "Signatures not equal:\n\t%v\n\n\t%v\n\ninput SignedPreVotes\n\t%v\n\npolka:\n\t%+v\n\nexpectedPolka:\n\t%+v",
					polka.Signatures,
					mock.expectedPolka.Signatures,
					mock.votes,
					polka,
					mock.expectedPolka,
				)

				Expect(polka.Equal(mock.expectedPolka) &&
					(found == mock.expectedFound)).Should(BeTrue(),
					"input SignedPreVotes: %+v\nthreshold: %+v\nexpectedFound: %+v\nbuilder map: %+v\nfound: %+v\n polka: %+v\nexpectedPolka: %+v",
					mock.votes,
					mock.consensusThreshold,
					mock.expectedFound,
					builder,
					found,
					polka,
					mock.expectedPolka)

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
	blocks := make([]Block, (rand.Intn(4) + 1))

	for i := range blocks {
		//FIXME should generate multiple blocks for the same height
		blocks[i] = GenerateBlock(rand, Height(rand.Intn(len(blocks))))
	}

	numPreVotes := rand.Uint32() % 5
	consensusThreshold := rand.Uint32() % 5

	signedPreVotes := make([]SignedPreVote, numPreVotes)

	var heighestPolka *Polka
	heighestPolka = &Polka{}
	found := false

	scratch := make(map[sig.Hash]*tuple)

	for i := uint32(0); i < numPreVotes; i++ {

		block := blocks[rand.Intn(len(blocks))]

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
