package model_test

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/bdlm/model"
	"github.com/bdlm/std/v2/iterator"
	stdModel "github.com/bdlm/std/v2/model"
	"github.com/bdlm/std/v2/sorter"
)

// ── Construction ──────────────────────────────────────────────────────────────

// ExampleNew demonstrates creating a HASH model from JSON and iterating it.
func ExampleNew() {
	mdl, _ := model.New(model.HASH, nil)
	json.Unmarshal(
		[]byte(`{"key1":"value1","key2":2,"key3":["one","two","three"],"key4":{"k1":"v1","k2":"v2"}}`),
		&mdl,
	)
	var key, val any
	for mdl.Next(&key, &val) {
		if "key3" == key.(string) || "key4" == key.(string) {
			var k2, v2 any
			m2, _ := val.(stdModel.Value).Model()
			fmt.Println(key)
			for m2.(iterator.Iterator).Next(&k2, &v2) {
				fmt.Println("   ", k2, v2.(stdModel.Value).Value())
			}
		} else {
			fmt.Println(key, val.(stdModel.Value).Value())
		}
	}

	// Output: key1 value1
	//key2 2
	//key3
	//     0 one
	//     1 two
	//     2 three
	//key4
	//     k1 v1
	//     k2 v2
}

// ExampleNew_withHashData demonstrates constructing a HASH model with initial
// data. importMap automatically sorts keys alphabetically.
func ExampleNew_withHashData() {
	m, _ := model.New(model.HASH, map[string]any{"b": 2, "a": 1})
	var k, v any
	for m.Next(&k, &v) {
		n, _ := v.(stdModel.Value).Int()
		fmt.Printf("%s=%d\n", k, n)
	}

	// Output:
	// a=1
	// b=2
}

// ExampleNew_withListData demonstrates constructing a LIST model with initial
// data. Insertion order is preserved.
func ExampleNew_withListData() {
	m, _ := model.New(model.LIST, []any{"x", "y", "z"})
	var k, v any
	for m.Next(&k, &v) {
		s, _ := v.(stdModel.Value).String()
		fmt.Printf("[%d]=%s\n", k, s)
	}

	// Output:
	// [0]=x
	// [1]=y
	// [2]=z
}

// ── Type ──────────────────────────────────────────────────────────────────────

// ExampleModel_GetType demonstrates reading the model type.
func ExampleModel_GetType() {
	hash, _ := model.New(model.HASH, nil)
	list, _ := model.New(model.LIST, nil)
	fmt.Println(hash.GetType() == model.HASH)
	fmt.Println(list.GetType() == model.LIST)

	// Output:
	// true
	// true
}

// ExampleModel_SetType demonstrates changing the type of an empty model.
func ExampleModel_SetType() {
	m, _ := model.New(model.HASH, nil)
	err := m.SetType(model.LIST) // only succeeds on an empty model
	fmt.Println(err)
	fmt.Println(m.GetType() == model.LIST)

	// Output:
	// <nil>
	// true
}

// ── Identity ──────────────────────────────────────────────────────────────────

// ExampleModel_SetID demonstrates setting a model identifier.
func ExampleModel_SetID() {
	m, _ := model.New(model.HASH, nil)
	m.SetID("my-model")
	fmt.Println(m.GetID())

	// Output:
	// my-model
}

// ExampleModel_GetID demonstrates retrieving the model identifier.
func ExampleModel_GetID() {
	m, _ := model.New(model.HASH, nil)
	fmt.Println(m.GetID()) // nil before any SetID call
	m.SetID(42)
	fmt.Println(m.GetID())

	// Output:
	// <nil>
	// 42
}

// ── CRUD — HASH ───────────────────────────────────────────────────────────────

// ExampleModel_Set demonstrates storing values in a HASH model. Hash keys are
// always cast to string, so integer and other key types are accepted.
func ExampleModel_Set() {
	m, _ := model.New(model.HASH, nil)
	m.Set("name", "Alice")
	m.Set("age", 30)

	v, _ := m.Get("name")
	s, _ := v.String()
	fmt.Println(s)

	v, _ = m.Get("age")
	n, _ := v.Int()
	fmt.Println(n)

	// Output:
	// Alice
	// 30
}

