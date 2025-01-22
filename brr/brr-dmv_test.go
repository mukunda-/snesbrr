// snesbrr
// Copyright 2025 Mukunda Johnson (mukunda.com)
// Licensed under MIT

package brr

import (
	"fmt"
	"io"
	"math/rand"
	"os"
	"os/exec"
	"strconv"
	"testing"

	"github.com/go-audio/wav"
	"github.com/stretchr/testify/assert"
)

// These tests only function on Windows currently. It tests against a build of snesbrr.exe
// It should work on Linux if you replace the binary with a Linux build.

func runCmd(t *testing.T, args ...string) {
	cmd := exec.Command(args[0], args[1:]...)
	err := cmd.Run()

	if err != nil {
		assert.Fail(t, err.Error())
		output, _ := cmd.CombinedOutput()
		t.Logf("Command output: %s", string(output))
	}
}

func runEncodeTest(t *testing.T, method string, length int, loop int) {
	defer os.Remove(".testfile_brr-test.wav")
	defer os.Remove(".testfile_brr-test.brr")
	defer os.Remove(".testfile_brr-test2.brr")

	println("testing", method, "with length", length, "and loop", loop)
	createWavFile(method, length)
	codec := NewCodec()
	codec.SetCodecImplementation("dmv")
	codec.SetCodecOption("compat", "1")

	codec.ReadWavFile(".testfile_brr-test.wav")
	codec.SetLoop(loop)
	//codec.loopStart = loop
	codec.Encode()
	f, err := os.Create(".testfile_brr-test.brr")
	assert.NoError(t, err)
	defer f.Close()
	codec.WriteBrr(f)
	f.Close()

	args := []string{"../test/snesbrr.exe", "--encode", ".testfile_brr-test.wav", ".testfile_brr-test2.brr"}
	if loop >= 0 {
		args = append(args, "--loop-start", strconv.Itoa(loop))
	}
	runCmd(t, args...)

	compareChunks(t, ".testfile_brr-test.brr", ".testfile_brr-test2.brr")
}

func compareChunks(t *testing.T, file1 string, file2 string) {
	f1, err := os.Open(file1)
	assert.NoError(t, err)
	defer f1.Close()
	f2, err := os.Open(file2)
	assert.NoError(t, err)
	defer f2.Close()

	brr1 := make([]byte, 9)
	brr2 := make([]byte, 9)

	address := 0

	for {
		n1, err1 := f1.Read(brr1)
		n2, err2 := f2.Read(brr2)

		if n1 != n2 {
			t.Errorf("Files %s and %s differ in size", file1, file2)
			return
		}

		if err1 == io.EOF && err2 == io.EOF {
			break
		}
		assert.False(t, err1 != nil && err1 != io.EOF)
		assert.False(t, err2 != nil && err2 != io.EOF)

		for i := 0; i < n1; i++ {
			if brr1[i] != brr2[i] {
				t.Errorf("Files %s and %s differ at byte %d", file1, file2, address+i)
				return
			}
		}

		address += 9
	}
}

// Tests against the original snesbrr.exe
func TestBrrEncoding(t *testing.T) {
	// Check for proper padding during encoding.
	runEncodeTest(t, "sine", 64, -1)
	runEncodeTest(t, "sine", 65, -1)

	runEncodeTest(t, "sine", 9001, -1)
	runEncodeTest(t, "sine", 9002, -1)
	runEncodeTest(t, "sine", 9003, -1)
	runEncodeTest(t, "sine", 9004, -1)
	runEncodeTest(t, "sine", 9005, -1)
	runEncodeTest(t, "sine", 9006, -1)
	runEncodeTest(t, "sine", 9007, -1)
	runEncodeTest(t, "sine", 9008, -1)
	runEncodeTest(t, "sine", 9009, -1)

	// Proper filter usage detected by different waveforms.
	runEncodeTest(t, "whitenoise", 64, -1)
	runEncodeTest(t, "whitenoise", 65, -1)
	runEncodeTest(t, "whitenoise", 9000, -1)
	runEncodeTest(t, "whitenoise", 9001, -1)
	runEncodeTest(t, "whitenoise", 9002, -1)
	runEncodeTest(t, "whitenoise", 9003, -1)
	runEncodeTest(t, "whitenoise", 9004, -1)
	runEncodeTest(t, "whitenoise", 9005, -1)
	runEncodeTest(t, "whitenoise", 9006, -1)
	runEncodeTest(t, "whitenoise", 9007, -1)
	runEncodeTest(t, "whitenoise", 9008, -1)
	runEncodeTest(t, "whitenoise", 9009, -1)
}

