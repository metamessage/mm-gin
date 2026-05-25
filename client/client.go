package client

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"time"

	mm "github.com/metamessage/metamessage"
)

// Client HTTP 客戶端
// 請求體發送 JSONC 格式，響應始終接收 MetaMessage 二進制格式
type Client struct {
	baseURL    string
	httpClient *http.Client
}

// NewClient 創建客戶端
func NewClient(baseURL string) *Client {
	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func SetDefaultClient(baseURL string) {
	defaultClient = NewClient(baseURL)
}

var defaultClient = NewClient("")

// DoRequest 執行 HTTP 請求（泛型）
// 對於 POST/PUT/PATCH 請求，會先發送 OPTIONS 請求驗證 Schema，
// 確保請求結構體與服務端期望的格式匹配後再發送實際請求
// GET/DELETE/OPTIONS 等無請求體的方法，T 可指定為 any
func DoRequest[REQ any, RESP any](c *Client, method, path string, body REQ) (resp RESP, err error) {
	shouldEncode := method == "POST" || method == "PUT" || method == "PATCH"

	if shouldEncode {
		if _, err = DoRequest[any, REQ](c, "OPTIONS", path, nil); err != nil {
			err = fmt.Errorf("schema request failed: %w", err)
			return
		}
	}

	var reqBody io.Reader
	if shouldEncode {
		var data []byte
		data, err = mm.EncodeFromValue(body, "")
		if err != nil {
			err = fmt.Errorf("encode metamessage failed: %w", err)
			return
		}
		reqBody = bytes.NewReader(data)
	}

	req, err := http.NewRequest(method, c.baseURL+path, reqBody)
	if err != nil {
		err = fmt.Errorf("create request failed: %w", err)
		return
	}

	req.Header.Set("Accept", "application/x-metamessage")
	if shouldEncode {
		req.Header.Set("Content-Type", "application/x-metamessage")
	}

	r, err := c.httpClient.Do(req)
	if err != nil {
		err = fmt.Errorf("do request failed: %w", err)
		return
	}
	defer r.Body.Close()

	data, err := io.ReadAll(r.Body)
	if err != nil {
		err = fmt.Errorf("read response failed: %w", err)
		return
	}

	if err = mm.DecodeToValue(data, &resp); err != nil {
		jsonc, _ := mm.DecodeToJsonc(data)
		err = fmt.Errorf("decode metamessage failed: %s", jsonc)
		return
	}

	jsonc, err := mm.DecodeToJsonc(data)
	if err != nil {
		err = fmt.Errorf("decode metamessage failed: %w", err)
		return
	}

	fmt.Println("res:\n", jsonc)
	return
}

func GET[REQ any, RESP any](path string, body REQ) (resp RESP, err error) {
	return DoRequest[REQ, RESP](defaultClient, "GET", path, body)
}

func POST[REQ any, RESP any](path string, body REQ) (resp RESP, err error) {
	return DoRequest[REQ, RESP](defaultClient, "POST", path, body)
}

func PUT[REQ any, RESP any](path string, body REQ) (resp RESP, err error) {
	return DoRequest[REQ, RESP](defaultClient, "PUT", path, body)
}

func DELETE[REQ any, RESP any](path string, body REQ) (resp RESP, err error) {
	return DoRequest[REQ, RESP](defaultClient, "DELETE", path, body)
}

func PATCH[REQ any, RESP any](path string, body REQ) (resp RESP, err error) {
	return DoRequest[REQ, RESP](defaultClient, "PATCH", path, body)
}

// func OPTIONS[REQ any, RESP any](path string, body REQ) (resp RESP, err error) {
// 	return DoRequest[REQ, RESP](defaultClient, "OPTIONS", path, body)
// }

// func HEAD[REQ any, RESP any](path string, body REQ) (resp RESP, err error) {
// 	return DoRequest[REQ, RESP](defaultClient, "HEAD", path, body)
// }

// func TRACE[REQ any, RESP any](path string, body REQ) (resp RESP, err error) {
// 	return DoRequest[REQ, RESP](defaultClient, "TRACE", path, body)
// }
