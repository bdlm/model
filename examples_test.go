package model_test

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/bdlm/log/v2"
	"github.com/bdlm/model"
	stdIterator "github.com/bdlm/std/v2/iterator"
	stdModel "github.com/bdlm/std/v2/model"
)

func ExampleNew() {
	mdl := model.New(stdModel.ModelTypeHash)
	json.Unmarshal(
		[]byte(`{"key1":"value1","key2":2,"key3":["one","two","three"],"key4":{"k1":"v1","k2":"v2"}}`),
		&mdl,
	)
	var key, val interface{}
	for mdl.Next(&key, &val) {
		if "key3" == key.(string) || "key4" == key.(string) {
			var k2, v2 interface{}
			var m2 stdModel.Model
			m2, _ = val.(stdModel.Value).Model()
			if nil == m2 {
				data, hash, index := mdl.Data()
				log.Debugf("\n\n\ndata: %v\nhash: %v\nindex: %v\n\n\n", data, hash, index)
				os.Exit(1)
			}

			fmt.Println(key)
			for m2.(stdIterator.Iterator).Next(&k2, &v2) {
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
