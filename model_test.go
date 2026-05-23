package model_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/bdlm/model"
	stdModel "github.com/bdlm/std/v2/model"
	stdSorter "github.com/bdlm/std/v2/sorter"
)

// helpers

func mustNew(t *testing.T, typ stdModel.ModelType, data any) *model.Model {
	t.Helper()
	m, err := model.New(typ, data)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return m
}

// mLen returns the number of elements in any stdModel.Model via GetData.
func mLen(m stdModel.Model) int {
	d, _, _ := m.GetData()
	return len(d)
}

func intVal(t *testing.T, v stdModel.Value) int {
	t.Helper()
	n, err := v.Int()
	if err != nil {
		t.Fatalf("Int(): %v", err)
	}
	return n
}

func strVal(t *testing.T, v stdModel.Value) string {
	t.Helper()
	s, err := v.String()
	if err != nil {
		t.Fatalf("String(): %v", err)
	}
	return s
}

// ── New ──────────────────────────────────────────────────────────────────────

func TestNewHash(t *testing.T) {
	m := mustNew(t, model.HASH, nil)
	if m.Len() != 0 {
		t.Errorf("expected empty model, got len=%d", m.Len())
	}
}

func TestNewList(t *testing.T) {
	m := mustNew(t, model.LIST, nil)
	if m.Len() != 0 {
		t.Errorf("expected empty model, got len=%d", m.Len())
	}
}

func TestNewWithHashData(t *testing.T) {
	m := mustNew(t, model.HASH, map[string]any{"a": 1, "b": 2})
	if m.Len() != 2 {
		t.Errorf("expected len=2, got %d", m.Len())
	}
	v, err := m.Get("a")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if intVal(t, v) != 1 {
		t.Errorf("expected 1, got %d", intVal(t, v))
	}
}

func TestNewWithListData(t *testing.T) {
	m := mustNew(t, model.LIST, []any{10, 20, 30})
	if m.Len() != 3 {
		t.Errorf("expected len=3, got %d", m.Len())
	}
	v, err := m.Get(1)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if intVal(t, v) != 20 {
		t.Errorf("expected 20, got %d", intVal(t, v))
	}
}

func TestNewUnsupportedDataType(t *testing.T) {
	_, err := model.New(model.HASH, struct{ X int }{X: 1})
	if err == nil {
		t.Fatal("expected error for unsupported data type, got nil")
	}
}

// ── Set / Get ─────────────────────────────────────────────────────────────────

func TestHashSetGet(t *testing.T) {
	m := mustNew(t, model.HASH, nil)
	if err := m.Set("key", "value"); err != nil {
		t.Fatalf("Set: %v", err)
	}
	v, err := m.Get("key")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if strVal(t, v) != "value" {
		t.Errorf("expected 'value', got %q", strVal(t, v))
	}
}

func TestHashSetUpdatesExisting(t *testing.T) {
	m := mustNew(t, model.HASH, nil)
	m.Set("k", 1)
	m.Set("k", 2)
	v, _ := m.Get("k")
	if intVal(t, v) != 2 {
		t.Errorf("expected 2, got %d", intVal(t, v))
	}
}

func TestHashKeyCoercion(t *testing.T) {
	m := mustNew(t, model.HASH, nil)
	m.Set(42, "forty-two")          // integer key coerced to "42"
	m.Set(3.14, "pi")               // float key coerced to string
	m.Set(fmt.Errorf("e"), "error") // error key coerced via cast
	v, err := m.Get("42")
	if err != nil {
		t.Fatalf("Get '42': %v", err)
	}
	if strVal(t, v) != "forty-two" {
		t.Errorf("expected 'forty-two', got %q", strVal(t, v))
	}
}

func TestListSetGet(t *testing.T) {
	m := mustNew(t, model.LIST, nil)
	m.Push("a")
	m.Push("b")
	v, err := m.Get(0)
	if err != nil {
		t.Fatalf("Get(0): %v", err)
	}
	if strVal(t, v) != "a" {
		t.Errorf("expected 'a', got %q", strVal(t, v))
	}
	if err := m.Set(0, "A"); err != nil {
		t.Fatalf("Set(0): %v", err)
	}
	v, _ = m.Get(0)
	if strVal(t, v) != "A" {
		t.Errorf("expected 'A', got %q", strVal(t, v))
	}
}

func TestGetInvalidIndex(t *testing.T) {
	m := mustNew(t, model.LIST, []any{1, 2})
	_, err := m.Get(5)
	if err == nil {
		t.Fatal("expected error for out-of-bounds index")
	}
	if !errors.Is(err, model.InvalidIndex) {
		t.Errorf("expected InvalidIndex, got %v", err)
	}
}

func TestGetInvalidKeyType(t *testing.T) {
	m := mustNew(t, model.LIST, []any{1})
	_, err := m.Get("not-an-int")
	if err == nil {
		t.Fatal("expected error for string key on list model")
	}
	if !errors.Is(err, model.InvalidIndexType) {
		t.Errorf("expected InvalidIndexType, got %v", err)
	}
}

func TestGetNegativeIndex(t *testing.T) {
	m := mustNew(t, model.LIST, []any{1, 2})
	_, err := m.Get(-1)
	if err == nil {
		t.Fatal("expected error for negative index")
	}
}

func TestGetIntegerWidths(t *testing.T) {
	m := mustNew(t, model.LIST, []any{"x"})
	for _, key := range []any{int8(0), int16(0), int32(0), int64(0), uint(0), uint8(0), uint16(0), uint32(0), uint64(0)} {
		v, err := m.Get(key)
		if err != nil {
			t.Errorf("Get(%T(0)): %v", key, err)
			continue
		}
		if strVal(t, v) != "x" {
			t.Errorf("Get(%T(0)): expected 'x', got %q", key, strVal(t, v))
		}
	}
}

// ── Has ───────────────────────────────────────────────────────────────────────

func TestHashHas(t *testing.T) {
	m := mustNew(t, model.HASH, nil)
	m.Set("exists", 1)
	if !m.Has("exists") {
		t.Error("Has('exists') should be true")
	}
	if m.Has("missing") {
		t.Error("Has('missing') should be false")
	}
}

func TestListHas(t *testing.T) {
	m := mustNew(t, model.LIST, []any{1, 2, 3})
	if !m.Has(0) {
		t.Error("Has(0) should be true")
	}
	if m.Has(5) {
		t.Error("Has(5) should be false")
	}
	if m.Has(-1) {
		t.Error("Has(-1) should be false")
	}
}

// ── Push / Delete ─────────────────────────────────────────────────────────────

func TestPushOnHash(t *testing.T) {
	m := mustNew(t, model.HASH, nil)
	err := m.Push("val")
	if err == nil {
		t.Fatal("expected error pushing to hash model")
	}
	if !errors.Is(err, model.InvalidMethodContext) {
		t.Errorf("expected InvalidMethodContext, got %v", err)
	}
}

