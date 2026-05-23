# model

<p align="center">
    <a href="https://gopherize.me/gopher/6a118c02102bbd6c01143238aa26b98dc73b5853"><img src="https://github.com/bdlm/model/wiki/assets/images/gopher.png" width="300px"></a>
</p>

<p align="center">
    <a href="https://github.com/bdlm/model/blob/main/CHANGELOG.md"><img src="https://img.shields.io/github/v/release/bdlm/model" alt="Release"></a>
    <a href="https://pkg.go.dev/github.com/bdlm/model"><img src="https://godoc.org/github.com/bdlm/model?status.svg" alt="GoDoc"></a>
    <a href="https://github.com/bdlm/model/actions/workflows/go.yml"><img src="https://github.com/bdlm/model/actions/workflows/go.yml/badge.svg"></a>
    <a href="https://goreportcard.com/report/github.com/bdlm/model"><img src="https://goreportcard.com/badge/github.com/bdlm/model" alt="Go Report Card"></a>
    <a href="https://github.com/bdlm/model/issues"><img src="https://img.shields.io/github/issues-raw/bdlm/model.svg" alt="Github issues"></a>
    <a href="https://github.com/bdlm/model/pulls"><img src="https://img.shields.io/github/issues-pr/bdlm/model.svg" alt="Github pull requests"></a>
    <a href="https://github.com/bdlm/model/blob/main/LICENSE"><img src="https://img.shields.io/github/license/bdlm/model" alt="MIT"></a>
</p>

<a href="https://github.com/mkenney/software-guides/blob/master/STABILITY-BADGES.md#release-candidate"><img src="https://img.shields.io/badge/stability-pre--release-48c9b0.svg" alt="Release Candidate"></a> Code is fairly settled and is in use in production systems. Backwards-compatibility will be maintained unless serious issues are discovered and a better solution is reached.

`bdlm/model` is a generic, type-agnostic data container for Go. A single `Model` can hold either a **hash** (string-keyed map) or a **list** (integer-indexed array) of arbitrary values, with full support for nested models, bidirectional cursor iteration, sorting, merging, functional transforms, and JSON marshaling.

All public methods are safe for concurrent use.

---

## Installation

```bash
go get github.com/bdlm/model
```

---

## Overview

A `Model` stores values wrapped in a `Value` type that provides typed accessors (`Bool`, `Int`, `Float`, `String`, `Model`, etc.). Two model types are available:

| Constant | Behavior | Keys |
|----------|----------|------|
| `model.HASH` | ordered map | `string` |
| `model.LIST` | indexed array | `int` (0-based) |

Nested structures are supported: any value can itself be a `*Model`, allowing arbitrarily deep trees. JSON unmarshaling builds this structure automatically.

---

## Creating a Model

```go
// Empty hash model
mdl, err := model.New(model.HASH, nil)

// Empty list model
mdl, err := model.New(model.LIST, nil)

// Pre-populated hash model — keys are sorted alphabetically on import
mdl, err := model.New(model.HASH, map[string]any{
    "name": "Alice",
    "age":  30,
})

// Pre-populated list model from a slice
mdl, err := model.New(model.LIST, []any{"one", "two", 3})
```

`New` returns an error if `data` is not `nil`, `map[string]any` (for `HASH`), or `[]any` (for `LIST`).

---

## CRUD Operations

### Set and Get

```go
mdl, _ := model.New(model.HASH, nil)

// Store values — hash keys are always strings (any type is cast via bdlm/cast)
mdl.Set("name", "Alice")
mdl.Set("score", 42)
mdl.Set(100, "numeric keys are cast to string")

// Retrieve a value
val, err := mdl.Get("name")
if err != nil {
    // model.InvalidIndex if key does not exist
}
str, _ := val.String() // "Alice"
```

For list models, keys must be integer types and must already exist (use `Push` to append):

```go
lst, _ := model.New(model.LIST, nil)
lst.Push("first")
lst.Push("second")

val, _ := lst.Get(0)
str, _ := val.String() // "first"

lst.Set(0, "updated")
```

### Push

`Push` appends to a list model. It returns `InvalidMethodContext` on a hash model.

```go
lst, _ := model.New(model.LIST, nil)
lst.Push(1)
lst.Push(2)
lst.Push("three")
// list is now [1, 2, "three"]
```

### Delete

```go
// Hash — delete by string key
mdl.Delete("name")

// List — delete by integer index; remaining elements shift left
lst.Delete(1)
```

`Delete` on a list model requires an integer key. Passing a non-integer type returns `InvalidIndexType`.

### Has

```go
if mdl.Has("name") {
    // key exists
}
```

---

## Value Accessors

