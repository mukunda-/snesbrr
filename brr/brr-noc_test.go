// snesbrr
// Copyright 2025 Mukunda Johnson (mukunda.com)
// Licensed under MIT

package brr

import (
	"math/rand"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNocEncoding(t *testing.T) {

	codec := NewCodec()
	codec.PcmData = []int16{16, 16, 16, 16, 16, 16, 16, 16, 16, 16, 16, 16, 16, 16, 16, 16}

	codec.SetCodecImplementation("noc")
	codec.Encode()
	encoded1 := codec.BrrData

	codec.SetCodecImplementation("dmv")
	codec.Encode()
	encoded2 := codec.BrrData

	assert.Equal(t, encoded1, encoded2)

}

func TestNocEncodingManual(t *testing.T) {
	// Todo: this test is manual verification currently. We should test by checking
	// the output to make sure that it has minimal error.

	// Comment these out to do manual verification, otherwise this is just a smoke test.
	defer os.Remove(".testfile_manual_original.wav")
	defer os.Remove(".testfile_manual_transcoded_noc_dmv.wav")
	defer os.Remove(".testfile_manual_transcoded_dmv_dmv.wav")

	codec := NewCodec()
	testData := createSinePcm16(5000, 25000)

	codec.PcmData = testData
	codec.WriteWavFile(".testfile_manual_original.wav")

	codec.SetCodecImplementation("noc")
	codec.Encode()
	codec.SetCodecImplementation("dmv")
	codec.Decode()
	codec.WriteWavFile(".testfile_manual_transcoded_noc_dmv.wav")

	codec.PcmData = testData
	codec.Encode()
	codec.Decode()
	codec.WriteWavFile(".testfile_manual_transcoded_dmv_dmv.wav")

}

// Todo: more tests:
// Check samples that cause overflow clipping during encoding.
//   i.e. coverage of if nextDecodedSample < -0x3FFA and if nextDecodedSample > 0x3FF8
// Random generations and error testing to make sure nothing goes crazy.

// func getRandomBrr() []byte {
// 	brr := []byte{}
// 	for i := 0; i < 9*2; i++ {
// 		if i%9 == 0 {
// 			// header
// 			shift := rand.Intn(13)
// 			filter := rand.Intn(4)
// 			byt := byte((shift << 4) | (filter << 2))
// 			brr = append(brr, byt)
// 		} else {
// 			brr = append(brr, byte(rand.Intn(256)))
// 		}
// 	}
// 	return brr
// }

func createLosslessPcm(chunks int) []int16 {
	// Outputs a sample of lossless PCM (it should have zero error during BRR encoding)
	output := []int16{}

	prev1 := 0
	prev2 := 0

	for c := 0; c < chunks; c++ {
		filter := rand.Intn(4)
		shift := rand.Intn(11)
		if c == 0 {
			filter = 0
		}

		for {
			chunk := []int16{}
			retry := false

			for i := 0; i < 16; i++ {
				brrSamp := (rand.Intn(16) - 8)

				pcmSamp := brrSamp << shift
				pcmSamp += filterBase(prev1, prev2, filter)
				if pcmSamp < -0x3FF0 || pcmSamp >= 0x3FF0 {
					filter = 0
					retry = true
					break
				}

				prev2 = prev1
				prev1 = pcmSamp
				chunk = append(chunk, int16(pcmSamp<<1))
			}

			if !retry {
				output = append(output, chunk...)
				break
			}
		}

	}

	return output
}

func TestNocDecoding(t *testing.T) {

	// Artificially generate a PCM file that should losslessly encode to BRR, and then
	// transcode it. There should be 0 error and an exact match.

	pcm := createLosslessPcm(1000)
	codec := NewCodec()
	codec.PcmData = pcm
	codec.SetCodecImplementation("noc")
	codec.Encode()
	assert.Equal(t, 0.0, codec.EncodingStats().TotalError)

	codec.Decode()
	assert.Equal(t, pcm, codec.PcmData)
}
