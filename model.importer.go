package model

import (
	"github.com/bdlm/errors/v2"
	stdModel "github.com/bdlm/std/v2/model"
	stdSorter "github.com/bdlm/std/v2/sorter"
)

func importMap(data map[string]any, node *Model) (*Model, error) {
	for k, v := range data {
		switch typedV := v.(type) {
		case map[string]any:
			n := New(stdModel.ModelTypeHash, nil)
			child, err := importMap(typedV, n)
			if err != nil {
				return nil, errors.Wrap(err, "failed to import nested map at key '%s'", k)
			}
			if err := node.Set(k, child); err != nil {
				return nil, errors.Wrap(err, "failed to set key '%s'", k)
			}
		case []any:
			n := New(stdModel.ModelTypeList, nil)
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

func importSlice(data []any, node *Model) (*Model, error) {
	for i, v := range data {
		switch typedV := v.(type) {
		case map[string]any:
			n := New(stdModel.ModelTypeHash, nil)
			child, err := importMap(typedV, n)
			if err != nil {
				return nil, errors.Wrap(err, "failed to import nested map at index '%d'", i)
			}
			if err := node.Push(child); err != nil {
				return nil, errors.Wrap(err, "failed to push at index '%d'", i)
			}
		case []any:
			n := New(stdModel.ModelTypeList, nil)
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

func (mdl *Model) importData(data any) (*Model, error) {
	switch typedData := data.(type) {
	case map[string]any:
		return importMap(typedData, mdl)
	case []any:
		return importSlice(typedData, mdl)
	}
	return nil, nil
}
