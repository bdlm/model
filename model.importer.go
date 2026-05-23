package model

import (
	"github.com/bdlm/errors/v2"
	stdSorter "github.com/bdlm/std/v2/sorter"
)

// importMap populates node with the contents of data, a Go map[string]any as
// produced by encoding/json. Each value is handled as follows:
//
//   - map[string]any: a new HASH child model is created and recursively
//     populated by importMap, then stored with node.Set.
//   - []any: a new LIST child model is created and populated by importSlice,
//     then stored with node.Set.
//   - any other type: stored as-is via node.Set.
//
// After all key-value pairs are inserted, the node is sorted alphabetically
// by key (Sort(SortByKey)) to ensure deterministic iteration order regardless
// of Go's non-deterministic map iteration.
//
// Callers must not hold node's mutex; importMap acquires it internally through
// node.Set and node.Sort.
func importMap(data map[string]any, node *Model) (*Model, error) {
	for k, v := range data {
		switch typedV := v.(type) {
		case map[string]any:
			n, _ := New(HASH, nil)
			child, err := importMap(typedV, n)
			if err != nil {
				return nil, errors.Wrap(err, "failed to import nested map at key '%s'", k)
			}
			if err := node.Set(k, child); err != nil {
				return nil, errors.Wrap(err, "failed to set key '%s'", k)
			}
		case []any:
			n, _ := New(LIST, nil)
			child, err := importSlice(typedV, n)
			if err != nil {
				return nil, errors.Wrap(err, "failed to import nested slice at key '%s'", k)
			}
			if err := node.Set(k, child); err != nil {
				return nil, errors.Wrap(err, "failed to set key '%s'", k)
			}
		default:
			if err := node.Set(k, v); err != nil {
				return nil, errors.Wrap(err, "failed to set key '%s'", k)
			}
		}
	}
	if err := node.Sort(stdSorter.SortByKey); err != nil {
		return nil, errors.Wrap(err, "failed to sort by key")
	}
	return node, nil
}

// importSlice populates node with the contents of data, a Go []any as produced
// by encoding/json. Each element is handled as follows:
//
//   - map[string]any: a new HASH child model is created and populated by
//     importMap, then appended via node.Push.
//   - []any: a new LIST child model is created and recursively populated by
//     importSlice, then appended via node.Push.
//   - any other type: appended as-is via node.Push.
//
// Insertion order is preserved; no sorting is applied to list models.
//
// Callers must not hold node's mutex; importSlice acquires it internally
// through node.Push.
func importSlice(data []any, node *Model) (*Model, error) {
	for i, v := range data {
		switch typedV := v.(type) {
		case map[string]any:
			n, _ := New(HASH, nil)
			child, err := importMap(typedV, n)
			if err != nil {
				return nil, errors.Wrap(err, "failed to import nested map at index '%d'", i)
			}
			if err := node.Push(child); err != nil {
				return nil, errors.Wrap(err, "failed to push at index '%d'", i)
			}
		case []any:
			n, _ := New(LIST, nil)
			child, err := importSlice(typedV, n)
			if err != nil {
				return nil, errors.Wrap(err, "failed to import nested slice at index '%d'", i)
			}
			if err := node.Push(child); err != nil {
				return nil, errors.Wrap(err, "failed to push at index '%d'", i)
			}
		default:
			if err := node.Push(v); err != nil {
				return nil, errors.Wrap(err, "failed to push at index '%d'", i)
			}
		}
	}
	return node, nil
}

// importData is the entry point for all data import operations triggered by
// [New] and [Model.UnmarshalJSON]. It dispatches to [importMap] or
// [importSlice] based on the concrete type of data.
//
// data must be map[string]any (for HASH models) or []any (for LIST models).
// Any other type returns a non-nil error. The receiver mdl is used directly as
// the import target; callers must not hold mdl's mutex.
func (mdl *Model) importData(data any) (*Model, error) {
	switch typedData := data.(type) {
	case map[string]any:
		return importMap(typedData, mdl)
	case []any:
		return importSlice(typedData, mdl)
	default:
		return nil, errors.Errorf("cannot import data of type '%T'", data)
	}
}
