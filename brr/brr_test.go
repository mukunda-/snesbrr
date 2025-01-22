// snesbrr
// Copyright 2025 Mukunda Johnson (mukunda.com)
// Licensed under MIT

package brr

import (
	"math"
	"math/rand"
	"os"

	"github.com/go-audio/audio"
	"github.com/go-audio/wav"
)

func createSinePcm16(length int, height float64) []int16 {
	pcm := make([]int16, length)
	for i := 0; i < length; i++ {
		pcm[i] = int16(height * math.Sin(440.0/22000.0*float64(i)))
	}
	return pcm
}

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

// func TestBrrCodec_ReadWavFile(t *testing.T) {

// }