// ExampleModel_Set_update demonstrates that Set overwrites an existing key.
func ExampleModel_Set_update() {
	m, _ := model.New(model.HASH, nil)
	m.Set("k", 1)
	m.Set("k", 2)
	v, _ := m.Get("k")
	n, _ := v.Int()
	fmt.Println(n)

	// Output:
	// 2
}

// ExampleModel_Get demonstrates retrieving values from HASH and LIST models.
func ExampleModel_Get() {
	h, _ := model.New(model.HASH, nil)
	h.Set("color", "blue")
	v, _ := h.Get("color")
	s, _ := v.String()
	fmt.Println(s)

	l, _ := model.New(model.LIST, []any{10, 20, 30})
	v, _ = l.Get(1)
	n, _ := v.Int()
	fmt.Println(n)

	// Output:
	// blue
	// 20
}

// ExampleModel_Has demonstrates checking whether a key exists.
func ExampleModel_Has() {
	h, _ := model.New(model.HASH, nil)
	h.Set("present", 1)
	fmt.Println(h.Has("present"))
	fmt.Println(h.Has("missing"))

	l, _ := model.New(model.LIST, []any{1, 2, 3})
	fmt.Println(l.Has(0))
	fmt.Println(l.Has(5))

	// Output:
	// true
	// false
	// true
	// false
}

// ExampleModel_Delete demonstrates removing a key from a HASH model and an
// element from a LIST model.
func ExampleModel_Delete() {
	h, _ := model.New(model.HASH, nil)
	h.Set("a", 1)
	h.Set("b", 2)
	h.Delete("a")
	fmt.Println(h.Has("a"))
	fmt.Println(h.Len())

	l, _ := model.New(model.LIST, []any{10, 20, 30})
	l.Delete(1) // removes element at index 1
	fmt.Println(l.Len())
	v, _ := l.Get(1) // former index 2 is now index 1
	n, _ := v.Int()
	fmt.Println(n)

	// Output:
	// false
	// 1
	// 2
	// 30
}

// ── CRUD — LIST ───────────────────────────────────────────────────────────────

// ExampleModel_Push demonstrates appending values to a LIST model.
func ExampleModel_Push() {
	m, _ := model.New(model.LIST, nil)
	m.Push("first")
	m.Push("second")
	m.Push("third")
	fmt.Println(m.Len())
	v, _ := m.Get(2)
	s, _ := v.String()
	fmt.Println(s)

	// Output:
	// 3
	// third
}

// ── Length ────────────────────────────────────────────────────────────────────

// ExampleModel_Len demonstrates returning the number of elements stored.
func ExampleModel_Len() {
	m, _ := model.New(model.LIST, []any{1, 2, 3})
	fmt.Println(m.Len())
	m.Push(4)
	fmt.Println(m.Len())
	m.Delete(0)
	fmt.Println(m.Len())

	// Output:
	// 3
	// 4
	// 3
}

// ── Data replacement ──────────────────────────────────────────────────────────

// ExampleModel_SetData demonstrates replacing all data in a LIST model.
func ExampleModel_SetData() {
	m, _ := model.New(model.LIST, []any{1, 2, 3})
	m.SetData([]any{10, 20})
	fmt.Println(m.Len())
	fmt.Println(m)

	// Output:
	// 2
	// [10,20]
}

// ExampleModel_SetData_hash demonstrates replacing all data in a HASH model.
// The map iteration order is non-deterministic, so Sort is called to ensure
// a consistent result.
func ExampleModel_SetData_hash() {
	m, _ := model.New(model.HASH, nil)
	m.Set("old", 99)
	m.SetData(map[string]any{"a": 1, "b": 2})
	m.Sort(sorter.SortByKey) // normalize order
	var k, v any
	for m.Next(&k, &v) {
		n, _ := v.(stdModel.Value).Int()
		fmt.Printf("%s=%d\n", k, n)
	}

	// Output:
	// a=1
	// b=2
}

// ExampleModel_GetData demonstrates retrieving a snapshot of the raw data
// slice and index maps. Mutations to the returned copies do not affect the
// model.
func ExampleModel_GetData() {
	m, _ := model.New(model.LIST, []any{"a", "b", "c"})
	data, _, _ := m.GetData()
	for i, item := range data {
		s, _ := item.(stdModel.Value).String()
		fmt.Printf("[%d]=%s\n", i, s)
	}

	// Output:
	// [0]=a
	// [1]=b
	// [2]=c
}

