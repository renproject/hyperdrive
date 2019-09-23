package block_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"reflect"
	"testing/quick"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/renproject/hyperdrive/block"
	. "github.com/renproject/hyperdrive/testutil"

	"github.com/renproject/id"
)

var _ = Describe("Block", func() {

	Context("Block Kind", func() {
		Context("when stringifying", func() {
			It("should return correct string for each Kind", func() {
				Expect(func() {
					_ = Invalid.String()
				}).Should(Panic())

				Expect(fmt.Sprintf("%v", Standard)).Should(Equal("standard"))
				Expect(fmt.Sprintf("%v", Rebase)).Should(Equal("rebase"))
				Expect(fmt.Sprintf("%v", Base)).Should(Equal("base"))

				randKind := func() bool {
					kind := rand.Intn(math.MaxUint8 - 4)
					invalidKind := Kind(kind + 4) // skip the valid kinds
					Expect(func() {
						_ = invalidKind.String()
					}).Should(Panic())
					return true
				}

				Expect(quick.Check(randKind, nil)).Should(Succeed())
			})
		})
	})

	Context("Block header", func() {
		Context("when stringifying random block headers", func() {
			Context("when block headers are equal", func() {
				It("should return equal strings", func() {
					test := func() bool {
						header := RandomBlockHeader(RandomBlockKind())
						newHeader := header
						return header.String() == newHeader.String()
					}
					Expect(quick.Check(test, nil)).Should(Succeed())
				})
			})

			Context("when block headers are unequal", func() {
				It("should return unequal strings", func() {
					test := func() bool {
						header1 := RandomBlockHeader(RandomBlockKind())
						header2 := RandomBlockHeader(RandomBlockKind())
						return header1.String() != header2.String()
					}
					Expect(quick.Check(test, nil)).Should(Succeed())
				})
			})
		})

		Context("when marshaling a random block header", func() {
			It("should equal itself after json marshaling and then unmarshaling", func() {
				test := func() bool {
					header := RandomBlockHeader(RandomBlockKind())
					data, err := json.Marshal(header)
					Expect(err).NotTo(HaveOccurred())

					var newHeader Header
					Expect(json.Unmarshal(data, &newHeader)).Should(Succeed())
					Expect(header.String()).Should(Equal(newHeader.String()))
					return reflect.DeepEqual(header, newHeader)
				}
				Expect(quick.Check(test, nil)).Should(Succeed())
			})

			It("should equal itself after binary marshaling and then unmarshaling", func() {
				test := func() bool {
					header := RandomBlockHeader(RandomBlockKind())
					data, err := header.MarshalBinary()
					Expect(err).NotTo(HaveOccurred())

					var newHeader Header
					Expect(newHeader.UnmarshalBinary(data)).Should(Succeed())
					Expect(header.String()).Should(Equal(newHeader.String()))
					return reflect.DeepEqual(header, newHeader)
				}
				Expect(quick.Check(test, nil)).Should(Succeed())
			})
		})

		Context("when initializing a new block header", func() {

			Context("when the block header is well-formed", func() {
				It("should return a block header with fields equal to those passed during creation", func() {
					test := func() bool {
						kind := RandomBlockKind()
						headerInit := RandomBlockHeaderJSON(kind)
						header := headerInit.ToBlockHeader()

						Expect(header.Kind()).Should(Equal(headerInit.Kind))
						Expect(header.ParentHash()).Should(Equal(headerInit.ParentHash))
						Expect(header.BaseHash()).Should(Equal(headerInit.BaseHash))
						Expect(header.Height()).Should(Equal(headerInit.Height))
						Expect(header.Round()).Should(Equal(headerInit.Round))
						Expect(header.Timestamp()).Should(Equal(headerInit.Timestamp))
						Expect(header.Signatories()).Should(Equal(headerInit.Signatories))

						return true
					}
					Expect(quick.Check(test, nil)).Should(Succeed())
				})
			})

			Context("when the block kind is invalid", func() {
				It("should panic", func() {
					test := func() bool {
						kind := RandomBlockKind()
						headerInit := RandomBlockHeaderJSON(kind)
						headerInit.Kind = Invalid
						Expect(func() {
							_ = headerInit.ToBlockHeader()
						}).Should(Panic())

						// Creat a invalid kind between [4, 255]
						headerInit.Kind = Kind(rand.Intn(math.MaxUint8-4) + 4)
						Expect(func() {
							_ = headerInit.ToBlockHeader()
						}).Should(Panic())
						return true
					}
					Expect(quick.Check(test, nil)).Should(Succeed())
				})
			})

			Context("when the parent header is invalid", func() {
				It("should panic", func() {
					test := func() bool {
						kind := RandomBlockKind()
						headerInit := RandomBlockHeaderJSON(kind)
						headerInit.ParentHash = InvalidHash
						Expect(func() {
							_ = headerInit.ToBlockHeader()
						}).Should(Panic())
						return true
					}
					Expect(quick.Check(test, nil)).Should(Succeed())
				})
			})

			Context("when the base header is invalid", func() {
				It("should panic", func() {
					test := func() bool {
						kind := RandomBlockKind()
						headerInit := RandomBlockHeaderJSON(kind)
						headerInit.BaseHash = InvalidHash
						Expect(func() {
							_ = headerInit.ToBlockHeader()
						}).Should(Panic())
						return true
					}
					Expect(quick.Check(test, nil)).Should(Succeed())
				})
			})

			Context("when the height is invalid", func() {
				It("should panic", func() {
					test := func() bool {
						kind := RandomBlockKind()
						headerInit := RandomBlockHeaderJSON(kind)
						headerInit.Height = InvalidHeight
						Expect(func() {
							_ = headerInit.ToBlockHeader()
						}).Should(Panic())
						return true
					}
					Expect(quick.Check(test, nil)).Should(Succeed())
				})
			})

			Context("when the timestamp is invalid", func() {
				It("should panic", func() {
					test := func() bool {
						kind := RandomBlockKind()
						headerInit := RandomBlockHeaderJSON(kind)
						headerInit.Timestamp = Timestamp(time.Now().Unix() + int64(rand.Intn(1e6)))
						Expect(func() {
							_ = headerInit.ToBlockHeader()
						}).Should(Panic())
						return true
					}
					Expect(quick.Check(test, nil)).Should(Succeed())
				})
			})

			Context("when the round is invalid", func() {
				It("should panic", func() {
					test := func() bool {
						kind := RandomBlockKind()
						headerInit := RandomBlockHeaderJSON(kind)
						headerInit.Round = InvalidRound
						Expect(func() {
							_ = headerInit.ToBlockHeader()
						}).Should(Panic())
						return true
					}
					Expect(quick.Check(test, nil)).Should(Succeed())
				})
			})

			Context("when creating standard block headers", func() {
				Context("when signatories are non-nil", func() {
					It("should panic", func() {
						test := func() bool {
							headerInit := RandomBlockHeaderJSON(Standard)
							for len(headerInit.Signatories) == 0 {
								headerInit.Signatories = RandomSignatories()
							}
							Expect(func() {
								_ = headerInit.ToBlockHeader()
							}).Should(Panic())
							return true
						}
						Expect(quick.Check(test, nil)).Should(Succeed())
					})
				})
			})

			Context("when creating rebase block headers", func() {
				Context("when signatories are nil or empty", func() {
					It("should panic", func() {
						test := func() bool {
							headerInit := RandomBlockHeaderJSON(Rebase)
							headerInit.Signatories = nil
							Expect(func() {
								_ = headerInit.ToBlockHeader()
							}).Should(Panic())

							headerInit.Signatories = id.Signatories{}
							Expect(func() {
								_ = headerInit.ToBlockHeader()
							}).Should(Panic())
							return true
						}
						Expect(quick.Check(test, nil)).Should(Succeed())
					})
				})
			})

			Context("when creating base block headers", func() {
				Context("when signatories are nil or empty", func() {
					It("should panic", func() {
						test := func() bool {
							headerInit := RandomBlockHeaderJSON(Base)
							headerInit.Signatories = nil
							Expect(func() {
								_ = headerInit.ToBlockHeader()
							}).Should(Panic())

							headerInit.Signatories = id.Signatories{}
							Expect(func() {
								_ = headerInit.ToBlockHeader()
							}).Should(Panic())
							return true
						}
						Expect(quick.Check(test, nil)).Should(Succeed())
					})
				})
			})
		})
	})

	Context("Block Data", func() {
		Context("when stringifying random block data", func() {
			Context("when block data is equal", func() {
				It("should return equal strings", func() {
					test := func(data Data) bool {
						dataCopy := make(Data, len(data))
						copy(dataCopy, data)

						return data.String() == dataCopy.String()
					}
					Expect(quick.Check(test, nil)).Should(Succeed())
				})
			})

			Context("when block data is unequal", func() {
				It("should return unequal strings", func() {
					test := func(data1, data2 Data) bool {
						if bytes.Equal(data1, data2) {
							return true
						}
						return data1.String() != data2.String()
					}
					Expect(quick.Check(test, nil)).Should(Succeed())
				})
			})
		})
	})

	Context("Block State", func() {
		Context("when stringifying random block state", func() {
			Context("when block state is equal", func() {
				It("should return equal strings", func() {
					test := func(state State) bool {
						stateCopy := make(State, len(state))
						copy(stateCopy, state)

						return state.String() == stateCopy.String()
					}
					Expect(quick.Check(test, nil)).Should(Succeed())
				})
			})

			Context("when block state is unequal", func() {
				It("should return unequal strings", func() {
					test := func(state1, state2 State) bool {
						if bytes.Equal(state1, state2) {
							return true
						}
						return state1.String() != state2.String()
					}
					Expect(quick.Check(test, nil)).Should(Succeed())
				})
			})
		})
	})

	Context("Block", func() {

		Context("when stringifying random blocks", func() {
			Context("when blocks are equal", func() {
				It("should return equal strings", func() {
					test := func() bool {
						block := RandomBlock(RandomBlockKind())
						newBlock := block
						return block.String() == newBlock.String()
					}
					Expect(quick.Check(test, nil)).Should(Succeed())
				})
			})

			Context("when blocks are unequal", func() {
				It("should return unequal strings", func() {
					test := func() bool {
						block1 := RandomBlock(RandomBlockKind())
						block2 := RandomBlock(RandomBlockKind())
						return block1.String() != block2.String()
					}
					Expect(quick.Check(test, nil)).Should(Succeed())
				})
			})
		})

		Context("when marshaling", func() {
			Context("when marshaling a random block", func() {
				It("should equal itself after json marshaling and then unmarshaling", func() {
					test := func() bool {
						block := RandomBlock(RandomBlockKind())
						data, err := json.Marshal(block)
						Expect(err).NotTo(HaveOccurred())

						var newBlock Block
						Expect(json.Unmarshal(data, &newBlock)).Should(Succeed())
						Expect(block.String()).Should(Equal(newBlock.String()))
						return block.Equal(newBlock)
					}
					Expect(quick.Check(test, nil)).Should(Succeed())
				})

				It("should equal itself after binary marshaling and then unmarshaling", func() {
					test := func() bool {
						block := RandomBlock(RandomBlockKind())
						data, err := block.MarshalBinary()
						Expect(err).NotTo(HaveOccurred())

						var newBlock Block
						Expect(newBlock.UnmarshalBinary(data)).Should(Succeed())
						Expect(block.String()).Should(Equal(newBlock.String()))
						return block.Equal(newBlock)
					}
					Expect(quick.Check(test, nil)).Should(Succeed())
				})
			})
		})

		Context("when creating blocks", func() {
			It("should return a block with fields equal to those passed during creation", func() {
				test := func() bool {
					header := RandomBlockHeader(RandomBlockKind())
					data, state := RandomBytesSlice(), RandomBytesSlice()

					// Expect the block has a valid hash
					block := New(header, data, state)
					Expect(block.Hash()).ShouldNot(Equal(InvalidHash))

					Expect(block.Header().String()).Should(Equal(header.String()))
					Expect(block.Data()).Should(Equal(Data(data)))
					Expect(block.PreviousState()).Should(Equal(State(state)))

					return true
				}
				Expect(quick.Check(test, nil)).Should(Succeed())
			})

			Context("when the header, data, and previous state are equal", func() {
				It("should return a block with computed hashes that are equal", func() {
					test := func() bool {
						header := RandomBlockHeader(RandomBlockKind())
						data, state := RandomBytesSlice(), RandomBytesSlice()

						block1 := New(header, data, state)
						block2 := New(header, data, state)

						Expect(block1.Hash()).Should(Equal(block2.Hash()))
						Expect(block1.Equal(block2)).Should(BeTrue())
						Expect(block2.Equal(block1)).Should(BeTrue())

						return true
					}
					Expect(quick.Check(test, nil)).Should(Succeed())
				})
			})

			Context("when the header, data, and previous state are unequal", func() {
				It("should return a block with computed hashes that are unequal", func() {
					test := func() bool {
						kind := RandomBlockKind()
						block1 := RandomBlock(kind)
						block2 := RandomBlock(kind)

						Expect(block1.Hash()).ShouldNot(Equal(block2.Hash()))
						Expect(block1.Equal(block2)).Should(BeFalse())
						Expect(block2.Equal(block1)).Should(BeFalse())

						return true
					}
					Expect(quick.Check(test, nil)).Should(Succeed())
				})
			})
		})
	})
})
