package tests

import (
	"CodeSandboxAPI/routes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

var (
	testServer *httptest.Server
	httpClient = &http.Client{Timeout: 30 * time.Second}
)

func TestMain(m *testing.M) {
	enforceRootAndCompiler()

	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	routes.Setup(router)
	testServer = httptest.NewServer(router)

	exitCode := m.Run()
	testServer.Close()
	os.Exit(exitCode)
}

func enforceRootAndCompiler() {
	if os.Geteuid() != 0 {
		fmt.Fprintln(os.Stderr, "integration tests must run as root (sudo go test -v)")
		os.Exit(1)
	}
	if _, err := exec.LookPath("gcc"); err != nil {
		fmt.Fprintf(os.Stderr, "gcc is required for integration tests: %v\n", err)
		os.Exit(1)
	}
}