// ExampleModel_GetData_hash demonstrates using the index maps returned by
// GetData to look up keys by position and positions by key.
func ExampleModel_GetData_hash() {
	m, _ := model.New(model.HASH, nil)
	m.Set("x", 10)
	m.Set("y", 20)
	_, hashIdx, idxHash := m.GetData()
	fmt.Println("x at index:", hashIdx["x"])
	fmt.Println("key at index 1:", idxHash[1])

	// Output:
	// x at index: 0
	// key at index 1: y
}

// ── Lock ──────────────────────────────────────────────────────────────────────

// ExampleModel_Lock demonstrates making a model permanently read-only. There
// is no Unlock — once locked a model accepts reads but rejects all mutations.
func ExampleModel_Lock() {
	m, _ := model.New(model.HASH, nil)
	m.Set("k", 1)
	m.Lock()

	err := m.Set("k", 2) // write rejected
	fmt.Println(err != nil)

	v, _ := m.Get("k") // reads still work
	n, _ := v.Int()
	fmt.Println(n)

	// Output:
	// true
	// 1
}

// ── Iteration ─────────────────────────────────────────────────────────────────

// ExampleModel_Next demonstrates forward iteration. When the end of the data
// is reached, Next resets the cursor to -1 and returns false.
func ExampleModel_Next() {
	m, _ := model.New(model.HASH, nil)
	m.Set("a", 1)
	m.Set("b", 2)
	m.Set("c", 3)
	var k, v any
	for m.Next(&k, &v) {
		n, _ := v.(stdModel.Value).Int()
		fmt.Printf("%s=%d\n", k, n)
	}

	// Output:
	// a=1
	// b=2
	// c=3
}

// ExampleModel_Prev demonstrates backward iteration starting from a Seek
// position. Prev clamps to -1 so that a subsequent Next restarts from the
// beginning.
func ExampleModel_Prev() {
	m, _ := model.New(model.LIST, []any{10, 20, 30})
	m.Seek(2) // start at the last element
	var k, v any
	for m.Prev(&k, &v) {
		n, _ := v.(stdModel.Value).Int()
		fmt.Println(n)
	}

	// Output:
	// 20
	// 10
}

// ExampleModel_Cur demonstrates reading the key and value at the current
// cursor position without advancing the cursor.
func ExampleModel_Cur() {
	m, _ := model.New(model.LIST, []any{10, 20, 30})
	var k, v any
	fmt.Println(m.Cur(&k, &v)) // false before any iteration
	m.Next(&k, &v)             // pos → 0
	m.Next(&k, &v)             // pos → 1
	m.Cur(&k, &v)
	n, _ := v.(stdModel.Value).Int()
	fmt.Println(n)

	// Output:
	// false
	// 20
}

// ExampleModel_Seek demonstrates positioning the cursor at a specific index
// so that Cur returns the element at that position.
func ExampleModel_Seek() {
	m, _ := model.New(model.LIST, []any{10, 20, 30})
	m.Seek(2)
	var k, v any
	m.Cur(&k, &v)
	n, _ := v.(stdModel.Value).Int()
	fmt.Println(n)

	// Output:
	// 30
}

// ExampleModel_Seek_hash demonstrates seeking to a hash key by name.
func ExampleModel_Seek_hash() {
	m, _ := model.New(model.HASH, nil)
	m.Set("a", 1)
	m.Set("b", 2)
	m.Seek("b")
	var k, v any
	m.Cur(&k, &v)
	n, _ := v.(stdModel.Value).Int()
	fmt.Println(k, n)

	// Output:
	// b 2
}

// ExampleModel_Reset demonstrates resetting the cursor to before the first
// element so that the next Next call returns the first element.
func ExampleModel_Reset() {
	m, _ := model.New(model.LIST, []any{1, 2, 3})
	var k, v any
	m.Next(&k, &v) // pos → 0
	m.Next(&k, &v) // pos → 1
	m.Reset()
	m.Next(&k, &v) // pos → 0 again
	n, _ := v.(stdModel.Value).Int()
	fmt.Println(n)

	// Output:
	// 1
}

// ── Sort & Reverse ────────────────────────────────────────────────────────────