Every value retrieved from a model is a `Value` with typed conversion methods. All conversions use [`bdlm/cast`](https://github.com/bdlm/cast) and return an error if the conversion is not possible.

```go
val, _ := mdl.Get("key")

b, err   := val.Bool()
i, err   := val.Int()
f, err   := val.Float()      // float64
f32, err := val.Float32()
f64, err := val.Float64()
s, err   := val.String()
raw      := val.Value()      // untyped any

// Nested model stored at this value
nested, err := val.Model()
```

`List()` and `Map()` are low-level accessors for the uncommon case where a raw `[]stdModel.Value` or `map[string]stdModel.Value` was stored directly in the model. For nested arrays and objects, use a child `*Model` and `val.Model()` instead.

```go
list, err := val.List()   // succeeds only if underlying data is []stdModel.Value
m, err    := val.Map()    // succeeds only if underlying data is map[string]stdModel.Value
```

---

## Iteration

`Model` implements a bidirectional cursor iterator. For hash models, iteration order depends on how the model was built:

- **Built via `New(HASH, map[string]any{...})` or `UnmarshalJSON`**: keys are sorted alphabetically on import.
- **Built via sequential `Set` calls**: keys are returned in insertion order. Call `Sort(SortByKey)` to normalize.

```go
// Keys sorted alphabetically because the model was built via New with a map.
mdl, _ := model.New(model.HASH, map[string]any{"b": 2, "a": 1})

var key, val any
for mdl.Next(&key, &val) {
    n, _ := val.(stdModel.Value).Int()
    fmt.Printf("%s = %d\n", key, n)
}
// a = 1
// b = 2
```

### Iterator Methods

| Method | Description |
|--------|-------------|
| `Next(pK, pV *any) bool` | Advance cursor; returns false and resets at end |
| `Prev(pK, pV *any) bool` | Retreat cursor; returns false at beginning |
| `Cur(pK, pV *any) bool` | Read current position without moving |
| `Seek(pos any) error` | Jump to a specific key (hash) or index (list) |
| `Reset()` | Reset cursor to before the first element |
| `Len() int` | Number of elements |

After `Seek(n)`, `Cur` returns element `n` and `Next` returns element `n+1`.

```go
// Reverse iteration — seek to the last element, then walk backward.
mdl.Seek(mdl.Len() - 1)   // list: seek to last index
mdl.Seek("z")             // hash: seek to key "z"

var key, val any
for mdl.Prev(&key, &val) {
    fmt.Println(key, val.(stdModel.Value).Value())
}
```

---

## Sorting

`Sort(flag)` accepts one or more `SortFlag` values combined with `|`. Invalid combinations return `InvalidSortFlagCombination`.

`SortByValue` is the zero-value default (`0`). Value-based sorting is triggered by providing a direction or modifier flag **without** `SortByKey`:

```go
import "github.com/bdlm/std/v2/sorter"

mdl.Sort(sorter.SortByKey)                               // HASH: alphabetical ascending
mdl.Sort(sorter.SortByKey  | sorter.SortDesc)            // HASH: alphabetical descending
mdl.Sort(sorter.SortAsc)                                 // sort by value, ascending
mdl.Sort(sorter.SortDesc)                                // sort by value, descending
mdl.Sort(sorter.SortAsString)                            // sort by value, string comparison
mdl.Sort(sorter.SortDesc | sorter.SortReverse)           // desc then reversed = ascending
```

### Sort Flags

| Flag | Value | Effect |
|------|-------|--------|
| `SortByValue` | `0` | Zero-value default. Value sorting is implicit; `Sort(SortByValue)` alone is a **no-op**. Trigger it with `SortAsc`, `SortDesc`, or `SortAsString`. |
| `SortByKey` | `1` | Hash: sort keys alphabetically. List: no-op unless combined with `SortAsString`. |
| `SortAsc` | `2` | Ascending order. Triggers value-based sorting when used without `SortByKey`. |
| `SortDesc` | `4` | Descending order. Triggers value-based sorting when used without `SortByKey`. |
| `SortAsString` | `8` | String comparison instead of type-stratified. Triggers value-based string sorting when used without `SortByKey`. With `SortByKey` on a list, sorts list values as strings. |
| `SortReverse` | `16` | Reverse the final result after all other sorting is complete. |

**Only one invalid combination** (returns `InvalidSortFlagCombination`):
- `SortAsc | SortDesc` — conflicting direction flags

### `SortDesc` vs `SortReverse`

`SortDesc` changes the comparison direction before sorting; `SortReverse` reverses the result after sorting. They are fully composable:

| Flags | Effect |
|-------|--------|
| `SortAsc` | ascending by value |
| `SortDesc` | descending by value |
| `SortAsc \| SortReverse` | ascending sorted then reversed = descending |
| `SortDesc \| SortReverse` | descending sorted then reversed = ascending |

### Type-Stratified Order

When sorting by value without `SortAsString`, the ascending sort order across types is:

```
*Model  <  nil  <  false  <  true  <  (numeric)  <  (string, lexicographic)  <  other
```

Within the `*Model` bucket, models are compared by `GetID()` then by element count.

### `SortByKey | SortAsString` on a List

Sorts the list elements by their string representations:

```go
lst, _ := model.New(model.LIST, []any{30, 10, 200})
lst.Sort(sorter.SortByKey | sorter.SortAsString)
// result: [10, 200, 30]  (lexicographic: "10" < "200" < "30")

lst.Sort(sorter.SortByKey | sorter.SortAsString | sorter.SortDesc)
// result: [30, 200, 10]
```

### Standalone `Reverse`

```go
mdl.Reverse() // reverses in place; maintains hash index consistency
```

---

## Merging

`Merge` merges all values from an incoming model into the receiver. Self-merge returns an error.

```go
base, _ := model.New(model.HASH, nil)
base.Set("a", 1)
base.Set("b", 2)

incoming, _ := model.New(model.HASH, nil)
incoming.Set("b", 99)  // overwrites
incoming.Set("c", 3)   // new key

base.Merge(incoming)
// base is now {a:1, b:99, c:3}
```

### Merge Rules

| Receiver type | Incoming type | Behavior |
|---------------|---------------|----------|
| HASH | HASH | Incoming wins on conflict; when both values are `*Model`, they are merged recursively |
| HASH | LIST | List indices cast to string become hash keys |
| LIST | LIST | Incoming elements appended |
| LIST | HASH | Hash values appended in insertion order; keys ignored |

---

## Functional Operations

The callback for `Filter`, `Map`, and `Reduce` is called without holding the model mutex, so it is safe for the callback to read or write the model being iterated.

### Filter

Returns a new model of the same type containing only elements for which the callback returns `true`. The original model is not modified.

```go
nums, _ := model.New(model.LIST, []any{1, 2, 3, 4, 5, 6})
evens := nums.Filter(func(v stdModel.Value) bool {
    n, _ := v.Int()
    return n%2 == 0
})
// evens: [2, 4, 6]
```

### Map

Returns a new model of the same type with each element replaced by the callback's return value. The original model is not modified.

```go
nums, _ := model.New(model.LIST, []any{1, 2, 3})
doubled := nums.Map(func(v stdModel.Value) stdModel.Value {
    n, _ := v.Int()
    tmp, _ := model.New(model.LIST, nil)
    tmp.Push(n * 2)
    result, _ := tmp.Get(0)
    return result
})
// doubled: [2, 4, 6]
```

### Reduce

Iteratively reduces the model to a single value. The first element is used as the initial carry; the callback is first invoked with that carry and the second element. Returns `nil` for an empty model.

```go
nums, _ := model.New(model.LIST, []any{3, 1, 4, 1, 5, 9})
maxVal := nums.Reduce(func(carry, cur stdModel.Value) stdModel.Value {
    a, _ := carry.Int()
    b, _ := cur.Int()
    if b > a {
        return cur
    }
    return carry
})
n, _ := maxVal.Int() // 9
```

---

## JSON Support

`Model` implements `json.Marshaler` and `json.Unmarshaler`. Nested JSON objects become child `HASH` models; nested arrays become child `LIST` models. Hash models are always sorted by key after import.

```go
mdl, _ := model.New(model.HASH, nil)
json.Unmarshal([]byte(`{
    "name": "Alice",
    "scores": [95, 87, 91],
    "address": {"city": "Portland", "state": "OR"}
}`), &mdl)

// Access top-level scalar
val, _ := mdl.Get("name")
name, _ := val.String() // "Alice"

// Access nested list
val, _ = mdl.Get("scores")
nested, _ := val.Model()     // *Model (LIST)
first, _ := nested.Get(0)
score, _ := first.Float()    // 95 (JSON numbers decode as float64)

// Access nested hash
val, _ = mdl.Get("address")
addr, _ := val.Model()       // *Model (HASH)
city, _ := addr.Get("city")
c, _ := city.String()        // "Portland"

// Marshal back to JSON
b, _ := json.Marshal(mdl)
```

### `MarshalModel` / `UnmarshalModel`

The package also defines `Marshaler` and `Unmarshaler` interfaces for custom serialization:

```go
bytes, err := mdl.MarshalModel()       // delegates to MarshalJSON
err        = mdl.UnmarshalModel(bytes) // delegates to UnmarshalJSON; no-op for "null"
```

---

## Locking

`Lock()` makes a model permanently read-only. There is no `Unlock`. All write operations (`Set`, `Push`, `Delete`, `Merge`, `Reverse`, `Sort`, `SetData`, `SetID`, `SetType`, `UnmarshalJSON`) return `ReadOnlyModel` on a locked model.

```go
mdl.Lock()
err := mdl.Set("key", "value") // returns ReadOnlyModel error
```

---

## SetData and SetType

`SetData` replaces the entire data store:

```go
// Replace hash model contents
mdl.SetData(map[string]any{"x": 1, "y": 2})

// Replace list model contents
lst.SetData([]any{10, 20, 30})
```

`SetType` changes the model type, but only while the model is empty:

```go
mdl, _ := model.New(model.HASH, nil)
mdl.SetType(model.LIST) // OK — model is empty
mdl.Push("item")
mdl.SetType(model.HASH) // error: ReadOnlyModel (model is not empty)
```

---

## Model Identity

Every model can carry an arbitrary identifier:

```go
mdl.SetID("user-42")
id := mdl.GetID() // any
```

IDs are used as the primary sort key when sorting a list of models by value, with element count as a tiebreaker.

---

## Errors

All sentinel errors are exported and compatible with the standard `errors.Is` function:

| Sentinel | When returned |
|----------|--------------|
| `InvalidIndex` | Key or index does not exist |
| `InvalidIndexType` | Wrong key type for this model type (e.g. string key on a LIST) |
| `InvalidMethodContext` | Operation invalid in context (e.g., `Push` on a hash, self-merge) |
| `ReadOnlyModel` | Model is locked, or type change attempted on non-empty model |
| `InvalidDataSet` | `SetData` called with a type incompatible with the model type |
| `InvalidSortFlagCombination` | `SortAsc` and `SortDesc` combined |

```go
val, err := mdl.Get("missing")
if errors.Is(err, model.InvalidIndex) {
    // key not found
}
```

---

## Thread Safety

All public methods are safe for concurrent use. Internally, `Model` uses a `sync.RWMutex` — readers (`Get`, `Has`, `Len`, `Cur`, `GetData`, `MarshalJSON`) acquire a read lock and run concurrently; writers acquire an exclusive lock.

The `locked` flag and model type are stored as `sync/atomic` values, making `Lock()` and `GetType()` safe to call from any context including from inside a lock.

The `Merge` operation holds the receiver's mutex for all direct writes. During recursive merges of nested models, the receiver mutex is briefly released and reacquired, so `Merge` is not fully atomic in the presence of concurrent writers on the receiver.

---

## Complete Example

```go
package main

import (
    "encoding/json"
    "fmt"

    "github.com/bdlm/model"
    "github.com/bdlm/std/v2/iterator"
    stdModel "github.com/bdlm/std/v2/model"
)

func main() {
    // Build a hash model from JSON. Nested objects become HASH models;
    // nested arrays become LIST models. Hash keys are sorted alphabetically.
    mdl, _ := model.New(model.HASH, nil)
    json.Unmarshal([]byte(`{
        "users": [
            {"name": "Charlie", "age": 25},
            {"name": "Alice",   "age": 30},
            {"name": "Bob",     "age": 22}
        ]
    }`), &mdl)

    // Get the nested list. val.Model() returns stdModel.Model.
    val, _ := mdl.Get("users")
    users, _ := val.Model()

    // Iterate using the iterator.Iterator interface.
    var key, v any
    for users.(iterator.Iterator).Next(&key, &v) {
        user, _ := v.(stdModel.Value).Model()
        nameVal, _ := user.Get("name")
        name, _ := nameVal.String()
        ageVal, _ := user.Get("age")
        age, _ := ageVal.Float() // JSON numbers decode as float64
        fmt.Printf("%s: %d\n", name, int(age))
    }
    // Charlie: 25
    // Alice: 30
    // Bob: 22

    // Filter to adults only.
    adults := users.Filter(func(v stdModel.Value) bool {
        user, err := v.Model()
        if err != nil {
            return false
        }
        ageVal, _ := user.Get("age")
        age, _ := ageVal.Float()
        return age >= 25
    })
    fmt.Println("Adults:", adults) // [{"age":25,...},{"age":30,...}]

    // Marshal the root model back to JSON.
    b, _ := json.Marshal(mdl)
    fmt.Println(string(b))
}
```

---

## Dependencies

| Package | Role |
|---------|------|
| [`bdlm/cast/v2`](https://github.com/bdlm/cast) | Type conversion for value accessors and key coercion |
| [`bdlm/errors/v2`](https://github.com/bdlm/errors) | Structured error wrapping with sentinel support |
| [`bdlm/std/v2`](https://github.com/bdlm/std) | Interface definitions: `Model`, `Value`, `Iterator`, `Sorter` |
