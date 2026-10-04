package tests

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

type simpleExecuteRequest struct {
	Language  string   `json:"language"`
	Code      string   `json:"code"`
	Timeout   uint     `json:"timeout,omitempty"`
	MaxMemory uint     `json:"max_memory,omitempty"`
	Inputs    []string `json:"inputs,omitempty"`
}

type simpleExecuteResponse struct {
	Output     string `json:"output"`
	Error      string `json:"error"`
	MemoryUsed string `json:"memory_used"`
	CPUTime    string `json:"cpu_time"`
}

type integrationHarness struct {
	baseURL string
	apiPort int
}

func newHarness(t *testing.T) integrationHarness {
	t.Helper()
	return integrationHarness{
		baseURL: testServer.URL,
		apiPort: mustExtractPort(t, testServer.URL),
	}
}

func buildLanguageRequest(language, code string, timeoutSec, maxMemoryKB uint) simpleExecuteRequest {
	return simpleExecuteRequest{Language: language, Code: code, Timeout: timeoutSec, MaxMemory: maxMemoryKB}
}

func buildCRequest(code string, timeoutSec, maxMemoryKB uint) simpleExecuteRequest {
	return buildLanguageRequest("c", code, timeoutSec, maxMemoryKB)
}

func callSimpleExecute(t *testing.T, baseURL string, req simpleExecuteRequest) simpleExecuteResponse {
	t.Helper()
	resp, err := callSimpleExecuteRaw(baseURL, req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	return resp
}

func callSimpleExecuteRaw(baseURL string, req simpleExecuteRequest) (simpleExecuteResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return simpleExecuteResponse{}, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequest(http.MethodPost, baseURL+"/execute", bytes.NewReader(body))
	if err != nil {
		return simpleExecuteResponse{}, fmt.Errorf("failed to create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	httpResp, err := httpClient.Do(httpReq)
	if err != nil {
		return simpleExecuteResponse{}, fmt.Errorf("request failed: %w", err)
	}
	defer httpResp.Body.Close()
	return decodeSimpleResponseRaw(httpResp)
}

func decodeSimpleResponseRaw(httpResp *http.Response) (simpleExecuteResponse, error) {
	rawBody, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return simpleExecuteResponse{}, fmt.Errorf("failed to read response body: %w", err)
	}
	if httpResp.StatusCode != http.StatusOK {
		return simpleExecuteResponse{}, fmt.Errorf("unexpected HTTP status %d: %s", httpResp.StatusCode, strings.TrimSpace(string(rawBody)))
	}

	var parsed simpleExecuteResponse
	if err := json.Unmarshal(rawBody, &parsed); err != nil {
		return simpleExecuteResponse{}, fmt.Errorf("failed to decode response JSON: %w; body=%q", err, strings.TrimSpace(string(rawBody)))
	}
	return parsed, nil
}

func mustExtractPort(t *testing.T, serverURL string) int {
	t.Helper()
	hostPort := strings.TrimPrefix(serverURL, "http://")
	_, portStr, err := net.SplitHostPort(hostPort)
	if err != nil {
		t.Fatalf("failed to parse test server URL %q: %v", serverURL, err)
	}
	port, err := strconv.Atoi(portStr)
	if err != nil {
		t.Fatalf("failed to parse server port %q: %v", portStr, err)
	}
	return port
}

// mustLoadSampleCode reads a fixture from tests/testdata/sample_code and
// applies optional placeholder replacements. relPath is relative to the
// fixture root, for example "hello_world/main.c".
func mustLoadSampleCode(t *testing.T, relPath string, replacements map[string]string) string {
	t.Helper()
	path := filepath.Join(sampleCodeDir, relPath)
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read sample code %s: %v", path, err)
	}

	code := string(content)
	for oldValue, newValue := range replacements {
		code = strings.ReplaceAll(code, oldValue, newValue)
	}
	return code
}