// ExampleModel_Sort demonstrates sorting a LIST model by value in ascending
// order. SortByValue (==0) is the default; pass SortAsc to trigger it.
func ExampleModel_Sort() {
	m, _ := model.New(model.LIST, nil)
	m.Push(3)
	m.Push(1)
	m.Push(2)
	m.Sort(sorter.SortAsc)
	var k, v any
	for m.Next(&k, &v) {
		n, _ := v.(stdModel.Value).Int()
		fmt.Println(n)
	}

	// Output:
	// 1
	// 2
	// 3
}

// ExampleModel_Sort_byKey demonstrates sorting a HASH model alphabetically
// by key.
func ExampleModel_Sort_byKey() {
	m, _ := model.New(model.HASH, nil)
	m.Set("c", 3)
	m.Set("a", 1)
	m.Set("b", 2)
	m.Sort(sorter.SortByKey)
	var k, v any
	for m.Next(&k, &v) {
		n, _ := v.(stdModel.Value).Int()
		fmt.Printf("%s=%d\n", k, n)
	}

	// Output:
	// a=1
	// b=2
	// c=3
}

// ExampleModel_Sort_descending demonstrates sorting a LIST model by value in
// descending order.
func ExampleModel_Sort_descending() {
	m, _ := model.New(model.LIST, nil)
	m.Push(1)
	m.Push(3)
	m.Push(2)
	m.Sort(sorter.SortDesc)
	var k, v any
	for m.Next(&k, &v) {
		n, _ := v.(stdModel.Value).Int()
		fmt.Println(n)
	}

	// Output:
	// 3
	// 2
	// 1
}

// ExampleModel_Sort_asString demonstrates sorting a LIST model by the string
// representation of each value (lexicographic order).
func ExampleModel_Sort_asString() {
	m, _ := model.New(model.LIST, []any{30, 10, 200})
	m.Sort(sorter.SortByKey | sorter.SortAsString) // SortByKey+SortAsString sorts list values as strings
	fmt.Println(m) // "10" < "200" < "30" lexicographically

	// Output:
	// [10,200,30]
}

// ExampleModel_Reverse demonstrates reversing the order of elements.
func ExampleModel_Reverse() {
	m, _ := model.New(model.LIST, []any{1, 2, 3})
	m.Reverse()
	fmt.Println(m)

	// Output:
	// [3,2,1]
}

// ── Functional ────────────────────────────────────────────────────────────────

// ExampleModel_Filter demonstrates filtering a LIST model, returning a new
// model containing only the elements for which the callback returns true. The
// original model is not modified.
func ExampleModel_Filter() {
	m, _ := model.New(model.LIST, []any{1, 2, 3, 4, 5})
	evens := m.Filter(func(v stdModel.Value) bool {
		n, _ := v.Int()
		return n%2 == 0
	})
	fmt.Println(evens)
	fmt.Println(m.Len()) // original unchanged

	// Output:
	// [2,4]
	// 5
}

// ExampleModel_Filter_hash demonstrates filtering a HASH model. Matching
// key→value pairs are preserved in the result.
func ExampleModel_Filter_hash() {
	m, _ := model.New(model.HASH, nil)
	m.Set("a", 1)
	m.Set("b", 20)
	m.Set("c", 3)
	m.Set("d", 40)
	big := m.Filter(func(v stdModel.Value) bool {
		n, _ := v.Int()
		return n > 10
	})
	fmt.Println(big)

	// Output:
	// {"b":20,"d":40}
}

// ExampleModel_Map demonstrates transforming every element of a LIST model
// with a callback, returning a new model with the mapped values.
func ExampleModel_Map() {
	m, _ := model.New(model.LIST, []any{1, 2, 3})
	doubled := m.Map(func(v stdModel.Value) stdModel.Value {
		n, _ := v.Int()
		tmp, _ := model.New(model.LIST, nil)
		tmp.Push(n * 2)
		r, _ := tmp.Get(0)
		return r
	})
	fmt.Println(doubled)

	// Output:
	// [2,4,6]
}

// ExampleModel_Reduce demonstrates reducing a LIST model to a single value
// using a callback. The first element is used as the initial carry value.
func ExampleModel_Reduce() {
	m, _ := model.New(model.LIST, []any{3, 1, 4, 1, 5, 9, 2, 6})
	maxVal := m.Reduce(func(carry, cur stdModel.Value) stdModel.Value {
		a, _ := carry.Int()
		b, _ := cur.Int()
		if b > a {
			return cur
		}
		return carry
	})
	n, _ := maxVal.Int()
	fmt.Println(n)

	// Output:
	// 9
}

