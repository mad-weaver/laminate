package testutil

import (
	"bytes"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// GetMainPath returns the absolute path to main.go
func GetMainPath(t *testing.T) string {
	t.Helper()
	// Get the module root using go list
	cmd := exec.Command("go", "list", "-m", "-f", "{{.Dir}}")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to get module root: %v", err)
	}
	moduleRoot := strings.TrimSpace(string(output))
	mainPath := filepath.Join(moduleRoot, "main.go")
	if _, err := os.Stat(mainPath); err != nil {
		t.Fatalf("main.go not found at %s: %v", mainPath, err)
	}
	return mainPath
}

// CreateTempDir creates a temporary directory for test files and returns its path
// and a cleanup function to pair with a defer for cleanup.
func CreateTempDir(t *testing.T, prefix string) (string, func()) {
	t.Helper()
	tmpDir, err := os.MkdirTemp("", prefix)
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	cleanup := func() {
		os.RemoveAll(tmpDir)
	}

	return tmpDir, cleanup
}

// GetTestDir returns the absolute path of the directory containing the test file.
// This is useful for locating test data files relative to the test.
func GetTestDir(t *testing.T) string {
	t.Helper()
	// Get the directory of the calling test file
	_, testFile, _, _ := runtime.Caller(1)
	testDir := filepath.Dir(testFile)
	return testDir
}

// CopyFile copies a file from src to dst, preserving the original filename.
// It returns the full path of the destination file.
func CopyFile(t *testing.T, src, dstDir string) string {
	t.Helper()
	srcFile, err := os.Open(src)
	if err != nil {
		t.Fatalf("Failed to open source file: %v", err)
	}
	defer srcFile.Close()

	dst := filepath.Join(dstDir, filepath.Base(src))
	dstFile, err := os.Create(dst)
	if err != nil {
		t.Fatalf("Failed to create destination file: %v", err)
	}
	defer dstFile.Close()

	if _, err := io.Copy(dstFile, srcFile); err != nil {
		t.Fatalf("Failed to copy file: %v", err)
	}

	return dst
}

// RunCLI runs the laminate CLI with the given arguments and returns its output
func RunCLI(t *testing.T, args ...string) (string, error) {
	t.Helper()

	// Get the path to the current working directory
	cwd, err := os.Getwd()
	require.NoError(t, err)

	// Build the command
	cmd := exec.Command(filepath.Join(cwd, "laminate"))
	cmd.Args = append([]string{cmd.Path}, args...)

	// Capture output
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Run the command
	err = cmd.Run()
	if err != nil {
		return "", err
	}

	return stdout.String(), nil
}
