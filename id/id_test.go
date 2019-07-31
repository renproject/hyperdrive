package id_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("ID", func() {
	Context("when stringifying", func() {
		Context("when stringifying random hashes", func() {
			Context("when hashes are equal", func() {
				It("should return equal strings", func() {
					Expect(true).To(BeFalse())
				})
			})

			Context("when hashes are unequal", func() {
				It("should return unequal strings", func() {
					Expect(true).To(BeFalse())
				})
			})
		})

		Context("when stringifying random signatures", func() {
			Context("when signatures are equal", func() {
				It("should return equal strings", func() {
					Expect(true).To(BeFalse())
				})
			})

			Context("when signatures are unequal", func() {
				It("should return unequal strings", func() {
					Expect(true).To(BeFalse())
				})
			})
		})

		Context("when stringifying random signatories", func() {
			Context("when signatories are equal", func() {
				It("should return equal strings", func() {
					Expect(true).To(BeFalse())
				})
			})

			Context("when signatories are unequal", func() {
				It("should return unequal strings", func() {
					Expect(true).To(BeFalse())
				})
			})
		})
	})

	Context("when comparing", func() {
		Context("when comparing random hashes", func() {
			Context("when hashes are equal", func() {
				It("should return true", func() {
					Expect(true).To(BeFalse())
				})
			})

			Context("when hashes are unequal", func() {
				It("should return false", func() {
					Expect(true).To(BeFalse())
				})
			})
		})

		Context("when comparing random signatures", func() {
			Context("when signatures are equal", func() {
				It("should return true", func() {
					Expect(true).To(BeFalse())
				})
			})

			Context("when signatures are unequal", func() {
				It("should return false", func() {
					Expect(true).To(BeFalse())
				})
			})
		})

		Context("when comparing random signatories", func() {
			Context("when signatories are equal", func() {
				It("should return true", func() {
					Expect(true).To(BeFalse())
				})
			})

			Context("when signatories are unequal", func() {
				It("should return false", func() {
					Expect(true).To(BeFalse())
				})
			})
		})
	})

	Context("when marshaling", func() {
		Context("when marshaling a random hash", func() {
			It("should equal itself after marshaling and then unmarshaling", func() {
				Expect(true).To(BeFalse())
			})
		})

		Context("when marshaling random hashes", func() {
			It("should equal itself after marshaling and then unmarshaling", func() {
				Expect(true).To(BeFalse())
			})
		})

		Context("when marshaling a random signature", func() {
			It("should equal itself after marshaling and then unmarshaling", func() {
				Expect(true).To(BeFalse())
			})
		})

		Context("when marshaling random signatures", func() {
			It("should equal itself after marshaling and then unmarshaling", func() {
				Expect(true).To(BeFalse())
			})
		})

		Context("when marshaling a random signatory", func() {
			It("should equal itself after marshaling and then unmarshaling", func() {
				Expect(true).To(BeFalse())
			})
		})

		Context("when marshaling random signatories", func() {
			It("should equal itself after marshaling and then unmarshaling", func() {
				Expect(true).To(BeFalse())
			})
		})
	})

	Context("when creating signatories", func() {
		Context("when using pubkeys that are equal", func() {
			It("should return signatories that are equal", func() {
				Expect(true).To(BeFalse())
			})
		})

		Context("when using pubkeys that are unequal", func() {
			It("should return signatories that are unequal", func() {
				Expect(true).To(BeFalse())
			})
		})
	})
})
