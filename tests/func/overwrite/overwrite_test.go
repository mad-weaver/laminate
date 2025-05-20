package overwrite

import (
	"encoding/json"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/mad-weaver/laminate/tests/func/testutil"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestOverwriteMergeStrategy(t *testing.T) {
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
			// Run laminate with overwrite merge strategy
			cmd := exec.Command("go", "run", mainPath,
				"--source", baseFile,
				"--patch", patchFile,
				"--merge-strategy", "overwrite",
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

			// Verify the list has been completely replaced
			require.Len(t, plugins, 2, "plugins list should have 2 items")

			// Check first plugin (auth with enabled=false)
			plugin0 := plugins[0].(map[string]interface{})
			require.Equal(t, "auth", plugin0["name"])
			require.Equal(t, false, plugin0["enabled"])
			require.Nil(t, plugin0["config"], "config should not exist for auth plugin")

			// Check second plugin (cache)
			plugin1 := plugins[1].(map[string]interface{})
			require.Equal(t, "cache", plugin1["name"])
			require.Equal(t, true, plugin1["enabled"])

			config1 := plugin1["config"].(map[string]interface{})

			// Handle both int and float64 types for size
			size := config1["size"]
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
