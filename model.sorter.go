package model

import (
	"sort"

	stdModel "github.com/bdlm/std/v2/model"
	stdSorter "github.com/bdlm/std/v2/sorter"
)

/*
Sort sorts the model data.
*/
func (mdl *Model) Sort(flag stdSorter.SortFlag) error {
	data := []interface{}{}
	hashIdx := map[string]int{}
	idxHash := map[int]string{}
	switch flag {
	case stdSorter.SortByKey:
		if stdModel.ModelTypeHash == mdl.GetType() {
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
		if stdModel.ModelTypeList == mdl.GetType() {
		}
	}
	return nil
}
