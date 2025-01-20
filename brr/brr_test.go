package brr

import (
	"fmt"
	"io"
	"math"
	"math/rand"
	"os"
	"os/exec"
	"strconv"
	"testing"

	"github.com/go-audio/audio"
	"github.com/go-audio/wav"
	"github.com/stretchr/testify/assert"
)

// These tests only function on Windows currently. It tests against a build of snesbrr.exe
// It should work on Linux if you replace the binary with a Linux build.

func createWavFile(method string, length int) {
	pcm := make([]int, length)

	if method == "whitenoise" {
		for i := 0; i < length; i++ {
			pcm[i] = int(rand.Intn(65536) - 32768)
		}
	} else if method == "sine" {
		for i := 0; i < length; i++ {
			pcm[i] = int(30000 * math.Sin(440.0/22000.0*float64(i)))
		}
	}

	f, err := os.Create(".testfile_brr-test.wav")
	if err != nil {
		panic(err)
	}
	defer f.Close()

	encoder := wav.NewEncoder(f, 22000, 16, 1, 1)
	defer encoder.Close()
	// buffer := audio.IntBuffer{Data: pcm, Format: &audio.Format{NumChannels: 1, SampleRate: 22000}}

	err = encoder.Write(&audio.IntBuffer{
		Data:   pcm,
		Format: &audio.Format{NumChannels: 1, SampleRate: 22000},
	})

	if err != nil {
		panic(err)
	}
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

func runCmd(t *testing.T, args ...string) {
	cmd := exec.Command(args[0], args[1:]...)
	err := cmd.Run()

	if err != nil {
		assert.Fail(t, err.Error())
		output, _ := cmd.CombinedOutput()
		t.Logf("Command output: %s", string(output))
	}
}

func runEncodeTest(t *testing.T, method string, length int, loop int, loopEnabled bool) {

	println("testing", method, "with length", length, "and loop", loop)
	createWavFile(method, length)
	codec := NewBrrCodec()
	codec.ReadWavFile(".testfile_brr-test.wav")
	codec.loopStart = loop
	codec.loopEnabled = loopEnabled
	codec.Encode()
	f, err := os.Create(".testfile_brr-test.brr")
	assert.NoError(t, err)
	defer f.Close()
	codec.WriteBrr(f)
	f.Close()

	args := []string{"../test/snesbrr.exe", "--encode", ".testfile_brr-test.wav", ".testfile_brr-test2.brr"}
	if loopEnabled {
		args = append(args, "--loop-start", strconv.Itoa(loop))
	}
	runCmd(t, args...)

	compareChunks(t, ".testfile_brr-test.brr", ".testfile_brr-test2.brr")
}

// Tests against the original snesbrr.exe
func TestBrrEncoding(t *testing.T) {
	// Check for proper padding during encoding.
	runEncodeTest(t, "sine", 64, 0, false)
	runEncodeTest(t, "sine", 65, 0, false)

	runEncodeTest(t, "sine", 9001, 0, false)
	runEncodeTest(t, "sine", 9002, 0, false)
	runEncodeTest(t, "sine", 9003, 0, false)
	runEncodeTest(t, "sine", 9004, 0, false)
	runEncodeTest(t, "sine", 9005, 0, false)
	runEncodeTest(t, "sine", 9006, 0, false)
	runEncodeTest(t, "sine", 9007, 0, false)
	runEncodeTest(t, "sine", 9008, 0, false)
	runEncodeTest(t, "sine", 9009, 0, false)

	// Proper filter usage detected by different waveforms.
	runEncodeTest(t, "whitenoise", 64, 0, false)
	runEncodeTest(t, "whitenoise", 65, 0, false)
	runEncodeTest(t, "whitenoise", 9000, 0, false)
	runEncodeTest(t, "whitenoise", 9001, 0, false)
	runEncodeTest(t, "whitenoise", 9002, 0, false)
	runEncodeTest(t, "whitenoise", 9003, 0, false)
	runEncodeTest(t, "whitenoise", 9004, 0, false)
	runEncodeTest(t, "whitenoise", 9005, 0, false)
	runEncodeTest(t, "whitenoise", 9006, 0, false)
	runEncodeTest(t, "whitenoise", 9007, 0, false)
	runEncodeTest(t, "whitenoise", 9008, 0, false)
	runEncodeTest(t, "whitenoise", 9009, 0, false)
}

func TestBrrLoops(t *testing.T) {
	runEncodeTest(t, "sine", 64, 0, true)
	runEncodeTest(t, "sine", 64, 32, true)
	runEncodeTest(t, "sine", 100, 0, true)
	runEncodeTest(t, "sine", 100, 40, true)
	runEncodeTest(t, "sine", 100, 41, true)
	runEncodeTest(t, "sine", 100, 42, true)

	runEncodeTest(t, "sine", 6000, 400, true)
	runEncodeTest(t, "sine", 6001, 400, true)
	runEncodeTest(t, "sine", 6003, 400, true)
	runEncodeTest(t, "sine", 6000, 401, true)
	runEncodeTest(t, "sine", 6001, 401, true)
	runEncodeTest(t, "sine", 6003, 401, true)
	runEncodeTest(t, "sine", 6000, 402, true)
	runEncodeTest(t, "sine", 6001, 402, true)
	runEncodeTest(t, "sine", 6003, 402, true)

	runEncodeTest(t, "whitenoise", 6000, 400, true)
	runEncodeTest(t, "whitenoise", 6001, 400, true)
	runEncodeTest(t, "whitenoise", 6003, 400, true)
	runEncodeTest(t, "whitenoise", 6000, 401, true)
	runEncodeTest(t, "whitenoise", 6001, 401, true)
	runEncodeTest(t, "whitenoise", 6003, 401, true)
	runEncodeTest(t, "whitenoise", 6000, 402, true)
	runEncodeTest(t, "whitenoise", 6001, 402, true)
	runEncodeTest(t, "whitenoise", 6003, 402, true)
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

func runDecodeTest(t *testing.T, length int, crazy bool, gauss bool, pitch uint16) {
	println("testing decode with length", length, "and crazy", crazy)
	createBrrFile(length, crazy)
	codec := NewBrrCodec()
	codec.ReadBrrFile(".testfile_brr-test.brr")
	codec.gaussEnabled = gauss
	if pitch > 0 {
		codec.userPitchEnabled = true
		codec.pitchStepBase = pitch
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

	assert.Equal(t, expected, codec.wavData)
}

func TestDecode(t *testing.T) {
	runDecodeTest(t, 9*50, false, false, 0)
	runDecodeTest(t, 9*500, true, false, 0)
	runDecodeTest(t, 9*50+1, false, false, 0)
	runDecodeTest(t, 9*50+2, true, false, 0)
	runDecodeTest(t, 9*50+3, true, false, 0)
	runDecodeTest(t, 9*50+4, true, false, 0)

	runDecodeTest(t, 9*3, false, true, 0)
	runDecodeTest(t, 9*500, true, true, 0)
	runDecodeTest(t, 9*50+1, false, true, 0)
	runDecodeTest(t, 9*50+2, true, true, 0)
	runDecodeTest(t, 9*50+3, true, true, 0)
	runDecodeTest(t, 9*50+4, true, true, 0)

	runDecodeTest(t, 9*3, false, true, 599)
	runDecodeTest(t, 9*500, true, true, 600)
	runDecodeTest(t, 9*50+1, false, true, 700)
	runDecodeTest(t, 9*50+2, true, true, 800)
	runDecodeTest(t, 9*50+3, true, true, 900)
	runDecodeTest(t, 9*50+4, true, true, 601)
}

func TestMain(m *testing.M) {
	// Setup code here (runs before all tests)

	exitCode := m.Run() // Run all tests

	// Teardown code here (runs after all tests)
	os.Remove(".testfile_brr-test.wav")
	os.Remove(".testfile_brr-test.brr")
	os.Remove(".testfile_brr-test2.brr")
	os.Remove(".testfile_brr-test2.wav")

	os.Exit(exitCode)
}
