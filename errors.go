package model

import (
	"github.com/bdlm/errors/v2"
	stdErrors "github.com/bdlm/std/v2/errors"
)

// Exported error sentinels. All errors returned by this package wrap one of
// these values and can be tested with errors.Is:
//
//	if errors.Is(err, model.InvalidIndex) { … }
var (
	// InvalidIndex is returned when a key or index that does not exist in the
	// model is used with Get, Delete, or Seek.
	InvalidIndex stdErrors.Error

	// InvalidIndexType is returned when a key of the wrong type is used to
	// access a model. For LIST models all keys must be integer types (int,
	// int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64); passing
	// any other type to Get, Delete, or Seek returns this error.
	InvalidIndexType stdErrors.Error

	// InvalidMethodContext is returned when a method is called in an
	// unsupported context. Examples: Push called on a HASH model, or Merge
	// called with the receiver as the incoming argument (self-merge).
	InvalidMethodContext stdErrors.Error

	// ReadOnlyModel is returned when a write operation is attempted on a model
	// that has been made read-only. A model becomes read-only in two ways:
	//   - Lock has been called (permanent; affects all write methods).
	//   - SetType is called on a non-empty model (the type is read-only while
	//     data is present, but the lock flag itself is not set).
	ReadOnlyModel stdErrors.Error

	// InvalidDataSet is returned by SetData when the supplied data type is
	// incompatible with the model type: a []any for a HASH model, or a
	// map[string]any for a LIST model.
	InvalidDataSet stdErrors.Error

	// InvalidSortFlagCombination is returned by Sort when mutually exclusive
	// flags are combined. The only invalid combination is SortAsc | SortDesc.
	InvalidSortFlagCombination stdErrors.Error
)

// init initializes the package-level error sentinel variables. Sentinels are
// constructed here rather than at var-declaration time so that the
// errors.New calls (which establish the sentinel identity used by errors.Is)
// run in a single, well-defined initialization pass.
func init() {
	InvalidIndex = errors.New("specified index does not exist")
	InvalidIndexType = errors.New("an invalid index datatype was used")
	InvalidMethodContext = errors.New("a method was used in an invalid context")
	ReadOnlyModel = errors.New("cannot update a read-only model")
	InvalidDataSet = errors.New("invalid data set for model type")
	InvalidSortFlagCombination = errors.New("invalid sort flag combination")
}
