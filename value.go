package model

import (
	"github.com/bdlm/cast/v2"
	"github.com/bdlm/errors/v2"
	stdModel "github.com/bdlm/std/v2/model"
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
	result, err := cast.ToE[bool](val.data)
	if nil != err {
		err = errors.Wrap(err, "could not convert value '%v' to a boolean", val.data)
	}
	return result, err
}

/*
Float returns the float64 representation of the value of this node, or an
error if the type conversion is not possible.
*/
func (val *Value) Float() (float64, error) {
	result, err := cast.ToE[float64](val.data)
	if nil != err {
		err = errors.Wrap(err, "could not convert value '%v' to a float64", val.data)
	}
	return result, err
}

/*
Float32 returns the float32 representation of the value of this node, or an
error if the type conversion is not possible.
*/
func (val *Value) Float32() (float32, error) {
	result, err := cast.ToE[float32](val.data)
	if nil != err {
		err = errors.Wrap(err, "could not convert value '%v' to a float32", val.data)
	}
	return result, err
}

/*
Float64 returns the float64 representation of the value of this node, or an
error if the type conversion is not possible.
*/
func (val *Value) Float64() (float64, error) {
	result, err := cast.ToE[float64](val.data)
	if nil != err {
		err = errors.Wrap(err, "could not convert value '%v' to a float64", val.data)
	}
	return result, err
}

/*
Int returns the int representation of the value of this node, or an error if
the type conversion is not possible.
*/
func (val *Value) Int() (int, error) {
	result, err := cast.ToE[int](val.data)
	if nil != err {
		err = errors.Wrap(err, "could not convert value '%v' to an int", val.data)
	}
	return result, err
}

/*
List returns the array of Values stored in this node, or an error if the
type conversion is not possible.
*/
func (val *Value) List() ([]stdModel.Value, error) {
	var err error
	result, ok := val.data.([]stdModel.Value)
	if !ok {
		err = errors.Errorf("could not convert value '%v' to an array", val.data)
	}
	return result, err
}

/*
Map returns the map[string]Value data stored in this node, or an error if
the type conversion is not possible.
*/
func (val *Value) Map() (map[string]stdModel.Value, error) {
	var err error
	result, ok := val.data.(map[string]stdModel.Value)
	if !ok {
		err = errors.Errorf("could not convert value '%v' to a map", val.data)
	}
	return result, err
}

/*
Model returns the Model stored at this node, or an error if the value does
not implement Model.
*/
func (val *Value) Model() (stdModel.Model, error) {
	var err error
	result, ok := val.data.(stdModel.Model)
	if !ok {
		err = errors.Errorf("could not convert value '%v' to a Model", val.data)
	}
	return result, err
}

/*
String returns the boolean representation of the value, or an error if the
type conversion is not possible.
*/
func (val *Value) String() (string, error) {
	result, err := cast.ToE[string](val.data)
	if nil != err {
		err = errors.Wrap(err, "could not convert value '%v' to a string", val.data)
	}
	return result, err
}

/*
Value returns the untyped value.
*/
func (val *Value) Value() interface{} {
	return val.data
}
