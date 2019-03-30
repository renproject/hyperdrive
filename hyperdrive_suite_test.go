package hyperdrive_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestHyperdrive(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Hyperdrive Suite")
}