func TestDeleteHash(t *testing.T) {
	m := mustNew(t, model.HASH, nil)
	m.Set("a", 1)
	m.Set("b", 2)
	m.Set("c", 3)
	if err := m.Delete("b"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if m.Has("b") {
		t.Error("'b' should be deleted")
	}
	if m.Len() != 2 {
		t.Errorf("expected len=2, got %d", m.Len())
	}
	// remaining keys should still be accessible
	v, err := m.Get("a")
	if err != nil || intVal(t, v) != 1 {
		t.Errorf("Get('a') after delete: %v %v", v, err)
	}
	v, err = m.Get("c")
	if err != nil || intVal(t, v) != 3 {
		t.Errorf("Get('c') after delete: %v %v", v, err)
	}
}

func TestDeleteList(t *testing.T) {
	m := mustNew(t, model.LIST, []any{10, 20, 30})
	if err := m.Delete(1); err != nil {
		t.Fatalf("Delete(1): %v", err)
	}
	if m.Len() != 2 {
		t.Errorf("expected len=2, got %d", m.Len())
	}
	v, _ := m.Get(0)
	if intVal(t, v) != 10 {
		t.Errorf("expected 10 at index 0, got %d", intVal(t, v))
	}
	v, _ = m.Get(1)
	if intVal(t, v) != 30 {
		t.Errorf("expected 30 at index 1, got %d", intVal(t, v))
	}
}

func TestDeleteInvalidIndex(t *testing.T) {
	m := mustNew(t, model.LIST, []any{1})
	if err := m.Delete(5); err == nil {
		t.Error("expected error deleting out-of-bounds index")
	}
}

// TestDeleteListInvalidKeyType verifies that Delete on a LIST model rejects
// non-integer keys with InvalidIndexType, consistent with Get behavior.
// Before the fix, cast.To[int]("foo")==0 caused Delete("foo") to silently
// delete index 0.
func TestDeleteListInvalidKeyType(t *testing.T) {
	m := mustNew(t, model.LIST, []any{10, 20, 30})
	for _, bad := range []any{"foo", "1", 1.5, true, struct{}{}} {
		err := m.Delete(bad)
		if err == nil {
			t.Errorf("Delete(%v): expected InvalidIndexType error, got nil", bad)
			continue
		}
		if !errors.Is(err, model.InvalidIndexType) {
			t.Errorf("Delete(%v): expected InvalidIndexType, got %v", bad, err)
		}
	}
	// Model must be unchanged.
	if m.Len() != 3 {
		t.Errorf("model was mutated by an invalid Delete: expected len=3, got %d", m.Len())
	}
}

// TestDeleteListGCFriendly verifies that deleting an element does not retain
// a reference to it via the backing array, and that the remaining elements
// are correct after deletion.
func TestDeleteListGCFriendly(t *testing.T) {
	m := mustNew(t, model.LIST, []any{1, 2, 3, 4, 5})
	if err := m.Delete(2); err != nil {
		t.Fatalf("Delete(2): %v", err)
	}
	if m.Len() != 4 {
		t.Fatalf("expected len=4 after Delete, got %d", m.Len())
	}
	expected := []int{1, 2, 4, 5}
	for i, want := range expected {
		v, err := m.Get(i)
		if err != nil {
			t.Fatalf("Get(%d): %v", i, err)
		}
		if intVal(t, v) != want {
			t.Errorf("index %d: expected %d, got %d", i, want, intVal(t, v))
		}
	}
}

// ── Iteration ────────────────────────────────────────────────────────────────

func TestHashIterator(t *testing.T) {
	m := mustNew(t, model.HASH, nil)
	m.Set("b", 2)
	m.Set("a", 1)
	m.Set("c", 3)

	var keys []string
	var key, val any
	for m.Next(&key, &val) {
		keys = append(keys, key.(string))
	}
	// Hash is sorted by key after import; Set appends in insertion order
	// but importMap sorts. Direct Set calls don't sort.
	if len(keys) != 3 {
		t.Errorf("expected 3 keys, got %d: %v", len(keys), keys)
	}
}

func TestListIterator(t *testing.T) {
	m := mustNew(t, model.LIST, []any{1, 2, 3})
	var sum int
	var key, val any
	for m.Next(&key, &val) {
		sum += intVal(t, val.(stdModel.Value))
	}
	if sum != 6 {
		t.Errorf("expected sum=6, got %d", sum)
	}
}

func TestPrevIterator(t *testing.T) {
	m := mustNew(t, model.LIST, []any{1, 2, 3})
	// exhaust forward
	var key, val any
	for m.Next(&key, &val) {
	}
	// seek to end and go backward
	m.Seek(2)
	var vals []int
	for m.Prev(&key, &val) {
		vals = append(vals, intVal(t, val.(stdModel.Value)))
	}
	if len(vals) != 2 || vals[0] != 2 || vals[1] != 1 {
		t.Errorf("expected [2 1], got %v", vals)
	}
}

func TestCurAfterSeek(t *testing.T) {
	m := mustNew(t, model.LIST, []any{10, 20, 30})
	m.Seek(1)
	var key, val any
	if !m.Cur(&key, &val) {
		t.Fatal("Cur returned false after Seek")
	}
	if intVal(t, val.(stdModel.Value)) != 20 {
		t.Errorf("expected 20, got %d", intVal(t, val.(stdModel.Value)))
	}
}

func TestSeekHashKey(t *testing.T) {
	m, _ := model.New(model.HASH, map[string]any{"x": 10, "y": 20})
	if err := m.Seek("x"); err != nil {
		t.Fatalf("Seek('x'): %v", err)
	}
	var key, val any
	m.Cur(&key, &val)
	if intVal(t, val.(stdModel.Value)) != 10 {
		t.Errorf("expected 10, got %d", intVal(t, val.(stdModel.Value)))
	}
}

func TestReset(t *testing.T) {
	m := mustNew(t, model.LIST, []any{1, 2, 3})
	var key, val any
	m.Next(&key, &val)
	m.Next(&key, &val)
	m.Reset()
	m.Next(&key, &val)
	if intVal(t, val.(stdModel.Value)) != 1 {
		t.Errorf("after Reset, expected first element=1, got %d", intVal(t, val.(stdModel.Value)))
	}
}

func TestSeekInvalidType(t *testing.T) {
	m := mustNew(t, model.LIST, []any{1})
	if err := m.Seek("not-an-int"); err == nil {
		t.Error("expected error seeking list with string key")
	}
}

func TestSeekOutOfBounds(t *testing.T) {
	m := mustNew(t, model.LIST, []any{1, 2})
	if err := m.Seek(5); err == nil {
		t.Error("expected error seeking beyond end")
	}
}

// ── Lock ─────────────────────────────────────────────────────────────────────

func TestLockPreventsWrites(t *testing.T) {
	m := mustNew(t, model.HASH, nil)
	m.Set("k", 1)
	m.Lock()

	ops := []struct {
		name string
		fn   func() error
	}{
		{"Set", func() error { return m.Set("k", 2) }},
		{"Delete", func() error { return m.Delete("k") }},
		{"SetID", func() error { return m.SetID("id") }},
		{"SetType", func() error { return m.SetType(model.LIST) }},
	}
	for _, op := range ops {
		err := op.fn()
		if err == nil {
			t.Errorf("%s: expected ReadOnlyModel error, got nil", op.name)
			continue
		}
		if !errors.Is(err, model.ReadOnlyModel) {
			t.Errorf("%s: expected ReadOnlyModel, got %v", op.name, err)
		}
	}
}

func TestLockPreventsListWrites(t *testing.T) {
	m := mustNew(t, model.LIST, []any{1, 2})
	m.Lock()
	if err := m.Push(3); err == nil {
		t.Error("expected ReadOnlyModel from Push on locked model")
	}
	if err := m.Delete(0); err == nil {
		t.Error("expected ReadOnlyModel from Delete on locked model")
	}
}

func TestLockAllowsReads(t *testing.T) {
	m := mustNew(t, model.HASH, nil)
	m.Set("k", 42)
	m.Lock()
	v, err := m.Get("k")
	if err != nil {
		t.Fatalf("Get on locked model: %v", err)
	}
	if intVal(t, v) != 42 {
		t.Errorf("expected 42, got %d", intVal(t, v))
	}
}

// ── ModelType / SetType ───────────────────────────────────────────────────────

func TestModelType(t *testing.T) {
	mdl, _ := model.New(model.HASH, nil)

	err := mdl.Push("val1")
	if err == nil {
		t.Fatal("expected error from Push on hash model")
	}
	if !errors.Is(err, model.InvalidMethodContext) {
		t.Errorf("expected InvalidMethodContext, got %v", err)
	}

	// Hash keys are always cast to string. Verify that string and integer keys
	// round-trip correctly (the cast result is well-defined for these types).
	mdl.Set("str", "str")
	mdl.Set(10, "10")

	var key, val any
	for mdl.Next(&key, &val) {
		v, err := val.(stdModel.Value).String()
		if err != nil {
			t.Errorf("String(): %v", err)
		}
		if key != v {
			t.Errorf("expected key==value, got key=%v value=%v", key, v)
		}
	}
}

func TestSetTypeFailsWhenNonEmpty(t *testing.T) {
	m := mustNew(t, model.HASH, nil)
	m.Set("k", 1)
	if err := m.SetType(model.LIST); err == nil {
		t.Error("expected error setting type on non-empty model")
	}
}

func TestSetTypeSucceedsWhenEmpty(t *testing.T) {
	m := mustNew(t, model.HASH, nil)
	if err := m.SetType(model.LIST); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

// ── SetData ────────────────────────────────────────────────────────────────────

func TestSetDataHash(t *testing.T) {
	m := mustNew(t, model.HASH, nil)
	m.Set("old", 99)
	if err := m.SetData(map[string]any{"new": 1}); err != nil {
		t.Fatalf("SetData: %v", err)
	}
	if m.Has("old") {
		t.Error("'old' key should be gone after SetData")
	}
	if !m.Has("new") {
		t.Error("'new' key should exist after SetData")
	}
}

// TestSetDataHashSortedOrder verifies that SetData on a HASH model inserts keys
// in ascending alphabetical order, matching the behaviour of importMap and
// UnmarshalJSON.
func TestSetDataHashSortedOrder(t *testing.T) {
	m := mustNew(t, model.HASH, nil)
	if err := m.SetData(map[string]any{"c": 3, "a": 1, "b": 2}); err != nil {
		t.Fatalf("SetData: %v", err)
	}
	want := []string{"a", "b", "c"}
	for i, wantKey := range want {
		v, err := m.Get(wantKey)
		if err != nil {
			t.Fatalf("Get(%q): %v", wantKey, err)
		}
		n, _ := v.Int()
		if n != i+1 {
			t.Errorf("key %q: want value %d, got %d", wantKey, i+1, n)
		}
	}
	// Iterate to confirm key order.
	var gotKeys []string
	var k, v any
	for m.Next(&k, &v) {
		gotKeys = append(gotKeys, k.(string))
	}
	for i, k := range gotKeys {
		if k != want[i] {
			t.Errorf("iteration position %d: want key %q, got %q", i, want[i], k)
		}
	}
}

func TestSetDataList(t *testing.T) {
	m := mustNew(t, model.LIST, []any{1, 2, 3})
	if err := m.SetData([]any{10, 20}); err != nil {
		t.Fatalf("SetData: %v", err)
	}
	if m.Len() != 2 {
		t.Errorf("expected len=2, got %d", m.Len())
	}
}

func TestSetDataWrongType(t *testing.T) {
	m := mustNew(t, model.HASH, nil)
	if err := m.SetData([]any{1, 2}); err == nil {
		t.Error("expected error setting []any on hash model")
	}
}

// ── SetID / GetID ─────────────────────────────────────────────────────────────

func TestSetGetID(t *testing.T) {
	m := mustNew(t, model.HASH, nil)
	if err := m.SetID("my-id"); err != nil {
		t.Fatalf("SetID: %v", err)
	}
	if m.GetID() != "my-id" {
		t.Errorf("expected 'my-id', got %v", m.GetID())
	}
}

// ── Reverse ────────────────────────────────────────────────────────────────────

func TestReverseList(t *testing.T) {
	m := mustNew(t, model.LIST, []any{1, 2, 3})
	if err := m.Reverse(); err != nil {
		t.Fatalf("Reverse: %v", err)
	}
	v0, _ := m.Get(0)
	v2, _ := m.Get(2)
	if intVal(t, v0) != 3 || intVal(t, v2) != 1 {
		t.Errorf("expected [3,2,1], got [%d,_,%d]", intVal(t, v0), intVal(t, v2))
	}
}

func TestReverseHash(t *testing.T) {
	m, _ := model.New(model.HASH, map[string]any{"a": 1, "b": 2, "c": 3})
	origData, origHashIdx, _ := m.GetData()
	m.Reverse()
	revData, revHashIdx, _ := m.GetData()
	n := len(origData)
	for k, idx := range origHashIdx {
		newIdx := n - 1 - idx
		if revHashIdx[k] != newIdx {
			t.Errorf("key %q: expected new index %d, got %d", k, newIdx, revHashIdx[k])
		}
	}
	_ = revData
}

// ── Sort ────────────────────────────────────────────────────────────────────

func TestSortByKeyHash(t *testing.T) {
	m := mustNew(t, model.HASH, nil)
	m.Set("c", 3)
	m.Set("a", 1)
	m.Set("b", 2)
	m.Sort(stdSorter.SortByKey)

	var keys []string
	var key, val any
	for m.Next(&key, &val) {
		keys = append(keys, key.(string))
	}
	if keys[0] != "a" || keys[1] != "b" || keys[2] != "c" {
		t.Errorf("expected [a b c], got %v", keys)
	}
}

func TestSortByKeyDesc(t *testing.T) {
	m := mustNew(t, model.HASH, nil)
	m.Set("a", 1)
	m.Set("c", 3)
	m.Set("b", 2)
	m.Sort(stdSorter.SortByKey | stdSorter.SortDesc)

	var keys []string
	var key, val any
	for m.Next(&key, &val) {
		keys = append(keys, key.(string))
	}
	if keys[0] != "c" || keys[1] != "b" || keys[2] != "a" {
		t.Errorf("expected [c b a], got %v", keys)
	}
}

func TestSortByValueList(t *testing.T) {
	m := mustNew(t, model.LIST, []any{3, 1, 2})
	m.Sort(stdSorter.SortAsc)
	v0, _ := m.Get(0)
	v1, _ := m.Get(1)
	v2, _ := m.Get(2)
	if intVal(t, v0) != 1 || intVal(t, v1) != 2 || intVal(t, v2) != 3 {
		t.Errorf("expected [1 2 3], got [%d %d %d]", intVal(t, v0), intVal(t, v1), intVal(t, v2))
	}
}

func TestSortByValueDesc(t *testing.T) {
	m := mustNew(t, model.LIST, []any{1, 3, 2})
	m.Sort(stdSorter.SortByValue | stdSorter.SortDesc)
	v0, _ := m.Get(0)
	if intVal(t, v0) != 3 {
		t.Errorf("expected 3 at index 0, got %d", intVal(t, v0))
	}
}

func TestSortWithReverse(t *testing.T) {
	m := mustNew(t, model.LIST, []any{3, 1, 2})
	// ascending then reversed = descending
	m.Sort(stdSorter.SortAsc | stdSorter.SortReverse)
	v0, _ := m.Get(0)
	if intVal(t, v0) != 3 {
		t.Errorf("expected 3 at index 0, got %d", intVal(t, v0))
	}
}

func TestSortDescAndReverse(t *testing.T) {
	m := mustNew(t, model.LIST, []any{3, 1, 2})
	// descending then reversed = ascending
	m.Sort(stdSorter.SortByValue | stdSorter.SortDesc | stdSorter.SortReverse)
	v0, _ := m.Get(0)
	if intVal(t, v0) != 1 {
		t.Errorf("expected 1 at index 0, got %d", intVal(t, v0))
	}
}

// TestSortByKeyWithDefaultValueIsValid verifies that SortByKey combined with
// SortByValue (the zero-value default) is treated as SortByKey alone and does
// not return an error. In the updated sorter API, SortByValue==0 contributes
// no bits, so the combination is always equivalent to SortByKey.
func TestSortByKeyWithDefaultValueIsValid(t *testing.T) {
	m := mustNew(t, model.HASH, nil)
	m.Set("b", 2)
	m.Set("a", 1)
	err := m.Sort(stdSorter.SortByKey | stdSorter.SortByValue)
	if err != nil {
		t.Fatalf("Sort(SortByKey|SortByValue) should be valid, got %v", err)
	}
	var keys []string
	var key, val any
	for m.Next(&key, &val) {
		keys = append(keys, key.(string))
	}
	if len(keys) != 2 || keys[0] != "a" || keys[1] != "b" {
		t.Errorf("expected [a b] after SortByKey, got %v", keys)
	}
}

func TestSortAscAndDescInvalid(t *testing.T) {
	m := mustNew(t, model.LIST, []any{1, 2})
	err := m.Sort(stdSorter.SortAsc | stdSorter.SortDesc)
	if err == nil {
		t.Fatal("expected InvalidSortFlagCombination")
	}
}

func TestSortByKeyAsStringList(t *testing.T) {
	m := mustNew(t, model.LIST, []any{30, 10, 200})
	m.Sort(stdSorter.SortByKey | stdSorter.SortAsString)
	v0, _ := m.Get(0)
	v1, _ := m.Get(1)
	v2, _ := m.Get(2)
	// lexicographic: "10" < "200" < "30"
	if intVal(t, v0) != 10 || intVal(t, v1) != 200 || intVal(t, v2) != 30 {
		t.Errorf("expected [10 200 30] (lexicographic), got [%d %d %d]", intVal(t, v0), intVal(t, v1), intVal(t, v2))
	}
}

func TestSortLockedModel(t *testing.T) {
	m := mustNew(t, model.LIST, []any{1, 2})
	m.Lock()
	if err := m.Sort(stdSorter.SortByValue); err == nil {
		t.Error("expected error sorting locked model")
	}
}

// ── Merge ─────────────────────────────────────────────────────────────────────

func TestMergeHashIntoHash(t *testing.T) {
	base, _ := model.New(model.HASH, nil)
	base.Set("a", 1)
	base.Set("b", 2)

	incoming, _ := model.New(model.HASH, nil)
	incoming.Set("b", 99) // overwrites
	incoming.Set("c", 3)  // new key

	if err := base.Merge(incoming); err != nil {
		t.Fatalf("Merge: %v", err)
	}
	v, _ := base.Get("a")
	if intVal(t, v) != 1 {
		t.Errorf("'a': expected 1, got %d", intVal(t, v))
	}
	v, _ = base.Get("b")
	if intVal(t, v) != 99 {
		t.Errorf("'b': expected 99, got %d", intVal(t, v))
	}
	v, _ = base.Get("c")
	if intVal(t, v) != 3 {
		t.Errorf("'c': expected 3, got %d", intVal(t, v))
	}
}

func TestMergeListIntoList(t *testing.T) {
	base := mustNew(t, model.LIST, []any{1, 2})
	inc := mustNew(t, model.LIST, []any{3, 4})
	if err := base.Merge(inc); err != nil {
		t.Fatalf("Merge: %v", err)
	}
	if base.Len() != 4 {
		t.Errorf("expected len=4, got %d", base.Len())
	}
}

func TestMergeListIntoHash(t *testing.T) {
	h := mustNew(t, model.HASH, nil)
	h.Set("existing", 0)
	l := mustNew(t, model.LIST, []any{"x", "y"})
	if err := h.Merge(l); err != nil {
		t.Fatalf("Merge: %v", err)
	}
	// list indices cast to string keys
	if !h.Has("0") {
		t.Error("expected key '0'")
	}
	if !h.Has("1") {
		t.Error("expected key '1'")
	}
}

func TestMergeHashIntoList(t *testing.T) {
	l := mustNew(t, model.LIST, []any{1})
	h := mustNew(t, model.HASH, nil)
	h.Set("a", 10)
	h.Set("b", 20)
	if err := l.Merge(h); err != nil {
		t.Fatalf("Merge: %v", err)
	}
	if l.Len() != 3 {
		t.Errorf("expected len=3, got %d", l.Len())
	}
}

func TestMergeSelf(t *testing.T) {
	m := mustNew(t, model.HASH, nil)
	m.Set("k", 1)
	err := m.Merge(m)
	if err == nil {
		t.Fatal("expected error for self-merge")
	}
	if !errors.Is(err, model.InvalidMethodContext) {
		t.Errorf("expected InvalidMethodContext, got %v", err)
	}
}

func TestMergeNestedRecursive(t *testing.T) {
	inner1, _ := model.New(model.HASH, nil)
	inner1.Set("x", 1)
	base, _ := model.New(model.HASH, nil)
	base.Set("nested", inner1)

	inner2, _ := model.New(model.HASH, nil)
	inner2.Set("y", 2)
	incoming, _ := model.New(model.HASH, nil)
	incoming.Set("nested", inner2)

	if err := base.Merge(incoming); err != nil {
		t.Fatalf("Merge: %v", err)
	}
	v, _ := base.Get("nested")
	nestedModel, err := v.Model()
	if err != nil {
		t.Fatalf("Model(): %v", err)
	}
	// both x and y should exist in the merged nested model
	if !nestedModel.Has("x") {
		t.Error("expected 'x' in merged nested model")
	}
	if !nestedModel.Has("y") {
		t.Error("expected 'y' in merged nested model")
	}
}

// ── Filter ────────────────────────────────────────────────────────────────────

func TestFilterList(t *testing.T) {
	m := mustNew(t, model.LIST, []any{1, 2, 3, 4, 5, 6})
	result := m.Filter(func(v stdModel.Value) bool {
		n, _ := v.Int()
		return n%2 == 0
	})
	if mLen(result) != 3 {
		t.Errorf("expected len=3, got %d", mLen(result))
	}
}

func TestFilterHash(t *testing.T) {
	m, _ := model.New(model.HASH, map[string]any{"a": 1, "b": 2, "c": 3, "d": 4})
	result := m.Filter(func(v stdModel.Value) bool {
		n, _ := v.Int()
		return n > 2
	})
	if mLen(result) != 2 {
		t.Errorf("expected len=2, got %d", mLen(result))
	}
	if result.GetType() != model.HASH {
		t.Error("Filter should return same model type")
	}
}

func TestFilterPreservesOriginal(t *testing.T) {
	m := mustNew(t, model.LIST, []any{1, 2, 3, 4})
	m.Filter(func(v stdModel.Value) bool { return true })
	if m.Len() != 4 {
		t.Errorf("Filter should not modify original; expected len=4, got %d", m.Len())
	}
}

func TestFilterCallbackCanReadModel(t *testing.T) {
	m := mustNew(t, model.LIST, []any{1, 2, 3})
	// Callback calls Get on the same model — would deadlock before the snapshot fix.
	result := m.Filter(func(v stdModel.Value) bool {
		n, _ := v.Int()
		// read from the model being filtered — safe with snapshot approach
		first, err := m.Get(0)
		if err != nil {
			return false
		}
		f, _ := first.Int()
		return n > f // include elements greater than the first
	})
	// first element is 1, so elements 2 and 3 pass
	if mLen(result) != 2 {
		t.Errorf("expected len=2, got %d", mLen(result))
	}
}

func TestFilterCallbackCanWriteModel(t *testing.T) {
	m := mustNew(t, model.LIST, []any{1, 2, 3})
	seen := 0
	m.Filter(func(v stdModel.Value) bool {
		seen++
		// write to the model being filtered — safe with snapshot approach
		m.Push(seen * 10)
		return false
	})
	// snapshot had 3 elements, so 3 pushes happened
	if m.Len() != 6 {
		t.Errorf("expected model len=6 after filter writes, got %d", m.Len())
	}
}

// ── Map ───────────────────────────────────────────────────────────────────────

func TestMapList(t *testing.T) {
	m := mustNew(t, model.LIST, []any{1, 2, 3})
	result := m.Map(func(v stdModel.Value) stdModel.Value {
		n, _ := v.Int()
		doubled, _ := model.New(model.LIST, nil)
		doubled.Push(n * 2)
		val, _ := doubled.Get(0)
		return val
	})
	if mLen(result) != 3 {
		t.Errorf("expected len=3, got %d", mLen(result))
	}
	v, _ := result.Get(0)
	if intVal(t, v) != 2 {
		t.Errorf("expected first element=2, got %d", intVal(t, v))
	}
}

func TestMapPreservesHashKeys(t *testing.T) {
	m, _ := model.New(model.HASH, map[string]any{"a": 1, "b": 2})
	result := m.Map(func(v stdModel.Value) stdModel.Value {
		n, _ := v.Int()
		tmp, _ := model.New(model.LIST, nil)
		tmp.Push(n * 10)
		r, _ := tmp.Get(0)
		return r
	})
	if !result.Has("a") || !result.Has("b") {
		t.Error("Map should preserve hash keys")
	}
}

func TestMapPreservesOriginal(t *testing.T) {
	m := mustNew(t, model.LIST, []any{1, 2, 3})
	m.Map(func(v stdModel.Value) stdModel.Value { return v })
	if m.Len() != 3 {
		t.Errorf("Map should not modify original; expected len=3, got %d", m.Len())
	}
}

func TestMapCallbackCanReadModel(t *testing.T) {
	m := mustNew(t, model.LIST, []any{10, 20, 30})
	// Callback reads the model — safe with snapshot approach.
	result := m.Map(func(v stdModel.Value) stdModel.Value {
		n, _ := v.Int()
		first, _ := m.Get(0)
		base, _ := first.Int()
		tmp, _ := model.New(model.LIST, nil)
		tmp.Push(n + base) // add first element (10) to each
		r, _ := tmp.Get(0)
		return r
	})
	v, _ := result.Get(0)
	if intVal(t, v) != 20 {
		t.Errorf("expected 20, got %d", intVal(t, v))
	}
	v, _ = result.Get(1)
	if intVal(t, v) != 30 {
		t.Errorf("expected 30, got %d", intVal(t, v))
	}
}

// ── Reduce ────────────────────────────────────────────────────────────────────

func TestReduceSum(t *testing.T) {
	m := mustNew(t, model.LIST, []any{1, 2, 3, 4, 5})
	result := m.Reduce(func(carry, cur stdModel.Value) stdModel.Value {
		a, _ := carry.Int()
		b, _ := cur.Int()
		tmp, _ := model.New(model.LIST, nil)
		tmp.Push(a + b)
		v, _ := tmp.Get(0)
		return v
	})
	if intVal(t, result) != 15 {
		t.Errorf("expected sum=15, got %d", intVal(t, result))
	}
}

func TestReduceEmpty(t *testing.T) {
	m := mustNew(t, model.LIST, nil)
	result := m.Reduce(func(carry, cur stdModel.Value) stdModel.Value {
		return carry
	})
	if result != nil {
		t.Errorf("expected nil for empty model, got %v", result)
	}
}

func TestReduceSingleElement(t *testing.T) {
	m := mustNew(t, model.LIST, []any{42})
	result := m.Reduce(func(carry, cur stdModel.Value) stdModel.Value {
		return cur
	})
	if intVal(t, result) != 42 {
		t.Errorf("expected 42, got %d", intVal(t, result))
	}
}

func TestReduceCallbackCanReadModel(t *testing.T) {
	m := mustNew(t, model.LIST, []any{1, 2, 3})
	// Callback reads the model — safe with snapshot approach.
	m.Reduce(func(carry, cur stdModel.Value) stdModel.Value {
		m.Len() // reading the model during reduce
		return carry
	})
}

// ── JSON marshal/unmarshal ────────────────────────────────────────────────────

func TestJSONRoundTrip(t *testing.T) {
	m := mustNew(t, model.HASH, nil)
	m.Set("str", "hello")
	m.Set("num", 42)

	b, err := json.Marshal(m)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	m2 := mustNew(t, model.HASH, nil)
	if err := json.Unmarshal(b, m2); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	v, _ := m2.Get("str")
	if strVal(t, v) != "hello" {
		t.Errorf("expected 'hello', got %q", strVal(t, v))
	}
	v, _ = m2.Get("num")
	if n, _ := v.Float(); n != 42 {
		t.Errorf("expected 42, got %v", n)
	}
}

func TestJSONNestedModel(t *testing.T) {
	m := mustNew(t, model.HASH, nil)
	json.Unmarshal([]byte(`{"users":[{"name":"Alice"},{"name":"Bob"}]}`), m)

	v, err := m.Get("users")
	if err != nil {
		t.Fatalf("Get('users'): %v", err)
	}
	users, err := v.Model()
	if err != nil {
		t.Fatalf("Model(): %v", err)
	}
	if mLen(users) != 2 {
		t.Errorf("expected 2 users, got %d", mLen(users))
	}
}

func TestUnmarshalReplacesExistingData(t *testing.T) {
	m := mustNew(t, model.HASH, nil)
	m.Set("old", 99)
	json.Unmarshal([]byte(`{"new":1}`), m)
	if m.Has("old") {
		t.Error("UnmarshalJSON should replace existing data, not append")
	}
	if !m.Has("new") {
		t.Error("'new' key should exist after unmarshal")
	}
}

func TestUnmarshalLockedModel(t *testing.T) {
	m := mustNew(t, model.HASH, nil)
	m.Lock()
	err := json.Unmarshal([]byte(`{"k":1}`), m)
	if err == nil {
		t.Error("expected error unmarshaling into locked model")
	}
}

// ── Concurrency ───────────────────────────────────────────────────────────────

func TestConcurrentReads(t *testing.T) {
	m := mustNew(t, model.HASH, nil)
	for i := 0; i < 100; i++ {
		m.Set(fmt.Sprintf("k%d", i), i)
	}
	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(key string) {
			defer wg.Done()
			m.Get(key)
			m.Has(key)
			m.Len()
		}(fmt.Sprintf("k%d", i))
	}
	wg.Wait()
}

func TestConcurrentWrites(t *testing.T) {
	m := mustNew(t, model.LIST, nil)
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			m.Push(n)
		}(i)
	}
	wg.Wait()
	if m.Len() != 100 {
		t.Errorf("expected len=100 after 100 concurrent pushes, got %d", m.Len())
	}
}

