// snesbrr
// Copyright 2025 Mukunda Johnson (mukunda.com)
// Licensed under MIT

package brr_test

import (
	"fmt"

	"go.mukunda.com/snesbrr/brr"
)

func ExampleBrrCodec_Encode() {
	codec := brr.NewCodec()

	codec.ReadWavFile("input.wav")
	codec.Encode()
	codec.WriteBrrFile("output.brr")
}

func ExampleBrrCodec_Decode() {
	codec := brr.NewCodec()

	codec.ReadBrrFile("input.brr")
	codec.Decode()
	codec.WriteWavFile("output.wav")
}

func ExampleBrrCodec_GetBrrData() {
	codec := brr.NewCodec()

	// 16 samples will become 9 bytes.

	// The library could still use some work to verify if that additional dummy block is
	// required.
	codec.PcmData = []int16{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15}
	codec.Encode()

	brrData := codec.GetBrrData()
	fmt.Println("BRR data length:", len(brrData))
	// Output:
	// BRR data length: 9
}

func ExampleBrrCodec_GetPcmData() {
	codec := brr.NewCodec()

	// One BRR block (9 bytes) will become 16 samples.
	codec.BrrData = []uint8{0x21, 0x11, 0x11, 0x11, 0x11, 0x11, 0x11, 0x11, 0x11}
	codec.Decode()

	pcmData := codec.GetPcmData()
	fmt.Printf("Decoded PCM: %02x", pcmData)
	// Output:
	// Decoded PCM: [04 04 04 04 04 04 04 04 04 04 04 04 04 04 04 04]
}
