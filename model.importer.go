package model

import (
	stdModel "github.com/bdlm/std/v2/model"
	stdSorter "github.com/bdlm/std/v2/sorter"
)

func importMap(data map[string]interface{}, node *Model) *Model {
	for k, v := range data {
		switch typedV := v.(type) {
		case map[string]interface{}:
			n := New(stdModel.ModelTypeHash)
			node.Set(k, importMap(typedV, n))
		case []interface{}:
			n := New(stdModel.ModelTypeList)
			node.Set(k, importSlice(typedV, n))
		default:
			if stdModel.ModelTypeHash == node.GetType() {
				node.Set(k, v)
			}
			if stdModel.ModelTypeList == node.GetType() {
				node.Push(v)
			}
		}
	}
	node.Sort(stdSorter.SortByKey)
	return node
}

func importSlice(data []interface{}, node *Model) *Model {
	for _, v := range data {
		switch typedV := v.(type) {
		case map[string]interface{}:
			n := New(stdModel.ModelTypeHash)
			node.Push(importMap(typedV, n))
		case []interface{}:
			n := New(stdModel.ModelTypeList)
			node.Push(importSlice(typedV, n))
		default:
			node.Push(v)
		}
	}
	node.Sort(stdSorter.SortByKey)
	return node
}

func (mdl *Model) importData(data interface{}) *Model {
	switch typedData := data.(type) {
	case map[string]interface{}:
		return importMap(typedData, mdl)
	case []interface{}:
		return importSlice(typedData, mdl)
	}
	return nil
}
