package tx_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestTx(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Tx Suite")
}
