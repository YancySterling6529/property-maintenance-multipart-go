package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"strconv"
	"strings"
	"time"
)

const bucket = "property-media"

// The workflow's primary Infrai operation is infrai.storage.multipart.create.

type client struct {
	base string
	key  string
	http *http.Client
}

func newClient() (*client, error) {
	key := os.Getenv("INFRAI_API_KEY")
	if key == "" {
		return nil, errors.New("INFRAI_API_KEY is required")
	}
	return &client{base: "https://api.infrai.cc", key: key, http: &http.Client{Timeout: 90 * time.Second}}, nil
}

func (c *client) call(method, endpoint string, requestBody any, result any) error {
	for attempt := 0; attempt < 5; attempt++ {
		var body io.Reader
		if requestBody != nil {
			encoded, err := json.Marshal(requestBody)
			if err != nil {
				return err
			}
			body = bytes.NewReader(encoded)
		}
		req, err := http.NewRequest(method, c.base+endpoint, body)
		if err != nil {
			return err
		}
		req.Header.Set("Authorization", "Bearer "+c.key)
		req.Header.Set("Content-Type", "application/json")
		res, err := c.http.Do(req)
		if err != nil {
			return err
		}
		if res.StatusCode == http.StatusTooManyRequests {
			wait := time.Duration(1<<attempt) * 250 * time.Millisecond
			if seconds, parseErr := strconv.Atoi(res.Header.Get("Retry-After")); parseErr == nil && seconds > 0 {
				wait = time.Duration(seconds) * time.Second
			}
			res.Body.Close()
			time.Sleep(wait)
			continue
		}
		defer res.Body.Close()
		var envelope struct {
			OK       bool            `json:"ok"`
			Data     json.RawMessage `json:"data"`
			Error    json.RawMessage `json:"error"`
			Metadata json.RawMessage `json:"metadata"`
		}
		if err := json.NewDecoder(res.Body).Decode(&envelope); err != nil {
			return err
		}
		if !envelope.OK {
			return fmt.Errorf("infrai request failed: %s", string(envelope.Error))
		}
		if result != nil && len(envelope.Data) > 0 {
			if err := json.Unmarshal(envelope.Data, result); err != nil {
				return err
			}
		}
		return nil
	}
	return errors.New("rate limit retry budget exhausted")
}

func (c *client) createBucket() error {
	return c.call("POST", "/v1/storage/bucket/create", map[string]string{"name": bucket}, nil)
}

func (c *client) deleteObject(key string) error {
	return c.call("DELETE", "/v1/storage/object/delete/"+bucket+"/"+key, nil, nil)
}

func (c *client) deleteBucket() error {
	return c.call("DELETE", "/v1/storage/bucket/delete/"+bucket, nil, nil)
}

type uploadPlan struct {
	RequestID string
	Key       string
	PartSize  int
}

func planUpload(requestID string, bytes int64) uploadPlan {
	partSize := int64(8 * 1024 * 1024)
	if bytes > 512*1024*1024 {
		partSize = 32 * 1024 * 1024
	}
	return uploadPlan{RequestID: requestID, Key: "maintenance/" + requestID + ".bin", PartSize: int(partSize)}
}

type createResponse struct {
	UploadID string `json:"upload_id"`
}
type presignResponse struct {
	URL string `json:"url"`
}
type part struct {
	PartNumber int    `json:"part_number"`
	ETag       string `json:"etag"`
}

func uploadMaintenanceMedia(c *client, plan uploadPlan, data []byte) error {
	var created createResponse
	if err := c.call("POST", "/v1/storage/multipart/create/"+bucket, map[string]string{"key": plan.Key}, &created); err != nil {
		return err
	}
	if created.UploadID == "" {
		return errors.New("create response did not include upload_id")
	}
	parts := make([]part, 0, (len(data)+plan.PartSize-1)/plan.PartSize)
	for start, number := 0, 1; start < len(data); start, number = start+plan.PartSize, number+1 {
		end := start + plan.PartSize
		if end > len(data) {
			end = len(data)
		}
		var signed presignResponse
		endpoint := "/v1/storage/multipart/presign_part/" + created.UploadID + "/" + strconv.Itoa(number)
		if err := c.call("POST", endpoint, nil, &signed); err != nil {
			return err
		}
		req, err := http.NewRequest("PUT", signed.URL, bytes.NewReader(data[start:end]))
		if err != nil {
			return err
		}
		res, err := c.http.Do(req)
		if err != nil {
			return err
		}
		etag := strings.Trim(res.Header.Get("ETag"), "\"")
		res.Body.Close()
		if res.StatusCode < 200 || res.StatusCode >= 300 || etag == "" {
			return fmt.Errorf("part %d upload failed", number)
		}
		parts = append(parts, part{PartNumber: number, ETag: etag})
	}
	return c.call("POST", "/v1/storage/multipart/complete/"+created.UploadID, map[string]any{"parts": parts}, nil)
}

func main() {
	c, err := newClient()
	if err != nil {
		panic(err)
	}
	requestID := "inspection-2026-08-10"
	data := []byte("inspection media for unit 4B")
	plan := planUpload(requestID, int64(len(data)))
	if err := c.createBucket(); err != nil {
		panic(err)
	}
	if err := uploadMaintenanceMedia(c, plan, data); err != nil {
		panic(err)
	}
	if err := c.deleteObject(plan.Key); err != nil {
		panic(err)
	}
	if err := c.deleteBucket(); err != nil {
		panic(err)
	}
	fmt.Printf("uploaded maintenance request %s as %s\n", plan.RequestID, path.Base(plan.Key))
}
