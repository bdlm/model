package model

import (
	"github.com/bdlm/errors/v2"
	stdModel "github.com/bdlm/std/v2/model"
)

/*
Cur implements stdModel.Iterator.

Cur reads the key and value at the current cursor postion into pK and pV
respectively. Cur will return false if no iteration has begun, including
following calls to Reset.
*/
func (mdl *Model) Cur(pK, pV *interface{}) bool {
	if mdl.pos < 0 || mdl.pos >= len(mdl.data) {
		return false
	}

	*pK = mdl.pos
	if stdModel.ModelTypeHash == mdl.GetType() {
		*pK = mdl.idxHash[mdl.pos]
	}
	if tmp, ok := mdl.data[mdl.pos].(*Value); ok && nil != tmp {
		*pV = tmp
	} else {
		*pV = &Value{mdl.data[mdl.pos]}
	}

	return true
}

/*
Next implements stdModel.Iterator.

Next moves the cursor forward one position before reading the key and value
at the cursor position into pK and pV respectively. If data is available at
that position and was written to pK and pV then Next returns true, else
false to signify the end of the data and resets the cursor postion to the
beginning of the data set (-1).
*/
func (mdl *Model) Next(pK, pV *interface{}) bool {
	mdl.mux.Lock()
	mdl.pos++

	// at the end of the data, reset.
	if len(mdl.data) <= mdl.pos {
		mdl.pos = -1
		mdl.mux.Unlock()
		return false
	}

	*pK = mdl.pos
	if stdModel.ModelTypeHash == mdl.GetType() {
		*pK = mdl.idxHash[mdl.pos]
	}
	if tmp, ok := mdl.data[mdl.pos].(*Value); ok && nil != tmp {
		*pV = tmp
	} else {
		*pV = &Value{mdl.data[mdl.pos]}
	}

	mdl.mux.Unlock()
	return true
}

/*
Prev implements stdModel.Iterator.

Prev moves the cursor backward one position before reading the key and value
at the cursor position into pK and pV respectively. If data is available at
that position and was written to pK and pV then Prev returns true, else
false to signify the beginning of the data.
*/
func (mdl *Model) Prev(pK, pV *interface{}) bool {
	mdl.mux.Lock()
	mdl.pos--

	// at the beginning of the data, stop.
	if mdl.pos < 0 {
		mdl.mux.Unlock()
		return false
	}

	*pK = mdl.pos
	if stdModel.ModelTypeHash == mdl.GetType() {
		*pK = mdl.idxHash[mdl.pos]
	}
	if tmp, ok := mdl.data[mdl.pos].(*Value); ok && nil != tmp {
		*pV = tmp
	} else {
		*pV = &Value{mdl.data[mdl.pos]}
	}

	mdl.mux.Unlock()
	return true
}

/*
Reset implements stdModel.Iterator.

Reset sets the iterator cursor position.
*/
func (mdl *Model) Reset() {
	mdl.pos = -1
}

/*
Seek implements stdModel.Iterator.

Seek sets the iterator cursor position.
*/
func (mdl *Model) Seek(pos interface{}) error {
	// List model
	if stdModel.ModelTypeList == mdl.GetType() {
		idx := pos.(int)
		if idx >= len(mdl.data) {
			return errors.WrapE(InvalidIndex, errors.Errorf("the specified position '%d' is beyond the end of the data", idx))
		} else if idx < 0 {
			return errors.WrapE(InvalidIndex, errors.Errorf("invalid index '%d'", idx))
		}
		mdl.pos = idx - 1
		return nil
	}

	// Hash model
	hashKey := pos.(string)
	if idx, ok := mdl.hashIdx[hashKey]; ok {
		mdl.pos = idx - 1
	}
	return errors.WrapE(InvalidIndex, errors.Errorf("the specified position '%s' does not exist", hashKey))
}
