package repository

import (
	"encoding/json"
)

// marshalJSON marshals a value to JSON bytes, handling nil values
func marshalJSON(v interface{}) ([]byte, error) {
	if v == nil {
		return []byte("{}"), nil
	}
	return json.Marshal(v)
}

// unmarshalJSON unmarshals JSON bytes to a value, handling empty/nil bytes
func unmarshalJSON(data []byte, v interface{}) error {
	if len(data) == 0 {
		return nil
	}
	return json.Unmarshal(data, v)
}