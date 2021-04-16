package common

import (
	"encoding/json"
	"io"
)

// MapFromJson decodes the key/value pair map
func MapFromJson(data io.Reader) map[string]string {
	decoder := json.NewDecoder(data)

	var res map[string]string
	if err := decoder.Decode(&res); err != nil {
		return make(map[string]string)
	}
	return res
}
