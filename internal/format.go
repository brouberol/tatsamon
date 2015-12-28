package internal

import (
	"encoding/json"
	"fmt"
)

// FormatOutputErrror return the "message" field of an API return
func FormatOutputErrror(data []byte) error {
	var errorDesc map[string]interface{}
	if err := json.Unmarshal(data, &errorDesc); err != nil {
		// sometimes, the API returns a string instead of a
		// JSON-object for the error. Let's fallback on that
		s := ""
		err := json.Unmarshal(data, &s)
		if err != nil {
			return err
		}
		errorDesc = map[string]interface{}{"message": s}
	}

	message := errorDesc["message"]
	if message == nil {
		message = errorDesc["error_details"]
	}

	if message != nil {
		return fmt.Errorf("Error: %s", message)
	}
	return fmt.Errorf("Error: %s", data)
}
