package main

// import (
// 	"fmt"
// 	"github.com/renproject/hyperdrive/block"
// 	"github.com/renproject/hyperdrive/consensus"
// 	"github.com/renproject/hyperdrive/sig"
// 	"github.com/renproject/hyperdrive/sig/ecdsa"
// 	"github.com/renproject/hyperdrive/tx"
// )

func main() {
// 	action := consensus.SignedPreVote{
// 		SignedPreVote: block.SignedPreVote{
// 			PreVote: block.PreVote{
// 				Block: buildSignedBlock(),
// 			},
// 		},
// 	}

// 	deepCopy := *action.SignedPreVote.Block

// 	copy := action
// 	copy.SignedPreVote.Block = &deepCopy

// 	action.SignedPreVote.Block.Header = sig.Hash{}
// 	fmt.Printf("%+v\n%+v\n%+v\n\n", action.SignedPreVote, copy.SignedPreVote, deepCopy)
// }

// func buildSignedBlock() *block.SignedBlock {
// 	// TODO: We should put more than one transaction into a block.
// 	transactions := tx.Transactions{}
// 	// transaction, ok := replica.txPool.Dequeue()
// 	// if ok {
// 	// 	transactions = append(transactions, transaction)
// 	// }

// 	// parent, ok := replica.blockchain.Head()
// 	// if !ok {
// 	parent := block.Genesis()
// 	// }
// 	// fmt.Printf("%x\n", parent.Header)

// 	block := block.New(
// 		0,
// 		0,
// 		parent.Header,
// 		transactions,
// 	)

// 	signer, _ := ecdsa.NewFromRandom()
// 	signedBlock, err := block.Sign(signer)
// 	if err != nil {
// 		// FIXME: We should handle this error properly. It would not make sense to propagate it, but there should at
// 		// least be some sane logging and recovery.
// 		panic(err)
// 	}

// 	fmt.Printf("here %x \n%x\n%+v\n\n", signedBlock.Signature, signedBlock.Signatory, signedBlock)
// 	return &signedBlock
}
