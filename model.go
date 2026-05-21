package model

import (
	"sync"

	"github.com/bdlm/cast/v2"
	"github.com/bdlm/errors/v2"
	stdModel "github.com/bdlm/std/v2/model"
)

// modelType is a data type for defining Model types.
type modelType int

const (
	// Dict defines a dictionary model type.
	Dict modelType = iota
	// List defines a list model type.
	List
)

// Model defines the model data structure.
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

// New returns a new stdModel.Model.
func New(modelType stdModel.ModelType, data any) *Model {
	model := &Model{
		mux:     &sync.Mutex{},
		typ:     modelType,
		hashIdx: map[string]int{},
		idxHash: map[int]string{},
		pos:     -1,
	}
	if data != nil {
		if _, err := model.importData(data); err != nil {
			panic(err)
		}
	}
	return model
}

// Delete removes a value from this model.
func (mdl *Model) Delete(key any) error {
	if mdl.locked {
		return errors.WrapE(ReadOnlyProperty, errors.Errorf("model is locked"))
	}
	mdl.mux.Lock()
	defer mdl.mux.Unlock()
	if stdModel.ModelTypeList == mdl.GetType() {
		k := cast.To[int](key)
		if k < 0 || k >= len(mdl.data) {
			return errors.WrapE(InvalidIndex, errors.Errorf("index '%d' out of range", k))
		}
		mdl.data = append(mdl.data[:k], mdl.data[k+1:]...)
		return nil
	}

	k := cast.To[string](key)
	if idx, ok := mdl.hashIdx[k]; ok {
		mdl.data = append(mdl.data[:idx], mdl.data[idx+1:]...)
		delete(mdl.hashIdx, k)
		delete(mdl.idxHash, idx)
		for i := idx; i < len(mdl.data); i++ {
			shifted := mdl.idxHash[i+1]
			mdl.hashIdx[shifted] = i
			mdl.idxHash[i] = shifted
			delete(mdl.idxHash, i+1)
		}
		return nil
	}
	return errors.WrapE(InvalidIndex, errors.Errorf("index '%s' out of range", k))
}

// Filter filters elements of the data using a callback function and returns
// the result.
func (mdl *Model) Filter(callback func(stdModel.Value) bool) stdModel.Model {
	result := New(mdl.GetType(), nil)
	mdl.mux.Lock()
	defer mdl.mux.Unlock()
	for i := 0; i < len(mdl.data); i++ {
		v := toValue(mdl.data[i])
		if callback(v) {
			if stdModel.ModelTypeHash == mdl.GetType() {
				result.Set(mdl.idxHash[i], v)
			} else {
				result.Push(v)
			}
		}
	}
	return result
}

// Get returns the specified data value in this model.
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
		k := cast.To[int](key)
		if k < 0 || k >= len(mdl.data) {
			return nil, errors.WrapE(InvalidIndex, errors.Errorf("invalid index '%d'", k))
		}
		return &Value{mdl.data[k]}, nil
	default:
		return nil, errors.WrapE(InvalidIndexType, errors.Errorf("key '%v' must be an integer", key))
	}
}

// GetData returns the current data set and indexes.
func (mdl *Model) GetData() ([]any, map[string]int, map[int]string) {
	mdl.mux.Lock()
	defer mdl.mux.Unlock()
	data := make([]any, len(mdl.data))
	copy(data, mdl.data)
	hashIdx := make(map[string]int, len(mdl.hashIdx))
	for k, v := range mdl.hashIdx {
		hashIdx[k] = v
	}
	idxHash := make(map[int]string, len(mdl.idxHash))
	for k, v := range mdl.idxHash {
		idxHash[k] = v
	}
	return data, hashIdx, idxHash
}

// GetID returns this model's id.
func (mdl *Model) GetID() any {
	return mdl.id
}

// GetType returns the model type.
func (mdl *Model) GetType() stdModel.ModelType {
	return mdl.typ
}

// Has tests to see if a specified data element exists in this model.
func (mdl *Model) Has(key any) bool {
	if stdModel.ModelTypeList == mdl.GetType() {
		switch key.(type) {
		case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
			k := cast.To[int](key)
			if k >= 0 && k < len(mdl.data) {
				return true
			}
		}
	} else if kstr, ok := key.(string); ok {
		if _, ok := mdl.hashIdx[kstr]; ok {
			return true
		}
	}
	return false
}

// Lock marks this model as read-only.
func (mdl *Model) Lock() {
	mdl.locked = true
}

// Map applies a callback to all elements in this model and returns the result.
func (mdl *Model) Map(callback func(stdModel.Value) stdModel.Value) stdModel.Model {
	result := New(mdl.GetType(), nil)
	mdl.mux.Lock()
	defer mdl.mux.Unlock()
	for i := 0; i < len(mdl.data); i++ {
		v := toValue(mdl.data[i])
		mapped := callback(v)
		if stdModel.ModelTypeHash == mdl.GetType() {
			result.Set(mdl.idxHash[i], mapped)
		} else {
			result.Push(mapped)
		}
	}
	return result
}

