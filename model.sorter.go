package model

import (
	"reflect"
	"sort"

	"github.com/bdlm/cast/v2"
	"github.com/bdlm/errors/v2"
	stdModel "github.com/bdlm/std/v2/model"
	stdSorter "github.com/bdlm/std/v2/sorter"
)

// Len returns the number of items stored in this model.
func (mdl *Model) Len() int {
	return len(mdl.data)
}

// Reverse reverses the order of the data store.
func (mdl *Model) Reverse() error {
	if mdl.locked {
		return errors.WrapE(ReadOnlyProperty, errors.Errorf("model is locked"))
	}
	mdl.mux.Lock()
	defer mdl.mux.Unlock()

	n := len(mdl.data)
	for i, j := 0, n-1; i < j; i, j = i+1, j-1 {
		mdl.data[i], mdl.data[j] = mdl.data[j], mdl.data[i]
	}
	if stdModel.ModelTypeHash == mdl.GetType() {
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
	return nil
}

// Sort sorts the model data by the specified flag.
func (mdl *Model) Sort(flag stdSorter.SortFlag) error {
	if mdl.locked {
		return errors.WrapE(ReadOnlyProperty, errors.Errorf("model is locked"))
	}

	if flag&stdSorter.SortByKey != 0 {
		if stdModel.ModelTypeHash == mdl.GetType() {
			keys := make([]string, 0, len(mdl.idxHash))
			for _, k := range mdl.idxHash {
				keys = append(keys, k)
			}
			sort.Strings(keys)
			mdl.data, mdl.hashIdx, mdl.idxHash = rebuildHashIndexes(keys, mdl)
		}
		// list: integer keys are always positional, SortByKey is a no-op
	}

	if flag&stdSorter.SortByValue != 0 {
		compare := stratifiedLess
		if flag&stdSorter.SortAsString != 0 {
			compare = stringLess
		}

		if stdModel.ModelTypeHash == mdl.GetType() {
			keys := make([]string, 0, len(mdl.idxHash))
			for _, k := range mdl.idxHash {
				keys = append(keys, k)
			}
			sort.SliceStable(keys, func(i, j int) bool {
				return compare(mdl.data[mdl.hashIdx[keys[i]]], mdl.data[mdl.hashIdx[keys[j]]])
			})
			mdl.data, mdl.hashIdx, mdl.idxHash = rebuildHashIndexes(keys, mdl)
		}

		if stdModel.ModelTypeList == mdl.GetType() {
			sort.SliceStable(mdl.data, func(i, j int) bool {
				return compare(mdl.data[i], mdl.data[j])
			})
		}
	}

	if flag&stdSorter.SortReverse != 0 {
		if err := mdl.Reverse(); err != nil {
			return errors.Wrap(err, "failed to reverse data")
		}
	}

	return nil
}

// modelLen returns the number of elements in a Model, or 0 for other types.
func modelLen(v any) int {
	if m, ok := v.(stdModel.Model); ok {
		d, _, _ := m.GetData()
		return len(d)
	}
	return 0
}

// rebuildHashIndexes rebuilds hashIdx and idxHash from an ordered key slice
// after the data slice has been repopulated in that order.
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

// stratifiedLess compares two values using type-stratified ordering:
// Model/*Model < nil < bool < numeric < string < other. Within each bucket,
// values are compared naturally (bool: false<true; numeric: float64; all else: string).
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

// stringLess compares two values by their string representations via cast.
func stringLess(a, b any) bool {
	return cast.To[string](unwrapValue(a)) < cast.To[string](unwrapValue(b))
}

// typeBucket assigns a sort-order bucket to a value for stratified comparison:
// 0=Model/*Model, 1=nil, 2=bool, 3=numeric, 4=string, 5=other.
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

// unwrapValue extracts the underlying data from a *Value wrapper, if present.
func unwrapValue(v any) any {
	if val, ok := v.(*Value); ok {
		return val.data
	}
	return v
}
