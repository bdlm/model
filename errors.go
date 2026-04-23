package model

import (
	"github.com/bdlm/errors/v2"
	stdErrors "github.com/bdlm/std/v2/errors"
)

/*
Internal errors
*/
var (
	// InvalidIndex - The specified index does not exist.
	InvalidIndex stdErrors.Error

	// InvalidIndexType - The specified index data type is invalid for this
	// model.
	InvalidIndexType stdErrors.Error

	// InvalidMethodContext - The requested method is not valid in the
	// current context. E.g. the Push() method on hash models.
	InvalidMethodContext stdErrors.Error

	// ReadOnlyProperty - An attempt was made to modify a read-only property.
	ReadOnlyProperty stdErrors.Error

	// InvalidDataSet - An attempt was made to store a data set that is
	// with the model type
	InvalidDataSet stdErrors.Error
)

func init() {
	InvalidIndex = errors.New("specified index does not exist")
	InvalidIndexType = errors.New("an invalid index datatype was used")
	InvalidMethodContext = errors.New("a method was used in an invalid context")
	ReadOnlyProperty = errors.New("cannot update a read-only property")
}
