package model

import (
	"sort"
	"sync"
	"sync/atomic"

	"github.com/bdlm/cast/v2"
	"github.com/bdlm/errors/v2"
	stdModel "github.com/bdlm/std/v2/model"
)

const (
	// HASH is the model type for a string-keyed ordered map. All keys are
	// normalized to string via bdlm/cast: Set, Get, Has, and Delete accept any
	// type and cast it to string before use. When a HASH model is created via
	// [New] with initial data or populated via [Model.UnmarshalJSON], keys are
	// sorted alphabetically. When populated via sequential [Model.Set] calls,
	// keys are stored in insertion order; call [Model.Sort] to normalize.
	//
	// Re-exported from github.com/bdlm/std/v2/model for caller convenience.
	HASH = stdModel.HASH

	// LIST is the model type for an integer-indexed array. Keys must be one of
	// the standard integer types (int, int8, int16, int32, int64, uint, uint8,
	// uint16, uint32, uint64). Indices are zero-based and contiguous; deleting
	// an element shifts all higher-indexed elements left. New elements are
	// added via [Model.Push]; [Model.Set] updates existing positions only.
	//
	// Re-exported from github.com/bdlm/std/v2/model for caller convenience.
	LIST = stdModel.LIST
)

// Model is the core data container. It holds an ordered collection of values
// in either HASH (string-keyed map) or LIST (integer-indexed array) mode.
//
// Internally, all elements are stored in a single contiguous slice (data).
// For HASH models two companion maps maintain the bidirectional key↔position
// relationship: hashIdx maps each string key to its position in data, and
// idxHash maps each position back to its key. This layout gives O(1) reads
// and stable insertion order independently of Go's non-deterministic map
// iteration.
//
// A sync.RWMutex guards all mutable fields. Read operations acquire a shared
// lock; write operations acquire an exclusive lock. The locked flag and model
// type are stored as atomic values so that [Model.Lock] and [Model.GetType]
// are always safe to call without holding the mutex.
//
// Model satisfies the following interfaces from github.com/bdlm/std/v2:
//   - model.Model — the full data-container interface
//   - iterator.Iterator — bidirectional cursor iteration
//   - sorter.Sorter — in-place sorting and reversal
//   - fmt.Stringer — JSON-formatted string representation
type Model struct {
	// id is the model's optional identifier, set by SetID and read by GetID.
	// It is not interpreted by the model; it serves as the primary sort key
	// when comparing nested models during value-based sorting.
	id any

	// locked is the read-only flag. Once set by Lock it is never cleared.
	// Stored atomically so that checkLocked can be called without the mutex.
	locked atomic.Bool

	// typ stores the model type (HASH or LIST) as an int64. Using an atomic
	// allows GetType to be called safely from within callbacks that already
	// hold the mutex.
	typ atomic.Int64

	// mux protects data, hashIdx, idxHash, pos, and id. Read operations
	// acquire RLock; write operations acquire Lock.
	mux *sync.RWMutex

	// data is the element store in model order (index 0 to Len-1). Each
	// element is a *Value. The slice may contain non-Value entries only
	// immediately after a raw SetData call; all other write paths go through
	// toValue, which normalizes entries to *Value.
	data []any

	// hashIdx maps string key → position in data. Only populated for HASH
	// models. Must always be consistent with idxHash and data.
	hashIdx map[string]int

	// idxHash maps position in data → string key. Only populated for HASH
	// models. Must always be consistent with hashIdx and data.
	idxHash map[int]string

	// pos is the cursor position used by Next, Prev, Cur, Seek, and Reset.
	// A value of -1 means "before the first element". Any successful mutation
	// (Delete, SetData, Sort, Reverse) resets pos to -1.
	pos int
}

// New creates and returns a new Model of the given modelType.
//
// If data is non-nil it is imported immediately:
//   - For HASH models, data must be map[string]any. Nested map values become
//     child HASH models; nested slice values become child LIST models. Keys
//     are sorted alphabetically after all key-value pairs are inserted.
//   - For LIST models, data must be []any. Nested maps and slices are
//     converted to child models in the same way.
//
// Any other non-nil data type returns a non-nil error and a nil *Model.
// Passing nil data creates an empty model ready for use.
func New(modelType stdModel.ModelType, data any) (*Model, error) {
	model := &Model{
		mux:     &sync.RWMutex{},
		hashIdx: map[string]int{},
		idxHash: map[int]string{},
		pos:     -1,
	}
	model.typ.Store(int64(modelType))
	if data != nil {
		if _, err := model.importData(data); err != nil {
			return nil, errors.Wrap(err, "failed to import data")
		}
	}
	return model, nil
}

