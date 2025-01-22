// snesbrr
// Copyright 2025 Mukunda Johnson (mukunda.com)
// Licensed under MIT

// Package brr provides codec functionality to convert between BRR and PCM. It also has
// basic WAV reading and writing functionality via the go-audio libraries.
package brr

import (
	"errors"
	"io"
	"os"
	"strconv"

	"github.com/go-audio/audio"
	"github.com/go-audio/wav"
)

type SampleRate int

// Returned when importing a wav file and finding invalid
// data. go-audio/wav is used to detect invalid wav files.
var ErrInvalidWav = errors.New("invalid wav file")

// Returned when importing a wav file with formats that are unsupported.
var ErrUnsupportedWav = errors.New("unsupported wav file")

// Returned when an unknown codec is specified.
var ErrUnknownCodec = errors.New("unknown codec")

// Returned when an unknown codec option is specified.
var ErrUnknownCodecOption = errors.New("unknown codec option")

// Returned when an invalid codec option value is specified.
var ErrInvalidCodecOptionValue = errors.New("invalid option value")

// Collects information about the last encoding operation.
type EncodingStats struct {
	TotalError float64
	AvgError   float64
	MinError   float64
	MaxError   float64
}

type codecImpl interface {
	Setopt(name string, value string) error
	Encode(data []int16) []byte
	Decode(data []byte) ([]int16, SampleRate)
	EncodingStats() EncodingStats
}

// Encodes and decodes between BRR and PCM. Buffers are kept entirely in memory.
type BrrCodec struct {

	// 16-bit signed PCM data, used during encoding. This can be set directly or through
	// the reading methods.
	PcmData []int16

	// The sample rate of the PCM data. Usually 32000 but codecs can change this depending
	// on the options during decoding. Not used for encoding.
	PcmRate SampleRate

	// 8-bit BRR data, used during decoding. This can be set directly or through the
	// reading methods.
	BrrData []byte

	// The underlying codec implementation.
	codec codecImpl
}

// Create a new BRR codec instance and initialize it.
func NewCodec() *BrrCodec {
	codec := BrrCodec{}
	codec.initialize()
	return &codec
}

// [Not Implemented] Returns some statistics about the last encoding. During encoding,
// statistics about the the differences in the encoded samples are recorded. Typically not
// very useful data unless you are doing some sort of obscure sound synthesis.
func (bc *BrrCodec) EncodingStats() EncodingStats {
	return bc.codec.EncodingStats()
}

// Enables gaussian filtering during decoding. The SNES applies a gaussian filter to sound
// output, and this simulates that in the resulting PCM. For when you want the authentic
// "SNES vibe" in your decoded sample.
// func (bc *BrrCodec) SetGaussEnabled(gaussEnabled bool) {
// 	str := "0"
// 	if gaussEnabled {
// 		str = "1"
// 	}
// 	bc.codec.Setopt("gauss", str)
// }

// Sets the loop start point in the given PCM, measured in samples. The loop start may be
// adjusted during encoding to be a multiple of 16. The loop length (remainder of the
// sample) is unrolled to align.
//
// Pass -1 to remove the loop.
//
// Loops affect the BRR encoding. There is a loop flag (2nd bit) in the BRR blocks, and
// BRR filters will not be used on the loop block to avoid corrupted output.
func (bc *BrrCodec) SetLoop(loopStart int) {
	if loopStart < 0 {
		bc.codec.Setopt("loop", "")
	}

	bc.codec.Setopt("loop", strconv.Itoa(loopStart))
}

// Returns the raw PCM data from the last decode operation.
func (bc *BrrCodec) GetPcmData() []int16 {
	return bc.PcmData
}

func (bc *BrrCodec) GetBrrData() []uint8 {
	return bc.BrrData
}

// Set an option for the underlying codec.
func (bc *BrrCodec) SetCodecOption(name string, value string) error {
	return bc.codec.Setopt(name, value)
}

// Sets the pitch to be used during decoding. (I'm not sure what this is used for.)
// func (bc *BrrCodec) SetPitch(pitch int) {
// 	bc.codec.Setopt("pitch", strconv.Itoa(pitch))
// }

// Executed on new instances to initialize the state.
func (bc *BrrCodec) initialize() {
	*bc = BrrCodec{}
	bc.PcmRate = 32000
	bc.SetCodecImplementation("noc")
}

