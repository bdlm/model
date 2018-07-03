package model

import (
	"github.com/bdlm/errors"
	"github.com/bdlm/std"
	"github.com/spf13/cast"
)

/*
Value implements github.com/bdlm/std/Value.
*/
type Value struct {
	data interface{}
}

/*
Bool returns the boolean representation of the value of this node, or an
error if the type conversion is not possible.
*/
func (val *Value) Bool() (bool, error) {
	result, err := cast.ToBoolE(val.data)
	if nil != err {
		err = errors.Wrap(
			err,
			errors.ErrTypeConversionFailed,
			"could not convert value '%v' to a boolean",
			val.data,
		)
	}
	return result, err
}

/*
Float returns the float64 representation of the value of this node, or an
error if the type conversion is not possible.
*/
func (val *Value) Float() (float64, error) {
	result, err := cast.ToFloat64E(val.data)
	if nil != err {
		err = errors.Wrap(
			err,
			errors.ErrTypeConversionFailed,
			"could not convert value '%v' to a float64",
			val.data,
		)
	}
	return result, err
}

/*
Float32 returns the float32 representation of the value of this node, or an
error if the type conversion is not possible.
*/
func (val *Value) Float32() (float32, error) {
	result, err := cast.ToFloat32E(val.data)
	if nil != err {
		err = errors.Wrap(
			err,
			errors.ErrTypeConversionFailed,
			"could not convert value '%v' to a float32",
			val.data,
		)
	}
	return result, err
}

/*
Float64 returns the float64 representation of the value of this node, or an
error if the type conversion is not possible.
*/
func (val *Value) Float64() (float64, error) {
	result, err := cast.ToFloat64E(val.data)
	if nil != err {
		err = errors.Wrap(
			err,
			errors.ErrTypeConversionFailed,
			"could not convert value '%v' to a float64",
			val.data,
		)
	}
	return result, err
}

/*
Int returns the int representation of the value of this node, or an error if
the type conversion is not possible.
*/
func (val *Value) Int() (int, error) {
	result, err := cast.ToIntE(val.data)
	if nil != err {
		err = errors.Wrap(
			err,
			errors.ErrTypeConversionFailed,
			"could not convert value '%v' to an int",
			val.data,
		)
	}
	return result, err
}

/*
List returns the array of Values stored in this node, or an error if the
type conversion is not possible.
*/
func (val *Value) List() ([]std.Value, error) {
	var err error
	result, ok := val.data.([]std.Value)
	if !ok {
		err = errors.New(
			errors.ErrTypeConversionFailed,
			"could not convert value '%v' to an array",
			val.data,
		)
	}
	return result, err
}

/*
Map returns the map[string]Value data stored in this node, or an error if
the type conversion is not possible.
*/
func (val *Value) Map() (map[string]std.Value, error) {
	var err error
	result, ok := val.data.(map[string]std.Value)
	if !ok {
		err = errors.New(
			errors.ErrTypeConversionFailed,
			"could not convert value '%v' to a map",
			val.data,
		)
	}
	return result, err
}

/*
Model returns the Model stored at this node, or an error if the value does
not implement Model.
*/
func (val *Value) Model() (std.Model, error) {
	var err error
	result, ok := val.data.(std.Model)
	if !ok {
		err = errors.New(
			errors.ErrTypeConversionFailed,
			"could not convert value '%v' to a Model",
			val.data,
		)
	}
	return result, err
}

/*
String returns the boolean representation of the value, or an error if the
type conversion is not possible.
*/
func (val *Value) String() (string, error) {
	result, err := cast.ToStringE(val.data)
	if nil != err {
		err = errors.Wrap(
			err,
			errors.ErrTypeConversionFailed,
			"could not convert value '%v' to a string",
			val.data,
		)
	}
	return result, err
}

/*
Value returns the untyped value.
*/
func (val *Value) Value() interface{} {
	return val.data
}
