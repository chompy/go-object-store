package store

import (
	"math/rand"
	"testing"

	"github.com/pkg/errors"
	"gitlab.com/contextualcode/go-object-store/types"
)

func TestGetSet(t *testing.T) {
	client := NewClient(nil)
	o := &types.Object{
		Data: map[string]interface{}{
			"test":  "hello world",
			"test2": 123,
		},
	}
	if err := client.Set(o, nil); err != nil {
		t.Error(err)
		return
	}
	storedObj, err := client.Get(o.UID, nil)
	if err != nil {
		t.Error(err)
		return
	}
	if storedObj.UID != o.UID || storedObj.Data["test"] != o.Data["test"] || int(storedObj.Data["test2"].(float64)) != o.Data["test2"] {
		t.Error("stored object does not match")
		return
	}
}

func TestDelete(t *testing.T) {
	client := NewClient(nil)
	o := &types.Object{
		Data: map[string]interface{}{
			"test": "hello world",
		},
	}
	if err := client.Set(o, nil); err != nil {
		t.Error(err)
		return
	}
	if err := client.Delete(o, nil); err != nil {
		t.Error(err)
		return
	}
	_, err := client.Get(o.UID, nil)
	if !errors.Is(err, ErrNotFound) {
		t.Error("expected not found error")
		return
	}
}

func TestIndexSet(t *testing.T) {
	client := NewClient(nil)
	o := &types.Object{
		Data: map[string]interface{}{
			"test":      "hello world",
			"test_long": "",
		},
	}
	for i := 0; i < 256; i++ {
		o.Data["test_long"] = o.Data["test_long"].(string) + "a"
	}
	if err := client.Set(o, nil); err != nil {
		t.Error(err)
		return
	}
	index, err := client.Index()
	if err != nil {
		t.Error(err)
		return
	}
	if index[0].UID != o.UID || index[0].Data["test"] != o.Data["test"] {
		t.Error("indexed object does not match")
	}
	if len(index[0].Data["test_long"].(string)) > types.IndexValueMaxSize {
		t.Error("unexpected long string indexed")
	}
}

func TestQuery(t *testing.T) {
	client := NewClient(nil)
	o := &types.Object{
		Data: map[string]interface{}{
			"test_int":    123,
			"test_float":  123.4,
			"test_bool":   false,
			"test_string": "hello world",
		},
	}
	if err := client.Set(o, nil); err != nil {
		t.Error(err)
		return
	}
	if err := client.Set(o, nil); err != nil {
		t.Error(err)
		return
	}
	res, err := client.Query("test_int = 123", nil)
	if err != nil {
		t.Error(err)
		return
	}
	if len(res) == 0 {
		t.Error("unexpected empty index")
	}
	if res[0].UID != o.UID {
		t.Error("unexpected item in index")
	}

	res, err = client.Query("test_int > 64 and test_int < 128", nil)
	if err != nil {
		t.Error(err)
		return
	}
	if len(res) == 0 {
		t.Error("unexpected empty index")
		return
	}
	if res[0].UID != o.UID {
		t.Error("unexpected item in index")
		return
	}

	res, err = client.Query("test_int > 123", nil)
	if err != nil {
		t.Error(err)
		return
	}
	if len(res) != 0 {
		t.Error("unexpected index")
		return
	}

	res, err = client.Query("test_string = 'hello world'", nil)
	if err != nil {
		t.Error(err)
		return
	}
	if len(res) == 0 {
		t.Error("unexpected empty index")
		return
	}
	if res[0].UID != o.UID {
		t.Error("unexpected item in index")
		return
	}

}

func TestQueryMulti(t *testing.T) {
	client := NewClient(nil)
	o1 := &types.Object{
		Data: map[string]interface{}{
			"test_str": "hello",
			"test_int": 1,
		},
	}
	client.Set(o1, nil)

	o2 := &types.Object{
		Data: map[string]interface{}{
			"test_str": "world",
			"test_int": 99,
		},
	}
	client.Set(o2, nil)

	o3 := &types.Object{
		Data: map[string]interface{}{
			"test_str":   "world",
			"test_float": 153.4,
		},
	}
	client.Set(o3, nil)

	res, err := client.Query("test_int >= 1", nil)
	if err != nil {
		t.Error(err)
		return
	}
	if len(res) != 2 {
		t.Error("unexpected query results")
		return
	}
}

func TestLargeIndex(t *testing.T) {
	client := NewClient(nil)
	// build very large index
	for i := 0; i < 4096; i++ {
		o := &types.Object{
			Data: map[string]interface{}{
				"test_int":    rand.Int(),
				"test_float":  rand.Float64(),
				"test_letter": string(byte(65 + (i % 24))),
			},
		}
		client.Set(o, nil)
	}
	index, _ := client.Index()
	if len(index) != 4096 {
		t.Error("unexpected index size")
	}
	res, _ := client.Query("test_int > 0", nil)
	if len(res) == 0 {
		t.Error("expected at least one result from query")
	}
	res, _ = client.Query("test_letter = 'A'", nil)
	if len(res) == 0 || len(res) == 4096 {
		t.Error("expected more than one result from query but less than 4096")
	}
}

func TestSyncIndex(t *testing.T) {
	client := NewClient(nil)
	o := &types.Object{
		Data: map[string]interface{}{
			"test_int":    123,
			"test_float":  123.4,
			"test_bool":   false,
			"test_string": "hello world",
		},
	}
	if err := client.Set(o, nil); err != nil {
		t.Error(err)
		return
	}

	// store object and sync index
	client.Set(o, nil)
	if err := client.Sync(); err != nil {
		t.Error(err)
		return
	}

	// update object without sync
	o.Data["test_string"] = "hello world two"
	client.Set(o, nil)

	// fetch remote index prior to sync to ensure
	// old value still remains
	remoteIndex := make([]*types.IndexObject, 0)
	client.getRaw(indexName, &remoteIndex)
	if len(remoteIndex) == 0 {
		t.Error("unexpected remote index length")
	}
	if remoteIndex[0].Data["test_string"] != "hello world" {
		t.Error("unexpected value in remote index")
	}

	// sync and ensure remote index is now updated
	client.Sync()
	remoteIndex = make([]*types.IndexObject, 0)
	client.getRaw(indexName, &remoteIndex)
	if remoteIndex[0].Data["test_string"] != "hello world two" {
		t.Error("unexpected value in remote index")
	}

}
