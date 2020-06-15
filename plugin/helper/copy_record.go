package helper

import "encoding/json"

func CopyRecord(r interface{}) interface{} {
	switch r := r.(type) {
	case map[string]interface{}:
		return copyMap(r)
	case string:
		return r
	case []byte:
		new := make([]byte, 0, len(r))
		copy(new, r)
		return new
	default:
		// fall back to JSON roundtrip
		var i interface{}
		b, err := json.Marshal(r)
		if err != nil {
			panic(err)
		}
		err = json.Unmarshal(b, &i)
		if err != nil {
			panic(err)
		}
		return i
	}
}

// Should this do something different with pointers or arrays?
func copyMap(m map[string]interface{}) map[string]interface{} {
	cp := make(map[string]interface{})
	for k, v := range m {
		vm, ok := v.(map[string]interface{})
		if ok {
			cp[k] = copyMap(vm)
		} else {
			cp[k] = v
		}
	}

	return cp
}
