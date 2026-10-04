package tests

import (
	"path/filepath"
	"testing"
)

// sampleCodeDir is the fixture root, relative to this package directory.
const sampleCodeDir = "testdata/sample_code"

// language describes how a supported language maps onto the fixture tree and
// onto the API's language identifier.
type language struct {
	// Name is the value sent in the request's "language" field.
	Name string
	// Ext is the fixture file extension, including the dot.
	Ext string
	// OrphanName is the process comm name used by the orphan-reaping scenario.
	OrphanName string
}

var (
	langC      = language{Name: "c", Ext: ".c", OrphanName: "orphanmakergc"}
	langPython = language{Name: "python3", Ext: ".py", OrphanName: "orphanpygc"}
	langCpp    = language{Name: "cpp", Ext: ".cpp", OrphanName: "orphanmakergc"}
	langJava   = language{Name: "java", Ext: ".java", OrphanName: "orphanjavagc"}
)

// load reads a sample program for this language, applying replacements.
func (l language) load(t *testing.T, name string, replacements map[string]string) string {
	t.Helper()
	return mustLoadSampleCode(t, filepath.Join(name, entryFileName(l.Ext)), replacements)
}

func entryFileName(ext string) string {
	if ext == ".java" {
		return "Main.java"
	}
	return "main" + ext
}