func TestFilterCallbackNoDeadlock(t *testing.T) {
	m := mustNew(t, model.LIST, []any{1, 2, 3, 4, 5})
	done := make(chan struct{})
	go func() {
		m.Filter(func(v stdModel.Value) bool {
			n, _ := v.Int()
			// Reading and writing the model from inside the callback.
			// Both would deadlock before the snapshot fix.
			m.Has(0)
			m.Push(n * 100)
			return n%2 == 0
		})
		close(done)
	}()
	select {
	case <-done:
	// passed
	case <-waitTimeout(2 * time.Second):
		t.Fatal("Filter deadlocked — callback blocked on model mutex")
	}
}

func TestMapCallbackNoDeadlock(t *testing.T) {
	m := mustNew(t, model.LIST, []any{1, 2, 3})
	done := make(chan struct{})
	go func() {
		m.Map(func(v stdModel.Value) stdModel.Value {
			m.Len() // would deadlock before fix
			return v
		})
		close(done)
	}()
	select {
	case <-done:
	case <-waitTimeout(2 * time.Second):
		t.Fatal("Map deadlocked — callback blocked on model mutex")
	}
}

func TestReduceCallbackNoDeadlock(t *testing.T) {
	m := mustNew(t, model.LIST, []any{1, 2, 3})
	done := make(chan struct{})
	go func() {
		m.Reduce(func(carry, cur stdModel.Value) stdModel.Value {
			m.Len() // would deadlock before fix
			return carry
		})
		close(done)
	}()
	select {
	case <-done:
	case <-waitTimeout(2 * time.Second):
		t.Fatal("Reduce deadlocked — callback blocked on model mutex")
	}
}

