package model

import (
	"sort"

	"github.com/bdlm/std"
)

/*
Sort sorts the model data.
*/
func (mdl *Model) Sort(flag std.SortFlag) error {
	data := []interface{}{}
	hashIdx := map[string]int{}
	idxHash := map[int]string{}
	switch flag {
	case std.SortByKey:
		if std.ModelTypeHash == mdl.GetType() {
			order := []string{}
			for _, v := range mdl.idxHash {
				order = append(order, v)
			}
			sort.Strings(order)
			for _, v := range order {
				hashIdx[v] = len(data)
				idxHash[len(data)] = v
				data = append(data, mdl.data[mdl.hashIdx[v]])
			}
			mdl.data = data
			mdl.hashIdx = hashIdx
			mdl.idxHash = idxHash
		}
		if std.ModelTypeList == mdl.GetType() {
		}
	}
	return nil
}
