package process_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestProcess(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Process Suite")
}