// ── Value accessors ────────────────────────────────────────────────────────────

func TestValueBoolSuccess(t *testing.T) {
	m := mustNew(t, model.LIST, nil)
	m.Push(1) // 1 → true via cast
	v, _ := m.Get(0)
	b, err := v.Bool()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !b {
		t.Error("expected true from int(1)")
	}
}

func TestValueBoolError(t *testing.T) {
	m := mustNew(t, model.LIST, nil)
	m.Push("not-a-bool")
	v, _ := m.Get(0)
	_, err := v.Bool()
	if err == nil {
		t.Error("expected error converting 'not-a-bool' to bool")
	}
}

func TestValueIntFromFloat(t *testing.T) {
	m := mustNew(t, model.LIST, nil)
	m.Push(3.7)
	v, _ := m.Get(0)
	n, err := v.Int()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if n != 3 { // truncated toward zero
		t.Errorf("expected 3 (truncated), got %d", n)
	}
}

func TestValueIntError(t *testing.T) {
	m := mustNew(t, model.LIST, nil)
	m.Push("not-a-number")
	v, _ := m.Get(0)
	_, err := v.Int()
	if err == nil {
		t.Error("expected error converting 'not-a-number' to int")
	}
}

func TestValueFloat32Success(t *testing.T) {
	m := mustNew(t, model.LIST, nil)
	m.Push(3.14)
	v, _ := m.Get(0)
	f, err := v.Float32()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if f < 3.13 || f > 3.15 {
		t.Errorf("expected ~3.14, got %v", f)
	}
}

