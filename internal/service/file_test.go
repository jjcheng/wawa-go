package service

import "testing"

func TestBlobNameFromURL(t *testing.T) {
	file := &File{bucketName: "test-bucket", endpoint: "https://oss.example.com"}
	for _, tc := range []struct {
		name string
		url  string
		key  string
	}{
		{"media", "https://test-bucket.oss.example.com/media/video.mp4", "media/video.mp4"},
		{"escaped key", "https://test-bucket.oss.example.com/media/my%20audio.mp3?signature=value", "media/my audio.mp3"},
		{"other bucket", "https://other-bucket.oss.example.com/media/video.mp4", ""},
		{"external host", "https://example.com/media/video.mp4", ""},
		{"missing key", "https://test-bucket.oss.example.com/", ""},
		{"relative path", "media/video.mp4", ""},
		{"credentials", "https://user@test-bucket.oss.example.com/media/video.mp4", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			key, err := file.blobNameFromURL(tc.url)
			if tc.key == "" {
				if err == nil {
					t.Fatal("expected an invalid file URL error")
				}
				return
			}
			if err != nil || key != tc.key {
				t.Fatalf("got key=%q err=%v, want %q", key, err, tc.key)
			}
		})
	}
}
