package model

import "github.com/bdlm/std"

func importMap(data map[string]interface{}, node *Model) *Model {
	for k, v := range data {
		switch typedV := v.(type) {
		case map[string]interface{}:
			n := New(std.ModelTypeHash)
			node.Set(k, importMap(typedV, n))
		case []interface{}:
			n := New(std.ModelTypeList)
			node.Set(k, importSlice(typedV, n))
		default:
			if std.ModelTypeHash == node.GetType() {
				node.Set(k, v)
			}
			if std.ModelTypeList == node.GetType() {
				node.Push(v)
			}
		}
	}
	node.Sort(std.SortByKey)
	return node
}

func importSlice(data []interface{}, node *Model) *Model {
	for _, v := range data {
		switch typedV := v.(type) {
		case map[string]interface{}:
			n := New(std.ModelTypeHash)
			node.Push(importMap(typedV, n))
		case []interface{}:
			n := New(std.ModelTypeList)
			node.Push(importSlice(typedV, n))
		default:
			node.Push(v)
		}
	}
	node.Sort(std.SortByKey)
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