func TestValueFloat64Success(t *testing.T) {
	m := mustNew(t, model.LIST, nil)
	m.Push(2.718281828)
	v, _ := m.Get(0)
	f, err := v.Float64()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if f < 2.71 || f > 2.72 {
		t.Errorf("expected ~2.718, got %v", f)
	}
}

func TestValueStringFromInt(t *testing.T) {
	m := mustNew(t, model.LIST, nil)
	m.Push(42)
	v, _ := m.Get(0)
	s, err := v.String()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if s != "42" {
		t.Errorf("expected '42', got %q", s)
	}
}

func TestValueListError(t *testing.T) {
	m := mustNew(t, model.LIST, nil)
	m.Push("not-a-list")
	v, _ := m.Get(0)
	_, err := v.List()
	if err == nil {
		t.Error("expected error from List() on a string value")
	}
}

func TestValueMapError(t *testing.T) {
	m := mustNew(t, model.LIST, nil)
	m.Push("not-a-map")
	v, _ := m.Get(0)
	_, err := v.Map()
	if err == nil {
		t.Error("expected error from Map() on a string value")
	}
}

func TestValueModelError(t *testing.T) {
	m := mustNew(t, model.LIST, nil)
	m.Push("not-a-model")
	v, _ := m.Get(0)
	_, err := v.Model()
	if err == nil {
		t.Error("expected error from Model() on a string value")
	}
}

func TestValueValueReturnsRaw(t *testing.T) {
	m := mustNew(t, model.LIST, nil)
	m.Push(42)
	v, _ := m.Get(0)
	raw := v.Value()
	n, ok := raw.(int)
	if !ok {
		t.Fatalf("expected int from Value(), got %T", raw)
	}
	if n != 42 {
		t.Errorf("expected 42, got %d", n)
	}
}

// ── Additional error cases ─────────────────────────────────────────────────────

func TestDeleteNonExistentHashKey(t *testing.T) {
	m := mustNew(t, model.HASH, nil)
	m.Set("a", 1)
	err := m.Delete("nonexistent")
	if err == nil {
		t.Fatal("expected error deleting non-existent key")
	}
	if !errors.Is(err, model.InvalidIndex) {
		t.Errorf("expected InvalidIndex, got %v", err)
	}
}

func TestSetDataNilList(t *testing.T) {
	m := mustNew(t, model.LIST, nil)
	if err := m.SetData(nil); err == nil {
		t.Fatal("expected error from SetData(nil) on list model")
	} else if !errors.Is(err, model.InvalidDataSet) {
		t.Errorf("expected InvalidDataSet, got %v", err)
	}
}

func TestSetDataNilHash(t *testing.T) {
	m := mustNew(t, model.HASH, nil)
	if err := m.SetData(nil); err == nil {
		t.Fatal("expected error from SetData(nil) on hash model")
	} else if !errors.Is(err, model.InvalidDataSet) {
		t.Errorf("expected InvalidDataSet, got %v", err)
	}
}

func TestSeekNegativeList(t *testing.T) {
	m := mustNew(t, model.LIST, []any{1, 2})
	err := m.Seek(-1)
	if err == nil {
		t.Fatal("expected error from Seek(-1)")
	}
	if !errors.Is(err, model.InvalidIndex) {
		t.Errorf("expected InvalidIndex, got %v", err)
	}
}

func TestSeekMissingHashKey(t *testing.T) {
	m, _ := model.New(model.HASH, map[string]any{"a": 1})
	err := m.Seek("missing")
	if err == nil {
		t.Fatal("expected error from Seek on missing hash key")
	}
	if !errors.Is(err, model.InvalidIndex) {
		t.Errorf("expected InvalidIndex, got %v", err)
	}
}

func TestCurBeforeIteration(t *testing.T) {
	m := mustNew(t, model.LIST, []any{1, 2, 3})
	var key, val any
	if m.Cur(&key, &val) {
		t.Error("Cur should return false before any iteration begins")
	}
}

// ── Additional method coverage ─────────────────────────────────────────────────

func TestMarshalModel(t *testing.T) {
	m := mustNew(t, model.HASH, nil)
	m.Set("key", "value")
	b, err := m.MarshalModel()
	if err != nil {
		t.Fatalf("MarshalModel: %v", err)
	}
	expected, _ := json.Marshal(m)
	if string(b) != string(expected) {
		t.Errorf("MarshalModel output differs from MarshalJSON\ngot:  %s\nwant: %s", b, expected)
	}
}

// TestModelStringErrorFallback verifies that Model.String() returns the error
// message when MarshalJSON fails (e.g., when the model contains a value that
// encoding/json cannot serialize, such as a channel).
func TestModelStringErrorFallback(t *testing.T) {
	m := mustNew(t, model.HASH, nil)
	// Channels are not JSON-serializable; this causes MarshalJSON to return an error.
	m.Set("bad", make(chan int))
	s := m.String()
	if s == "" {
		t.Fatal("String() returned empty string; expected an error message")
	}
	// The string must not be valid JSON — it should be an error message, not "{}".
	if s == "{}" || s[0] == '{' {
		t.Errorf("String() returned JSON-like output %q; expected error message", s)
	}
}

func TestUnmarshalModel(t *testing.T) {
	m := mustNew(t, model.HASH, nil)
	m.Set("x", 99)
	b, _ := json.Marshal(m)

	m2 := mustNew(t, model.HASH, nil)
	if err := m2.UnmarshalModel(b); err != nil {
		t.Fatalf("UnmarshalModel: %v", err)
	}
	v, err := m2.Get("x")
	if err != nil {
		t.Fatalf("Get('x'): %v", err)
	}
	if n, _ := v.Float(); n != 99 {
		t.Errorf("expected 99, got %v", n)
	}
}

func TestUnmarshalModelNull(t *testing.T) {
	m := mustNew(t, model.HASH, nil)
	m.Set("existing", 1)
	if err := m.UnmarshalModel([]byte("null")); err != nil {
		t.Fatalf("UnmarshalModel(null) should be a no-op: %v", err)
	}
	if !m.Has("existing") {
		t.Error("UnmarshalModel(null) should not modify the model")
	}
}

func TestUnmarshalModelLockedModel(t *testing.T) {
	m := mustNew(t, model.HASH, nil)
	m.Lock()
	err := m.UnmarshalModel([]byte(`{"k":1}`))
	if err == nil {
		t.Fatal("expected error from UnmarshalModel on locked model")
	}
	if !errors.Is(err, model.ReadOnlyModel) {
		t.Errorf("expected ReadOnlyModel, got %v", err)
	}
}

func TestGetTypeAfterSetType(t *testing.T) {
	m := mustNew(t, model.HASH, nil)
	if m.GetType() != model.HASH {
		t.Errorf("expected HASH, got %v", m.GetType())
	}
	if err := m.SetType(model.LIST); err != nil {
		t.Fatalf("SetType: %v", err)
	}
	if m.GetType() != model.LIST {
		t.Errorf("expected LIST after SetType, got %v", m.GetType())
	}
}

func TestFilterEmpty(t *testing.T) {
	m := mustNew(t, model.LIST, nil)
	result := m.Filter(func(v stdModel.Value) bool { return true })
	if mLen(result) != 0 {
		t.Errorf("expected empty result from Filter on empty model, got %d", mLen(result))
	}
}

func TestFilterListValues(t *testing.T) {
	m := mustNew(t, model.LIST, []any{1, 2, 3, 4, 5, 6})
	result := m.Filter(func(v stdModel.Value) bool {
		n, _ := v.Int()
		return n%2 == 0
	})
	if mLen(result) != 3 {
		t.Fatalf("expected 3 elements, got %d", mLen(result))
	}
	for i, expected := range []int{2, 4, 6} {
		v, err := result.Get(i)
		if err != nil {
			t.Fatalf("Get(%d): %v", i, err)
		}
		if n, _ := v.Int(); n != expected {
			t.Errorf("index %d: expected %d, got %d", i, expected, n)
		}
	}
}

func TestFilterHashValues(t *testing.T) {
	m, _ := model.New(model.HASH, map[string]any{"a": 1, "b": 2, "c": 3, "d": 4})
	result := m.Filter(func(v stdModel.Value) bool {
		n, _ := v.Int()
		return n > 2
	})
	if !result.Has("c") || !result.Has("d") {
		t.Error("expected keys 'c' and 'd' in filter result")
	}
	if result.Has("a") || result.Has("b") {
		t.Error("keys 'a' and 'b' should be filtered out")
	}
}

func TestMapEmpty(t *testing.T) {
	m := mustNew(t, model.LIST, nil)
	result := m.Map(func(v stdModel.Value) stdModel.Value { return v })
	if mLen(result) != 0 {
		t.Errorf("expected empty result from Map on empty model, got %d", mLen(result))
	}
}