// toValue returns v wrapped in a *Value. If v is already a non-nil *Value it
// is returned as-is to avoid double-wrapping. A nil *Value is still wrapped
// so that callers always receive a valid, non-nil *Value.
func toValue(v any) *Value {
	if val, ok := v.(*Value); ok && val != nil {
		return val
	}
	return &Value{v}
}

// asModel extracts a stdModel.Model from v. If v is a *Value the extraction
// looks through the wrapper to the underlying data before attempting the type
// assertion, so both raw stdModel.Model values and *Value-wrapped ones are
// handled transparently.
func asModel(v any) (stdModel.Model, bool) {
	if val, ok := v.(*Value); ok {
		v = val.data
	}
	m, ok := v.(stdModel.Model)
	return m, ok
}

// checkLocked returns a [ReadOnlyModel] error if the model's locked flag is
// set. It must be called both before acquiring the write mutex (fast path for
// already-locked models) and again after acquiring it (to close the TOCTOU
// window between the pre-check and the mux.Lock call; another goroutine could
// call Lock between those two points).
func (mdl *Model) checkLocked() error {
	if mdl.locked.Load() {
		return errors.WrapE(ReadOnlyModel, errors.Errorf("model is locked"))
	}
	return nil
}

// Delete removes the element identified by key from the model.
//
// For LIST models, key must be one of the integer types (int, int8, int16,
// int32, int64, uint, uint8, uint16, uint32, uint64); any other type returns
// [InvalidIndexType]. A valid integer key that is out of range returns
// [InvalidIndex]. After deletion all elements at higher indices shift left,
// maintaining contiguous zero-based indexing.
//
// For HASH models, key is cast to string via bdlm/cast. If the resulting key
// does not exist, [InvalidIndex] is returned. After deletion, the positions of
// all elements that had a higher index than the deleted element are decremented
// by one; both index maps are updated atomically within the write lock.
//
// A successful delete resets the cursor to -1. A failed delete (wrong type or
// out-of-range key) leaves the cursor unchanged.
//
// Returns [ReadOnlyModel] if the model is locked.
func (mdl *Model) Delete(key any) error {
	if err := mdl.checkLocked(); err != nil {
		return err
	}
	mdl.mux.Lock()
	defer mdl.mux.Unlock()
	if err := mdl.checkLocked(); err != nil {
		return err
	}
	if LIST == mdl.GetType() {
		switch key.(type) {
		case int, int8, int16, int32, int64,
			uint, uint8, uint16, uint32, uint64:
		default:
			return errors.WrapE(InvalidIndexType, errors.Errorf("key '%v' must be an integer", key))
		}
		k := cast.To[int](key)
		if k < 0 || k >= len(mdl.data) {
			return errors.WrapE(InvalidIndex, errors.Errorf("index '%d' out of range", k))
		}
		mdl.pos = -1 // successful mutation invalidates cursor
		n := len(mdl.data)
		copy(mdl.data[k:], mdl.data[k+1:])
		mdl.data[n-1] = nil // zero last slot to allow GC of the moved element
		mdl.data = mdl.data[:n-1]
		return nil
	}

	k := cast.To[string](key)
	if idx, ok := mdl.hashIdx[k]; ok {
		mdl.pos = -1 // successful mutation invalidates cursor
		n := len(mdl.data)
		copy(mdl.data[idx:], mdl.data[idx+1:])
		mdl.data[n-1] = nil // zero last slot to allow GC of the moved element
		mdl.data = mdl.data[:n-1]
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

// Filter returns a new model of the same type containing only the elements for
// which callback returns true. The original model is not modified.
//
// Filter operates on a point-in-time snapshot obtained via [Model.GetData]
// before the first callback invocation. The callback is called without holding
// the model mutex, so it is safe for the callback to read or write the model
// being filtered without deadlocking.
//
// For HASH models the result preserves the keys and their snapshot order. For
// LIST models elements are appended to the result in snapshot order.
func (mdl *Model) Filter(callback func(stdModel.Value) bool) stdModel.Model {
	modelType := mdl.GetType()
	data, _, idxHash := mdl.GetData()
	result, _ := New(modelType, nil)
	for i, v := range data {
		val := toValue(v)
		if callback(val) {
			if HASH == modelType {
				result.Set(idxHash[i], val)
			} else {
				result.Push(val)
			}
		}
	}
	return result
}

// Get returns the value stored at key.
//
// For HASH models, key is cast to string via bdlm/cast. If the resulting key
// does not exist, [InvalidIndex] is returned.
//
// For LIST models, key must be one of the integer types (int, int8, int16,
// int32, int64, uint, uint8, uint16, uint32, uint64); any other type returns
// [InvalidIndexType]. An integer key that is negative or >= [Model.Len]
// returns [InvalidIndex].
//
// On success the returned value is always a non-nil *[Value].
func (mdl *Model) Get(key any) (stdModel.Value, error) {
	if HASH == mdl.GetType() {
		var ok bool
		var idx int

		// hash keys are always strings
		hashIdx := cast.To[string](key)

		mdl.mux.RLock()
		defer mdl.mux.RUnlock()

		if idx, ok = mdl.hashIdx[hashIdx]; !ok {
			return nil, errors.WrapE(InvalidIndex, errors.Errorf("invalid index '%s'", hashIdx))
		}

		return toValue(mdl.data[idx]), nil
	}

	// List model
	switch key.(type) {
	case int, int8, int16, int32, int64,
		uint, uint8, uint16, uint32, uint64:
		mdl.mux.RLock()
		defer mdl.mux.RUnlock()
		k := cast.To[int](key)
		if k < 0 || k >= len(mdl.data) {
			return nil, errors.WrapE(InvalidIndex, errors.Errorf("invalid index '%d'", k))
		}
		return toValue(mdl.data[k]), nil
	default:
		return nil, errors.WrapE(InvalidIndexType, errors.Errorf("key '%v' must be an integer", key))
	}
}

// GetData returns isolated copies of the model's internal data slice and both
// index maps. Callers may read or modify the returned values without affecting
// the model's internal state.
//
// The first return value is a shallow copy of the data slice in current model
// order (insertion order, or whatever order Sort or Reverse last established).
// The second is a copy of hashIdx (key → position) and the third is a copy of
// idxHash (position → key). For LIST models, both maps are empty but non-nil.
//
// GetData is the recommended way to snapshot state before iterating with an
// external loop or before passing data to a concurrent worker. It is also used
// internally by [Model.Filter], [Model.Map], and [Model.Reduce] to take a
// snapshot before invoking user callbacks.
func (mdl *Model) GetData() ([]any, map[string]int, map[int]string) {
	mdl.mux.RLock()
	defer mdl.mux.RUnlock()
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

// GetID returns the model's identifier as set by [Model.SetID]. Returns nil if
// SetID has never been called. The identifier is not interpreted by the model;
// it is stored and returned as-is. It is used as the primary sort key when
// comparing nested models during value-based sorting (see [Model.Sort]).
func (mdl *Model) GetID() any {
	mdl.mux.RLock()
	defer mdl.mux.RUnlock()
	return mdl.id
}

// GetType returns the model type, either [HASH] or [LIST]. The type is read
// atomically and is safe to call from any goroutine, including from within a
// callback or a method that already holds the model mutex.
func (mdl *Model) GetType() stdModel.ModelType {
	return stdModel.ModelType(mdl.typ.Load())
}

// Has reports whether an element identified by key exists in the model.
//
// For LIST models, key must be one of the integer types (int, int8, int16,
// int32, int64, uint, uint8, uint16, uint32, uint64); any other type returns
// false. A valid integer key that is negative or >= [Model.Len] returns false.
//
// For HASH models, key is cast to string via bdlm/cast and compared against
// the key index.
func (mdl *Model) Has(key any) bool {
	mdl.mux.RLock()
	defer mdl.mux.RUnlock()
	if LIST == mdl.GetType() {
		switch key.(type) {
		case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
			k := cast.To[int](key)
			if k >= 0 && k < len(mdl.data) {
				return true
			}
		}
	} else {
		if _, ok := mdl.hashIdx[cast.To[string](key)]; ok {
			return true
		}
	}
	return false
}

// Lock marks the model permanently read-only. Once called, all write
// operations (Set, Push, Delete, Merge, Reverse, Sort, SetData, SetID,
// SetType, UnmarshalJSON) return [ReadOnlyModel] and leave the model
// unchanged. There is no corresponding Unlock; the decision is irreversible.
//
// Lock stores the locked flag atomically, so any goroutine that subsequently
// reads the flag — whether under the write mutex or not — observes the locked
// state immediately.
func (mdl *Model) Lock() {
	mdl.locked.Store(true)
}

// Map returns a new model of the same type with each element replaced by the
// value returned by callback. The original model is not modified.
//
// Map operates on a point-in-time snapshot obtained via [Model.GetData] before
// the first callback invocation. The callback is called without holding the
// model mutex, so it is safe for the callback to read or write the model being
// mapped without deadlocking.
//
// For HASH models the result preserves the original keys and their snapshot
// order. For LIST models elements appear in snapshot order.
func (mdl *Model) Map(callback func(stdModel.Value) stdModel.Value) stdModel.Model {
	modelType := mdl.GetType()
	data, _, idxHash := mdl.GetData()
	result, _ := New(modelType, nil)
	for i, v := range data {
		val := toValue(v)
		mapped := callback(val)
		if HASH == modelType {
			result.Set(idxHash[i], mapped)
		} else {
			result.Push(mapped)
		}
	}
	return result
}

// Merge merges all values from incoming into the receiver model. The merge
// strategy depends on the combination of model types:
//
//   - HASH ← HASH: For each key in incoming, if the key exists in both models
//     and both values are themselves models, the sub-models are merged
//     recursively. Otherwise the incoming value overwrites the receiver's value
//     for that key. Keys present only in the receiver are preserved; keys
//     present only in incoming are added.
//
//   - HASH ← LIST: Each list element's integer index is cast to string and
//     used as a hash key. Existing hash keys with matching string-cast indices
//     are overwritten.
//
//   - LIST ← LIST: Incoming elements are appended to the receiver in order.
//
//   - LIST ← HASH: Hash values are appended to the receiver in their current
//     insertion order; keys are ignored.
//
// Merge returns [InvalidMethodContext] if incoming is the same pointer as the
// receiver (self-merge). It returns [ReadOnlyModel] if the receiver is locked.
//
// During recursive sub-model merges the receiver's write lock is released and
// reacquired. Within that window a concurrent Lock call can take effect; the
// lock state is rechecked after reacquisition and the merge is aborted if the
// receiver has been locked.
func (mdl *Model) Merge(incoming stdModel.Model) error {
	if err := mdl.checkLocked(); err != nil {
		return err
	}
	if m, ok := incoming.(*Model); ok && m == mdl {
		return errors.WrapE(InvalidMethodContext, errors.Errorf("cannot merge a model into itself"))
	}

	inData, inHashIdx, _ := incoming.GetData()

	held := true
	mdl.mux.Lock()
	defer func() {
		if held {
			mdl.mux.Unlock()
		}
	}()
	if err := mdl.checkLocked(); err != nil { // double-check inside lock
		return err
	}

	switch mdl.GetType() {
	case HASH:
		if incoming.GetType() == HASH {
			// hash into hash: incoming wins; merge nested models recursively
			for key, inIdx := range inHashIdx {
				inVal := inData[inIdx]
				myIdx, exists := mdl.hashIdx[key]
				var myVal any
				if exists {
					myVal = mdl.data[myIdx]
				}
				if exists {
					myModel, myIsModel := asModel(myVal)
					inModel, inIsModel := asModel(inVal)
					if myIsModel && inIsModel {
						held = false
						mdl.mux.Unlock()
						err := myModel.Merge(inModel)
						mdl.mux.Lock()
						held = true
						if err2 := mdl.checkLocked(); err2 != nil {
							return err2
						}
						if err != nil {
							return err
						}
						continue
					}
				}
				if err := mdl.set(key, inVal); err != nil {
					return err
				}
			}
		} else {
			// list into hash: string-cast indices become keys
			for i, v := range inData {
				if err := mdl.set(cast.To[string](i), v); err != nil {
					return err
				}
			}
		}

	case LIST:
		if incoming.GetType() == LIST {
			// list into list: append
			for _, v := range inData {
				if err := mdl.push(v); err != nil {
					return err
				}
			}
		} else {
			// hash into list: append values in insertion order, ignore keys
			for i := 0; i < len(inData); i++ {
				if err := mdl.push(inData[i]); err != nil {
					return err
				}
			}
		}
	}

	return nil
}

// Push appends value to the end of a LIST model and returns nil on success.
// Returns [InvalidMethodContext] for HASH models. Returns [ReadOnlyModel] if
// the model is locked.
func (mdl *Model) Push(value any) error {
	if err := mdl.checkLocked(); err != nil {
		return err
	}
	mdl.mux.Lock()
	defer mdl.mux.Unlock()
	if err := mdl.checkLocked(); err != nil {
		return err
	}
	return mdl.push(value)
}

// push is the write-lock-held implementation of [Model.Push]. Callers must
// hold mdl.mux exclusively. Returns [InvalidMethodContext] if the model type
// is not LIST; does not check the locked flag (callers are responsible for
// that check before acquiring the lock).
func (mdl *Model) push(value any) error {
	if LIST != mdl.GetType() {
		return errors.WrapE(InvalidMethodContext, errors.Errorf("Push() is only valid for LIST model types"))
	}
	mdl.data = append(mdl.data, toValue(value))
	return nil
}

// Reduce iteratively reduces the model to a single value by applying callback
// to each element in sequence. The first element serves as the initial carry
// value; the callback is first invoked with that carry and the second element.
// The return value of each invocation becomes the carry for the next. After
// all elements are processed the final carry is returned.
//
// For an empty model Reduce returns nil without invoking the callback.
//
// Reduce operates on a point-in-time snapshot obtained via [Model.GetData]
// before the first callback invocation. The callback is called without holding
// the model mutex, so it is safe for the callback to read or write the model.
func (mdl *Model) Reduce(callback func(carry, cur stdModel.Value) stdModel.Value) stdModel.Value {
	data, _, _ := mdl.GetData()
	if len(data) == 0 {
		return nil
	}
	var carry stdModel.Value = toValue(data[0])
	for i := 1; i < len(data); i++ {
		carry = callback(carry, toValue(data[i]))
	}
	return carry
}

// Set stores value at key in the model.
//
// For HASH models, key is cast to string via bdlm/cast. If the key already
// exists its value is overwritten in place, preserving the existing insertion
// order. New keys are appended at the end. Hash keys are not automatically
// re-sorted after a Set call; call [Model.Sort] if a specific order is needed.
//
// For LIST models, key must be one of the integer types (int, int8, int16,
// int32, int64, uint, uint8, uint16, uint32, uint64); any other type returns
// [InvalidIndexType]. The index must be in [0, Len); out-of-range indices
// return [InvalidIndex]. To append new elements to a list, use [Model.Push].
//
// Set does not reset the cursor; the cursor position remains valid after a
// Set call regardless of which element was updated.
//
// Returns [ReadOnlyModel] if the model is locked.
func (mdl *Model) Set(key any, value any) error {
	if err := mdl.checkLocked(); err != nil {
		return err
	}
	mdl.mux.Lock()
	defer mdl.mux.Unlock()
	if err := mdl.checkLocked(); err != nil {
		return err
	}
	return mdl.set(key, value)
}

// set is the write-lock-held implementation of [Model.Set]. Callers must hold
// mdl.mux exclusively and must have verified the locked flag before acquiring
// the lock.
func (mdl *Model) set(key any, value any) error {
	if HASH == mdl.GetType() {
		idx := cast.To[string](key)
		if _, ok := mdl.hashIdx[idx]; !ok {
			mdl.hashIdx[idx] = len(mdl.data)
			mdl.idxHash[len(mdl.data)] = idx
			mdl.data = append(mdl.data, toValue(value))
			return nil
		}
		mdl.data[mdl.hashIdx[idx]] = toValue(value)
		return nil
	}
	switch key.(type) {
	case int, int8, int16, int32, int64,
		uint, uint8, uint16, uint32, uint64:
		k := cast.To[int](key)
		if k >= len(mdl.data) || k < 0 {
			return errors.WrapE(InvalidIndex, errors.Errorf("invalid index '%d'", k))
		}
		mdl.data[k] = toValue(value)
		return nil
	default:
		return errors.WrapE(InvalidIndexType, errors.Errorf("key '%v' must be an integer", key))
	}
}

// SetID sets the model's identifier to id. The identifier is stored as-is and
// returned unchanged by [Model.GetID]. It has no semantic meaning to the model
// itself, but is used as the primary sort key when comparing nested models
// during value-based sorting (see [Model.Sort]).
//
// Returns [ReadOnlyModel] if the model is locked.
func (mdl *Model) SetID(id any) error {
	if err := mdl.checkLocked(); err != nil {
		return err
	}
	mdl.mux.Lock()
	defer mdl.mux.Unlock()
	if err := mdl.checkLocked(); err != nil {
		return err
	}
	mdl.id = id
	return nil
}

// SetData replaces the entire contents of the model with data.
//
// For LIST models, data must be []any. The input slice is copied so that
// subsequent mutations to the original slice do not affect the model.
//
// For HASH models, data must be map[string]any. The key-value pairs are
// inserted into a fresh data store in ascending alphabetical key order,
// matching the deterministic ordering applied by [Model.UnmarshalJSON] and
// [importMap]. Values are stored without wrapping in *[Value]; they are wrapped
// transparently on retrieval by [Model.Get] and the iterator methods.
//
// A successful SetData resets the cursor to -1. A failed call (wrong type for
// the model type) leaves the cursor unchanged and returns [InvalidDataSet].
//
// Returns [ReadOnlyModel] if the model is locked.
func (mdl *Model) SetData(data any) error {
	if err := mdl.checkLocked(); err != nil {
		return err
	}
	mdl.mux.Lock()
	defer mdl.mux.Unlock()
	if err := mdl.checkLocked(); err != nil {
		return err
	}
	if LIST == mdl.GetType() {
		d, ok := data.([]any)
		if !ok {
			return errors.WrapE(InvalidDataSet, errors.Errorf("invalid data set for list model"))
		}
		mdl.pos = -1 // successful mutation invalidates cursor
		mdl.data = make([]any, len(d))
		copy(mdl.data, d)
		return nil
	}

	d, ok := data.(map[string]any)
	if !ok {
		return errors.WrapE(InvalidDataSet, errors.Errorf("invalid data set for hash model"))
	}

	keys := make([]string, 0, len(d))
	for k := range d {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	mdl.pos = -1 // successful mutation invalidates cursor
	mdl.data = make([]any, 0, len(d))
	mdl.hashIdx = make(map[string]int, len(d))
	mdl.idxHash = make(map[int]string, len(d))
	for _, k := range keys {
		mdl.hashIdx[k] = len(mdl.data)
		mdl.idxHash[len(mdl.data)] = k
		mdl.data = append(mdl.data, d[k])
	}
	return nil
}

// SetType changes the model type to typ. It succeeds only when the model is
// empty; if any data has been stored, SetType returns [ReadOnlyModel] to
// signal that the type is effectively read-only while data is present.
//
// Returns [ReadOnlyModel] if the model is locked or non-empty.
func (mdl *Model) SetType(typ stdModel.ModelType) error {
	if err := mdl.checkLocked(); err != nil {
		return err
	}
	mdl.mux.Lock()
	defer mdl.mux.Unlock()
	if err := mdl.checkLocked(); err != nil {
		return err
	}
	if len(mdl.data) > 0 {
		return errors.WrapE(ReadOnlyModel, errors.Errorf("model is not empty, type cannot be modified"))
	}
	mdl.typ.Store(int64(typ))
	return nil
}

// String implements [fmt.Stringer] by returning the JSON representation of the
// model. If [Model.MarshalJSON] returns an error, String returns the error
// message instead of valid JSON.
func (mdl *Model) String() string {
	byts, err := mdl.MarshalJSON()
	if err != nil {
		return err.Error()
	}
	return string(byts)
}
