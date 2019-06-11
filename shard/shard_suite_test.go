package shard_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestShard(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Shard Suite")
}
