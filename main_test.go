package main

import (
	"math/rand"
	"os"
	"os/exec"
	"testing"

	"github.com/stretchr/testify/assert"
)

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

	createTestBrr(".testfile_main.brr")

	// First, let's create proper data by sanitizing the
	// randomly generated BRR through transcoding.
	assert.NoError(t, exec.Command("go", "run", "main.go", "--decode", ".testfile_main.brr", ".testfile_main.brr.wav").Run())
	assert.NoError(t, exec.Command("go", "run", "main.go", "--encode", ".testfile_main.brr.wav", ".testfile_main.brr").Run())

	{
		// When the output filename isn't given, it's derived from the input filename with a suffix.
		// Existing files will not be overwritten when the output filename is not explicitly given.
		err := exec.Command("go", "run", "main.go", "--decode", ".testfile_main.brr").Run()
		assert.Error(t, err)

		os.Remove(".testfile_main.brr.wav")
		err = exec.Command("go", "run", "main.go", "--decode", ".testfile_main.brr").Run()
		assert.NoError(t, err)
		assert.FileExists(t, ".testfile_main.brr.wav", "decoded file should be created")
	}

	{
		// When using encode, the suffix is ".brr".
		// Transcoding back to a valid BRR sample should be exactly equal.
		err := exec.Command("go", "run", "main.go", "--encode", ".testfile_main.brr.wav", ".testfile_main.brr.wav.brr").Run()
		assert.NoError(t, err)
		assert.FileExists(t, ".testfile_main.brr.wav.brr")
	}

	file1, _ := os.ReadFile(".testfile_main.brr")
	file2, _ := os.ReadFile(".testfile_main.brr.wav.brr")
	assert.Equal(t, file2, file1, "transcoded content should equal original")
}

func TestUsage(t *testing.T) {

	cmd := exec.Command("go", "run", "main.go")
	output, err := cmd.CombinedOutput()
	assert.NoError(t, err)
	assert.Contains(t, string(output), "Usage: ")
	assert.Contains(t, string(output), "Use --help")

	cmd = exec.Command("go", "run", "main.go", "--help")
	output, err = cmd.CombinedOutput()
	assert.NoError(t, err)
	assert.Contains(t, string(output), "Print a summary of the command line options")

	cmd = exec.Command("go", "run", "main.go", "dummyfile")
	output, err = cmd.CombinedOutput()
	assert.Error(t, err)
	assert.Contains(t, string(output), "Must specify --encode or --decode")
}

func TestMain(m *testing.M) {

	code := m.Run()

	os.Remove(".testfile_main.brr")
	os.Remove(".testfile_main.brr.wav")
	os.Remove(".testfile_main.brr.wav.brr")

	os.Exit(code)
}
