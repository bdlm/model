package model

import (
	"github.com/bdlm/cast/v2"
	"github.com/bdlm/errors/v2"
)

// Cur reads the key and value at the current cursor position into *pK and *pV
// and returns true. It returns false — and leaves *pK and *pV unchanged — when
// the cursor is before the first element (position -1) or beyond the last
// element.
//
// The cursor is at -1 immediately after construction, after [Model.Reset], and
// after any successful mutation (Delete, SetData, Sort, Reverse). Cur returns
// false in all those states.
//
// For HASH models *pK is set to the string key; for LIST models *pK is set to
// the integer index.
//
// Cur acquires a read lock and may run concurrently with other read operations.
// Note that the cursor (pos) is a shared mutable value written by Next, Prev,
// Reset, and Seek; concurrent iteration from multiple goroutines will
// interleave cursor movements non-deterministically.
func (mdl *Model) Cur(pK, pV *any) bool {
	mdl.mux.RLock()
	defer mdl.mux.RUnlock()
	if mdl.pos < 0 || mdl.pos >= len(mdl.data) {
		return false
	}
	*pK = mdl.pos
	if HASH == mdl.GetType() {
		*pK = mdl.idxHash[mdl.pos]
	}
	*pV = toValue(mdl.data[mdl.pos])
	return true
}

// Next advances the cursor by one position and reads the key and value at the
// new position into *pK and *pV. It returns true if an element was written, or
// false when the cursor moves past the last element. When Next returns false it
// also resets the cursor to -1, so the next Next call restarts from the
// beginning of the data.
//
// A typical forward-iteration loop:
//
//	var key, val any
//	for m.Next(&key, &val) {
//	    // process key and val.(stdModel.Value)
//	}
//
// For HASH models *pK is set to the string key; for LIST models *pK is set to
// the integer index.
//
// Next acquires an exclusive write lock to update the cursor atomically with
// the read of the current element.
func (mdl *Model) Next(pK, pV *any) bool {
	mdl.mux.Lock()
	mdl.pos++

	// at the end of the data, reset.
	if len(mdl.data) <= mdl.pos {
		mdl.pos = -1
		mdl.mux.Unlock()
		return false
	}

	*pK = mdl.pos
	if HASH == mdl.GetType() {
		*pK = mdl.idxHash[mdl.pos]
	}
	*pV = toValue(mdl.data[mdl.pos])

	mdl.mux.Unlock()
	return true
}

// Prev retreats the cursor by one position and reads the key and value at the
// new position into *pK and *pV. It returns true if an element was written, or
// false when the cursor would move before the first element. When Prev returns
// false it clamps the cursor to -1 so that a subsequent [Model.Next] call
// restarts from the beginning of the data rather than attempting a negative
// slice index.
//
// A typical backward-iteration loop:
//
//	m.Seek(m.Len() - 1) // start at the last element
//	var key, val any
//	for m.Prev(&key, &val) {
//	    // process key and val.(stdModel.Value)
//	}
//
// For HASH models *pK is set to the string key; for LIST models *pK is set to
// the integer index.
//
// Prev acquires an exclusive write lock to update the cursor atomically with
// the read of the current element.
func (mdl *Model) Prev(pK, pV *any) bool {
	mdl.mux.Lock()
	mdl.pos--

	// at the beginning of the data, stop and clamp to -1 so that a
	// subsequent Next() does not attempt a negative slice index.
	if mdl.pos < 0 {
		mdl.pos = -1
		mdl.mux.Unlock()
		return false
	}

	*pK = mdl.pos
	if HASH == mdl.GetType() {
		*pK = mdl.idxHash[mdl.pos]
	}
	*pV = toValue(mdl.data[mdl.pos])

	mdl.mux.Unlock()
	return true
}

// Reset sets the cursor to position -1, which is before the first element.
// After Reset, [Model.Cur] returns false and the next [Model.Next] call
// returns the first element.
//
// Reset acquires an exclusive write lock to update the cursor atomically.
func (mdl *Model) Reset() {
	mdl.mux.Lock()
	mdl.pos = -1
	mdl.mux.Unlock()
}

// Seek positions the cursor at pos so that [Model.Cur] immediately returns the
// element at that position and the next [Model.Next] call returns the following
// element.
//
// For LIST models, pos is converted to int via cast.ToE[int]; non-integer types
// that cannot be cast return [InvalidIndexType]. A negative value or a value
// >= [Model.Len] returns [InvalidIndex].
//
// For HASH models, pos is cast to string via bdlm/cast. If no element with
// that key exists, [InvalidIndex] is returned.
//
// Seek acquires an exclusive write lock to update the cursor atomically.
func (mdl *Model) Seek(pos any) error {
	mdl.mux.Lock()
	defer mdl.mux.Unlock()

	// List model
	if LIST == mdl.GetType() {
		idx, err := cast.ToE[int](pos)
		if err != nil {
			return errors.WrapE(InvalidIndexType, errors.Errorf("position '%v' must be an integer", pos))
		}
		if idx < 0 {
			return errors.WrapE(InvalidIndex, errors.Errorf("invalid index '%d'", idx))
		}
		if idx >= len(mdl.data) {
			return errors.WrapE(InvalidIndex, errors.Errorf("the specified position '%d' is beyond the end of the data", idx))
		}
		mdl.pos = idx
		return nil
	}

	// Hash model
	hashKey := cast.To[string](pos)
	if idx, ok := mdl.hashIdx[hashKey]; ok {
		mdl.pos = idx
		return nil
	}
	return errors.WrapE(InvalidIndex, errors.Errorf("the specified position '%s' does not exist", hashKey))
}
