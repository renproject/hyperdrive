package processutil_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestProcessutil(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Process Utilities Suite")
}