func TestMapHashValues(t *testing.T) {
	m, _ := model.New(model.HASH, map[string]any{"a": 1, "b": 2})
	result := m.Map(func(v stdModel.Value) stdModel.Value {
		n, _ := v.Int()
		tmp, _ := model.New(model.LIST, nil)
		tmp.Push(n * 10)
		r, _ := tmp.Get(0)
		return r
	})
	va, _ := result.Get("a")
	if intVal(t, va) != 10 {
		t.Errorf("expected 'a'→10, got %d", intVal(t, va))
	}
	vb, _ := result.Get("b")
	if intVal(t, vb) != 20 {
		t.Errorf("expected 'b'→20, got %d", intVal(t, vb))
	}
}

func TestReduceHash(t *testing.T) {
	// Hash is sorted by key after import (a=1, b=2, c=3), so iteration is deterministic.
	m, _ := model.New(model.HASH, map[string]any{"a": 1, "b": 2, "c": 3})
	result := m.Reduce(func(carry, cur stdModel.Value) stdModel.Value {
		a, _ := carry.Int()
		b, _ := cur.Int()
		tmp, _ := model.New(model.LIST, nil)
		tmp.Push(a + b)
		v, _ := tmp.Get(0)
		return v
	})
	if intVal(t, result) != 6 {
		t.Errorf("expected sum=6, got %d", intVal(t, result))
	}
}

// ── Additional sort coverage ───────────────────────────────────────────────────

func TestSortByValueHash(t *testing.T) {
	m := mustNew(t, model.HASH, nil)
	m.Set("b", 3)
	m.Set("a", 1)
	m.Set("c", 2)
	m.Sort(stdSorter.SortAsc)
	var vals []int
	var key, val any
	for m.Next(&key, &val) {
		n, _ := val.(stdModel.Value).Int()
		vals = append(vals, n)
	}
	if len(vals) != 3 || vals[0] != 1 || vals[1] != 2 || vals[2] != 3 {
		t.Errorf("expected [1 2 3] after SortByValue on hash, got %v", vals)
	}
}

func TestSortByValueAsString(t *testing.T) {
	m := mustNew(t, model.LIST, []any{30, 10, 200})
	m.Sort(stdSorter.SortByValue | stdSorter.SortAsString)
	v0, _ := m.Get(0)
	v1, _ := m.Get(1)
	v2, _ := m.Get(2)
	// lexicographic: "10" < "200" < "30"
	if intVal(t, v0) != 10 || intVal(t, v1) != 200 || intVal(t, v2) != 30 {
		t.Errorf("expected [10 200 30] (lexicographic), got [%d %d %d]", intVal(t, v0), intVal(t, v1), intVal(t, v2))
	}
}

func TestSortByKeyListNoOp(t *testing.T) {
	m := mustNew(t, model.LIST, []any{3, 1, 2})
	m.Sort(stdSorter.SortByKey)
	v0, _ := m.Get(0)
	v1, _ := m.Get(1)
	v2, _ := m.Get(2)
	if intVal(t, v0) != 3 || intVal(t, v1) != 1 || intVal(t, v2) != 2 {
		t.Errorf("SortByKey on list should be a no-op, got [%d %d %d]", intVal(t, v0), intVal(t, v1), intVal(t, v2))
	}
}

func TestSortByValueMixedTypes(t *testing.T) {
	// Stratified order: bool(false) < numeric(1) < string("apple") < string("banana")
	m := mustNew(t, model.LIST, nil)
	m.Push("banana")
	m.Push(1)
	m.Push(false)
	m.Push("apple")
	m.Sort(stdSorter.SortAsc)

	v0, _ := m.Get(0)
	b, _ := v0.Bool()
	if b != false {
		t.Errorf("expected false at index 0 (bool bucket), got %v", v0.Value())
	}
	v1, _ := m.Get(1)
	if intVal(t, v1) != 1 {
		t.Errorf("expected 1 at index 1 (numeric bucket), got %v", v1.Value())
	}
	v2, _ := m.Get(2)
	s2, _ := v2.String()
	v3, _ := m.Get(3)
	s3, _ := v3.String()
	if s2 != "apple" || s3 != "banana" {
		t.Errorf("expected [apple banana] at indices 2-3 (string bucket), got [%v %v]", s2, s3)
	}
}

func TestSortAscExplicit(t *testing.T) {
	m := mustNew(t, model.LIST, []any{3, 1, 2})
	m.Sort(stdSorter.SortByValue | stdSorter.SortAsc)
	v0, _ := m.Get(0)
	if intVal(t, v0) != 1 {
		t.Errorf("expected 1 at index 0 with explicit SortAsc, got %d", intVal(t, v0))
	}
}

// ── Additional lock enforcement ────────────────────────────────────────────────

func TestLockPreventsMerge(t *testing.T) {
	base := mustNew(t, model.HASH, nil)
	base.Lock()
	inc := mustNew(t, model.HASH, nil)
	inc.Set("k", 1)
	err := base.Merge(inc)
	if err == nil {
		t.Fatal("expected error from Merge on locked model")
	}
	if !errors.Is(err, model.ReadOnlyModel) {
		t.Errorf("expected ReadOnlyModel, got %v", err)
	}
}

func TestLockPreventsReverse(t *testing.T) {
	m := mustNew(t, model.LIST, []any{1, 2, 3})
	m.Lock()
	err := m.Reverse()
	if err == nil {
		t.Fatal("expected error from Reverse on locked model")
	}
	if !errors.Is(err, model.ReadOnlyModel) {
		t.Errorf("expected ReadOnlyModel, got %v", err)
	}
}

func TestLockPreventsSetData(t *testing.T) {
	m := mustNew(t, model.HASH, nil)
	m.Lock()
	err := m.SetData(map[string]any{"k": 1})
	if err == nil {
		t.Fatal("expected error from SetData on locked model")
	}
	if !errors.Is(err, model.ReadOnlyModel) {
		t.Errorf("expected ReadOnlyModel, got %v", err)
	}
}

// ── Additional concurrency ─────────────────────────────────────────────────────

func TestConcurrentReadWrite(t *testing.T) {
	m := mustNew(t, model.HASH, nil)
	for i := 0; i < 10; i++ {
		m.Set(fmt.Sprintf("k%d", i), i)
	}
	var wg sync.WaitGroup
	for i := 0; i < 40; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			if i%2 == 0 {
				m.Get(fmt.Sprintf("k%d", i%10))
				m.Has(fmt.Sprintf("k%d", i%10))
				m.Len()
			} else {
				m.Set(fmt.Sprintf("extra%d", i), i)
			}
		}(i)
	}
	wg.Wait()
	if m.Len() < 10 {
		t.Errorf("expected at least 10 elements after concurrent access, got %d", m.Len())
	}
}

func TestConcurrentReadersAreParallel(t *testing.T) {
	// With RWMutex, multiple readers should not block each other.
	// Load the model, then fire many goroutines reading simultaneously
	// and verify all reads return correct values.
	m := mustNew(t, model.HASH, nil)
	for i := 0; i < 50; i++ {
		m.Set(fmt.Sprintf("k%d", i), i)
	}
	var wg sync.WaitGroup
	errs := make(chan string, 100)
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			key := fmt.Sprintf("k%d", idx%50)
			v, err := m.Get(key)
			if err != nil {
				errs <- fmt.Sprintf("Get(%s): %v", key, err)
				return
			}
			n, _ := v.Int()
			if n != idx%50 {
				errs <- fmt.Sprintf("Get(%s): expected %d, got %d", key, idx%50, n)
			}
		}(i)
	}
	wg.Wait()
	close(errs)
	for e := range errs {
		t.Error(e)
	}
}

// ── GetData isolation ─────────────────────────────────────────────────────────

func TestGetDataReturnsCopies(t *testing.T) {
	m := mustNew(t, model.HASH, nil)
	m.Set("a", 1)

	data, hashIdx, idxHash := m.GetData()

	// Mutate the returned copies — the model must be unaffected.
	data[0] = "mutated"
	hashIdx["injected"] = 99
	idxHash[99] = "injected"

	if m.Has("injected") {
		t.Error("mutating the returned hashIdx should not affect the model")
	}
	v, err := m.Get("a")
	if err != nil {
		t.Fatalf("Get('a') after GetData mutation: %v", err)
	}
	if intVal(t, v) != 1 {
		t.Errorf("model data was affected by mutation of GetData result; expected 1, got %d", intVal(t, v))
	}
}

// ── Prev() cursor clamp regression tests ─────────────────────────────────────

func TestPrevClampToMinusOne(t *testing.T) {
	// Calling Prev() when cursor is already at -1 must not send pos below -1.
	// Before the fix: pos would go to -2, -3, etc., causing Next() to panic
	// with a negative slice index.
	m := mustNew(t, model.LIST, []any{1, 2, 3})

	// Prev on a fresh model (pos=-1)
	var key, val any
	if m.Prev(&key, &val) {
		t.Error("Prev on fresh model should return false")
	}
	// Call Next after Prev — must NOT panic and must return the first element
	if !m.Next(&key, &val) {
		t.Fatal("Next after Prev-from-start should return true")
	}
	if intVal(t, val.(stdModel.Value)) != 1 {
		t.Errorf("expected element 1, got %d", intVal(t, val.(stdModel.Value)))
	}
}

func TestPrevMultipleCallsPastStart(t *testing.T) {
	// Multiple Prev() calls past the start must not corrupt the cursor.
	m := mustNew(t, model.LIST, []any{10, 20, 30})

	// Call Prev many times from the beginning
	var key, val any
	for i := 0; i < 10; i++ {
		m.Prev(&key, &val)
	}
	// Cursor must be clamped at -1; Next should restart from element 0
	if !m.Next(&key, &val) {
		t.Fatal("Next after repeated Prev should return true")
	}
	if intVal(t, val.(stdModel.Value)) != 10 {
		t.Errorf("expected element 10, got %d", intVal(t, val.(stdModel.Value)))
	}
}

// ── Delete / SetData cursor regression tests ──────────────────────────────────

