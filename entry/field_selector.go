package entry

type FieldSelector interface {
	Get(Record) (interface{}, bool)
	SetSafe(*Record, interface{}) bool
	Set(*Record, interface{})
	// Merge(Record, map[string]interface{})
}

// TODO support arrays?
type SingleFieldSelector []string

func (s SingleFieldSelector) Get(record Record) (interface{}, bool) {
	var current interface{} = record
	for _, str := range s {
		mapNext, ok := current.(map[string]interface{})
		if !ok {
			// The current level is not a map,
			return nil, false
		}

		current, ok = mapNext[str]
		if !ok {
			// The current level's key does not exist
			return nil, false
		}
	}

	return current, true
}

// SetSafe sets a key without overwriting any other records. It returns
// whether the key was set
func (s SingleFieldSelector) SetSafe(record *Record, val interface{}) bool {
	if record == nil {
		return false
	}

	if len(s) == 0 {
		if *record != Record(nil) {
			// don't overwrite record if it exists
			return false
		}
		*record = Record(val)
		return true
	}

	var current interface{} = *record
	for i, str := range s {
		c, ok := current.(map[string]interface{})
		if !ok {
			return false
		}

		if i == len(s)-1 {
			c[str] = val
			return true
		}

		current, ok = c[str]
		if !ok {
			return false
		}
	}

	// we should never get here
	return false

}

// Set sets a value, overwriting any intermediate values as necessary
func (s SingleFieldSelector) Set(record *Record, val interface{}) {
	if len(s) == 0 {
		*record = Record(val)
		return
	}

	var currentMap map[string]interface{}
	var ok bool
	currentMap, ok = (*record).(map[string]interface{})
	if !ok {
		currentMap = map[string]interface{}{}
		*record = currentMap
	}

	for i, str := range s {
		if i == len(s)-1 {
			currentMap[str] = val
			return
		}

		current, ok := currentMap[str]
		if !ok {
			current = map[string]interface{}{}
			currentMap[str] = current
		}

		next, ok := current.(map[string]interface{})
		if !ok {
			next = map[string]interface{}{}
			currentMap[str] = next
		}

		currentMap = next
	}

	return
}
