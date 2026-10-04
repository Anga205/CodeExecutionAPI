package tests

import (
	"CodeSandboxAPI/config"
	"CodeSandboxAPI/resourcemanager"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"CodeSandboxAPI/routes"
)

type concurrentSimpleExecuteResult struct {
	response simpleExecuteResponse
	elapsed  time.Duration
	err      error
}

func TestOverloadedRejectsHalfRequestsUnderLoad(t *testing.T) {
	baseURL := startOverloadLoadTestServer(t)
	req := buildLanguageRequest("python", "import time;time.sleep(2)", 5, 131072)
	requestCount := 10

	results := runConcurrentSimpleExecuteRequests(baseURL, req, requestCount)

	overloaded := 0
	for i, result := range results {
		if result.err != nil {
			t.Fatalf("request %d failed unexpectedly: %v", i, result.err)
		}
		if result.response.Error == "Server is overloaded, please try again later" {
			overloaded++
		}
	}

	maxConcurrent := int(config.Config.Globals.RAM_LIMIT / req.MaxMemory)
	if maxConcurrent > requestCount {
		maxConcurrent = requestCount
	}
	expectedOverloaded := requestCount - maxConcurrent

	if overloaded != expectedOverloaded {
		t.Fatalf("expected exactly %d overloaded responses, got %d", expectedOverloaded, overloaded)
	}
}

func startOverloadLoadTestServer(t *testing.T) string {
	t.Helper()

	t.Setenv("GLOBAL_RAM_LIMIT", "524288")

	originalRAMLimit := config.Config.Globals.RAM_LIMIT

	config.Config.Globals.RAM_LIMIT = 524288
	resourcemanager.ResetRAMForTests(config.Config.Globals.RAM_LIMIT)

	t.Cleanup(func() {
		config.Config.Globals.RAM_LIMIT = originalRAMLimit
		resourcemanager.ResetRAMForTests(config.Config.Globals.RAM_LIMIT)
	})

	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	routes.Setup(router)
	server := httptest.NewServer(router)
	t.Cleanup(server.Close)

	return server.URL
}

func runConcurrentSimpleExecuteRequests(baseURL string, req simpleExecuteRequest, count int) []concurrentSimpleExecuteResult {
	results := make([]concurrentSimpleExecuteResult, count)

	start := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(count)

	for i := 0; i < count; i++ {
		go func(index int) {
			defer wg.Done()
			<-start

			startedAt := time.Now()
			resp, err := callSimpleExecuteRaw(baseURL, req)
			results[index] = concurrentSimpleExecuteResult{
				response: resp,
				elapsed:  time.Since(startedAt),
				err:      err,
			}
		}(i)
	}

	close(start)
	wg.Wait()
	return results
}