func TestDeleteFailedDoesNotResetPos(t *testing.T) {
	// A failed Delete must not reset the cursor.
	m := mustNew(t, model.LIST, []any{1, 2, 3})
	var key, val any
	m.Next(&key, &val) // pos = 0
	m.Next(&key, &val) // pos = 1

	// Delete with out-of-bounds index — should fail
	if err := m.Delete(99); err == nil {
		t.Fatal("expected error from Delete(99)")
	}
	// Cursor must still be at position 1
	if !m.Cur(&key, &val) {
		t.Fatal("Cur should still return true — cursor should be unchanged after failed Delete")
	}
	if intVal(t, val.(stdModel.Value)) != 2 {
		t.Errorf("expected element 2 at current position, got %d", intVal(t, val.(stdModel.Value)))
	}
}

func TestDeleteHashFailedDoesNotResetPos(t *testing.T) {
	m := mustNew(t, model.HASH, nil)
	m.Set("a", 1)
	m.Set("b", 2)
	m.Seek("a")

	if err := m.Delete("nonexistent"); err == nil {
		t.Fatal("expected error from Delete('nonexistent')")
	}
	// Cursor must still be at 'a'
	var key, val any
	if !m.Cur(&key, &val) {
		t.Fatal("Cur should still return true after failed hash Delete")
	}
	if key.(string) != "a" {
		t.Errorf("expected cursor at 'a', got %v", key)
	}
}

func TestSetDataFailedDoesNotResetPos(t *testing.T) {
	m := mustNew(t, model.LIST, []any{1, 2, 3})
	var key, val any
	m.Next(&key, &val) // pos = 0
	m.Next(&key, &val) // pos = 1

	// SetData with wrong type — should fail
	if err := m.SetData("not-a-slice"); err == nil {
		t.Fatal("expected error from SetData with wrong type")
	}
	// Cursor must still be at position 1
	if !m.Cur(&key, &val) {
		t.Fatal("Cur should still return true — cursor should be unchanged after failed SetData")
	}
	if intVal(t, val.(stdModel.Value)) != 2 {
		t.Errorf("expected element 2, got %d", intVal(t, val.(stdModel.Value)))
	}
}

// ── Sort no-op does not reset cursor ──────────────────────────────────────────

func TestSortNoFlagsDoesNotResetPos(t *testing.T) {
	m := mustNew(t, model.LIST, []any{1, 2, 3})
	var key, val any
	m.Next(&key, &val) // pos = 0

	// Sort with zero flags — no-op, cursor must not be reset
	m.Sort(0)
	if !m.Cur(&key, &val) {
		t.Error("Cur should still return true after Sort(0)")
	}
	if intVal(t, val.(stdModel.Value)) != 1 {
		t.Errorf("expected element 1 after Sort(0), got %d", intVal(t, val.(stdModel.Value)))
	}
}

func TestSortByKeyOnListNoOpDoesNotResetPos(t *testing.T) {
	m := mustNew(t, model.LIST, []any{1, 2, 3})
	var key, val any
	m.Next(&key, &val) // pos = 0

	// SortByKey on list is a no-op (no SortAsString), cursor must not be reset
	m.Sort(stdSorter.SortByKey)
	if !m.Cur(&key, &val) {
		t.Error("Cur should still return true after SortByKey no-op on list")
	}
}

// ── Cursor reset after mutation ───────────────────────────────────────────────

func TestSetDataResetsPos(t *testing.T) {
	m := mustNew(t, model.LIST, []any{1, 2, 3})
	// advance cursor to position 1
	var key, val any
	m.Next(&key, &val)
	m.Next(&key, &val)
	// replace data — cursor must reset
	m.SetData([]any{10, 20})
	// Cur must return false (pos == -1)
	if m.Cur(&key, &val) {
		t.Error("Cur should return false after SetData resets the cursor")
	}
	// Next must restart from the beginning
	if !m.Next(&key, &val) {
		t.Fatal("Next returned false after SetData")
	}
	if intVal(t, val.(stdModel.Value)) != 10 {
		t.Errorf("expected first element 10 after SetData, got %d", intVal(t, val.(stdModel.Value)))
	}
}

func TestDeleteResetsPos(t *testing.T) {
	m := mustNew(t, model.LIST, []any{1, 2, 3})
	var key, val any
	m.Next(&key, &val) // pos = 0
	m.Next(&key, &val) // pos = 1
	m.Delete(0)
	// cursor must have been reset
	if m.Cur(&key, &val) {
		t.Error("Cur should return false after Delete resets the cursor")
	}
}

func TestSortResetsPos(t *testing.T) {
	m := mustNew(t, model.LIST, []any{3, 1, 2})
	var key, val any
	m.Next(&key, &val) // pos = 0
	m.Sort(stdSorter.SortAsc)
	// cursor must have been reset
	if m.Cur(&key, &val) {
		t.Error("Cur should return false after Sort resets the cursor")
	}
	// Next should start from the new beginning
	if !m.Next(&key, &val) {
		t.Fatal("Next returned false after Sort")
	}
	if intVal(t, val.(stdModel.Value)) != 1 {
		t.Errorf("expected sorted first element 1 after Sort, got %d", intVal(t, val.(stdModel.Value)))
	}
}

func TestReverseResetsPos(t *testing.T) {
	m := mustNew(t, model.LIST, []any{1, 2, 3})
	var key, val any
	m.Next(&key, &val) // pos = 0
	m.Next(&key, &val) // pos = 1
	m.Reverse()
	if m.Cur(&key, &val) {
		t.Error("Cur should return false after Reverse resets the cursor")
	}
}

// ── UnmarshalJSON atomicity ───────────────────────────────────────────────────

// TestUnmarshalJSONNeverEmpty verifies that concurrent readers never observe
// the model in an empty state during UnmarshalJSON. With the old two-phase
// reset+import approach, readers could see Len()==0 briefly. The temp-model
// swap eliminates that window.
func TestUnmarshalJSONNeverEmpty(t *testing.T) {
	m := mustNew(t, model.HASH, nil)
	for i := 0; i < 50; i++ {
		m.Set(fmt.Sprintf("k%d", i), i)
	}

	sawEmpty := int32(0)
	var wg sync.WaitGroup

	// Concurrent readers
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 200; j++ {
				if m.Len() == 0 {
					atomic.StoreInt32(&sawEmpty, 1)
				}
			}
		}()
	}

	// Concurrent unmarshalers replacing data
	newData := []byte(`{"a":1,"b":2,"c":3,"d":4,"e":5}`)
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 20; j++ {
				json.Unmarshal(newData, m)
			}
		}()
	}

	wg.Wait()
	if atomic.LoadInt32(&sawEmpty) == 1 {
		t.Error("concurrent reader observed model in empty state during UnmarshalJSON")
	}
}

// ── TOCTOU: locked check before vs inside the write lock ──────────────────────

// TestLockTOCTOUDoubleCheck verifies that after Lock() is called, no write
// method can succeed — even writes that checked locked before Lock() was
// called. The double-check inside the mutex closes this window.
func TestLockTOCTOUDoubleCheck(t *testing.T) {
	for attempt := 0; attempt < 50; attempt++ {
		m := mustNew(t, model.HASH, nil)

		var wg sync.WaitGroup
		wg.Add(2)

		// Goroutine A: hammers Set
		go func() {
			defer wg.Done()
			for i := 0; i < 500; i++ {
				m.Set("k", i)
			}
		}()

		// Goroutine B: calls Lock() after a brief yield
		go func() {
			defer wg.Done()
			runtime.Gosched()
			m.Lock()
		}()

		wg.Wait()

		// After both goroutines finish, Lock() has definitely been called.
		// All subsequent writes MUST fail.
		for i := 0; i < 20; i++ {
			if err := m.Set("k", i); err == nil {
				t.Errorf("attempt %d: Set succeeded after Lock() was called", attempt)
			}
			if err := m.Push("v"); err == nil {
				t.Errorf("attempt %d: Push succeeded after Lock() was called", attempt)
			}
			if err := m.Delete("k"); err == nil {
				t.Errorf("attempt %d: Delete succeeded after Lock() was called", attempt)
			}
		}
	}
}

// TestLockIsAtomic verifies that the model's locked state is observed
// consistently by all goroutines immediately after Lock() returns.
func TestLockIsAtomic(t *testing.T) {
	m := mustNew(t, model.HASH, nil)
	m.Lock()

	var wg sync.WaitGroup
	failures := int32(0)
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := m.Set("k", 1); err == nil {
				atomic.AddInt32(&failures, 1)
			}
		}()
	}
	wg.Wait()
	if failures > 0 {
		t.Errorf("%d goroutines succeeded writing to a locked model", failures)
	}
}

// ── Concurrent Merge safety ───────────────────────────────────────────────────

// TestConcurrentMergeReadsDoNotPanic verifies that concurrent reads during a
// Merge (including during the recursive nested-model unlock window) do not
// panic or corrupt the model.
func TestConcurrentMergeReadsDoNotPanic(t *testing.T) {
	// Build a model with deeply nested sub-models to exercise the unlock window
	inner1, _ := model.New(model.HASH, nil)
	for i := 0; i < 10; i++ {
		inner1.Set(fmt.Sprintf("x%d", i), i)
	}
	base, _ := model.New(model.HASH, nil)
	base.Set("nested", inner1)
	for i := 0; i < 10; i++ {
		base.Set(fmt.Sprintf("k%d", i), i)
	}

	inner2, _ := model.New(model.HASH, nil)
	for i := 0; i < 10; i++ {
		inner2.Set(fmt.Sprintf("y%d", i), i*10)
	}
	incoming, _ := model.New(model.HASH, nil)
	incoming.Set("nested", inner2)
	for i := 0; i < 5; i++ {
		incoming.Set(fmt.Sprintf("new%d", i), i*100)
	}

	var wg sync.WaitGroup

	// Concurrent readers during Merge
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				base.Len()
				base.Has("nested")
				base.Get("k0")
			}
		}()
	}

	// The merge itself
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := base.Merge(incoming); err != nil {
			t.Errorf("Merge failed: %v", err)
		}
	}()

	wg.Wait()
	// Verify model is still consistent after concurrent access
	if base.Len() == 0 {
		t.Error("model is empty after concurrent Merge+reads")
	}
}

