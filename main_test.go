// snesbrr
// Copyright 2025 Mukunda Johnson (mukunda.com)
// Licensed under MIT

package main

import (
	"bytes"
	"io"
	"math/rand"
	"os"
	"os/exec"
	"testing"

	"github.com/stretchr/testify/assert"
)

type runResult struct {
	ret    returnCode
	output string
}

func runArgs(args ...string) runResult {
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	defer func() {
		os.Stdout = old
		r.Close()
		w.Close()
	}()

	ret := run(args)

	w.Close()
	var buf bytes.Buffer
	io.Copy(&buf, r)

	return runResult{
		ret:    ret,
		output: buf.String(),
	}
}

func createTestBrr(path string) {
	f, err := os.Create(path)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	for i := 0; i < 10; i++ {
		// header (clamp range to 0-7) (remove END and LOOP bits)
		f.Write([]byte{byte(rand.Intn(256)) &^ 0x83})
		for j := 0; j < 8; j++ {
			// data
			f.Write([]byte{byte(rand.Intn(256))})
		}
	}
}

func TestCodec(t *testing.T) {
	defer os.Remove(".testfile_main.brr")
	defer os.Remove(".testfile_main.brr.wav")
	defer os.Remove(".testfile_main.brr.wav.brr")

	createTestBrr(".testfile_main.brr")

	// First, let's create proper data by sanitizing the
	// randomly generated BRR through transcoding.

	assert.Zero(t, runArgs("--decode", ".testfile_main.brr", ".testfile_main.brr.wav").ret)
	assert.Zero(t, runArgs("--encode", ".testfile_main.brr.wav", ".testfile_main.brr").ret)

	//assert.NoError(t, exec.Command("go", "run", "main.go", "--decode", ".testfile_main.brr", ".testfile_main.brr.wav").Run())
	//assert.NoError(t, exec.Command("go", "run", "main.go", "--encode", ".testfile_main.brr.wav", ".testfile_main.brr").Run())

	{
		// When the output filename isn't given, it's derived from the input filename with a suffix.
		// Existing files will not be overwritten when the output filename is not explicitly given.
		assert.Equal(t, 1, runArgs("--decode", ".testfile_main.brr").ret)

		os.Remove(".testfile_main.brr.wav")
		//err = exec.Command("go", "run", "main.go", "--decode", ".testfile_main.brr").Run()
		assert.Zero(t, 0, runArgs("--decode", ".testfile_main.brr").ret)
		//assert.NoError(t, err)
		assert.FileExists(t, ".testfile_main.brr.wav", "decoded file should be created")
	}

	{
		// When using encode, the suffix is ".brr".
		// Transcoding back to a valid BRR sample should be exactly equal.
		//err := exec.Command("go", "run", "main.go", "--encode", ".testfile_main.brr.wav", ".testfile_main.brr.wav.brr").Run()
		assert.Zero(t, 0, runArgs("--encode", ".testfile_main.brr.wav", ".testfile_main.brr.wav.brr").ret)
		//assert.NoError(t, err)
		assert.FileExists(t, ".testfile_main.brr.wav.brr")
	}

	file1, _ := os.ReadFile(".testfile_main.brr")
	file2, _ := os.ReadFile(".testfile_main.brr.wav.brr")
	assert.Equal(t, file2, file1, "transcoded content should equal original")
}

func TestUsage(t *testing.T) {

	// When no args are specified, a short usage message is printed.
	cmd := exec.Command("go", "run", "main.go")
	output, err := cmd.CombinedOutput()
	assert.NoError(t, err)
	assert.Contains(t, string(output), "Usage: ")
	assert.Contains(t, string(output), "Use --help")

	// When --help is used, help documentation is printed.
	cmd = exec.Command("go", "run", "main.go", "--help")
	output, err = cmd.CombinedOutput()
	assert.NoError(t, err)
	assert.Contains(t, string(output), "Show this help.")

	// When required options are missing, the user is informed.
	cmd = exec.Command("go", "run", "main.go", "dummyfile")
	output, err = cmd.CombinedOutput()
	assert.Error(t, err)
	assert.Contains(t, string(output), "Must specify --encode or --decode")
}

func TestInvalidCodecOption(t *testing.T) {
	// Codec options differ between codecs. Each codec is responsible for parsing and
	// validating their options and reporting incorrect usage.

	{
		// Error is reported if an unknown codec option is used.
		r := runArgs("--encode", "--codec", "noc", "--opt", "badopt", ".testfile_dummy")
		assert.NotZero(t, r.ret)
		assert.Contains(t, r.output, "unknown codec option")
	}

	// Same case for an option that only exists for a specific codec.
	{
		r := runArgs("--encode", "--codec", "noc", "--opt", "pitch", ".testfile_dummy")
		assert.NotZero(t, r.ret)
		assert.Contains(t, r.output, "unknown codec option")
	}
	{
		// The option is okay here, so it will complain about the file.
		r := runArgs("--encode", "--codec", "dmv", "--opt", "pitch", ".testfile_dummy")
		assert.Contains(t, r.output, "The system cannot find the file specified.")
		assert.NotZero(t, r.ret)
	}

	// Errors from codec options are forwarded to the user.
	{
		r := runArgs("--encode", "--codec", "dmv", "--opt", "pitch=0", ".testfile_dummy")
		assert.Contains(t, r.output, "invalid option value")
		assert.NotZero(t, r.ret)
	}
}
