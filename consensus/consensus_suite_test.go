package consensus_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestConsensus(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Consensus Suite")
}
