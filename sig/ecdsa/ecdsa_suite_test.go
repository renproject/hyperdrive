package ecdsa_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestEcdsa(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "ECDSA Suite")
}
