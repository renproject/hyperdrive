package replica

import (
	"github.com/renproject/hyperdrive/process"
	"github.com/renproject/hyperdrive/process/block"
)

// BlockHeaderValidator returns a `process.Validator` that validates the
// `block.Header` of a `block.Block` and, if it is valid, passes validation on
// to another `process.Validator`.
func BlockHeaderValidator(nextValidator process.Validator) process.Validator {
	return &blockHeaderValidator{
		nextValidator: nextValidator,
	}
}

type blockHeaderValidator struct {
	nextValidator process.Validator
}

func (validator *blockHeaderValidator) Validate(block block.Block) bool {
	res := false

	if res {
		if validator.nextValidator != nil {
			return validator.nextValidator.Validate(block)
		}
		return true
	}
	return false
}