// ── Has key coercion ──────────────────────────────────────────────────────────

// TestHasHashKeyCoercion verifies that Has uses the same key coercion as Set/Get/Delete.
// Before the fix, Has used a direct string type assertion so Has(42) returned false even
// when key "42" existed (set via Set(42, "v")).
func TestHasHashKeyCoercion(t *testing.T) {
	m := mustNew(t, model.HASH, nil)
	m.Set(42, "forty-two")
	m.Set(3.14, "pi")

	if !m.Has("42") {
		t.Error("Has('42') should be true after Set(42, ...)")
	}
	if !m.Has(42) {
		t.Error("Has(42) should be true — same coercion as Set/Get/Delete")
	}
	if !m.Has(3.14) {
		t.Error("Has(3.14) should be true — same coercion as Set/Get/Delete")
	}
}

// ── SetData LIST slice isolation ──────────────────────────────────────────────

// TestSetDataListDoesNotAliasInputSlice verifies that SetData copies the input
// slice so that external mutations do not corrupt the model's internal state.
func TestSetDataListDoesNotAliasInputSlice(t *testing.T) {
	d := []any{1, 2, 3}
	m := mustNew(t, model.LIST, nil)
	if err := m.SetData(d); err != nil {
		t.Fatalf("SetData: %v", err)
	}

	// Mutate the original slice after SetData.
	d[0] = 99

	v, err := m.Get(0)
	if err != nil {
		t.Fatalf("Get(0): %v", err)
	}
	if intVal(t, v) != 1 {
		t.Errorf("SetData aliased input slice: expected 1 at index 0, got %d", intVal(t, v))
	}
}

// ── Merge TOCTOU: Lock during recursive sub-model merge ───────────────────────

// TestMergeLockDuringRecursiveUnlockWindow verifies that if Lock() is called on
// the outer model during the recursive sub-model merge (which releases and
// re-acquires the outer mutex), subsequent keys in the outer merge are NOT
// written. Before the fix, the re-check after relocking was missing.
func TestMergeLockDuringRecursiveUnlockWindow(t *testing.T) {
	inner1, _ := model.New(model.HASH, nil)
	inner1.Set("a", 1)
	base, _ := model.New(model.HASH, nil)
	base.Set("nested", inner1)
	base.Set("plain", 10)

	inner2, _ := model.New(model.HASH, nil)
	inner2.Set("b", 2)
	incoming, _ := model.New(model.HASH, nil)
	incoming.Set("nested", inner2)
	incoming.Set("extra", 99) // key that would be written after the recursive merge

	// Lock the base model. Merge should detect this and return an error rather
	// than writing "extra" to a locked model.
	base.Lock()

	err := base.Merge(incoming)
	if err == nil {
		t.Fatal("Merge on locked model should return an error")
	}
	if !errors.Is(err, model.ReadOnlyModel) {
		t.Errorf("expected ReadOnlyModel, got %v", err)
	}
	// Confirm "extra" was not written.
	if base.Has("extra") {
		t.Error("Merge wrote 'extra' to a locked model")
	}
}

// TestSortByValueModelBucket verifies that stratifiedLess correctly sorts nested
// *Model values in a LIST using the model bucket (bucket 0). Models are compared
// by GetID() string and then by element count as a tiebreaker.
func TestSortByValueModelBucket(t *testing.T) {
	// Build three child models with distinct IDs.
	mC, _ := model.New(model.HASH, nil)
	mC.SetID("c")
	mC.Set("x", 1)

	mA, _ := model.New(model.HASH, nil)
	mA.SetID("a")
	mA.Set("x", 1)
	mA.Set("y", 2)

	mB, _ := model.New(model.HASH, nil)
	mB.SetID("b")

	// Insert in reverse-alphabetical order: c, a, b.
	parent := mustNew(t, model.LIST, nil)
	parent.Push(mC)
	parent.Push(mA)
	parent.Push(mB)

	if err := parent.Sort(stdSorter.SortAsc); err != nil {
		t.Fatalf("Sort: %v", err)
	}

	// After ascending sort the order should be: mA ("a") < mB ("b") < mC ("c").
	for i, wantID := range []string{"a", "b", "c"} {
		v, err := parent.Get(i)
		if err != nil {
			t.Fatalf("Get(%d): %v", i, err)
		}
		mdl, err := v.Model()
		if err != nil {
			t.Fatalf("Model() at index %d: %v", i, err)
		}
		gotID := fmt.Sprintf("%v", mdl.GetID())
		if gotID != wantID {
			t.Errorf("index %d: want ID %q, got %q", i, wantID, gotID)
		}
	}
}

// TestSortByValueModelTiebreakByLen verifies that when two *Model values share
// the same GetID, the shorter model sorts before the longer one (modelLen tiebreak).
func TestSortByValueModelTiebreakByLen(t *testing.T) {
	mFew, _ := model.New(model.HASH, nil)
	mFew.SetID("same")
	mFew.Set("a", 1)

	mMany, _ := model.New(model.HASH, nil)
	mMany.SetID("same")
	mMany.Set("a", 1)
	mMany.Set("b", 2)
	mMany.Set("c", 3)

	parent := mustNew(t, model.LIST, nil)
	parent.Push(mMany)
	parent.Push(mFew)

	if err := parent.Sort(stdSorter.SortAsc); err != nil {
		t.Fatalf("Sort: %v", err)
	}

	v0, _ := parent.Get(0)
	m0, _ := v0.Model()
	data0, _, _ := m0.GetData()
	if len(data0) != 1 {
		t.Errorf("expected shorter model (len 1) at index 0, got len %d", len(data0))
	}
}

// TestImportNestedSliceInSlice verifies that importSlice correctly handles
// arrays-of-arrays by creating nested LIST child models.
func TestImportNestedSliceInSlice(t *testing.T) {
	jsn := []byte(`[[1,2],[3,4]]`)
	m, err := model.New(model.LIST, nil)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if err := m.UnmarshalJSON(jsn); err != nil {
		t.Fatalf("UnmarshalJSON: %v", err)
	}
	if m.Len() != 2 {
		t.Fatalf("expected 2 child models, got %d", m.Len())
	}
	// Each child should be a LIST model containing 2 numeric elements.
	for i := 0; i < 2; i++ {
		v, err := m.Get(i)
		if err != nil {
			t.Fatalf("Get(%d): %v", i, err)
		}
		child, err := v.Model()
		if err != nil {
			t.Fatalf("Model() at index %d: %v", i, err)
		}
		childData, _, _ := child.GetData()
		if len(childData) != 2 {
			t.Errorf("child[%d]: expected len 2, got %d", i, len(childData))
		}
	}
	// Verify specific values: child[0][0]==1, child[0][1]==2, child[1][0]==3, child[1][1]==4.
	expected := [][]float64{{1, 2}, {3, 4}}
	for i, row := range expected {
		cv, _ := m.Get(i)
		child, _ := cv.Model()
		for j, want := range row {
			ev, err := child.Get(j)
			if err != nil {
				t.Fatalf("child[%d].Get(%d): %v", i, j, err)
			}
			got, err := ev.Float64()
			if err != nil {
				t.Fatalf("child[%d][%d] Float64: %v", i, j, err)
			}
			if got != want {
				t.Errorf("child[%d][%d]: want %v, got %v", i, j, want, got)
			}
		}
	}
}

// ── Benchmarks ─────────────────────────────────────────────────────────────────

// BenchmarkSortByKeyHash measures Sort(SortByKey) on a HASH model with 100 entries.
func BenchmarkSortByKeyHash(b *testing.B) {
	m, _ := model.New(model.HASH, nil)
	for i := 0; i < 100; i++ {
		m.Set(fmt.Sprintf("key%03d", i), i)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		m.Sort(stdSorter.SortByKey)
	}
}

// BenchmarkSortByValueList measures Sort(SortAsc) on a LIST model with 100 numeric elements.
func BenchmarkSortByValueList(b *testing.B) {
	m, _ := model.New(model.LIST, nil)
	for i := 99; i >= 0; i-- {
		m.Push(i)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		m.Sort(stdSorter.SortAsc)
	}
}

// BenchmarkMergeHash measures merging two 50-key HASH models.
func BenchmarkMergeHash(b *testing.B) {
	base, _ := model.New(model.HASH, nil)
	for i := 0; i < 50; i++ {
		base.Set(fmt.Sprintf("base%03d", i), i)
	}
	inc, _ := model.New(model.HASH, nil)
	for i := 0; i < 50; i++ {
		inc.Set(fmt.Sprintf("inc%03d", i), i)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		clone, _ := model.New(model.HASH, nil)
		var k, v any
		base.Reset()
		for base.Next(&k, &v) {
			clone.Set(k.(string), v)
		}
		clone.Merge(inc)
	}
}

// BenchmarkMergeDeepHash measures merging nested HASH models two levels deep.
func BenchmarkMergeDeepHash(b *testing.B) {
	makeNested := func(prefix string, depth, width int) *model.Model {
		root, _ := model.New(model.HASH, nil)
		for i := 0; i < width; i++ {
			child, _ := model.New(model.HASH, nil)
			for j := 0; j < width; j++ {
				child.Set(fmt.Sprintf("%s_c%d_k%d", prefix, i, j), j)
			}
			root.Set(fmt.Sprintf("%s_k%d", prefix, i), child)
		}
		_ = depth
		return root
	}
	base := makeNested("base", 2, 5)
	inc := makeNested("inc", 2, 5)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		clone, _ := model.New(model.HASH, nil)
		var k, v any
		base.Reset()
		for base.Next(&k, &v) {
			clone.Set(k.(string), v)
		}
		clone.Merge(inc)
	}
}

// waitTimeout returns a channel that fires after d, used to detect deadlocks in tests.
func waitTimeout(d time.Duration) <-chan struct{} {
	ch := make(chan struct{})
	go func() {
		time.Sleep(d)
		close(ch)
	}()
	return ch
}
