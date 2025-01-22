// snesbrr
// Copyright 2025 Mukunda Johnson (mukunda.com)
// Licensed under MIT

// This source code is a Go port of the original codec from DMV47. Original source is
// licensed under Common Development and Distribution License, Version 1.0 only.

package brr

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

type dmvCodec struct {
	stats        EncodingStats
	loopPoint    int
	hasLoop      bool
	pitch        int
	gaussEnabled bool
	compat       bool
}

func createDmvCodec() *dmvCodec {
	return &dmvCodec{}
}

var ErrBadGaussOption = errors.New("gauss must be 0 or 1")
var ErrBadPitch = errors.New("invalid pitch value")

func parsePitch(pitch string) (int, error) {
	var pitchValue int
	var parsed int
	var err error
	if strings.HasPrefix(pitch, "0x") {
		parsed, err = fmt.Sscanf(pitch, "%x", &pitchValue)
	} else {
		parsed, err = fmt.Sscanf(pitch, "%d", &pitchValue)
	}

	if parsed == 0 || pitchValue < 0x0001 || pitchValue > 0x3FFF || err != nil {
		return 0, ErrBadPitch
	}

	return pitchValue, nil
}

func (c *dmvCodec) Setopt(name string, value string) error {
	switch name {
	case "loop":
		if lp, err := strconv.Atoi(value); err != nil {
			return fmt.Errorf("%w: loop=%s", ErrInvalidCodecOptionValue, value)
		} else {
			c.loopPoint = lp
			c.hasLoop = c.loopPoint >= 0
		}
	case "gauss":
		if value != "0" && value != "1" {
			return fmt.Errorf("%w: gauss must be 0 or 1", ErrInvalidCodecOptionValue)
		}
		c.gaussEnabled = value == "1"
	case "pitch":
		if pitchValue, err := parsePitch(value); err != nil {
			return fmt.Errorf("%w: pitch=%s", ErrInvalidCodecOptionValue, value)
		} else {
			c.pitch = pitchValue
		}
	case "compat":
		if value != "0" && value != "1" {
			return fmt.Errorf("%w: compat must be 0 or 1", ErrInvalidCodecOptionValue)
		}
		c.compat = value == "1"
	default:
		return fmt.Errorf("%w: %s", ErrUnknownCodecOption, name)
	}

	return nil
}

func (c *dmvCodec) getLoopOpt() int {
	if !c.hasLoop {
		return -1
	} else {
		return c.loopPoint
	}
}

