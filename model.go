package model

import (
	"fmt"
	"strings"
	"sync"

	"github.com/bdlm/errors"
	"github.com/bdlm/std"
)

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
	id     interface{}   // model identifier
	locked bool          // model read-only flag
	typ    std.ModelType // model type, either std.ModelTypeHash or std.ModelTypeList

	mux     *sync.Mutex    // goroutine-safe
	data    []interface{}  // data store
	hashIdx map[string]int // std.ModelTypeHash data index
	idxHash map[int]string // std.ModelTypeHash hash index
	pos     int            // current std.Iterator cursor position
}

/*
New returns a new std.Model.
*/
func New(modelType std.ModelType) std.Model {
	return &Model{
		mux:     &sync.Mutex{},
		typ:     modelType,
		hashIdx: map[string]int{},
		idxHash: map[int]string{},
		pos:     -1,
	}
}

/*
Delete removes a value from this model.
*/
func (mdl *Model) Delete(key interface{}) error {
	mdl.mux.Lock()
	defer mdl.mux.Unlock()
	if std.ModelTypeList == mdl.typ {
		k := key.(int)
		if k > len(mdl.data) {
			return errors.New(InvalidIndex, "index '%d' out of range", k)
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
	return errors.New(InvalidIndex, "index '%s' out of range", k)
}

/*
Filter filters elements of the data using a callback function and returns
the result.
*/
func (mdl *Model) Filter(callback func(std.Value) std.Model) std.Model {
	return mdl
}

/*
Get returns the specified data value in this model.
*/
func (mdl *Model) Get(key interface{}) (std.Value, error) {
	if std.ModelTypeHash == mdl.typ {
		var ok bool
		var idx int

		// hash keys are always strings
		hashIdx := toString(key)

		mdl.mux.Lock()
		defer mdl.mux.Unlock()

		if idx, ok = mdl.hashIdx[hashIdx]; !ok {
			return nil, errors.New(InvalidIndex, "invalid index '%s'", hashIdx)
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
			return nil, errors.New(InvalidIndex, "invalid index %d", key.(int))
		}
		ret := mdl.data[key.(int)]
		return &Value{ret}, nil
	default:
		return nil, errors.New(InvalidIndexType, "key '%v' is must be an integer", key)
	}
}

/*
GetID returns returns this model's id.
*/
func (mdl *Model) GetID() interface{} {
	return mdl.id
}

/*
GetType returns the model type.
*/
func (mdl *Model) GetType() std.ModelType {
	return mdl.typ
}

/*
Has tests to see of a specified data element exists in this model.
*/
func (mdl *Model) Has(key interface{}) bool {
	if std.ModelTypeList == mdl.GetType() {
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
func (mdl *Model) Map(callback func(std.Value) std.Model) std.Model {
	return nil
}

/*
Merge merges data from any Model into this Model.
*/
func (mdl *Model) Merge(model std.Model) error {
	return nil
}

/*
Next implements std.Iterator.

Next reads the key and value at the current cursor position into pK and pV
respectively and moves the cursor forward one position. If more data is
available Next returns true, else false and resets the cursor postion to the
beginning of the data.
*/
func (mdl *Model) Next(pK, pV *interface{}) bool {
	ret := true
	mdl.mux.Lock()

	mdl.pos++

	// at the end of the data, reset.
	if len(mdl.data) <= mdl.pos {
		mdl.pos = -1
		mdl.mux.Unlock()
		return false
	}

	*pK = mdl.pos
	if std.ModelTypeHash == mdl.typ {
		*pK = mdl.idxHash[mdl.pos]
	}
	*pV = &Value{mdl.data[mdl.pos]}

	mdl.mux.Unlock()
	return true
}

/*
Push a value to the end of the internal data store.
*/
func (mdl *Model) Push(value interface{}) error {
	// std.ModelTypeList only
	if std.ModelTypeList != mdl.typ {
		return errors.New(InvalidMethodContext, "Push() is only valid for std.ModelTypeList model types")
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
func (mdl *Model) Reduce(callback func(std.Value) bool) std.Value {
	return nil
}

/*
Reset implements std.Iterator.

Reset sets the iterator cursor position.
*/
func (mdl *Model) Reset() {
	mdl.pos = -1
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
func (mdl *Model) Set(key interface{}, value interface{}) error {
	// type Hash
	if std.ModelTypeHash == mdl.typ {
		// hash keys are always strings
		idx := toString(key)
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

	// type List
	switch key.(type) {
	case int, int8, int16, int32, int64,
		uint, uint8, uint16, uint32, uint64:
		k := key.(int)
		mdl.mux.Lock()
		defer mdl.mux.Unlock()
		if k >= len(mdl.data) || k < 0 {
			return errors.New(InvalidIndex, "invalid index '%d'", k)
		}
		mdl.data[k] = value
		return nil
	default:
		return errors.New(InvalidIndexType, "key '%v' is must be an integer", key)
	}
}

/*
SetID sets this Model's identifier property.
*/
func (mdl *Model) SetID(id interface{}) {
	mdl.id = id
}

/*
SetType sets the model type. If any data is stored in this model, this
property becomes read-only.
*/
func (mdl *Model) SetType(typ std.ModelType) error {
	if len(mdl.data) > 0 {
		return errors.New(ReadOnlyProperty, "model is not empty, type cannot be modified")
	}
	mdl.typ = typ
	return nil
}

/*
Sort sorts the model data.
*/
func (mdl *Model) Sort(flags uintptr) error {
	return nil
}

/*
modelType is a data type for defining Model types.
*/
type modelType int

func toString(v interface{}) string {
	switch v.(type) {
	case string, []byte, []rune:
	case int, int8, int16, int32, int64:
		v = fmt.Sprintf("%d", v.(int))
	case float32:
		p := strings.Split(fmt.Sprintf("%.25f", v.(float32)), ".")
		v = p[0]
		if 2 == len(p) {
			v = v.(string) + string([]rune(p[1])[:10])
		}
	case float64:
		p := strings.Split(fmt.Sprintf("%.25f", v.(float64)), ".")
		v = p[0]
		if 2 == len(p) {
			v = v.(string) + "." + string([]rune(p[1])[:10])
		}
	default:
		v = fmt.Sprintf("%v", v)
	}
	return v.(string)
}
