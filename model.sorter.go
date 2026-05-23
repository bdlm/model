package model

import (
	"reflect"
	"sort"

	"github.com/bdlm/cast/v2"
	"github.com/bdlm/errors/v2"
	stdModel "github.com/bdlm/std/v2/model"
	stdSorter "github.com/bdlm/std/v2/sorter"
)

// Len returns the number of elements currently stored in the model. It
// acquires a read lock and may run concurrently with other read operations.
func (mdl *Model) Len() int {
	mdl.mux.RLock()
	defer mdl.mux.RUnlock()
	return len(mdl.data)
}

// Reverse reverses the order of all elements in the model in place.
//
// For HASH models both index maps (hashIdx and idxHash) are rebuilt to reflect
// the new positions; all existing keys remain accessible at their reversed
// positions. For LIST models the data slice is reversed in place.
//
// A successful Reverse resets the cursor to -1. Returns [ReadOnlyModel] if the
// model is locked.
func (mdl *Model) Reverse() error {
	if err := mdl.checkLocked(); err != nil {
		return err
	}
	mdl.mux.Lock()
	defer mdl.mux.Unlock()
	if err := mdl.checkLocked(); err != nil {
		return err
	}
	mdl.reverse()
	mdl.pos = -1 // mutation invalidates cursor
	return nil
}

// reverse is the write-lock-held implementation of [Model.Reverse]. Callers
// must hold mdl.mux exclusively. It reverses the data slice and, for HASH
// models, rebuilds both index maps so that each key continues to resolve to
// its (now-reversed) position. The cursor is not touched here; callers are
// responsible for resetting it.
func (mdl *Model) reverse() {
	n := len(mdl.data)
	for i, j := 0, n-1; i < j; i, j = i+1, j-1 {
		mdl.data[i], mdl.data[j] = mdl.data[j], mdl.data[i]
	}
	if HASH == mdl.GetType() {
		newHashIdx := make(map[string]int, n)
		newIdxHash := make(map[int]string, n)
		for oldIdx, key := range mdl.idxHash {
			newIdx := n - 1 - oldIdx
			newHashIdx[key] = newIdx
			newIdxHash[newIdx] = key
		}
		mdl.hashIdx = newHashIdx
		mdl.idxHash = newIdxHash
	}
}

// Sort reorders the model's elements according to flag. It returns
// [ReadOnlyModel] if the model is locked, and [InvalidSortFlagCombination] if
// mutually exclusive flags are combined.
//
// # Flag semantics
//
// SortByValue (== 0) is the implicit default for value-based sorting. It
// contributes no bits of its own; value-based sorting is triggered by
// providing SortAsc, SortDesc, or SortAsString without SortByKey. Calling
// Sort(0) or Sort(SortByValue) is therefore a no-op.
//
//   - SortByKey (1): For HASH models, sort alphabetically by key, ascending
//     unless SortDesc is also set. For LIST models, SortByKey alone is a
//     no-op; combine with SortAsString to sort list elements by their string
//     representations.
//   - SortAsc (2): Ascending order. Triggers value-based sorting when used
//     without SortByKey.
//   - SortDesc (4): Descending order. Triggers value-based sorting when used
//     without SortByKey.
//   - SortAsString (8): Compare using string representations via bdlm/cast
//     rather than the default type-stratified ordering. Triggers value-based
//     sorting when used without SortByKey.
//   - SortReverse (16): Reverse the final result after all other sorting is
//     complete. May be combined with any other flag.
//
// The only invalid combination is SortAsc | SortDesc.
//
// # Value-based type-stratified ordering (ascending)
//
//	*Model  <  nil  <  false  <  true  <  numeric  <  string  <  other
//
// Within the *Model bucket, models are compared first by GetID() (cast to
// string) then by element count. Within the numeric bucket, all values are
// widened to float64 before comparison.
//
// # Cursor behavior
//
// The cursor is reset to -1 only when data ordering actually changes. A no-op
// call (e.g., Sort(0) or SortByKey on a LIST without SortAsString) leaves the
// cursor unchanged.
func (mdl *Model) Sort(flag stdSorter.SortFlag) error {
	if err := mdl.checkLocked(); err != nil {
		return err
	}
	if flag&stdSorter.SortAsc != 0 && flag&stdSorter.SortDesc != 0 {
		return errors.WrapE(InvalidSortFlagCombination, errors.Errorf("SortAsc and SortDesc cannot be combined"))
	}

	desc := flag&stdSorter.SortDesc != 0

	mdl.mux.Lock()
	defer mdl.mux.Unlock()
	if err := mdl.checkLocked(); err != nil { // double-check inside lock
		return err
	}

	// pos is reset only when data ordering actually changes.
	didSort := false

	if flag&stdSorter.SortByKey != 0 {
		if HASH == mdl.GetType() {
			keys := make([]string, 0, len(mdl.idxHash))
			for _, k := range mdl.idxHash {
				keys = append(keys, k)
			}
			sort.Strings(keys) // always collect ascending first
			if desc {
				for i, j := 0, len(keys)-1; i < j; i, j = i+1, j-1 {
					keys[i], keys[j] = keys[j], keys[i]
				}
			}
			mdl.data, mdl.hashIdx, mdl.idxHash = rebuildHashIndexes(keys, mdl)
			didSort = true
		}
		// For list models, SortByKey alone is a no-op (integer keys are always
		// positional). SortByKey|SortAsString sorts list values as strings.
		if LIST == mdl.GetType() && flag&stdSorter.SortAsString != 0 {
			compare := lessFn(stringLess, desc)
			sort.SliceStable(mdl.data, func(i, j int) bool {
				return compare(mdl.data[i], mdl.data[j])
			})
			didSort = true
		}
	}

	// SortByValue is the zero-value default: sort by value whenever SortByKey is
	// not set and at least one sort-relevant modifier (direction or SortAsString)
	// is present. A bare flag==0 (or flag==SortByValue) is a no-op.
	if flag&stdSorter.SortByKey == 0 && flag&(stdSorter.SortAsc|stdSorter.SortDesc|stdSorter.SortAsString) != 0 {
		base := stratifiedLess
		if flag&stdSorter.SortAsString != 0 {
			base = stringLess
		}
		compare := lessFn(base, desc)

		if HASH == mdl.GetType() {
			keys := make([]string, 0, len(mdl.idxHash))
			for _, k := range mdl.idxHash {
				keys = append(keys, k)
			}
			sort.SliceStable(keys, func(i, j int) bool {
				return compare(mdl.data[mdl.hashIdx[keys[i]]], mdl.data[mdl.hashIdx[keys[j]]])
			})
			mdl.data, mdl.hashIdx, mdl.idxHash = rebuildHashIndexes(keys, mdl)
			didSort = true
		}

		if LIST == mdl.GetType() {
			sort.SliceStable(mdl.data, func(i, j int) bool {
				return compare(mdl.data[i], mdl.data[j])
			})
			didSort = true
		}
	}

	if flag&stdSorter.SortReverse != 0 {
		mdl.reverse()
		didSort = true
	}

	if didSort {
		mdl.pos = -1
	}

	return nil
}

