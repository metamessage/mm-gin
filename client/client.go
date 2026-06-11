package client

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"slices"
	"time"

	"github.com/metamessage/mm-web-go/web"

	mm "github.com/metamessage/metamessage"
)

// Client is an HTTP client for MetaMessage protocol communication.
// Request bodies are encoded in MetaMessage binary format;
// responses are always expected in MetaMessage format.
type Client struct {
	baseURL    string
	httpClient *http.Client
	debug      bool
}

// NewClient creates a new Client with the given base URL.
func NewClient(baseURL string, debug bool) *Client {
	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		debug: debug,
	}
}

// SetDefaultClient sets the global default client with the given base URL.
func SetDefaultClient(baseURL string, debug bool) {
	defaultClient = NewClient(baseURL, debug)
}

// defaultClient is the global default client used by package-level functions.
var defaultClient = NewClient("", false)

// DoRequest executes an HTTP request with generic request/response types.
// For POST/PUT/PATCH requests, it first sends an OPTIONS preflight to validate
// the request schema against the server, ensuring the request struct matches
// the expected format before sending the actual request.
// For GET/DELETE/OPTIONS (bodyless methods), specify T as any.
func DoRequest[REQ any, RESP any](c *Client, method, path string, body *REQ) (resp RESP, err error) {
	var reqBody io.Reader
	var query string
	var contentType string
	if slices.Contains([]string{http.MethodGet, http.MethodDelete}, method) {
		if _, err = DoRequest[any, REQ](c, http.MethodOptions, path, nil); err != nil {
			err = fmt.Errorf("schema request failed: %w", err)
			return
		}

		// if body != nil {
		// 	query, err = mm.ValueToJsonc(*body, "")
		// 	if err != nil {
		// 		err = fmt.Errorf("encode metamessage failed: %w", err)
		// 		return
		// 	}
		// }
		// contentType = web.ContentTypeJSONC

		if body != nil {
			var data []byte
			data, err = mm.EncodeFromValue(*body, "")
			if err != nil {
				err = fmt.Errorf("encode metamessage failed: %w", err)
				return
			}
			query = hex.EncodeToString(data)
		}
		contentType = web.ContentTypeMetaMessage
	} else if slices.Contains([]string{http.MethodPost, http.MethodPut, http.MethodPatch}, method) {
		if _, err = DoRequest[any, REQ](c, http.MethodOptions, path, nil); err != nil {
			err = fmt.Errorf("schema request failed: %w", err)
			return
		}

		if body != nil {
			var data []byte
			data, err = mm.EncodeFromValue(*body, "")
			if err != nil {
				err = fmt.Errorf("encode metamessage failed: %w", err)
				return
			}
			reqBody = bytes.NewReader(data)
		}
		contentType = web.ContentTypeMetaMessage
	}

	url := c.baseURL + path
	if query != "" {
		url = fmt.Sprintf("%s?data=%s", url, query)
	}

	req, err := http.NewRequest(method, url, reqBody)
	if err != nil {
		err = fmt.Errorf("create request failed: %w", err)
		return
	}

	if contentType != "" {
		req.Header.Set("Accept", web.ContentTypeMetaMessage)
		req.Header.Set("Content-Type", contentType)
	}

	r, err := c.httpClient.Do(req)
	if err != nil {
		err = fmt.Errorf("do request failed: %w", err)
		return
	}
	defer r.Body.Close()

	if r.StatusCode != 200 {
		err = fmt.Errorf("do request failed: %d %s %s", r.StatusCode, r.Request.Method, r.Request.URL.String())
		return
	}

	data, err := io.ReadAll(r.Body)
	if err != nil {
		err = fmt.Errorf("read response failed: %w", err)
		return
	}

	if err = mm.DecodeToValue(data, &resp); err != nil {
		jsonc, _ := mm.DecodeToJsonc(data)
		err = fmt.Errorf("decode metamessage failed: %s: %s", err, jsonc)
		return
	}

	if c.debug {
		jsonc, _ := mm.DecodeToJsonc(data)
		fmt.Printf("%s %s:\n%s\n", method, path, jsonc)
	}
	return
}

// GET sends a GET request using the default client.
func GET[REQ any, RESP any](path string, body *REQ) (resp RESP, err error) {
	return DoRequest[REQ, RESP](defaultClient, http.MethodGet, path, body)
}

// POST sends a POST request using the default client.
func POST[REQ any, RESP any](path string, body *REQ) (resp RESP, err error) {
	return DoRequest[REQ, RESP](defaultClient, http.MethodPost, path, body)
}

// PUT sends a PUT request using the default client.
func PUT[REQ any, RESP any](path string, body *REQ) (resp RESP, err error) {
	return DoRequest[REQ, RESP](defaultClient, http.MethodPut, path, body)
}

// DELETE sends a DELETE request using the default client.
func DELETE[REQ any, RESP any](path string, body *REQ) (resp RESP, err error) {
	return DoRequest[REQ, RESP](defaultClient, http.MethodDelete, path, body)
}

// PATCH sends a PATCH request using the default client.
func PATCH[REQ any, RESP any](path string, body *REQ) (resp RESP, err error) {
	return DoRequest[REQ, RESP](defaultClient, http.MethodPatch, path, body)
}

// func OPTIONS[REQ any, RESP any](path string, body *REQ) (resp RESP, err error) {
// 	return DoRequest[*REQ, RESP](defaultClient, "OPTIONS", path, body)
// }

// func HEAD[REQ any, RESP any](path string, body *REQ) (resp RESP, err error) {
// 	return DoRequest[*REQ, RESP](defaultClient, "HEAD", path, body)
// }

// func TRACE[REQ any, RESP any](path string, body *REQ) (resp RESP, err error) {
// 	return DoRequest[*REQ, RESP](defaultClient, "TRACE", path, body)
// }