// ── Merge ─────────────────────────────────────────────────────────────────────

// ExampleModel_Merge demonstrates merging one HASH model into another.
// Keys present in both models are overwritten by the incoming values.
func ExampleModel_Merge() {
	base, _ := model.New(model.HASH, nil)
	base.Set("a", 1)
	base.Set("b", 2)

	incoming, _ := model.New(model.HASH, nil)
	incoming.Set("b", 99) // overwrites
	incoming.Set("c", 3)  // new key

	base.Merge(incoming)
	fmt.Println(base)

	// Output:
	// {"a":1,"b":99,"c":3}
}

// ExampleModel_Merge_list demonstrates merging one LIST model into another.
// Incoming elements are appended to the base.
func ExampleModel_Merge_list() {
	a, _ := model.New(model.LIST, []any{1, 2})
	b, _ := model.New(model.LIST, []any{3, 4})
	a.Merge(b)
	fmt.Println(a)

	// Output:
	// [1,2,3,4]
}

// ExampleModel_Merge_nested demonstrates that when both models share a key
// whose value is itself a Model, Merge recurses into the sub-models rather
// than replacing one with the other. Non-overlapping keys on both sides are
// preserved.
func ExampleModel_Merge_nested() {
	inner1, _ := model.New(model.HASH, nil)
	inner1.Set("x", 1)
	base, _ := model.New(model.HASH, nil)
	base.Set("shared", inner1)
	base.Set("base_only", "hello")

	inner2, _ := model.New(model.HASH, nil)
	inner2.Set("y", 2)
	incoming, _ := model.New(model.HASH, nil)
	incoming.Set("shared", inner2)
	incoming.Set("new_key", "world")

	base.Merge(incoming)

	// "shared" now contains both x (from base) and y (from incoming).
	v, _ := base.Get("shared")
	shared, _ := v.Model()
	fmt.Println(shared)
	fmt.Println(base.Has("base_only"))
	fmt.Println(base.Has("new_key"))

	// Output:
	// {"x":1,"y":2}
	// true
	// true
}

// ── JSON ──────────────────────────────────────────────────────────────────────

// ExampleModel_MarshalJSON demonstrates serializing a model to JSON bytes.
func ExampleModel_MarshalJSON() {
	m, _ := model.New(model.LIST, []any{1, "two", true})
	b, _ := m.MarshalJSON()
	fmt.Println(string(b))

	// Output:
	// [1,"two",true]
}

// ExampleModel_UnmarshalJSON demonstrates deserializing JSON into a model.
// Calling UnmarshalJSON replaces all existing data. JSON numbers decode as
// float64, so use Float() to retrieve them precisely.
func ExampleModel_UnmarshalJSON() {
	m, _ := model.New(model.LIST, nil)
	m.UnmarshalJSON([]byte(`[10,20,30]`))
	v, _ := m.Get(1)
	n, _ := v.Float()
	fmt.Println(int(n))

	// Output:
	// 20
}

// ExampleModel_UnmarshalJSON_replaces demonstrates that UnmarshalJSON replaces
// all existing data rather than merging.
func ExampleModel_UnmarshalJSON_replaces() {
	m, _ := model.New(model.HASH, nil)
	m.Set("old", 1)
	m.UnmarshalJSON([]byte(`{"new":2}`))
	fmt.Println(m.Has("old"))
	fmt.Println(m.Has("new"))

	// Output:
	// false
	// true
}

// ExampleModel_MarshalModel demonstrates the MarshalModel method, which is an
// alias for MarshalJSON and satisfies the Marshaler interface.
func ExampleModel_MarshalModel() {
	m, _ := model.New(model.HASH, nil)
	m.Set("count", 3)
	b, _ := m.MarshalModel()
	fmt.Println(string(b))

	// Output:
	// {"count":3}
}

// ExampleModel_UnmarshalModel demonstrates UnmarshalModel, which skips
// deserialization when given the JSON null literal.
func ExampleModel_UnmarshalModel() {
	m, _ := model.New(model.HASH, nil)
	m.UnmarshalModel([]byte(`{"x":1,"y":2}`))
	fmt.Println(m.Has("x"), m.Has("y"))

	// Output:
	// true true
}

