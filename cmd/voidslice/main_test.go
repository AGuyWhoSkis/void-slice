package main_test

import (
	"flag"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	binaryPath   string
	updateGolden = flag.Bool("update", false, "overwrite golden files with current output")
)

func TestMain(m *testing.M) {
	flag.Parse()

	tmp, err := os.MkdirTemp("", "voidslice-test-*")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(tmp)

	name := "voidslice"
	if runtime.GOOS == "windows" {
		name = "voidslice.exe"
	}
	binaryPath = filepath.Join(tmp, name)

	build := exec.Command("go", "build", "-o", binaryPath, ".")
	build.Stdout = os.Stderr
	build.Stderr = os.Stderr
	if err := build.Run(); err != nil {
		panic("go build failed: " + err.Error())
	}

	os.Exit(m.Run())
}

// run executes the binary with args and returns stdout, stderr, and exit code.
func run(args ...string) (stdout, stderr string, exitCode int) {
	cmd := exec.Command(binaryPath, args...)
	var outBuf, errBuf strings.Builder
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf
	err := cmd.Run()
	stdout = outBuf.String()
	stderr = errBuf.String()
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			exitCode = ee.ExitCode()
		} else {
			exitCode = -1
		}
	}
	return
}

func goldenPath(name string) string {
	return filepath.Join("testdata", "golden", name+".txt")
}

func checkGolden(t *testing.T, name, got string) {
	t.Helper()
	path := goldenPath(name)
	if *updateGolden {
		require.NoError(t, os.MkdirAll(filepath.Dir(path), 0755))
		require.NoError(t, os.WriteFile(path, []byte(got), 0644))
		return
	}
	want, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		t.Fatalf("golden file missing: %s — run with -update to create it", path)
	}
	require.NoError(t, err)
	assert.Equal(t, string(want), got)
}

func TestLint_ExitCode_Error(t *testing.T) {
	// missing-semicolon.decl produces PARSE_EXPECTED_SEMICOLON (Error severity)
	_, _, code := run("lint", "../../testdata/broken/missing-semicolon.decl")
	assert.Equal(t, 1, code)
}

func TestLint_ExitCode_Clean(t *testing.T) {
	tmp, err := os.CreateTemp("", "clean-*.decl")
	require.NoError(t, err)
	defer os.Remove(tmp.Name())
	_, err = tmp.WriteString("Version 1\ncomponent {\n\tcpntTest myTest {\n\t\tedit = {\n\t\t\tm_val = \"hello\";\n\t\t}\n\t}\n}\n")
	require.NoError(t, err)
	tmp.Close()

	_, _, code := run("lint", tmp.Name())
	assert.Equal(t, 0, code)
}

func TestLint_HumanFormat(t *testing.T) {
	stdout, _, _ := run("lint", "../../testdata/broken/index-oob.decl")
	checkGolden(t, "index-oob", stdout)
}

func TestLint_JSONFormat(t *testing.T) {
	stdout, _, _ := run("lint", "--json", "../../testdata/broken/index-oob.decl")
	checkGolden(t, "index-oob-json", stdout)
}

// The usage banner advertises `voidslice lint <file> [--json]`, so the flag
// must work after the filename too. The stdlib flag package stops at the
// first non-flag arg by default; parseInterspersed is the workaround.
func TestLint_JSONFormat_FlagAfterFile(t *testing.T) {
	stdout, _, _ := run("lint", "../../testdata/broken/index-oob.decl", "--json")
	checkGolden(t, "index-oob-json", stdout)
}

func TestLint_Binary(t *testing.T) {
	stdout, _, code := run("lint", "../../testdata/binary/sample.bwm")
	assert.Equal(t, 1, code)
	assert.Contains(t, stdout, "binary")
}

func TestLint_FileNotFound(t *testing.T) {
	_, stderr, code := run("lint", "nonexistent-file-xyz.decl")
	assert.Equal(t, 2, code)
	assert.NotEmpty(t, stderr)
}
