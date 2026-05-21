package model

import (
	"encoding/json"

	"github.com/bdlm/errors/v2"
	stdModel "github.com/bdlm/std/v2/model"
)

// MarshalJSON implements json.Marshaler.
func (mdl *Model) MarshalJSON() ([]byte, error) {
	mdl.mux.Lock()
	defer mdl.mux.Unlock()
	if stdModel.ModelTypeList == mdl.GetType() {
		return json.Marshal(mdl.data)
	}
	d := map[string]any{}
	for k, v := range mdl.data {
		d[mdl.idxHash[k]] = v
	}
	return json.Marshal(d)
}

// MarshalModel implements Marshaler.
func (mdl *Model) MarshalModel() ([]byte, error) {
	return mdl.MarshalJSON()
}

// UnmarshalJSON implements json.Unmarshaler.
func (mdl *Model) UnmarshalJSON(jsn []byte) error {
	var data any

	if err := json.Unmarshal(jsn, &data); err != nil {
		return errors.Wrap(err, "unmarshaling failed")
	}
	if _, err := mdl.importData(data); err != nil {
		return errors.Wrap(err, "import failed")
	}
	return nil
}

// UnmarshalModel implements Unmarshaler.
func (mdl *Model) UnmarshalModel(bytes []byte) error {
	if string(bytes) == "null" {
		return nil
	}
	return mdl.UnmarshalJSON(bytes)
}
