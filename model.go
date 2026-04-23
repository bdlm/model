package model

import (
	"sync"

	"github.com/bdlm/cast/v2"
	"github.com/bdlm/errors/v2"
	stdModel "github.com/bdlm/std/v2/model"
)

/*
modelType is a data type for defining Model types.
*/
type modelType int

const (
	// Dict defines a dictionary model type.
	Dict modelType = iota
	// List defines a list model type.
	List
)

/*
Model defines the model data structure.
*/
type Model struct {
	id     any                // model identifier
	locked bool               // model read-only flag
	typ    stdModel.ModelType // model type, either stdModel.ModelTypeHash or stdModel.ModelTypeList

	mux     *sync.Mutex    // goroutine-safe
	data    []any          // data store
	hashIdx map[string]int // stdModel.ModelTypeHash data index
	idxHash map[int]string // stdModel.ModelTypeHash hash index
	pos     int            // current stdModel.Iterator cursor position
}

/*
New returns a new stdModel.Model.
*/
func New(modelType stdModel.ModelType) *Model {
	return &Model{
		mux:     &sync.Mutex{},
		typ:     modelType,
		hashIdx: map[string]int{},
		idxHash: map[int]string{},
		pos:     -1,
	}
}

/*
Data returns the current data set and indexes.
*/
func (mdl *Model) Data() ([]any, map[string]int, map[int]string) {
	return mdl.data, mdl.hashIdx, mdl.idxHash
}

/*
Delete removes a value from this model.
*/
func (mdl *Model) Delete(key any) error {
	mdl.mux.Lock()
	defer mdl.mux.Unlock()
	if stdModel.ModelTypeList == mdl.GetType() {
		k := key.(int)
		if k > len(mdl.data) {
			return errors.WrapE(InvalidIndex, errors.Errorf("index '%d' out of range", k))
		}
		mdl.data = append(mdl.data[:key.(int)-1], mdl.data[key.(int):]...)
		return nil
	}

	k := key.(string)
	if idx, ok := mdl.hashIdx[k]; ok {
		mdl.data = append(mdl.data[:idx-1], mdl.data[idx:]...)
		delete(mdl.hashIdx, k)
		delete(mdl.idxHash, idx)
		return nil
	}
	return errors.WrapE(InvalidIndex, errors.Errorf("index '%s' out of range", k))
}

/*
Filter filters elements of the data using a callback function and returns
the result.
*/
func (mdl *Model) Filter(callback func(stdModel.Value) stdModel.Model) stdModel.Model {
	return mdl
}

/*
Get returns the specified data value in this model.
*/
func (mdl *Model) Get(key any) (stdModel.Value, error) {
	if stdModel.ModelTypeHash == mdl.GetType() {
		var ok bool
		var idx int

		// hash keys are always strings
		hashIdx := cast.To[string](key)

		mdl.mux.Lock()
		defer mdl.mux.Unlock()

		if idx, ok = mdl.hashIdx[hashIdx]; !ok {
			return nil, errors.WrapE(InvalidIndex, errors.Errorf("invalid index '%s'", hashIdx))
		}

		ret := mdl.data[idx]
		return &Value{ret}, nil
	}

	// List model
	switch key.(type) {
	case int, int8, int16, int32, int64:
		mdl.mux.Lock()
		defer mdl.mux.Unlock()
		if key.(int) >= int(len(mdl.data)) {
			return nil, errors.WrapE(InvalidIndex, errors.Errorf("invalid index '%d'", key.(int)))
		}
		ret := mdl.data[key.(int)]
		return &Value{ret}, nil
	default:
		return nil, errors.WrapE(InvalidIndexType, errors.Errorf("key '%v' must be an integer", key))
	}
}

/*
GetID returns returns this model's id.
*/
func (mdl *Model) GetID() any {
	return mdl.id
}

/*
GetType returns the model type.
*/
func (mdl *Model) GetType() stdModel.ModelType {
	return mdl.typ
}