// Sets the implementation to be used. Default is "noc" which is based on the no$ Fullsnes
// information. The other implementation is "dmv" which is based on the original snesbrr
// codec from DMV47.
func (bc *BrrCodec) SetCodecImplementation(codec string) error {
	switch codec {
	case "noc":
		bc.codec = createNocCodec()
	case "dmv":
		bc.codec = createDmvCodec()
	default:
		return ErrUnknownCodec
	}

	return nil
}

// Decode the data in the BRR buffer into the PCM buffer.
func (bc *BrrCodec) Decode() {
	bc.PcmData, bc.PcmRate = bc.codec.Decode(bc.BrrData)
}

// Encode the data in the PCM buffer into the BRR buffer.
func (bc *BrrCodec) Encode() {
	bc.BrrData = bc.codec.Encode(bc.PcmData)
}

// Load the codec with the given BRR data from a stream.
func (bc *BrrCodec) ReadBrr(is io.Reader) error {
	var err error
	bc.BrrData, err = io.ReadAll(is)
	if err != nil {
		return err
	}

	// Pad to a multiple of 9 (BRR chunk size).
	for (len(bc.BrrData) % 9) != 0 {
		bc.BrrData = append(bc.BrrData, 0)
	}

	return nil
}

// Load the codec with the given BRR data from a file. The data in the file is the raw BRR
// data without any header.
func (bc *BrrCodec) ReadBrrFile(filename string) error {
	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer file.Close()
	return bc.ReadBrr(file)
}

// Copy the BRR buffer into the given stream.
func (bc *BrrCodec) WriteBrr(os io.Writer) error {
	_, err := os.Write(bc.BrrData)
	if err != nil {
		return err
	}
	return nil
}

// Copy the BRR buffer to a file. Existing files will be truncated/overwritten.
func (bc *BrrCodec) WriteBrrFile(filename string) error {
	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	return bc.WriteBrr(f)
}

// Read the given wav file from a stream into the PCM buffer.
func (bc *BrrCodec) ReadWav(file io.ReadSeeker) error {
	decoder := wav.NewDecoder(file)
	if !decoder.IsValidFile() {
		return ErrInvalidWav
	}

	data, err := decoder.FullPCMBuffer()
	if err != nil {
		return err
	}

	intData := data.AsIntBuffer()

	if intData.SourceBitDepth == 8 {
		for i := 0; i < len(intData.Data); i += intData.Format.NumChannels {
			bc.PcmData = append(bc.PcmData, int16(intData.Data[i])<<8)
		}
	} else if intData.SourceBitDepth == 16 {
		for i := 0; i < len(intData.Data); i += intData.Format.NumChannels {
			bc.PcmData = append(bc.PcmData, int16(intData.Data[i]))
		}
	} else if intData.SourceBitDepth == 24 {
		for i := 0; i < len(intData.Data); i += intData.Format.NumChannels {
			bc.PcmData = append(bc.PcmData, int16(intData.Data[i]>>8))
		}
	} else if intData.SourceBitDepth == 32 {
		for i := 0; i < len(intData.Data); i += intData.Format.NumChannels {
			bc.PcmData = append(bc.PcmData, int16(intData.Data[i]>>16))
		}
	} else {
		return ErrUnsupportedWav
	}

	return nil
}

// Read the given wav file into the PCM buffer.
func (bc *BrrCodec) ReadWavFile(filename string) error {
	// TODO: multichannel wavs need to be mixed into one channel.

	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer file.Close()
	return bc.ReadWav(file)
}

// Copy the contents of the PCM buffer to the given stream.
func (bc *BrrCodec) WriteWav(os io.WriteSeeker) error {
	enc := wav.NewEncoder(os, int(bc.PcmRate), 16, 1, 1)
	defer enc.Close()

	outputData := make([]int, len(bc.PcmData))
	for i := 0; i < len(bc.PcmData); i++ {
		outputData[i] = int(bc.PcmData[i])
	}

	err := enc.Write(&audio.IntBuffer{
		Data: outputData,
		Format: &audio.Format{
			SampleRate: int(bc.PcmRate), NumChannels: 1,
		},
		SourceBitDepth: 16,
	})
	if err != nil {
		return err
	}

	return nil
}

// Copy the contents of the PCM buffer to the given wav file. Existing files will be
// truncated/overwritten.
func (bc *BrrCodec) WriteWavFile(filename string) error {
	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer f.Close()
	return bc.WriteWav(f)
}
