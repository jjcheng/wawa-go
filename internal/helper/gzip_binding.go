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
	raw, err := io.ReadAll(r)
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
	return decodeJSON(r, obj)
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
