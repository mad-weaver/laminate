package patch

import (
	"encoding/json"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/mad-weaver/laminate/tests/func/testutil"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestPatchFunctionality(t *testing.T) {
	// Get the path to main.go
	mainPath := testutil.GetMainPath(t)

	// Get paths to test data files
	testDataDir := filepath.Join("testdata")
	baseFile := filepath.Join(testDataDir, "base.yaml")
	patch1File := filepath.Join(testDataDir, "patch1.json")
	patch2File := filepath.Join(testDataDir, "patch2.yaml")

	// Test with different output formats
	testCases := []struct {
		name         string
		outputFormat string
		decoder      func([]byte, interface{}) error
	}{
		{
			name:         "yaml_output_format",
			outputFormat: "yaml",
			decoder:      yaml.Unmarshal,
		},
		{
			name:         "json_output_format",
			outputFormat: "json",
			decoder:      json.Unmarshal,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Run laminate with both patches
			cmd := exec.Command("go", "run", mainPath,
				"--source", baseFile,
				"--patch", patch1File,
				"--patch", patch2File,
				"--format", tc.outputFormat)

			output, err := cmd.CombinedOutput()
			require.NoError(t, err, "laminate command failed: %s", string(output))

			// Parse the output
			var result map[string]interface{}
			err = tc.decoder(output, &result)
			require.NoError(t, err, "failed to decode output")

			// Verify server configuration
			server, ok := result["server"].(map[string]interface{})
			require.True(t, ok, "server configuration not found")
			require.Equal(t, "localhost", server["host"])

			// Handle both int and float64 types for port
			port := server["port"]
			switch v := port.(type) {
			case float64:
				require.Equal(t, float64(9090), v)
			case int:
				require.Equal(t, 9090, v)
			default:
				t.Errorf("port is neither int nor float64: %T", v)
			}

			// Verify database configuration
			db, ok := result["database"].(map[string]interface{})
			require.True(t, ok, "database configuration not found")
			require.Equal(t, "myapp_prod", db["name"], "database name mismatch")
			require.Equal(t, "admin", db["user"])
			require.Equal(t, "secret", db["password"])
		})
	}
}