func (c *dmvCodec) Encode(pcmData []int16) []byte {
	output := []byte{}
	c.stats = EncodingStats{}

	compat := c.compat
	loopPoint := c.getLoopOpt()

	if loopPoint >= len(pcmData) {
		loopPoint = -1
	}

	if loopPoint >= 0 {
		start_align := (16 - (loopPoint & 15)) & 15
		loop_size := len(pcmData) - loopPoint
		end_align := loop_size

		for (end_align & 15) != 0 {
			end_align <<= 1
		}

		// remove the existing loop block from the alignment
		end_align -= loop_size

		// also include the loop_start alignment
		end_align += start_align

		if end_align != 0 {
			// TODO: make sure that loop_start + loop_size is the end of wav_data
			//wav_data.resize(wav_data.size() + end_align, 0);

			var src = loopPoint
			var dst = loopPoint + loop_size
			var end = loopPoint + loop_size + end_align

			for dst != end {
				pcmData = append(pcmData, pcmData[src])
				//wav_data[dst] = wav_data[src];
				dst++
				src++
			}

			// 16-sample align loop_start
			loopPoint += start_align
		}
	} else {
		for (len(pcmData) & 15) != 0 {
			pcmData = append(pcmData, 0)
		}
	}

	const base_adjust_rate float64 = 0.0004
	var adjust_rate float64 = base_adjust_rate
	var loop_block int = loopPoint / 16 // -1 (disabled) will also result in the first block
	var wimax int = len(pcmData) / 16
	var wi int = 0
	var best_samp [18]int16

	//best_samp[0] = 0; already inited
	//best_samp[1] = 0;

	//totalBlocks := wimax
	totalError := float64(0)
	avgError := float64(0)
	minError := float64(0)
	maxError := float64(0)

	for wi != wimax {
		var p = pcmData[wi*16:]
		var best_err float64 = 1e20
		var blk_samp [18]int16
		var best_data [9]uint8

		blk_samp[0] = best_samp[0]
		blk_samp[1] = best_samp[1]

		for filter := 0; filter <= 3; filter++ {
			// Only use filter 0 for the start block or loop block
			if filter != 0 {
				if (wi == 0) || (wi == loop_block) {
					continue
				}
			}

			// Ranges 0, 13, 14, 15 are "invalid", so they are not used for encoding.
			// The values produced by these ranges are fully covered by the other
			// range values, so there will be no loss in quality.
			for brange := 12; brange >= 1; brange-- {
				var rhalf int32 = (1 << brange) >> 1
				var blk_err float64 = 0
				var blk_data [16]uint8

				for n := 0; n < 16; n++ {
					//int16* blk_ls = blk_samp + n;
					var filter_s int32

					a, b := int32(blk_samp[n+1]), int32(blk_samp[n+0])
					switch filter {
					case 0:
						// Coefficients: 0, 0
						filter_s = 0

					case 1:
						// Coefficients: 15/16, 0
						filter_s = a + ((-a) >> 4)

					case 2:
						// Coefficients: 61/32, -15/16
						filter_s = a << 1                // add 64/32
						filter_s += -(a + (a << 1)) >> 5 // add (-3)/32
						filter_s += -b                   // add (-16)/16
						filter_s += b >> 4               // add 1/16

					case 3:
						filter_s = a << 1                           // add 128/64
						filter_s += -(a + (a << 2) + (a << 3)) >> 6 // add (-13)/64
						filter_s += -b                              // add (-16)/16
						filter_s += (b + (b << 1)) >> 4             // add 3/16

					default: // should never happen
						filter_s = 0
					}

					// undo 15 -> 16 bit conversion
					var xs int32 = int32(p[n]) >> 1

					// undo 16 -> 15 bit wrapping
					// check both possible 16-bit values
					var s1 int32 = int32(int16(xs & 0x7FFF))
					var s2 int32 = int32(int16(xs | 0x8000))

					// undo filtering
					s1 -= filter_s
					s2 -= filter_s

					// restore low bit lost during range decoding
					s1 <<= 1
					s2 <<= 1

					// reduce s to range with nearest value rounding
					// range = 2, rhalf = 2
					// s(-6, -5, -4, -3) = -1
					// s(-2, -1,  0,  1) =  0
					// s( 2,  3,  4,  5) =  1
					s1 = (s1 + rhalf) >> brange
					s2 = (s2 + rhalf) >> brange

					s1 = clamp(s1, 4)
					s2 = clamp(s2, 4)

					var rs1 uint8 = uint8(s1 & 0x0F)
					var rs2 uint8 = uint8(s2 & 0x0F)

					// -16384 to 16383
					s1 = (s1 << brange) >> 1
					s2 = (s2 << brange) >> 1

					// BRR accumulates to 17 bits, saturates to 16 bits, and then wraps to 15 bits

					if filter >= 2 {
						s1 = clamp(s1+filter_s, 16)
						s2 = clamp(s2+filter_s, 16)
					} else {
						// don't clamp - result does not overflow 16 bits
						s1 += filter_s
						s2 += filter_s
					}

					// wrap to 15 bits, sign-extend to 16 bits
					s1 = int32(int16(s1<<1) >> 1)
					s2 = int32(int16(s2<<1) >> 1)

					var d1 float64 = float64(xs - s1)
					var d2 float64 = float64(xs - s2)

					d1 *= d1
					d2 *= d2

					// If d1 == d2, prefer s2 over s1.
					if d1 < d2 {
						blk_err += d1
						blk_samp[n+2] = int16(s1)
						blk_data[n] = rs1
					} else {
						blk_err += d2
						blk_samp[n+2] = int16(s2)
						blk_data[n] = rs2
					}
				}

				// Use < for comparison. This will cause the encoder to prefer
				// less complex filters and higher ranges when error rates are equal.
				// This will then result in a slightly lower average error rate.
				if blk_err < best_err {
					best_err = blk_err

					for n := 0; n < 16; n++ {
						best_samp[n+2] = blk_samp[n+2]
					}
					best_data[0] = uint8((brange << 4) | (filter << 2))

					for n := 0; n < 8; n++ {
						best_data[n+1] = (blk_data[n*2] << 4) | blk_data[n*2+1]
					}
				}
			}
		}

		var overflow uint16 = 0

		for n := 0; n < 16; n++ {
			var b uint8 = testOverflow(best_samp[n:])
			overflow = (overflow << 1) | uint16(b)
		}

		if overflow != 0 {
			var f [16]float64

			for n := 0; n < 16; n++ {
				f[n] = adjust_rate
			}

			for n := 0; n < 16; n++ {
				if (overflow & 0x8000) != 0 {
					var t float64 = 0.05

					for i := n; i >= 0; i-- {
						f[i] *= 1.0 + t
						t *= 0.1
					}

					t = 0.05 * 0.1
					for i := n + 1; i < 16; i++ {
						f[i] *= 1.0 + t
						t *= 0.1
					}
				}
				overflow <<= 1
			}

			for n := 0; n < 16; n++ {
				p[n] = int16(float64(p[n]) * (1.0 - f[n]))
			}

			adjust_rate *= 1.1
		} else {
			adjust_rate = base_adjust_rate
			best_samp[0] = best_samp[16]
			best_samp[1] = best_samp[17]

			totalError += best_err

			if best_err < minError {
				minError = best_err
			}
			if best_err > maxError {
				maxError = best_err
			}
			output = append(output, best_data[:]...)

			wi += 1
		}
	}

	if wimax == 0 {
		minError = 0
	} else {
		avgError = totalError / float64(wimax)
	}

	// Original BrrCodec adds an additional block if the loop is not enabled (which doesn't make sense to me.)
	if len(output) == 0 || (compat && loopPoint < 0) {
		output = append(output, 0, 0, 0, 0, 0, 0, 0, 0, 0)
	}

	var last_header_set_bits uint8 = 1
	if loopPoint >= 0 {
		last_header_set_bits |= 2
	}
	output[len(output)-9] |= last_header_set_bits

	// if !bc.userPitchEnabled {
	// 	// 0.128 = 0x1000 / 32000
	// 	x := int(float64(bc.inputSampleRate)*0.128 + 0.5)

	// 	if x < 1 {
	// 		x = 1
	// 	} else if x > 0x3FFF {
	// 		x = 0x3FFF
	// 	}

	// 	bc.pitchStepBase = x
	// }

	c.stats = EncodingStats{
		AvgError:   avgError,
		MinError:   minError,
		MaxError:   maxError,
		TotalError: totalError,
	}

	return output
}

