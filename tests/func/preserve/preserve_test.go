package preserve

import (
	"encoding/json"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/mad-weaver/laminate/tests/func/testutil"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestPreserveMergeStrategy(t *testing.T) {
	// Get the path to main.go
	mainPath := testutil.GetMainPath(t)

	// Get paths to test data files
	testDataDir := filepath.Join("testdata")
	baseFile := filepath.Join(testDataDir, "base_list.yaml")
	patchFile := filepath.Join(testDataDir, "patch_list.yaml")

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
			// Run laminate with preserve merge strategy
			cmd := exec.Command("go", "run", mainPath,
				"--source", baseFile,
				"--patch", patchFile,
				"--merge-strategy", "preserve",
				"--format", tc.outputFormat)

			output, err := cmd.CombinedOutput()
			require.NoError(t, err, "laminate command failed: %s", string(output))

			// Parse the output
			var result map[string]interface{}
			err = tc.decoder(output, &result)
			require.NoError(t, err, "failed to decode output")

			// Get the plugins list
			plugins, ok := result["server"].(map[string]interface{})["plugins"].([]interface{})
			require.True(t, ok, "failed to get plugins list")

			// Verify all items are preserved (3 original + 1 new = 4 total)
			require.Len(t, plugins, 4, "plugins list should have 4 items")

			// Create a map to store plugins by name for easier verification
			pluginMap := make(map[string]map[string]interface{})
			for _, p := range plugins {
				plugin := p.(map[string]interface{})
				pluginMap[plugin["name"].(string)] = plugin
			}

			// Check auth plugin (should preserve original config)
			auth := pluginMap["auth"]
			require.NotNil(t, auth)
			require.Equal(t, true, auth["enabled"])
			authConfig := auth["config"].(map[string]interface{})

			// Handle both int and float64 types for timeout
			timeout := authConfig["timeout"]
			switch v := timeout.(type) {
			case float64:
				require.Equal(t, float64(30), v)
			case int:
				require.Equal(t, 30, v)
			default:
				t.Errorf("timeout is neither int nor float64: %T", v)
			}

			// Check logger plugin (should be unchanged)
			logger := pluginMap["logger"]
			require.NotNil(t, logger)
			require.Equal(t, true, logger["enabled"])
			loggerConfig := logger["config"].(map[string]interface{})
			require.Equal(t, "info", loggerConfig["level"])

			// Check metrics plugin (should be unchanged)
			metrics := pluginMap["metrics"]
			require.NotNil(t, metrics)
			require.Equal(t, false, metrics["enabled"])

			// Check new cache plugin (should be added)
			cache := pluginMap["cache"]
			require.NotNil(t, cache)
			require.Equal(t, true, cache["enabled"])
			cacheConfig := cache["config"].(map[string]interface{})

			// Handle both int and float64 types for size
			size := cacheConfig["size"]
			switch v := size.(type) {
			case float64:
				require.Equal(t, float64(1024), v)
			case int:
				require.Equal(t, 1024, v)
			default:
				t.Errorf("size is neither int nor float64: %T", v)
			}
		})
	}
}