// lessFn adapts a less-than comparator for ascending or descending order. When
// desc is false the original function is returned unchanged. When desc is true
// a wrapper is returned that swaps the arguments, effectively reversing the
// comparison without rewriting the underlying comparator.
func lessFn(fn func(a, b any) bool, desc bool) func(a, b any) bool {
	if !desc {
		return fn
	}
	return func(a, b any) bool { return fn(b, a) }
}

// modelLen returns the number of elements in v if v implements stdModel.Model,
// or 0 otherwise. It is used as a tiebreaker in [stratifiedLess] when two
// models have the same GetID value.
func modelLen(v any) int {
	if m, ok := v.(stdModel.Model); ok {
		d, _, _ := m.GetData()
		return len(d)
	}
	return 0
}

// rebuildHashIndexes constructs a new data slice and both index maps from an
// ordered key slice, reading the associated values from src. It is called
// after any in-place reordering of a HASH model (Sort, Reverse) to keep
// data, hashIdx, and idxHash mutually consistent.
//
// Callers must hold src's write lock and must ensure that every key in keys is
// present in src.hashIdx.
func rebuildHashIndexes(keys []string, src *Model) ([]any, map[string]int, map[int]string) {
	data := make([]any, 0, len(keys))
	hashIdx := make(map[string]int, len(keys))
	idxHash := make(map[int]string, len(keys))
	for _, k := range keys {
		hashIdx[k] = len(data)
		idxHash[len(data)] = k
		data = append(data, src.data[src.hashIdx[k]])
	}
	return data, hashIdx, idxHash
}

// stratifiedLess reports whether a < b using type-stratified ordering. Values
// are first unwrapped from any *Value wrapper by [unwrapValue], then assigned
// to a type bucket by [typeBucket], and finally compared within their bucket.
//
// Cross-bucket comparison returns bucket(a) < bucket(b). Ascending bucket
// order: 0=Model, 1=nil, 2=bool, 3=numeric, 4=string, 5=other.
//
// Within-bucket rules:
//   - Model: compared by GetID() cast to string; ties broken by element count.
//   - nil: all nil values are equal; returns false.
//   - bool: false < true.
//   - numeric: all values widened to float64 for comparison.
//   - string / other: compared as string via bdlm/cast.
func stratifiedLess(a, b any) bool {
	a, b = unwrapValue(a), unwrapValue(b)
	ba, bb := typeBucket(a), typeBucket(b)
	if ba != bb {
		return ba < bb
	}
	switch ba {
	case 0: // both Model/*Model — compare by ID, tiebreak by element count
		ma, mb := a.(stdModel.Model), b.(stdModel.Model)
		aID, bID := cast.To[string](ma.GetID()), cast.To[string](mb.GetID())
		if aID != bID {
			return aID < bID
		}
		return modelLen(a) < modelLen(b)
	case 1: // both nil
		return false
	case 2: // both bool
		return !cast.To[bool](a) && cast.To[bool](b)
	case 3: // both numeric
		return cast.To[float64](a) < cast.To[float64](b)
	default: // string or other
		return cast.To[string](a) < cast.To[string](b)
	}
}

// stringLess reports whether the string representation of a is less than that
// of b. Both values are unwrapped from any *Value wrapper before conversion.
// String representations are produced by bdlm/cast.
func stringLess(a, b any) bool {
	return cast.To[string](unwrapValue(a)) < cast.To[string](unwrapValue(b))
}

// typeBucket returns the sort-order bucket index for v:
//
//	0 — stdModel.Model (any value satisfying the interface)
//	1 — nil
//	2 — bool
//	3 — any integer or floating-point numeric type
//	4 — string
//	5 — any other type
//
// v must already be unwrapped from any *Value before calling typeBucket; use
// [unwrapValue] first.
func typeBucket(v any) int {
	if _, ok := v.(stdModel.Model); ok {
		return 0
	}
	if v == nil {
		return 1
	}
	switch reflect.TypeOf(v).Kind() {
	case reflect.Bool:
		return 2
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64:
		return 3
	case reflect.String:
		return 4
	default:
		return 5
	}
}

// unwrapValue returns the data held inside a *Value wrapper, or v itself if v
// is not a *Value. It normalizes values before comparison so that raw data and
// *Value-wrapped data sort identically.
func unwrapValue(v any) any {
	if val, ok := v.(*Value); ok {
		return val.data
	}
	return v
}
