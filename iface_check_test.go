package model_test

import (
	"github.com/bdlm/model"
	stdIterator "github.com/bdlm/std/v2/iterator"
	stdModel "github.com/bdlm/std/v2/model"
	stdSorter "github.com/bdlm/std/v2/sorter"
)

// Compile-time assertions that *Model and *Value satisfy the intended interfaces.
var (
	_ stdModel.Model       = (*model.Model)(nil)
	_ stdModel.Value       = (*model.Value)(nil)
	_ stdSorter.Sorter     = (*model.Model)(nil)
	_ stdIterator.Iterator = (*model.Model)(nil)
)
