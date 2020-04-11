package helper

type FieldSelector interface {

	// Get returns the value of the field and whether the field exists
	Get(map[string]interface{}) (interface{}, bool)

	// Set overwrites the value of the field and returns the previous
	// value of field before it was replaced. The boolean returned
	// indicates whether the field existed before it was overwritten.
	Set(map[string]interface{}, interface{}) (interface{}, bool)

	// Delete removes the field from the record,
	Delete(map[string]interface{}) (interface{}, bool)
}
