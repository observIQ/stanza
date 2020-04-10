package entry

type FieldSelector interface {
	Get(Record) (interface{}, bool)
	// Set(Record, interface{}) interface{}
	// Merge(Record, map[string]interface{})
}

type SingleFieldSelector []string

func (s SingleFieldSelector) Get(record Record) (interface{}, bool) {
	var next interface{} = record
	for _, str := range s {
		mapNext, ok := next.(map[string]interface{})
		if !ok {
			// The current level is not a map,
			return nil, false
		}

		next, ok = mapNext[str]
		if !ok {
			return nil, false
		}
	}

	return next, true
}

// func (s SingleFieldSelector) Set(record Record, val interface{}) interface{} {
// 	var current interface{} = record
// 	for i, str := range s {
// 		if i == len(str)-1 {
// 			// last string in selector
// 			switch c := current.(type) {
// 			case map[string]interface{}:
// 				c[str] = val
// 			default:
// 				current = map[string]interface{}{
// 					str: val,
// 				}
// 			}

// 		}
// 	}

// 	old := record
// 	record = val
// 	return old
// }

// func (s SingleFieldSelector) Merge(record Record, val map[string]interface{}) {
// 	var next map[string]interface{} = record
// 	for i, str := range s {
// 		if i == len(s) {
// 			for k, v := range val {
// 				next[k] = v
// 			}
// 		} else {
// 			nextInterface, ok := next[str]
// 			if !ok {
// 				newMap := make(map[string]interface{})
// 				next[str] = newMap
// 				next = newMap
// 			} else {
// 				next, ok = nextInterface.(map[string]interface{})
// 				if !ok {
// 					newMap := make(map[string]interface{})
// 					next[str] = newMap
// 					next = newMap
// 				}
// 			}
// 		}
// 	}

// 	for k, v := range val {
// 		next[k] = v
// 	}
// }
