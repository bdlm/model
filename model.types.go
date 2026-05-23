// Package model provides a generic, concurrent-safe data container for Go.
//
// A [Model] holds either a HASH (string-keyed ordered map) or a LIST
// (integer-indexed array) of arbitrary values. Both modes share a single API
// and may be nested to form trees of arbitrary depth. JSON unmarshaling
// constructs the nested structure automatically.
//
// # Creating models
//
// Use [New] to construct a model, optionally pre-populated with data:
//
//	m, err := model.New(model.HASH, map[string]any{"key": "value"})
//	m, err := model.New(model.LIST, []any{1, 2, 3})
//
// # Values
//
// All stored values are wrapped in a [Value] struct that exposes typed
// accessors ([Value.Bool], [Value.Int], [Value.Float], [Value.String],
// [Value.Model], …). Conversions are performed by the bdlm/cast package and
// return an error when the target type is not representable.
//
// # Iteration
//
// [Model] implements a bidirectional cursor iterator. [Model.Next] advances
// the cursor and reads the current element; [Model.Prev] retreats it;
// [Model.Cur] reads without moving; [Model.Seek] positions the cursor at a
// specific key or index; [Model.Reset] returns to before the first element.
// Successful mutations (Delete, SetData, Sort, Reverse) reset the cursor to
// -1.
//
// # Sorting
//
// [Model.Sort] accepts a [github.com/bdlm/std/v2/sorter.SortFlag] bitmask.
// SortByValue (== 0) is the implicit default; value-based sorting is triggered
// by passing SortAsc, SortDesc, or SortAsString without SortByKey. The only
// invalid flag combination is SortAsc | SortDesc.
//
// # Merging
//
// [Model.Merge] combines two models according to their types:
//   - HASH into HASH: incoming wins on conflict; both-model keys are merged recursively.
//   - LIST into LIST: incoming elements are appended.
//   - LIST into HASH: hash values appended in insertion order, keys ignored.
//   - HASH into LIST: list indices cast to string become hash keys.
//
// # Functional transforms
//
// [Model.Filter], [Model.Map], and [Model.Reduce] operate on a snapshot taken
// before the callback runs, without holding the model mutex. The callback may
// safely read or write the model being transformed.
//
// # JSON
//
// [Model] implements [encoding/json.Marshaler] and [encoding/json.Unmarshaler].
// [Model.UnmarshalJSON] builds into a temporary model and swaps internals
// under one write lock, so concurrent readers never observe an empty
// intermediate state. Hash models are always sorted by key after import.
//
// # Concurrency
//
// All public methods are safe for concurrent use. Read operations ([Model.Get],
// [Model.Has], [Model.Len], [Model.Cur], [Model.GetData], [Model.MarshalJSON])
// acquire a shared read lock and run in parallel with other readers. Write
// operations acquire an exclusive write lock. The locked flag and model type
// are stored as atomic values and are safe to read from any goroutine without
// holding the mutex.
//
// [Model.Merge] releases the receiver's write lock while recursively merging
// nested models and reacquires it afterward; it is not fully atomic in the
// presence of concurrent writers on the receiver.
//
// # Error sentinels
//
// All error returns wrap a package-level sentinel testable with [errors.Is]:
//
//	if errors.Is(err, model.InvalidIndex) { … }
//
// The available sentinels are [InvalidIndex], [InvalidIndexType],
// [InvalidMethodContext], [ReadOnlyModel], [InvalidDataSet], and
// [InvalidSortFlagCombination].
package model
