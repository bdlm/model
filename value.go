package model

import (
	"encoding/json"

	"github.com/bdlm/cast/v2"
	"github.com/bdlm/errors/v2"
	stdModel "github.com/bdlm/std/v2/model"
)

// Value is the element type stored inside a [Model]. It wraps an arbitrary Go
// value and exposes typed accessors that convert the underlying data using the
// bdlm/cast package.
//
// Value implements the github.com/bdlm/std/v2/model.Value interface.
// Conversion methods return a non-nil error when the underlying data cannot be
// represented in the target type.
//
// The data field is intentionally unexported. All access to the raw underlying
// value goes through [Value.Value] or one of the typed conversion methods.
// Values are obtained from a Model via [Model.Get] or the iterator methods;
// they should not be constructed directly by callers.
type Value struct {
	data any
}

// To converts val to the target type TTo using the bdlm/cast package. On
// conversion failure the zero value of TTo is returned. For an error-aware
// conversion use [ToE].
//
// TTo is constrained to the set of types supported by cast.Types.
func To[TTo cast.Types](val any, ops ...cast.Op) TTo {
	return cast.To[TTo](val, ops...)
}

// ToE converts val to the target type TTo using the bdlm/cast package,
// returning the converted value and a non-nil error if the conversion fails
// or would be lossy.
//
// TTo is constrained to the set of types supported by cast.Types.
func ToE[TTo cast.Types](val any, ops ...cast.Op) (TTo, error) {
	return cast.ToE[TTo](val, ops...)
}

// MarshalJSON implements json.Marshaler. It marshals the underlying data
// directly so that a *Value stored inside a Model serializes as its content
// rather than as a struct literal with an unexported field.
func (val *Value) MarshalJSON() ([]byte, error) {
	return json.Marshal(val.data)
}

// Bool returns the boolean representation of the underlying value. Conversion
// follows the bdlm/cast rules: numeric zero is false, any non-zero number is
// true; strings are parsed according to strconv.ParseBool. Returns a non-nil
// error if the conversion is not possible.
func (val *Value) Bool() (bool, error) {
	result, err := ToE[bool](val.data)
	if nil != err {
		err = errors.Wrap(err, "could not convert value '%v' to a boolean", val.data)
	}
	return result, err
}

// Float is an alias for [Value.Float64].
func (val *Value) Float() (float64, error) {
	return val.Float64()
}

// Float32 returns the float32 representation of the underlying value. Returns
// a non-nil error if the conversion is not possible.
func (val *Value) Float32() (float32, error) {
	result, err := ToE[float32](val.data)
	if nil != err {
		err = errors.Wrap(err, "could not convert value '%v' to a float32", val.data)
	}
	return result, err
}

// Float64 returns the float64 representation of the underlying value. Returns
// a non-nil error if the conversion is not possible.
func (val *Value) Float64() (float64, error) {
	result, err := ToE[float64](val.data)
	if nil != err {
		err = errors.Wrap(err, "could not convert value '%v' to a float64", val.data)
	}
	return result, err
}

// Int returns the int representation of the underlying value. Floating-point
// values are truncated toward zero. Returns a non-nil error if the conversion
// is not possible (for example, a non-numeric string).
func (val *Value) Int() (int, error) {
	result, err := ToE[int](val.data)
	if nil != err {
		err = errors.Wrap(err, "could not convert value '%v' to an int", val.data)
	}
	return result, err
}

// List returns the underlying value as a []stdModel.Value. This succeeds only
// when the stored data is exactly a []stdModel.Value — it is not a general
// accessor for nested LIST models. For a nested LIST model stored inside a
// Model, retrieve the Value with [Model.Get] and call [Value.Model] instead.
func (val *Value) List() ([]stdModel.Value, error) {
	var err error
	result, ok := val.data.([]stdModel.Value)
	if !ok {
		err = errors.Errorf("could not convert value '%v' to an array", val.data)
	}
	return result, err
}

// Map returns the underlying value as a map[string]stdModel.Value. This
// succeeds only when the stored data is exactly a map[string]stdModel.Value —
// it is not a general accessor for nested HASH models. For a nested HASH model
// stored inside a Model, retrieve the Value with [Model.Get] and call
// [Value.Model] instead.
func (val *Value) Map() (map[string]stdModel.Value, error) {
	var err error
	result, ok := val.data.(map[string]stdModel.Value)
	if !ok {
		err = errors.Errorf("could not convert value '%v' to a map", val.data)
	}
	return result, err
}

// Model returns the underlying value as a stdModel.Model. This succeeds when
// the stored data implements the stdModel.Model interface, which is true for
// any *[Model] stored via [Model.Set] or [Model.Push] and for nested models
// created automatically during JSON import.
func (val *Value) Model() (stdModel.Model, error) {
	var err error
	result, ok := val.data.(stdModel.Model)
	if !ok {
		err = errors.Errorf("could not convert value '%v' to a Model", val.data)
	}
	return result, err
}

// String returns the string representation of the underlying value using the
// bdlm/cast package. Numeric values are formatted as their decimal string
// equivalents. Returns a non-nil error if the conversion is not possible.
func (val *Value) String() (string, error) {
	result, err := ToE[string](val.data)
	if nil != err {
		err = errors.Wrap(err, "could not convert value '%v' to a string", val.data)
	}
	return result, err
}

// Value returns the raw underlying data without any type conversion. The
// returned value is the same any that was originally passed to [Model.Set] or
// [Model.Push], or that was decoded from JSON.
func (val *Value) Value() any {
	return val.data
}
