package helper

import (
	"bytes"
	"encoding/json"
	"errors"
	"strings"
)

func IsJSONString(s string) bool {
	var js map[string]any
	return json.Unmarshal([]byte(s), &js) == nil
}

func ParseJSONData(data []byte) (*map[string]any, bool) {
	var m map[string]any
	if error := json.Unmarshal(data, &m); error != nil {
		return nil, false
	}
	return &m, true
}

func IsJSONData(data []byte) bool {
	return json.Valid(data)
}

func SerializeJSON(item any) (*string, error) {
	if item == nil {
		return nil, errors.New("invalid input")
	}
	buffer := &bytes.Buffer{}
	encoder := json.NewEncoder(buffer)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(item); err != nil {
		return nil, err
	}
	// Remove trailing newline added by Encode
	jsonStr := buffer.String()
	jsonStr = strings.TrimSpace(jsonStr)
	return &jsonStr, nil
}

func DeserializeJSON[T any](string string) (*T, error) {
	var item T
	if err := json.Unmarshal([]byte(string), &item); err != nil {
		return nil, err
	}
	return &item, nil
}
