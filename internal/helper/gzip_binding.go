package helper

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/gin-gonic/gin/binding"
)

// MaxJSONBodySizeBytes caps any request JSON body (before or after gzip decompression) to guard against memory-exhaustion/decompression-bomb DoS.
const MaxJSONBodySizeBytes = 5 * 1024 * 1024

type GzipJSONBinding struct {
}

func (GzipJSONBinding) Name() string {
	return "gzipjson"
}

func (GzipJSONBinding) Bind(req *http.Request, obj any) error {
	if req == nil || req.Body == nil {
		return errors.New("invalid request")
	}
	r, err := gzip.NewReader(req.Body)
	if err != nil {
		return err
	}
	raw, err := readLimited(r)
	if err != nil {
		return err
	}
	return json.Unmarshal(raw, obj)
}

func (GzipJSONBinding) BindBody(body []byte, obj any) error {
	r, err := gzip.NewReader(bytes.NewReader(body))
	if err != nil {
		return err
	}
	raw, err := readLimited(r)
	if err != nil {
		return err
	}
	return decodeJSON(bytes.NewReader(raw), obj)
}

// readLimited reads at most MaxJSONBodySizeBytes+1 so oversized decompressed payloads are rejected rather than fully buffered.
func readLimited(r io.Reader) ([]byte, error) {
	raw, err := io.ReadAll(io.LimitReader(r, MaxJSONBodySizeBytes+1))
	if err != nil {
		return nil, err
	}
	if len(raw) > MaxJSONBodySizeBytes {
		return nil, errors.New("decompressed body too large")
	}
	return raw, nil
}

func decodeJSON(r io.Reader, obj any) error {
	decoder := json.NewDecoder(r)
	if err := decoder.Decode(obj); err != nil {
		return err
	}
	return validate(obj)
}

func validate(obj any) error {
	if binding.Validator == nil {
		return nil
	}
	return binding.Validator.ValidateStruct(obj)
}
