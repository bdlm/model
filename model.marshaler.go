package model

import (
	"encoding/json"

	"github.com/bdlm/errors/v2"
)

// MarshalJSON implements json.Marshaler.
//
// For LIST models the data slice is marshaled as a JSON array. For HASH models
// a map[string]any is assembled from the index maps and marshaled as a JSON
// object. Because encoding/json sorts map keys alphabetically, the JSON key
// order is always alphabetical regardless of the model's current internal
// element order.
//
// Each stored *[Value] is marshaled via its own MarshalJSON method, which
// serializes the underlying data directly rather than as a struct literal with
// unexported fields.
//
// MarshalJSON acquires a read lock and may run concurrently with other read
// operations.
func (mdl *Model) MarshalJSON() ([]byte, error) {
	mdl.mux.RLock()
	defer mdl.mux.RUnlock()
	if LIST == mdl.GetType() {
		return json.Marshal(mdl.data)
	}
	d := map[string]any{}
	for k, v := range mdl.data {
		d[mdl.idxHash[k]] = v
	}
	return json.Marshal(d)
}

// MarshalModel implements the Marshaler interface. It delegates directly to
// [Model.MarshalJSON] and is provided to satisfy custom serialization
// interfaces that distinguish between the standard json.Marshaler and a
// package-specific Marshaler.
func (mdl *Model) MarshalModel() ([]byte, error) {
	return mdl.MarshalJSON()
}

// UnmarshalJSON implements json.Unmarshaler.
//
// The input bytes are first decoded into a raw Go value (map[string]any for
// objects, []any for arrays) using encoding/json. The result is then imported
// into a temporary model of the same type as the receiver. Once import
// succeeds, the receiver's data, hashIdx, idxHash, and pos fields are replaced
// atomically under a single write lock. This swap ensures that concurrent
// readers never observe an empty intermediate state between the old and new
// data.
//
// Calling UnmarshalJSON on a non-empty model replaces all existing data; it
// does not merge. Hash keys are sorted alphabetically after import.
//
// Returns [ReadOnlyModel] if the model is locked. Returns an error if the JSON
// cannot be decoded or if the decoded type is incompatible with the model type
// (for example, a JSON array passed to a HASH model).
func (mdl *Model) UnmarshalJSON(jsn []byte) error {
	if err := mdl.checkLocked(); err != nil {
		return err
	}
	var raw any
	if err := json.Unmarshal(jsn, &raw); err != nil {
		return errors.Wrap(err, "unmarshaling failed")
	}
	// Build into a temporary model (same type), then atomically swap internals.
	tmp, _ := New(mdl.GetType(), nil)
	if _, err := tmp.importData(raw); err != nil {
		return errors.Wrap(err, "import failed")
	}
	mdl.mux.Lock()
	if err := mdl.checkLocked(); err != nil { // double-check inside lock
		mdl.mux.Unlock()
		return err
	}
	mdl.data = tmp.data
	mdl.hashIdx = tmp.hashIdx
	mdl.idxHash = tmp.idxHash
	mdl.pos = -1
	mdl.mux.Unlock()
	return nil
}

// UnmarshalModel implements the Unmarshaler interface. It is a thin wrapper
// around [Model.UnmarshalJSON] that treats the JSON null literal as a no-op,
// leaving the model's existing data intact. All other input is forwarded to
// UnmarshalJSON.
func (mdl *Model) UnmarshalModel(bytes []byte) error {
	if string(bytes) == "null" {
		return nil
	}
	return mdl.UnmarshalJSON(bytes)
}
