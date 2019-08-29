package testutil

import (
	"crypto/rand"
	"fmt"
	mrand "math/rand"
	"time"

	"github.com/renproject/id"
)

func init() {
	mrand.Seed(time.Now().Unix())
}

func RandomHash() id.Hash {
	hash := id.Hash{}
	_, err := rand.Read(hash[:])
	if err != nil {
		panic(fmt.Sprintf("cannot create random hash, err = %v", err))
	}
	return hash
}

func RandomHashes() id.Hashes {
	length := mrand.Intn(30)
	hashes := make(id.Hashes, length)
	for i := 0; i < length; i++ {
		_, err := rand.Read(hashes[i][:])
		if err != nil {
			panic(fmt.Sprintf("cannot create random hash, err = %v", err))
		}
	}
	return hashes
}

func RandomSignature() id.Signature {
	signature := id.Signature{}
	_, err := rand.Read(signature[:])
	if err != nil {
		panic(fmt.Sprintf("cannot create random signature, err = %v", err))
	}
	return signature
}

func RandomSignatures() id.Signatures {
	length := mrand.Intn(30)
	sigs := make(id.Signatures, length)
	for i := 0; i < length; i++ {
		_, err := rand.Read(sigs[i][:])
		if err != nil {
			panic(fmt.Sprintf("cannot create random signature, err = %v", err))
		}
	}
	return sigs
}

func RandomSignatory() id.Signatory {
	signatory := id.Signatory{}
	_, err := rand.Read(signatory[:])
	if err != nil {
		panic(fmt.Sprintf("cannot create random signatory, err = %v", err))
	}
	return signatory
}

func RandomSignatories() id.Signatories {
	length := mrand.Intn(30)
	sigs := make(id.Signatories, length)
	for i := 0; i < length; i++ {
		_, err := rand.Read(sigs[i][:])
		if err != nil {
			panic(fmt.Sprintf("cannot create random signatory, err = %v", err))
		}
	}
	return sigs
}
