package timer_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestTimer(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Timer Suite")
}
