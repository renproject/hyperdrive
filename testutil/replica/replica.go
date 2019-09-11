package testutil_replica

import (
	"crypto/rand"
	"fmt"

	"github.com/renproject/hyperdrive/replica"
)

func RandomShard() replica.Shard {
	shard := replica.Shard{}
	_, err := rand.Read(shard[:])
	if err != nil {
		panic(fmt.Sprintf("cannot create random shard, err = %v", err))
	}
	return shard
}
