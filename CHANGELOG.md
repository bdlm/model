All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/), and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

- **Major**: backwards incompatible package updates
- **Minor**: feature additions, removal of deprecated features
- **Patch**: bug fixes, backward compatible protobuf model changes, etc.

# v1.0.0 - 2026-05-23

Initial stable release of `github.com/bdlm/model`.

## Overview

`bdlm/model` is a generic, concurrent-safe data container for Go. A single
`Model` holds either a **hash** (string-keyed, ordered map) or a **list**
(integer-indexed array) of arbitrary values. Values are wrapped in a `Value`
type that provides typed accessors. Nested models are supported at arbitrary
depth; JSON unmarshaling builds the nested structure automatically.

## Core API

### Construction

- `New(modelType, data) (*Model, error)` — creates a HASH or LIST model,
  optionally pre-populated from `map[string]any` or `[]any`. Hash models built
  via `New` or `UnmarshalJSON` are automatically sorted by key.

### CRUD

- `Set(key, value) error` — store a value by key; hash keys accept any type
  and are cast to `string` via `bdlm/cast`.
- `Get(key) (Value, error)` — retrieve a value by key or index.
- `Has(key) bool` — check existence without retrieving.
- `Push(value) error` — append to a LIST model.
- `Delete(key) error` — remove by key or index; hash index is rebuilt
  atomically and the backing-array slot is zeroed to release the GC reference.
- `Len() int` — number of elements.

### Metadata

- `GetType() / SetType()` — model type (HASH or LIST); type can only be
  changed while the model is empty.
- `GetID() / SetID()` — arbitrary model identifier; used as the primary sort
  key when comparing nested models.
- `GetData() ([]any, map[string]int, map[int]string)` — returns isolated
  copies of the internal data slice and both index maps.
- `SetData(data) error` — replaces the entire data store atomically.

### Locking

- `Lock()` — makes the model permanently read-only; there is no Unlock.
  All write operations return `ReadOnlyModel` on a locked model.

### Iteration

Bidirectional cursor iterator:

- `Next(pK, pV *any) bool` — advance; resets and returns false at end.
- `Prev(pK, pV *any) bool` — retreat; clamps to -1 and returns false at start.
- `Cur(pK, pV *any) bool` — read current position without moving.
- `Seek(pos any) error` — jump to a key (HASH) or index (LIST).
- `Reset()` — reset cursor to before the first element.

The cursor is reset to -1 on any successful mutation (Delete, SetData, Sort,
Reverse) and is left unchanged on failed operations.

### Sorting

`Sort(SortFlag)` sorts in place. Uses `sync.RWMutex` write lock; does not reset
the cursor unless data ordering actually changes.

Sort flags (`github.com/bdlm/std/v2/sorter`):

| Flag | Value | Meaning |
|------|-------|---------|
| `SortByValue` | `0` | Zero-value default; `Sort(SortByValue)` is a no-op. Trigger value sorting with `SortAsc`, `SortDesc`, or `SortAsString`. |
| `SortByKey` | `1` | Alphabetical key sort (HASH), or no-op on LIST unless combined with `SortAsString`. |
| `SortAsc` | `2` | Ascending; triggers value sorting when used without `SortByKey`. |
| `SortDesc` | `4` | Descending; triggers value sorting when used without `SortByKey`. |
| `SortAsString` | `8` | String comparison via cast; triggers value sorting when used without `SortByKey`. |
| `SortReverse` | `16` | Reverse the final result after all other sorting. |

Only one invalid combination: `SortAsc | SortDesc`.

Type-stratified value ordering (ascending): `*Model < nil < bool < numeric < string < other`.

- `Reverse() error` — reverse in place; rebuilds hash index maps correctly.

### Functional Operations

All three operate on a snapshot taken before the callback is invoked, so
callbacks may safely read or write the model without deadlocking.

- `Filter(func(Value) bool) Model` — returns a new model of the same type
  containing only elements for which the callback returns true.
- `Map(func(Value) Value) Model` — returns a new model with each element
  replaced by the callback's return value.
- `Reduce(func(carry, cur Value) Value) Value` — iteratively reduces to a
  single value; the first element is the initial carry. Returns nil for an
  empty model.

### Merge

`Merge(incoming Model) error` merges all values from `incoming` into the
receiver.

| Receiver | Incoming | Behavior |
|----------|----------|----------|
| HASH | HASH | Incoming wins on conflict; both-model keys are merged recursively |
| HASH | LIST | List indices cast to string become hash keys |
| LIST | LIST | Incoming elements appended |
| LIST | HASH | Hash values appended in insertion order |

Self-merge returns `InvalidMethodContext`.

### JSON

- `MarshalJSON() / UnmarshalJSON()` — standard `encoding/json` interfaces.
  `UnmarshalJSON` builds into a temporary model and swaps internals under a
  single write lock, so concurrent readers never observe an empty intermediate
  state. Repeated calls replace data rather than accumulating.
- `MarshalModel() / UnmarshalModel()` — `Marshaler`/`Unmarshaler` interface
  methods. `UnmarshalModel` treats the JSON `null` literal as a no-op.

## Value Accessors

`Value` wraps `any` and exposes:
`Bool`, `Int`, `Float`, `Float32`, `Float64`, `String`, `Value` (raw),
`Model` (nested model), `List` (`[]stdModel.Value`), `Map` (`map[string]stdModel.Value`).

All numeric/string conversions use `bdlm/cast`; they return an error when the
conversion is not possible.

Package-level generics `To[T]` and `ToE[T]` expose the cast helpers directly.

## Error Sentinels

All errors are compatible with the standard `errors.Is` function:

| Sentinel | Condition |
|----------|-----------|
| `InvalidIndex` | Key or index does not exist |
| `InvalidIndexType` | Wrong key type for this model type |
| `InvalidMethodContext` | Method invalid in context (Push on hash, self-merge) |
| `ReadOnlyModel` | Model is locked or SetType called on non-empty model |
| `InvalidDataSet` | SetData type does not match model type |
| `InvalidSortFlagCombination` | SortAsc and SortDesc combined |

## Concurrency

All public methods are safe for concurrent use. The implementation uses
`sync.RWMutex` — read operations (`Get`, `Has`, `Len`, `Cur`, `GetData`,
`MarshalJSON`) acquire a read lock and run concurrently; write operations
acquire an exclusive lock. The `locked` flag and model type are stored as
`sync/atomic` values. The double-checked locking pattern is used on all write
paths to close the TOCTOU window between the pre-lock check and lock
acquisition.

## Dependencies

| Package | Role |
|---------|------|
| `bdlm/cast/v2` | Type conversion for value accessors and key coercion |
| `bdlm/errors/v2` | Structured error wrapping with sentinel support |
| `bdlm/std/v2` | Interface definitions: `Model`, `Value`, `Iterator`, `Sorter` |