// ExampleModel_UnmarshalModel_null demonstrates that UnmarshalModel treats the
// JSON null literal as a no-op, leaving existing data intact.
func ExampleModel_UnmarshalModel_null() {
	m, _ := model.New(model.HASH, nil)
	m.Set("existing", 1)
	m.UnmarshalModel([]byte("null"))
	fmt.Println(m.Has("existing"))

	// Output:
	// true
}

// ExampleModel_String demonstrates the String method, which returns the JSON
// representation of the model and satisfies fmt.Stringer.
func ExampleModel_String() {
	m, _ := model.New(model.LIST, []any{1, 2, 3})
	fmt.Println(m)

	// Output:
	// [1,2,3]
}

// ── Value accessors ───────────────────────────────────────────────────────────

// ExampleValue_Bool demonstrates converting a Value to bool.
func ExampleValue_Bool() {
	m, _ := model.New(model.LIST, nil)
	m.Push(true)
	v, _ := m.Get(0)
	b, _ := v.Bool()
	fmt.Println(b)

	// Output:
	// true
}

// ExampleValue_Float demonstrates converting a Value to float64.
func ExampleValue_Float() {
	m, _ := model.New(model.LIST, nil)
	m.Push(2.5)
	v, _ := m.Get(0)
	f, _ := v.Float()
	fmt.Println(f)

	// Output:
	// 2.5
}

// ExampleValue_Float32 demonstrates converting a Value to float32.
func ExampleValue_Float32() {
	m, _ := model.New(model.LIST, nil)
	m.Push(float32(1.5))
	v, _ := m.Get(0)
	f, _ := v.Float32()
	fmt.Println(f)

	// Output:
	// 1.5
}

// ExampleValue_Float64 demonstrates converting a Value to float64 explicitly.
func ExampleValue_Float64() {
	m, _ := model.New(model.LIST, nil)
	m.Push(3.14)
	v, _ := m.Get(0)
	f, _ := v.Float64()
	fmt.Printf("%.2f\n", f)

	// Output:
	// 3.14
}

// ExampleValue_Int demonstrates converting a Value to int.
func ExampleValue_Int() {
	m, _ := model.New(model.LIST, nil)
	m.Push(42)
	v, _ := m.Get(0)
	n, _ := v.Int()
	fmt.Println(n)

	// Output:
	// 42
}

// ExampleValue_String demonstrates converting a Value to its string
// representation. Non-string types are cast via the cast package.
func ExampleValue_String() {
	m, _ := model.New(model.LIST, nil)
	m.Push(42)
	v, _ := m.Get(0)
	s, _ := v.String()
	fmt.Println(s)

	// Output:
	// 42
}

// ExampleValue_Model demonstrates extracting a nested Model stored as a Value.
func ExampleValue_Model() {
	inner, _ := model.New(model.HASH, nil)
	inner.Set("x", 1)

	outer, _ := model.New(model.LIST, nil)
	outer.Push(inner)

	v, _ := outer.Get(0)
	m, _ := v.Model()
	fmt.Println(m)

	// Output:
	// {"x":1}
}

// ExampleValue_Value demonstrates retrieving the raw untyped value stored
// inside a Value node.
func ExampleValue_Value() {
	m, _ := model.New(model.LIST, nil)
	m.Push("hello")
	v, _ := m.Get(0)
	fmt.Println(v.Value())

	// Output:
	// hello
}

// ExampleValue_List demonstrates the List accessor, which succeeds only when
// the underlying data is exactly a []stdModel.Value. The idiomatic way to
// store a nested list is to use a LIST model; List() is for the uncommon case
// where a raw []stdModel.Value slice was stored directly.
func ExampleValue_List() {
	// Build a []stdModel.Value from a model's snapshot.
	src, _ := model.New(model.LIST, []any{1, 2, 3})
	rawData, _, _ := src.GetData()
	vals := make([]stdModel.Value, len(rawData))
	for i, d := range rawData {
		vals[i] = d.(stdModel.Value)
	}

	// Store the slice and retrieve it via List().
	m, _ := model.New(model.LIST, nil)
	m.Push(vals)
	v, _ := m.Get(0)
	list, err := v.List()
	fmt.Println(err)
	fmt.Println(len(list))
	for i, item := range list {
		n, _ := item.Int()
		fmt.Printf("[%d]=%d\n", i, n)
	}

	// Output:
	// <nil>
	// 3
	// [0]=1
	// [1]=2
	// [2]=3
}