// Merge merges data from any Model into this Model.
func (mdl *Model) Merge(incoming stdModel.Model) error {
	if mdl.locked {
		return errors.WrapE(ReadOnlyProperty, errors.Errorf("model is locked"))
	}

	inData, inHashIdx, _ := incoming.GetData()

	switch mdl.GetType() {
	case stdModel.ModelTypeHash:
		if incoming.GetType() == stdModel.ModelTypeHash {
			// hash into hash: incoming wins; merge nested models recursively
			for key, inIdx := range inHashIdx {
				inVal := inData[inIdx]
				mdl.mux.Lock()
				myIdx, exists := mdl.hashIdx[key]
				var myVal any
				if exists {
					myVal = mdl.data[myIdx]
				}
				mdl.mux.Unlock()
				if exists {
					myModel, myIsModel := asModel(myVal)
					inModel, inIsModel := asModel(inVal)
					if myIsModel && inIsModel {
						if err := myModel.Merge(inModel); err != nil {
							return err
						}
						continue
					}
				}
				if err := mdl.Set(key, inVal); err != nil {
					return err
				}
			}
		} else {
			// list into hash: string-cast indices become keys
			for i, v := range inData {
				if err := mdl.Set(cast.To[string](i), v); err != nil {
					return err
				}
			}
		}

	case stdModel.ModelTypeList:
		if incoming.GetType() == stdModel.ModelTypeList {
			// list into list: append
			for _, v := range inData {
				if err := mdl.Push(v); err != nil {
					return err
				}
			}
		} else {
			// hash into list: append values in insertion order, ignore keys
			for i := 0; i < len(inData); i++ {
				if err := mdl.Push(inData[i]); err != nil {
					return err
				}
			}
		}
	}

	return nil
}

// Push a value to the end of the internal data store.
func (mdl *Model) Push(value any) error {
	if mdl.locked {
		return errors.WrapE(ReadOnlyProperty, errors.Errorf("model is locked"))
	}
	// stdModel.ModelTypeList only
	if stdModel.ModelTypeList != mdl.GetType() {
		return errors.WrapE(InvalidMethodContext, errors.Errorf("Push() is only valid for stdModel.ModelTypeList model types"))
	}

	mdl.mux.Lock()
	mdl.data = append(mdl.data, toValue(value))
	mdl.mux.Unlock()
	return nil
}

// Reduce iteratively reduces the data to a single value using a callback
// function and returns the result.
//
// The callback takes two stdModel.Value arguments: carry (the result of the
// previous iteration, or the first element on the first iteration) and cur
// (the current element). The callback returns a stdModel.Value which becomes
// the carry for the next iteration. After all iterations, Reduce returns the
// final carry value. Returns nil for an empty model.
func (mdl *Model) Reduce(callback func(carry, cur stdModel.Value) stdModel.Value) stdModel.Value {
	mdl.mux.Lock()
	defer mdl.mux.Unlock()
	if len(mdl.data) == 0 {
		return nil
	}
	var carry stdModel.Value = toValue(mdl.data[0])
	for i := 1; i < len(mdl.data); i++ {
		carry = callback(carry, toValue(mdl.data[i]))
	}
	return carry
}

// Set stores a value in the internal data store. All values must be identified
// by key.
func (mdl *Model) Set(key any, value any) error {
	if mdl.locked {
		return errors.WrapE(ReadOnlyProperty, errors.Errorf("model is locked"))
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
			mdl.data = append(mdl.data, toValue(value))
			return nil
		}
		mdl.data[mdl.hashIdx[idx]] = toValue(value)
		return nil
	}

	// List model
	switch key.(type) {
	case int, int8, int16, int32, int64,
		uint, uint8, uint16, uint32, uint64:
		k := cast.To[int](key)
		mdl.mux.Lock()
		defer mdl.mux.Unlock()
		if k >= len(mdl.data) || k < 0 {
			return errors.WrapE(InvalidIndex, errors.Errorf("invalid index '%d'", k))
		}
		mdl.data[k] = toValue(value)
		return nil
	default:
		return errors.WrapE(InvalidIndexType, errors.Errorf("key '%v' must be an integer", key))
	}
}

// SetID sets this Model's identifier property.
func (mdl *Model) SetID(id any) error {
	if mdl.locked {
		return errors.WrapE(ReadOnlyProperty, errors.Errorf("model is locked"))
	}
	mdl.id = id
	return nil
}

// SetData replaces the current data stored in the model with the provided data.
func (mdl *Model) SetData(data any) error {
	if mdl.locked {
		return errors.WrapE(ReadOnlyProperty, errors.Errorf("model is locked"))
	}
	if stdModel.ModelTypeList == mdl.GetType() {
		d, ok := data.([]any)
		if !ok {
			return errors.WrapE(InvalidDataSet, errors.Errorf("invalid data set for list model"))
		}
		mdl.data = d
		return nil
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

// SetType sets the model type. If any data is stored in this model, this
// property becomes read-only.
func (mdl *Model) SetType(typ stdModel.ModelType) error {
	if mdl.locked {
		return errors.WrapE(ReadOnlyProperty, errors.Errorf("model is locked"))
	}
	if len(mdl.data) > 0 {
		return errors.WrapE(ReadOnlyProperty, errors.Errorf("model is not empty, type cannot be modified"))
	}
	mdl.typ = typ
	return nil
}

// toValue wraps v in a *Value, or returns it directly if it is already one.
func toValue(v any) *Value {
	if val, ok := v.(*Value); ok && val != nil {
		return val
	}
	return &Value{v}
}

// asModel extracts a stdModel.Model from v, unwrapping a *Value if necessary.
func asModel(v any) (stdModel.Model, bool) {
	if val, ok := v.(*Value); ok {
		v = val.data
	}
	m, ok := v.(stdModel.Model)
	return m, ok
}
