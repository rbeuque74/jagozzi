package config

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
)

// UnmarshalConfig will decode a JSON structure while disallowing unknown fields, to prevent configuration file errors
func UnmarshalConfig(b []byte, obj interface{}) error {
	if ok := json.Valid(b); !ok {
		return fmt.Errorf("json: not valid configuration")
	}

	reader := bytes.NewReader(b)
	dec := json.NewDecoder(reader)
	dec.DisallowUnknownFields()

	if err := dec.Decode(obj); err != nil && err != io.EOF {
		return err
	}

	if more := dec.More(); more {
		return fmt.Errorf("json: not valid configuration")
	}
	return nil
}
