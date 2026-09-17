package service

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestResumableUploadPreservesSessionSignature(t *testing.T) {
	t.Chdir(t.TempDir())
	const signature = "sig=test%2Bsignature%2Fvalue%3D"
	const content = "sample image bytes"
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		response.Header().Set("Content-Type", "application/json")
		if request.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", request.Method)
		}
		switch request.URL.Path {
		case "/v26.0/test-app/uploads":
			if request.URL.Query().Get("file_name") != "sample.jpg" || request.URL.Query().Get("file_type") != "image/jpeg" || request.URL.Query().Get("file_length") != "18" {
				t.Errorf("unexpected session parameters: %v", request.URL.Query())
			}
			_ = json.NewEncoder(response).Encode(WhatsAppUploadSessionResponse{ID: "upload:test-session?" + signature})
		case "/v26.0/upload:test-session":
			if request.URL.RawQuery != signature {
				t.Errorf("query = %q, want %q", request.URL.RawQuery, signature)
			}
			if request.Header.Get("Authorization") != "OAuth test-token" || request.Header.Get("file_offset") != "0" || request.Header.Get("Content-Type") != "image/jpeg" {
				t.Error("unexpected upload headers")
			}
			body, err := io.ReadAll(request.Body)
			if err != nil || string(body) != content {
				t.Errorf("upload body = %q, error = %v", body, err)
			}
			_ = json.NewEncoder(response).Encode(WhatsAppTemplateHeaderSampleUploadResponse{Handle: "test-handle"})
		default:
			t.Errorf("unexpected upload path: %s", request.URL.Path)
			http.Error(response, "unexpected path", http.StatusBadRequest)
		}
	}))
	defer server.Close()
	whatsapp := &Whatsapp{baseURL: server.URL, apiVersion: "v26.0", appID: "test-app", logger: NewLogger()}
	handle, err := whatsapp.UploadTemplateHeaderSample(context.Background(), "sample.jpg", "image/jpeg", []byte(content), "test-token")
	if err != nil {
		t.Fatal(err)
	}
	if handle != "test-handle" {
		t.Errorf("handle = %q, want test-handle", handle)
	}
}
