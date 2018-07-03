package model

import (
	"encoding/json"

	"github.com/bdlm/errors"
	"github.com/bdlm/std"
)

/*
MarshalJSON implements json.Marshaler.
*/
func (mdl *Model) MarshalJSON() ([]byte, error) {
	if std.ModelTypeList == mdl.GetType() {
		return json.Marshal(mdl.data)
	}
	d := map[string]interface{}{}
	for k, v := range mdl.data {
		d[mdl.idxHash[k]] = v
	}
	return json.Marshal(d)
}

/*
MarshalModel implements Marshaler.
*/
func (mdl *Model) MarshalModel() ([]byte, error) {
	return mdl.MarshalJSON()
}

/*
UnmarshalJSON implements json.Unmarshaler.
*/
func (mdl *Model) UnmarshalJSON(jsn []byte) error {
	var data interface{}

	err := json.Unmarshal(jsn, &data)
	if nil != err {
		return errors.Wrap(
			err,
			errors.ErrInvalidJSON,
			"unmarshaling failed",
		)
	}
	mdl.importData(data)

	return nil
}

/*
UnmarshalModel implements Marshaler.
*/
func (mdl *Model) UnmarshalModel() ([]byte, error) {
	return mdl.MarshalJSON()
}