func (c *dmvCodec) Decode(brrData []byte) ([]int16, SampleRate) {

	compat := c.compat
	gaussEnabled := c.gaussEnabled
	outputSampleRate := SampleRate(32000)

	pitchStepBase := c.pitch
	if pitchStepBase == 0 {
		pitchStepBase = 0x1000
	}

	if !gaussEnabled {
		//7.8125 = 32000 / 0x1000

		outputSampleRate = SampleRate(float64(pitchStepBase)*7.8125 + 0.5)
	}

	pcmData := []int16{}

	if len(brrData) == 0 {
		return pcmData, outputSampleRate
	}

	// Length should be 9 bytes. Pad it if it isn't (corrupted data).
	for len(brrData)%9 != 0 {
		brrData = append(brrData, 0)
	}
	// Make sure that the last block has the "END" flag set to stop decoding.
	brrData[len(brrData)-9] |= 1

	data := 0
	sample := [8]int16{} // 4 samples stored twice
	last_sample := [2]int16{}
	var header uint8
	var samp_i int
	brr_counter := 1 // minus 1 = 0 to trigger next block decoding
	pitch := 0x3000  // decode 4 samples
	if !compat && !gaussEnabled {
		pitch = 0
	}

	for {
		for pitch >= 0 {
			pitch -= 0x1000

			brr_counter--
			if brr_counter == 0 {
				// End of block

				if header&1 != 0 {
					// End of sample (END set)
					return pcmData, outputSampleRate
				}

				header = brrData[data]
				data++
				brr_counter = 16

				if compat {
					// Original source returns here, but this doesn't make sense, given that
					// this may skip the data on the last block.
					if (header & 3) == 1 {
						return pcmData, outputSampleRate
					}
				}
			}

			brange := header >> 4
			filter := (header >> 2) & 3

			var samplePair uint8 = brrData[data]
			var s int

			// the high nybble is decoded before the low nybble
			if (brr_counter & 1) == 1 {
				data++
				s = int((samplePair&0x0F)^8) - 8
			} else {
				s = int((samplePair>>4)^8) - 8
			}

			if brange > 12 {
				s &= ^0x07FF // -2048 or 0
			} else {
				s = (s << brange) >> 1 // -16384 to 16383
			}

			// BRR accumulates to 17 bits, saturates to 16 bits, and then wraps to 15 bits

			switch filter {
			// last[1] * 15 / 16
			case 1:
				s += int(last_sample[1])       // add 16/16
				s += -int(last_sample[1]) >> 4 // add (-1)/16
				// don't clamp - result does not overflow 16 bits
			case 2:
				// last[1] * 61 / 32 - last[0] * 15 / 16
				s += int(last_sample[1]) << 1                                 // add 64/32
				s += -(int(last_sample[1]) + (int(last_sample[1]) << 1)) >> 5 // add (-3)/32
				s += -int(last_sample[0])                                     // add (-16)/16
				s += int(last_sample[0]) >> 4                                 // add 1/16
				s = clamp(s, 16)
			case 3:
				// last[1] * 115 / 64 - last[0] * 13 / 16
				s += int(last_sample[1]) << 1                                                              // add 128/64
				s += -(int(last_sample[1]) + (int(last_sample[1]) << 2) + (int(last_sample[1]) << 3)) >> 6 // add (-13)/64
				s += -int(last_sample[0])                                                                  // add (-16)/16
				s += (int(last_sample[0]) + (int(last_sample[0]) << 1)) >> 4                               // add 3/16
				s = clamp(s, 16)
			}

			s = int(int16(s<<1) >> 1) // wrap to 15 bits, sign-extend to 16 bits
			last_sample[0] = last_sample[1]
			last_sample[1] = int16(s) // 15-bit

			samp_i = (samp_i - 1) & 3   // do this before storing the sample
			sample[samp_i] = int16(s)   // 15-bit
			sample[samp_i+4] = int16(s) // store twice
		} // pitch loop

		var samp []int16 = sample[samp_i : samp_i+4]
		var s int

		if gaussEnabled {
			p := pitch >> 4
			np := -p
			G4 := int(kGaussTable[-1+np])
			G3 := int(kGaussTable[255+np])
			G2 := int(kGaussTable[512+p])
			G1 := int(kGaussTable[256+p])

			// p is always < 0 and >= -256
			// the first 3 steps wrap using a 15-bit accumulator
			// the last step accumulates to 16-bits then saturates to 15-bits

			s = ((G4) * int(samp[3])) >> 11
			s += ((G3) * int(samp[2])) >> 11
			s += ((G2) * int(samp[1])) >> 11
			s = int(int16(s<<1) >> 1)
			s += (G1) * int(samp[0]) >> 11
			s = clamp(s, 15)

			s = (s * 0x07FF) >> 11 // envx
			s = (s * 0x7F) >> 7    // volume

			pitch += pitchStepBase
		} else {
			s = int(samp[3])
			pitch += 0x1000
		}

		s <<= 1
		pcmData = append(pcmData, int16(s))
	}
}

