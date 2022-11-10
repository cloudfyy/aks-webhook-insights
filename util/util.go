package util

import (
	"encoding/json"
	"fmt"
	"net/http"
)

func LoadJSON(r *http.Request, v interface{}) error {
	err := json.NewDecoder(r.Body).Decode(v)
	if err != nil {
		return fmt.Errorf("invalid JSON input")
	}

	return nil
}
