package narrative

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type Client struct {
	baseURL string
	http    *http.Client
}

func NewClient(baseURL string) *Client {
	return &Client{
		baseURL: baseURL,
		http: &http.Client{
			Timeout: 30 * time.Second,
			Transport: &http.Transport{
				ResponseHeaderTimeout: 10 * time.Second,
				IdleConnTimeout:       90 * time.Second,
			},
		},
	}
}

func (c *Client) AnalyzeScene(ctx context.Context, req AnalysisRequest) (*SceneAnalysis, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("narrative marshal: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/api/v1/analysis/scene", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("narrative new request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	res, err := c.http.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("narrative request: %w", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(res.Body)
		return nil, fmt.Errorf("narrative api status %d: %s", res.StatusCode, string(respBody))
	}

	var analysis SceneAnalysis
	if err := json.NewDecoder(res.Body).Decode(&analysis); err != nil {
		return nil, fmt.Errorf("narrative decode: %w", err)
	}
	return &analysis, nil
}

func (c *Client) GetSceneAnalysis(ctx context.Context, sceneID string) (*SceneAnalysis, error) {
	httpReq, err := http.NewRequestWithContext(ctx, "GET", c.baseURL+"/api/v1/analysis/scene/"+sceneID, nil)
	if err != nil {
		return nil, fmt.Errorf("narrative new request: %w", err)
	}

	res, err := c.http.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("narrative request: %w", err)
	}
	defer res.Body.Close()

	if res.StatusCode == http.StatusNotFound {
		return nil, nil
	}
	if res.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(res.Body)
		return nil, fmt.Errorf("narrative api status %d: %s", res.StatusCode, string(respBody))
	}

	var analysis SceneAnalysis
	if err := json.NewDecoder(res.Body).Decode(&analysis); err != nil {
		return nil, fmt.Errorf("narrative decode: %w", err)
	}
	return &analysis, nil
}
