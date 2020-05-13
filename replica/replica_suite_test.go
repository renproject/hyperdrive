package replica

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestReplica(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Replica Suite")
}
