package block_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Block", func() {
	Context("when stringifying", func() {
		Context("when stringifying block kinds", func() {
			It("should return `standard` for standard blocks", func() {
				Expect(true).To(BeFalse())
			})
			It("should return `rebase` for rebase blocks", func() {
				Expect(true).To(BeFalse())
			})
			It("should return `base` for base blocks", func() {
				Expect(true).To(BeFalse())
			})
		})

		Context("when stringifying random block headers", func() {
			Context("when block headers are equal", func() {
				It("should return equal strings", func() {
					Expect(true).To(BeFalse())
				})
			})

			Context("when block headers are unequal", func() {
				It("should return unequal strings", func() {
					Expect(true).To(BeFalse())
				})
			})
		})

		Context("when stringifying random block data", func() {
			Context("when block data is equal", func() {
				It("should return equal strings", func() {
					Expect(true).To(BeFalse())
				})
			})

			Context("when block data is unequal", func() {
				It("should return unequal strings", func() {
					Expect(true).To(BeFalse())
				})
			})
		})

		Context("when stringifying random block states", func() {
			Context("when block states are equal", func() {
				It("should return equal strings", func() {
					Expect(true).To(BeFalse())
				})
			})

			Context("when block states are unequal", func() {
				It("should return unequal strings", func() {
					Expect(true).To(BeFalse())
				})
			})
		})

		Context("when stringifying random blocks", func() {
			Context("when blocks are equal", func() {
				It("should return equal strings", func() {
					Expect(true).To(BeFalse())
				})
			})

			Context("when blocks are unequal", func() {
				It("should return unequal strings", func() {
					Expect(true).To(BeFalse())
				})
			})
		})
	})

	Context("when marshaling", func() {
		Context("when marshaling a random block header", func() {
			It("should equal itself after marshaling and then unmarshaling", func() {
				Expect(true).To(BeFalse())
			})
		})

		Context("when marshaling a random block", func() {
			It("should equal itself after marshaling and then unmarshaling", func() {
				Expect(true).To(BeFalse())
			})
		})
	})

	Context("when creating block headers", func() {
		Context("when the block header is well-formed", func() {
			It("should return a block header with fields equal to those passed during creation", func() {
				Expect(true).To(BeFalse())
			})
		})

		Context("when the parent header is invalid", func() {
			It("should panic", func() {
				Expect(true).To(BeFalse())
			})
		})

		Context("when the base header is invalid", func() {
			It("should panic", func() {
				Expect(true).To(BeFalse())
			})
		})

		Context("when the height is invalid", func() {
			It("should panic", func() {
				Expect(true).To(BeFalse())
			})
		})

		Context("when the timestamp is invalid", func() {
			It("should panic", func() {
				Expect(true).To(BeFalse())
			})
		})

		Context("when the round is invalid", func() {
			It("should panic", func() {
				Expect(true).To(BeFalse())
			})
		})

		Context("when creating standard block headers", func() {
			Context("when signatories are non-nil", func() {
				It("should panic", func() {
					Expect(true).To(BeFalse())
				})
			})
		})

		Context("when creating rebase block headers", func() {
			Context("when signatories are nil or empty", func() {
				It("should panic", func() {
					Expect(true).To(BeFalse())
				})
			})
		})

		Context("when creating base block headers", func() {
			Context("when signatories are nil or empty", func() {
				It("should panic", func() {
					Expect(true).To(BeFalse())
				})
			})
		})
	})

	Context("when creating blocks", func() {
		It("should return blocks with computed hashes", func() {
			Expect(true).To(BeFalse())
		})

		It("should return a block with fields equal to those passed during creation", func() {
			Expect(true).To(BeFalse())
		})

		Context("when the header, data, and previous state are equal", func() {
			It("should return a block with computed hashes that are equal", func() {
				Expect(true).To(BeFalse())
			})

			It("should return true when checking block equality", func() {
				Expect(true).To(BeFalse())
			})
		})

		Context("when the header, data, and previous state are unequal", func() {
			It("should return a block with computed hashes that are unequal", func() {
				Expect(true).To(BeFalse())
			})

			It("should return false when checking block equality", func() {
				Expect(true).To(BeFalse())
			})
		})
	})
})