func TestBrrLoops(t *testing.T) {
	runEncodeTest(t, "sine", 64, 0)
	runEncodeTest(t, "sine", 64, 32)
	runEncodeTest(t, "sine", 100, 0)
	runEncodeTest(t, "sine", 100, 40)
	runEncodeTest(t, "sine", 100, 41)
	runEncodeTest(t, "sine", 100, 42)

	runEncodeTest(t, "sine", 6000, 400)
	runEncodeTest(t, "sine", 6001, 400)
	runEncodeTest(t, "sine", 6003, 400)
	runEncodeTest(t, "sine", 6000, 401)
	runEncodeTest(t, "sine", 6001, 401)
	runEncodeTest(t, "sine", 6003, 401)
	runEncodeTest(t, "sine", 6000, 402)
	runEncodeTest(t, "sine", 6001, 402)
	runEncodeTest(t, "sine", 6003, 402)

	runEncodeTest(t, "whitenoise", 6000, 400)
	runEncodeTest(t, "whitenoise", 6001, 400)
	runEncodeTest(t, "whitenoise", 6003, 400)
	runEncodeTest(t, "whitenoise", 6000, 401)
	runEncodeTest(t, "whitenoise", 6001, 401)
	runEncodeTest(t, "whitenoise", 6003, 401)
	runEncodeTest(t, "whitenoise", 6000, 402)
	runEncodeTest(t, "whitenoise", 6001, 402)
	runEncodeTest(t, "whitenoise", 6003, 402)
}

func createBrrFile(length int, crazy bool) {
	f, err := os.Create(".testfile_brr-test.brr")
	if err != nil {
		panic(err)
	}
	defer f.Close()

	for i := 0; i < length; i++ {
		if !crazy {
			if i%9 == 0 {
				brange := rand.Intn(13)
				filter := rand.Intn(4)
				end := 0
				if i >= length-9 {
					end = 1
				}
				f.Write([]byte{byte(brange<<4 | filter<<2 | end)})
				continue
			}
		}
		f.Write([]byte{byte(rand.Intn(256))})
	}
}

func readPcm16(filename string) []int16 {
	f, err := os.Open(filename)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	decoder := wav.NewDecoder(f)

	buf, err := decoder.FullPCMBuffer()
	if err != nil {
		panic(err)
	}

	buffer := make([]int16, len(buf.AsIntBuffer().Data))
	for i, v := range buf.AsIntBuffer().Data {
		buffer[i] = int16(v)
	}
	return buffer
}

func runDmvDecodeTest(t *testing.T, length int, crazy bool, gauss bool, pitch int) {

	defer os.Remove(".testfile_brr-test.brr")
	defer os.Remove(".testfile_brr-test2.wav")

	println("testing decode with length", length, "and crazy", crazy)
	createBrrFile(length, crazy)
	codec := NewCodec()
	codec.SetCodecImplementation("dmv")
	codec.SetCodecOption("compat", "1")

	codec.ReadBrrFile(".testfile_brr-test.brr")
	if gauss {
		codec.SetCodecOption("gauss", "1")
	}

	if pitch > 0 {
		codec.SetCodecOption("pitch", strconv.Itoa(pitch))
	}

	codec.Decode()

	args := []string{"../test/snesbrr.exe", "--decode", ".testfile_brr-test.brr", ".testfile_brr-test2.wav"}
	if gauss {
		args = append(args, "--enable-gauss")
	}
	if pitch > 0 {
		args = append(args, "--pitch", fmt.Sprintf("%x", pitch))
	}
	runCmd(t, args...)
	expected := readPcm16(".testfile_brr-test2.wav")

	assert.Equal(t, expected, codec.PcmData)
}

func TestDmvDecode(t *testing.T) {
	runDmvDecodeTest(t, 9*2, false, false, 0)

	runDmvDecodeTest(t, 9*50, false, false, 0)
	runDmvDecodeTest(t, 9*500, true, false, 0)
	runDmvDecodeTest(t, 9*50+1, false, false, 0)
	runDmvDecodeTest(t, 9*50+2, true, false, 0)
	runDmvDecodeTest(t, 9*50+3, true, false, 0)
	runDmvDecodeTest(t, 9*50+4, true, false, 0)

	runDmvDecodeTest(t, 9*3, false, true, 3) //
	runDmvDecodeTest(t, 9*3, false, true, 0)
	runDmvDecodeTest(t, 9*500, true, true, 0)
	runDmvDecodeTest(t, 9*50+1, false, true, 0)
	runDmvDecodeTest(t, 9*50+2, true, true, 0)
	runDmvDecodeTest(t, 9*50+3, true, true, 0)
	runDmvDecodeTest(t, 9*50+4, true, true, 0)

	runDmvDecodeTest(t, 9*3, false, true, 599)
	runDmvDecodeTest(t, 9*500, true, true, 600)
	runDmvDecodeTest(t, 9*50+1, false, true, 700)
	runDmvDecodeTest(t, 9*50+2, true, true, 800)
	runDmvDecodeTest(t, 9*50+3, true, true, 900)
	runDmvDecodeTest(t, 9*50+4, true, true, 601)
}

func TestDmvDecodeLength(t *testing.T) {
	brr := NewCodec()
	brr.SetCodecImplementation("dmv")
	brr.SetCodecOption("compat", "1")
	brr.BrrData = []byte{
		0x20, 0x11, 0x11, 0x11, 0x11, 0x11, 0x11, 0x11, 0x11,
		0x21, 0x22, 0x22, 0x22, 0x22, 0x22, 0x22, 0x22, 0x22,
	}

	brr.Decode()
	// "compat" bug:
	// The last block is also skipped.
	// The first 3 samples are skipped.

	assert.Equal(t, 13, len(brr.PcmData))

	brr.SetCodecOption("compat", "0")
	brr.Decode()
	assert.Equal(t, 32, len(brr.PcmData))
}