// ExampleValue_Map demonstrates the Map accessor, which succeeds only when the
// underlying data is exactly a map[string]stdModel.Value. The idiomatic way to
// store a nested map is to use a HASH model; Map() is for the uncommon case
// where a raw map was stored directly.
func ExampleValue_Map() {
	// Build a map[string]stdModel.Value from a model's snapshot.
	src, _ := model.New(model.HASH, nil)
	src.Set("a", 10)
	src.Set("b", 20)
	rawData, hashIdx, _ := src.GetData()
	vals := make(map[string]stdModel.Value, len(hashIdx))
	for key, idx := range hashIdx {
		vals[key] = rawData[idx].(stdModel.Value)
	}

	// Store the map and retrieve it via Map().
	m, _ := model.New(model.LIST, nil)
	m.Push(vals)
	v, _ := m.Get(0)
	mp, err := v.Map()
	fmt.Println(err)
	fmt.Println(len(mp))
	na, _ := mp["a"].Int()
	nb, _ := mp["b"].Int()
	fmt.Println(na)
	fmt.Println(nb)

	// Output:
	// <nil>
	// 2
	// 10
	// 20
}

// ── Conversion helpers ────────────────────────────────────────────────────────

// ExampleTo demonstrates the To generic helper for type conversions using the
// cast package. Returns the zero value on conversion failure.
func ExampleTo() {
	fmt.Println(model.To[int]("42"))
	fmt.Println(model.To[string](100))
	fmt.Println(model.To[bool](1))

	// Output:
	// 42
	// 100
	// true
}

// ExampleToE demonstrates the ToE generic helper, which returns the converted
// value and an error instead of a zero value on failure.
func ExampleToE() {
	n, err := model.ToE[int]("42")
	fmt.Println(n, err)
	_, err = model.ToE[int]("not-a-number")
	fmt.Println(err != nil)

	// Output:
	// 42 <nil>
	// true
}

// ── Error sentinels ───────────────────────────────────────────────────────────

// ExampleInvalidIndex demonstrates the sentinel returned when Get, Delete, or
// Seek is called with a key or index that does not exist in the model.
func ExampleInvalidIndex() {
	list, _ := model.New(model.LIST, []any{1, 2, 3})
	_, err := list.Get(99)
	fmt.Println(errors.Is(err, model.InvalidIndex))

	hash, _ := model.New(model.HASH, nil)
	_, err = hash.Get("missing")
	fmt.Println(errors.Is(err, model.InvalidIndex))

	// Output:
	// true
	// true
}

// ExampleInvalidIndexType demonstrates the sentinel returned when a LIST model
// is accessed with a non-integer key type.
func ExampleInvalidIndexType() {
	m, _ := model.New(model.LIST, []any{1, 2, 3})

	_, err := m.Get("not-an-int")
	fmt.Println(errors.Is(err, model.InvalidIndexType))

	err = m.Delete("also-not-an-int")
	fmt.Println(errors.Is(err, model.InvalidIndexType))

	// Output:
	// true
	// true
}

// ExampleInvalidMethodContext demonstrates the sentinel returned when a method
// is invoked in an unsupported context, such as calling Push on a HASH model.
func ExampleInvalidMethodContext() {
	h, _ := model.New(model.HASH, nil)
	err := h.Push("value")
	fmt.Println(errors.Is(err, model.InvalidMethodContext))

	// Merging a model into itself also returns this sentinel.
	m, _ := model.New(model.HASH, nil)
	err = m.Merge(m)
	fmt.Println(errors.Is(err, model.InvalidMethodContext))

	// Output:
	// true
	// true
}

// ExampleReadOnlyModel demonstrates the sentinel returned when a mutation is
// attempted on a locked model, or when SetType is called on a non-empty model.
func ExampleReadOnlyModel() {
	m, _ := model.New(model.HASH, nil)
	m.Set("k", 1)
	m.Lock()

	err := m.Set("k", 2)
	fmt.Println(errors.Is(err, model.ReadOnlyModel))

	// SetType also returns ReadOnlyModel when the model already holds data.
	m2, _ := model.New(model.HASH, nil)
	m2.Set("k", 1)
	err = m2.SetType(model.LIST)
	fmt.Println(errors.Is(err, model.ReadOnlyModel))

	// Output:
	// true
	// true
}

