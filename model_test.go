package model_test

import (
	"fmt"
	"testing"

	"github.com/bdlm/errors"
	"github.com/bdlm/logfmt"
	"github.com/bdlm/model"
	"github.com/bdlm/std"
	log "github.com/sirupsen/logrus"
)

func init() {
	level, _ := log.ParseLevel("debug")
	log.SetFormatter(&logfmt.TextFormat{})
	log.SetLevel(level)
}

func TestNewHash(t *testing.T) {
	//mdl := model.New(std.ModelTypeHash)
	//mdl.Set("key1", "val1")
	//val, err := mdl.Get("key")
	//str, err2 := val.String()
	//t.Errorf("%v - %v (%v)", str, err2, err)

	//for mdl.(std.Iterator).Next(func(k interface{}, v std.Value) {
	//	t.Errorf("key: %v; val: %v;", k, v.Value())
	//}) {
	//}
	//t.Errorf("done")
}

func TestHashIterator(t *testing.T) {
	//	mdl := model.New(std.ModelTypeHash)
	//	mdl.Set("key1", "val1")
	//	mdl.Set("key2", "val2")
	//
	//	var key, val interface{}
	//	for mdl.(std.Iterator).Next(&key, &val) {
	//		t.Errorf("key: %v; val: %v;", key, val)
	//	}
	//	t.Errorf("done")
}

func TestModelType(t *testing.T) {
	mdl := model.New(std.ModelTypeHash)

	// Push is only valid for std.ModelTypeList model types
	err := mdl.Push("val1")
	if nil == err {
		t.Errorf("expected error, received nil")
	}
	if e, ok := err.(errors.Err); !ok {
		t.Errorf("expected errors.Err, received '%v'", err)
	} else if e.Code() != model.InvalidMethodContext {
		t.Errorf("expected model.InvalidMethodContext, received '%v'", e.Code())
	}

	// hash keys are always strings
	str := "string-key"
	integer := 10
	float := float64(1234567890)
	err = fmt.Errorf("err-key")
	mdl.Set(str, "string-key")
	mdl.Set(integer, "10")
	mdl.Set(float, "1234567890.0000000000") // 10-digit precision...
	mdl.Set(err, "err-key")
	var key, val interface{}
	for mdl.Next(&key, &val) {
		v, err := val.(std.Value).String()
		if nil != err {
			t.Errorf("expected nil, received error: '%v'", err)
		}
		if key != v {
			t.Errorf("expected '%v', received '%v'", v, key)
		}
	}
}
