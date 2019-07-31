package process_test

import (
	"fmt"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Messages", func() {

	table := []string{"propose", "prevote", "precommit"}

	for _, entry := range table {
		entry := entry

		Context(fmt.Sprintf("when stringifying random %vs", entry), func() {
			Context("when equal", func() {
				It("should return equal strings", func() {
					Expect(true).To(BeFalse())
				})
			})

			Context("when unequal", func() {
				It("should return unequal strings", func() {
					Expect(true).To(BeFalse())
				})
			})
		})

		Context(fmt.Sprintf("when marshaling random %vs", entry), func() {
			It("should equal itself after marshaling and then unmarshaling", func() {
				Expect(true).To(BeFalse())
			})
		})

		Context(fmt.Sprintf("when creating %v messages", entry), func() {
			It("should return a message with fields equal to those passed during creation", func() {
				Expect(true).To(BeFalse())
			})
		})

		Context(fmt.Sprintf("when signing and then verifying random %vs", entry), func() {
			It("should return no errors", func() {
				Expect(true).To(BeFalse())
			})

			Context("when randomly changing the signatory after signing", func() {
				It("should return an error", func() {
					Expect(true).To(BeFalse())
				})
			})

			Context("when randomly changing the signature after signing", func() {
				It("should return an error", func() {
					Expect(true).To(BeFalse())
				})
			})

			Context("when randomly changing the sighash after signing", func() {
				It("should return an error", func() {
					Expect(true).To(BeFalse())
				})
			})
		})
	}

	Context("when marshaling a random inbox", func() {
		It("should equal itself after marshaling and then unmarshaling", func() {
			Expect(true).To(BeFalse())
		})
	})

	Context("when inserting messages into an inbox", func() {
		Context("when 1 message is inserted", func() {
			Context("when F=1", func() {
				It("should return n=1, firstTime=true, firstTimeExceedingF=false, and firstTimeExceeding2F=false", func() {
					Expect(true).To(BeFalse())
				})
			})

			Context("when F>1", func() {
				It("should return n=1, firstTime=true, firstTimeExceedingF=false, and firstTimeExceeding2F=false", func() {
					Expect(true).To(BeFalse())
				})
			})
		})

		Context("when F messages are inserted", func() {
			Context("when F>1", func() {
				It("should return n=F, firstTime=false, firstTimeExceedingF=false, and firstTimeExceeding2F=false", func() {
					Expect(true).To(BeFalse())
				})
			})
		})

		Context("when F+1 messages are inserted", func() {
			Context("when F=1", func() {
				It("should return n=F+1, firstTime=false, firstTimeExceedingF=true, and firstTimeExceeding2F=false", func() {
					Expect(true).To(BeFalse())
				})
			})

			Context("when F>1", func() {
				It("should return n=F+1, firstTime=false, firstTimeExceedingF=true, and firstTimeExceeding2F=false", func() {
					Expect(true).To(BeFalse())
				})
			})
		})

		Context("when 2F messages are inserted", func() {
			Context("when F=1", func() {
				It("should return n=2F, firstTime=false, firstTimeExceedingF=true, and firstTimeExceeding2F=false", func() {
					Expect(true).To(BeFalse())
				})
			})

			Context("when F>1", func() {
				It("should return n=2F, firstTime=false, firstTimeExceedingF=false, and firstTimeExceeding2F=false", func() {
					Expect(true).To(BeFalse())
				})
			})
		})

		Context("when 2F+1 messages are inserted", func() {
			Context("when F=1", func() {
				It("should return n=3, firstTime=false, firstTimeExceedingF=false, and firstTimeExceeding2F=true", func() {
					Expect(true).To(BeFalse())
				})
			})

			Context("when F>1", func() {
				It("should return n=2F+1, firstTime=false, firstTimeExceedingF=false, and firstTimeExceeding2F=true", func() {
					Expect(true).To(BeFalse())
				})
			})
		})

		Context("when 3F messages are inserted", func() {
			Context("when F=1", func() {
				It("should return n=3F, firstTime=false, firstTimeExceedingF=false, and firstTimeExceeding2F=true", func() {
					Expect(true).To(BeFalse())
				})
			})

			Context("when F>1", func() {
				It("should return n=3F, firstTime=false, firstTimeExceedingF=false, and firstTimeExceeding2F=false", func() {
					Expect(true).To(BeFalse())
				})
			})
		})

		Context("when 3F+1 messages are inserted", func() {
			Context("when F=1", func() {
				It("should return n=4, firstTime=false, firstTimeExceedingF=false, and firstTimeExceeding2F=false", func() {
					Expect(true).To(BeFalse())
				})
			})

			Context("when F>1", func() {
				It("should return n=3F+1, firstTime=false, firstTimeExceedingF=false, and firstTimeExceeding2F=false", func() {
					Expect(true).To(BeFalse())
				})
			})
		})

		Context("when inserting N messages at the same height", func() {
			Context("when all rounds are equal", func() {
				Context("when all block hashes are equal", func() {
					Context("when querying by height, round, and block hash", func() {
						Context("when the queried height, round, and block hash is the same as the inserted height, round, and block hash", func() {
							Context("when the signatories are all equal", func() {
								It("should return 1", func() {
									Expect(true).To(BeFalse())
								})
							})

							Context("when the signatories are all unequal", func() {
								It("should return N", func() {
									Expect(true).To(BeFalse())
								})
							})
						})

						Context("when the queried height is different from the inserted height", func() {
							It("should return 0", func() {
								Expect(true).To(BeFalse())
							})
						})

						Context("when the queried round is different from the inserted round", func() {
							It("should return 0", func() {
								Expect(true).To(BeFalse())
							})
						})

						Context("when the queried block hash is different from the inserted block hash", func() {
							It("should return 0", func() {
								Expect(true).To(BeFalse())
							})
						})
					})
				})
			})

			Context("when all rounds are unequal", func() {
				Context("when querying by height and round", func() {
					Context("when the queried height and round are same as the inserted height and round", func() {
						Context("when the signatories are all equal", func() {
							It("should return 1", func() {
								Expect(true).To(BeFalse())
							})
						})

						Context("when the signatories are all unequal", func() {
							It("should return N", func() {
								Expect(true).To(BeFalse())
							})
						})

						Context("when the queried height is different from the inserted height", func() {
							It("should return 0", func() {
								Expect(true).To(BeFalse())
							})
						})

						Context("when the queried round is different from the inserted round", func() {
							It("should return 0", func() {
								Expect(true).To(BeFalse())
							})
						})
					})
				})

				Context("when querying by height, round, and signatory", func() {
					Context("when the queried height and round is the same as the inserted height and round", func() {
						Context("when the queried signatory is one of the inserted signatories", func() {
							It("should return the inserted message", func() {
								Expect(true).To(BeFalse())
							})
						})

						Context("when the queried signatory is not one of the inserted signatories", func() {
							It("should return nil", func() {
								Expect(true).To(BeFalse())
							})
						})
					})

					Context("when the queried height is different from the inserted height", func() {
						It("should return nil", func() {
							Expect(true).To(BeFalse())
						})
					})

					Context("when the queried round is different from the inserted round", func() {
						It("should return nil", func() {
							Expect(true).To(BeFalse())
						})
					})
				})
			})
		})
	})
})
