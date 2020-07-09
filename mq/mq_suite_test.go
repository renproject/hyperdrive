package mq_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestMq(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Message Queue Suite")
}
