package plugin

import (
	"encoding/json"
	"fmt"
	"time"
)

type Duration struct {
	time.Duration
}

func (d *Duration) Raw() time.Duration {
	return d.Duration
}

func (d Duration) MarshalJSON() ([]byte, error) {
	return []byte(`"` + d.Duration.String() + `"`), nil
}

func (d *Duration) UnmarshalJSON(raw []byte) error {
	var v interface{}
	err := json.Unmarshal(raw, &v)
	if err != nil {
		return err
	}
	d.Duration, err = durationFromInterface(v)
	return err
}

func (d Duration) MarshalYAML() (interface{}, error) {
	return d.Duration.String(), nil
}

func (d *Duration) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var v interface{}
	err := unmarshal(&v)
	if err != nil {
		return err
	}
	d.Duration, err = durationFromInterface(v)
	if d.Duration < 0 {
		d.Duration *= -1
	}
	return err
}

func durationFromInterface(val interface{}) (time.Duration, error) {
	switch value := val.(type) {
	case float64:
		return time.Duration(value * float64(time.Second)), nil
	case int:
		return time.Duration(value) * time.Second, nil
	case string:
		var err error
		d, err := time.ParseDuration(value)
		return d, err
	default:
		return 0, fmt.Errorf("cannot unmarshal value of type %T into a duration", val)
	}
}
