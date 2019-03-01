package sig_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestSig(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Sig Suite")
}
