package model

import (
	"github.com/bdlm/errors"
)

/*
Internal errors
*/
const (
	// InvalidIndex - The specified index does not exist.
	InvalidIndex errors.Code = iota + 1000

	// InvalidIndexType - The specified index data type is invalid for this
	// model.
	InvalidIndexType

	// InvalidMethodContext - The requested method is not valid in the
	// current context. E.g. the Push() method on hash models.
	InvalidMethodContext

	// ReadOnlyProperty - An attempt was made to modify a read-only property.
	ReadOnlyProperty

	// InvalidDataSet - An attempt was made to store a data set that is
	// with the model type
	InvalidDataSet
)

func init() {
	errors.Codes[InvalidIndex] = errors.ErrCode{
		Int: "specified index does not exist",
	}
	errors.Codes[InvalidIndexType] = errors.ErrCode{
		Int: "an invalid index datatype was used",
	}
	errors.Codes[InvalidMethodContext] = errors.ErrCode{
		Int: "a method was used in an invalid context",
	}
	errors.Codes[ReadOnlyProperty] = errors.ErrCode{
		Int: "cannot update a read-only property",
	}
}
