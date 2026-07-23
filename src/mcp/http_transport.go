package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

type HttpTransport struct {
	url     string
	client  *http.Client
	headers map[string]string
}

func NewHttpTransport(config *MCPServerConfig) (*HttpTransport, error) {

	return &HttpTransport{
		url:     config.URL,
		client:  &http.Client{},
		headers: config.Headers,
	}, nil
}

func (h *HttpTransport) Send(ctx context.Context, req *Request) (*Response, error) {
	data, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, h.url, bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	for k, v := range h.headers {
		httpReq.Header.Set(k, v)
	}
	httpResp, err := h.client.Do(httpReq)
	if err != nil {
		fmt.Printf("do request error: %v, url: %s\n", err, h.url)

		return nil, err
	}
	defer httpResp.Body.Close()
	var resp Response
	if err := json.NewDecoder(httpResp.Body).Decode(&resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (h *HttpTransport) Notify(ctx context.Context, req *Request) error {
	data, err := json.Marshal(req)
	if err != nil {
		return err
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, h.url, bytes.NewReader(data))
	if err != nil {
		return err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	for k, v := range h.headers {
		httpReq.Header.Set(k, v)
	}
	httpResp, err := h.client.Do(httpReq)
	if err != nil {
		return err
	}
	defer httpResp.Body.Close()
	return nil
}

func (h *HttpTransport) Close() error {
	return nil
}