/*
Has tests to see of a specified data element exists in this model.
*/
func (mdl *Model) Has(key any) bool {
	if stdModel.ModelTypeList == mdl.GetType() {
		if k, ok := key.(int); ok && k < len(mdl.data) {
			return true
		}
	} else if kstr, ok := key.(string); ok {
		if _, ok := mdl.hashIdx[kstr]; ok {
			return true
		}
	}
	return false
}

/*
Lock marks this model as read-only.
*/
func (mdl *Model) Lock() {
	mdl.locked = true
}

/*
Map applies a callback to all elements in this model and returns the result.
*/
func (mdl *Model) Map(callback func(stdModel.Value) stdModel.Model) stdModel.Model {
	return nil
}

/*
Merge merges data from any Model into this Model.
*/
func (mdl *Model) Merge(model stdModel.Model) error {
	return nil
}

/*
Push a value to the end of the internal data store.
*/
func (mdl *Model) Push(value any) error {
	if raw, ok := value.(stdModel.Value); ok {
		value = raw
	}

	// stdModel.ModelTypeList only
	if stdModel.ModelTypeList != mdl.GetType() {
		return errors.WrapE(InvalidMethodContext, errors.Errorf("Push() is only valid for stdModel.ModelTypeList model types"))
	}

	mdl.mux.Lock()
	mdl.data = append(mdl.data, &Value{value})
	mdl.mux.Unlock()
	return nil
}

/*
Reduce iteratively reduces the data to a single value using a callback
function and returns the result.
*/
func (mdl *Model) Reduce(callback func(stdModel.Value) bool) stdModel.Value {
	return nil
}

/*
Reverse reverses the order of the data store.
*/
func (mdl *Model) Reverse() {
	return
}

/*
Set stores a value in the internal data store. All values must be identified
by key.
*/
func (mdl *Model) Set(key any, value any) error {
	if raw, ok := value.(stdModel.Value); ok {
		value = raw
	}

	// Hash model
	if stdModel.ModelTypeHash == mdl.GetType() {
		// hash keys are always strings
		idx := cast.To[string](key)
		mdl.mux.Lock()
		defer mdl.mux.Unlock()
		if _, ok := mdl.hashIdx[idx]; !ok {
			mdl.hashIdx[idx] = len(mdl.data)
			mdl.idxHash[len(mdl.data)] = idx
			mdl.data = append(mdl.data, value)
			return nil
		}
		mdl.data[mdl.hashIdx[idx]] = value
		return nil
	}

	// List model
	switch key.(type) {
	case int, int8, int16, int32, int64,
		uint, uint8, uint16, uint32, uint64:
		k := key.(int)
		mdl.mux.Lock()
		defer mdl.mux.Unlock()
		if k >= len(mdl.data) || k < 0 {
			return errors.WrapE(InvalidIndex, errors.Errorf("invalid index '%d'", k))
		}
		mdl.data[k] = value
		return nil
	default:
		return errors.WrapE(InvalidIndexType, errors.Errorf("key '%v' is must be an integer", key))
	}
}

/*
SetID sets this Model's identifier property.
*/
func (mdl *Model) SetID(id any) {
	mdl.id = id
}

/*
SetData replaces the current data stored in the model with the provided
data.
*/
func (mdl *Model) SetData(data any) error {
	if stdModel.ModelTypeList == mdl.GetType() {
		d, ok := data.([]any)
		if !ok {
			return errors.WrapE(InvalidDataSet, errors.Errorf("invalid data set for list model"))
		}
		mdl.data = d
	}

	d, ok := data.(map[string]any)
	if !ok {
		return errors.WrapE(InvalidDataSet, errors.Errorf("invalid data set for hash model"))
	}

	mdl.data = []any{}
	for k, v := range d {
		mdl.hashIdx[k] = len(mdl.data)
		mdl.idxHash[len(mdl.data)] = k
		mdl.data = append(mdl.data, v)
	}
	return nil
}

/*
SetType sets the model type. If any data is stored in this model, this
property becomes read-only.
*/
func (mdl *Model) SetType(typ stdModel.ModelType) error {
	if len(mdl.data) > 0 {
		return errors.WrapE(ReadOnlyProperty, errors.Errorf("model is not empty, type cannot be modified"))
	}
	mdl.typ = typ
	return nil
}
