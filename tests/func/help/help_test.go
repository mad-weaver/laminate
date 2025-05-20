package help

import (
	"os/exec"
	"testing"

	"github.com/mad-weaver/laminate/tests/func/testutil"
	"github.com/stretchr/testify/require"
)

func TestLaminateHelp(t *testing.T) {
	mainPath := testutil.GetMainPath(t)

	cmd := exec.Command("go", "run", mainPath, "--help")
	output, err := cmd.CombinedOutput()
	require.NoError(t, err, "laminate help command failed")

	// Log the output for manual verification
	t.Logf("Command output:\n%s", string(output))

	// Basic verification that help output contains expected sections
	outputStr := string(output)
	require.Contains(t, outputStr, "NAME:")
	require.Contains(t, outputStr, "USAGE:")
	require.Contains(t, outputStr, "GLOBAL OPTIONS:")
}
