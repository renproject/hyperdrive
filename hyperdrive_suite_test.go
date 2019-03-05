package hyperdrive_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestHyperdrive(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Hyperdrive Suite")
}
