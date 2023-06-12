package utils

import (
	"bytes"
	"io"
	"net/http"
	"time"
)

func HttpPost(url string, data []byte, contentType string) ([]byte, error) {
	return HttpDo("POST", url, data, contentType, 30*time.Second)
}

func HttpDo(method string, url string, data []byte, contentType string, timeout time.Duration) ([]byte, error) {
	req, err := http.NewRequest(method, url, bytes.NewBuffer(data))
	req.Header.Set("Content-Type", contentType)

	client := &http.Client{Timeout: timeout}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()
	return io.ReadAll(resp.Body)
}