// ExampleInvalidDataSet demonstrates the sentinel returned when SetData is
// called with a value whose type does not match the model type.
func ExampleInvalidDataSet() {
	list, _ := model.New(model.LIST, nil)
	err := list.SetData(map[string]any{"key": "val"}) // map not valid for LIST
	fmt.Println(errors.Is(err, model.InvalidDataSet))

	hash, _ := model.New(model.HASH, nil)
	err = hash.SetData([]any{1, 2, 3}) // slice not valid for HASH
	fmt.Println(errors.Is(err, model.InvalidDataSet))

	// Output:
	// true
	// true
}

// ExampleInvalidSortFlagCombination demonstrates the sentinel returned when
// mutually exclusive sort flags are combined.
func ExampleInvalidSortFlagCombination() {
	m, _ := model.New(model.LIST, []any{1, 2, 3})
	err := m.Sort(sorter.SortAsc | sorter.SortDesc)
	fmt.Println(errors.Is(err, model.InvalidSortFlagCombination))

	// Output:
	// true
}

// ── Comprehensive walkthrough ─────────────────────────────────────────────────

// ExampleModel_complete demonstrates the full model lifecycle: building from
// JSON, iterating, sorting, filtering, reducing, and marshaling back to JSON.
func ExampleModel_complete() {
	// Build a hash model from JSON
	mdl, _ := model.New(model.HASH, nil)
	json.Unmarshal([]byte(`{
        "users": [
            {"name": "Charlie", "age": 25},
            {"name": "Alice",   "age": 30},
            {"name": "Bob",     "age": 22}
        ]
    }`), &mdl)
	fmt.Println("Model:", mdl)

	// Get the nested list
	val, _ := mdl.Get("users")
	users, _ := val.Model()

	// SortByValue (==0) is a no-op: model values are compared by ID then by
	// element count, and all users have the same (empty) ID and the same number
	// of fields, so the sort leaves the original insertion order intact.
	users.(sorter.Sorter).Sort(sorter.SortByValue)
	fmt.Println("Users:", users)

	// Iterate and print
	var key, v any
	for users.(iterator.Iterator).Next(&key, &v) {
		user, _ := v.(stdModel.Value).Model()
		nameVal, _ := user.Get("name")
		name, _ := nameVal.String()
		ageVal, _ := user.Get("age")
		age, _ := ageVal.Float() // JSON numbers decode as float64
		fmt.Printf("%s: %d\n", name, int(age))
	}

	// Filter to adults only
	adults := users.Filter(func(v stdModel.Value) bool {
		user, err := v.Model()
		if err != nil {
			return false
		}
		ageVal, _ := user.Get("age")
		age, _ := ageVal.Float()
		return age >= 25
	})
	fmt.Println("Adults:", adults)

	// Reduce to find the oldest adult.
	oldestVal := adults.Reduce(func(carry, cur stdModel.Value) stdModel.Value {
		if nil == carry {
			return cur
		}
		oldest, err := carry.Model()
		if err != nil {
			return cur
		}
		curUser, err := cur.Model()
		if err != nil {
			return carry
		}
		oldVal, _ := oldest.Get("age")
		oldestAge, _ := oldVal.Int()
		curVal, _ := curUser.Get("age")
		curAge, _ := curVal.Int()
		if curAge > oldestAge {
			return cur
		}
		return carry
	})
	oldest, _ := oldestVal.Model()
	fmt.Println("Oldest:", oldest)

	// Marshal back to JSON
	b, _ := json.Marshal(mdl)
	fmt.Println(string(b))

	// Output: Model: {"users":[{"age":25,"name":"Charlie"},{"age":30,"name":"Alice"},{"age":22,"name":"Bob"}]}
	// Users: [{"age":25,"name":"Charlie"},{"age":30,"name":"Alice"},{"age":22,"name":"Bob"}]
	// Charlie: 25
	// Alice: 30
	// Bob: 22
	// Adults: [{"age":25,"name":"Charlie"},{"age":30,"name":"Alice"}]
	// Oldest: {"age":30,"name":"Alice"}
	// {"users":[{"age":25,"name":"Charlie"},{"age":30,"name":"Alice"},{"age":22,"name":"Bob"}]}
}