func testGauss(G4, G3, G2 int, ls []int16) uint8 {
	s := (G4 * int(ls[0])) >> 11
	s += (G3 * int(ls[1])) >> 11
	s += (G2 * int(ls[2])) >> 11
	if s > 0x3FFF || s < -0x4000 {
		return 1
	}
	return 0
}

/*
There are 3 pitch values that can cause sign inversion in the gaussian
filtering by overflowing the 15-bit accumulator if the input samples are
too close to the min/max value.

The sum of the first 3 gauss_table values for each of these 3 pitch values
is 2049 while all other pitch values are 2048 or less.
*/

func testOverflow(ls []int16) uint8 {
	var r uint8

	// p = -256; gauss_table[255, 511, 256]
	r = testGauss(370, 1305, 374, ls)

	// p = -255; gauss_table[254, 510, 257]
	r |= testGauss(366, 1305, 378, ls)

	// p = -247; gauss_table[246, 502, 265]
	r |= testGauss(336, 1303, 410, ls)

	return r
}

func (c *dmvCodec) EncodingStats() EncodingStats {
	return EncodingStats{}
}

// Clamp an integer value to the given number of bits. e.g., clamp(x, 8) clamps to
// [0,255].
func clamp[T int8 | int16 | int32 | int64 | int](value T, bits int) T {
	var low T = -1 << (bits - 1)
	var high T = (1 << (bits - 1)) - 1

	if value > high {
		return high
	} else if value < low {
		return low
	}

	return value
}
