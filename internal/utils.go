package internal

import (
	"encoding/json"
	"regexp"
)

// Message type
type Message struct {
	Message string `json:"message"`
	Type    string `json:"type"`
}

// DecodeMessage return a Message struct from json
func DecodeMessage(data []byte) *Message {
	var m Message

	err := json.Unmarshal(data, &m)
	if err != nil {
		return nil
	}

	if m.Message == "" && m.Type == "" {
		return nil
	}
	return &m
}

// RemoveHTTPPrefix removes http:// or https:// in string
func RemoveHTTPPrefix(url string) string {
	r, _ := regexp.Compile("http.?://")
	return r.ReplaceAllString(url, "")
}
