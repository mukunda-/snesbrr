// snesbrr
// Copyright 2025 Mukunda Johnson (mukunda.com)
// Licensed under MIT

// This implementation is based off of the excellent info in Fullsnes by no$ no$.
// All glories to Martin Korth.
// https://problemkaputt.de/fullsnes.htm#snesapudspbrrsamples

package brr

import (
	"fmt"
	"strconv"
)

type nocCodec struct {
	opts      map[string]string
	stats     EncodingStats
	loopPoint int
	hasLoop   bool
}

func createNocCodec() *nocCodec {
	return &nocCodec{}
}

func (c *nocCodec) Setopt(name string, value string) error {
	switch name {
	case "loop":
		if lp, err := strconv.Atoi(value); err != nil {
			return fmt.Errorf("%w: loop=%s", ErrInvalidCodecOptionValue, value)
		} else {
			c.loopPoint = lp
			c.hasLoop = c.loopPoint >= 0
		}
	default:
		return fmt.Errorf("%w: %s", ErrUnknownCodecOption, name)
	}

	return nil
}

func (c *nocCodec) getLoopOpt() int {
	if !c.hasLoop {
		return -1
	} else {
		return c.loopPoint
	}
}

func filterBase(prev1 int, prev2 int, filter int) int {
	// https://problemkaputt.de/fullsnes.htm#snesapudspbrrsamples
	//	More precisely, the exact formulas are:
	//   Filter 0: new = sample
	//   Filter 1: new = sample + old*1+((-old*1) SAR 4)
	//   Filter 2: new = sample + old*2+((-old*3) SAR 5)  - older+((older*1) SAR 4)
	//   Filter 3: new = sample + old*2+((-old*13) SAR 6) - older+((older*3) SAR 4)

	if filter == 1 {
		return prev1*1 + ((-prev1 * 1) >> 4)
	}
	if filter == 2 {
		return (prev1 * 2) + ((-prev1 * 3) >> 5) - prev2 + ((prev2 * 1) >> 4)
	}
	if filter == 3 {
		return (prev1 * 2) + ((-prev1 * 13) >> 6) - prev2 + ((prev2 * 3) >> 4)
	}

	return 0
}

// Encode a block. prev1 & prev2 are the previous two decoded 15-bit samples.
//
// noFilter forces use of filter 0, to avoid unexpected output for the start and loop
// point (when prev1 and prev2 are variable).
func (c *nocCodec) encodeBlock(pcmData []int16, prev1 int, prev2 int, noFilter bool,
	stats *EncodingStats) ([]byte, int, int) {

	bestOutput := []byte{}
	bestError := 0x7FFFFFFF
	bestPrev1 := 0
	bestPrev2 := 0

	filterEnd := 3
	if noFilter {
		filterEnd = 0
	}

	for filter := 0; filter <= filterEnd; filter++ {

		// Shift range = 1 + 0-11. Range 0 is unused.
		for shift := 11; shift >= 0; shift-- {
			half := 1 << shift >> 1

			fprev1 := prev1
			fprev2 := prev2

			foutput := []byte{byte(((shift + 1) << 4) | (filter << 2))}
			fErrorSum := 0
			failEncoding := false

			for p := 0; p < 16; p++ {
				desiredSample := int(pcmData[p]) >> 1

				base := filterBase(fprev1, fprev2, filter)
				delta := desiredSample - base

				brrSample := (delta + half) >> shift
				if brrSample < -8 {
					brrSample = -8
				} else if brrSample > 7 {
					brrSample = 7
				}

				for {
					nextDecodedSample := base + (brrSample << shift)

					// If the sample is out of range, try again with a lesser value. Note this
					// is naive and we could do more efficient math than
					// incrementing/decrementing.
					if nextDecodedSample < -0x3FFA {
						if brrSample < 7 {
							brrSample++
							continue
						} else {
							failEncoding = true
							break
						}
					} else if nextDecodedSample > 0x3FF8 {
						if brrSample > -8 {
							brrSample--
							continue
						} else {
							failEncoding = true
							break
						}
					}
					break
				}

				if failEncoding {
					break
				}

				nextDecodedSample := base + (brrSample << shift)
				// Todo: a better formula to determine severity of error?
				errAmount := (desiredSample - nextDecodedSample)
				if errAmount < 0 {
					errAmount = -errAmount
				}
				fErrorSum += errAmount
				foutput = append(foutput, byte(brrSample&0xF))
				fprev2 = fprev1
				fprev1 = nextDecodedSample
			}

			if failEncoding {
				continue
			}

			if fErrorSum < bestError {
				bestError = fErrorSum
				bestOutput = foutput
				bestPrev1 = fprev1
				bestPrev2 = fprev2
			}
		}
	}

	for i := 0; i < 8; i++ {
		bestOutput[1+i] = (bestOutput[1+i*2] << 4) | (bestOutput[1+i*2+1] & 0xF)
	}

	stats.TotalError += float64(bestError)
	stats.AvgError += float64(bestError / 16)

	return bestOutput[0:9], bestPrev1, bestPrev2
}

func (c *nocCodec) Encode(pcmData []int16) []byte {
	output := []byte{}
	c.stats = EncodingStats{}

	loopPoint := c.getLoopOpt()

	if loopPoint >= 0 {
		// Align loop start to 16 samples
		for loopPoint&15 != 0 {
			pcmData = append(pcmData, pcmData[loopPoint])
			loopPoint++
		}

		// Unroll loop to match 16 samples
		loopRegion := pcmData[loopPoint:]

		for len(pcmData)&15 != 0 {
			pcmData = append(pcmData, loopRegion...)
		}

	} else {
		// Pad end to 16 samples
		for len(pcmData)&15 != 0 {
			pcmData = append(pcmData, 0)
		}
	}

	prev1 := 0
	prev2 := 0

	for readPos := 0; readPos < len(pcmData); readPos += 16 {
		noFilter := readPos == 0 || readPos == loopPoint
		block, p1, p2 := c.encodeBlock(pcmData[readPos:readPos+16], prev1, prev2, noFilter, &c.stats)
		prev1 = p1
		prev2 = p2
		output = append(output, block...)
	}

	if len(output) == 0 {
		output = []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}
	}

	output[len(output)-9] |= 0x01

	if loopPoint >= 0 {
		output[len(output)-9] |= 0x02
	}

	return output
}

func (c *nocCodec) decodeBlock(block []byte, prev1 int, prev2 int) ([]int16, int, int) {
	output := []int16{}

	shift := block[0] >> 4
	filter := int((block[0] >> 2) & 0x03)

	for i := 0; i < 16; i++ {

		brrSample := int(block[1+(i>>1)])
		if i&1 == 0 {
			brrSample = (brrSample >> 4) & 0x0F
		} else {
			brrSample = brrSample & 0x0F
		}

		// Sign extend
		brrSample = int(int8(brrSample<<4) >> 4)

		unpackedSample := (brrSample << shift) >> 1

		base := filterBase(prev1, prev2, filter)
		nextDecodedSample := (base + unpackedSample)

		// Undefined behavior if nextDecodedSample is overflowing a sane range. We don't
		// clamp it in this implementation.

		pcmSample := nextDecodedSample << 1

		// Clamp output to 16 bit signed.
		if pcmSample < -0x8000 {
			pcmSample = -0x8000
		} else if pcmSample > 0x7FFF {
			pcmSample = 0x7FFF
		}
		output = append(output, int16(pcmSample))

		prev2 = prev1
		prev1 = nextDecodedSample
	}

	return output, prev1, prev2
}

func (c *nocCodec) Decode(brrData []byte) ([]int16, SampleRate) {
	output := []int16{}
	prev1 := 0
	prev2 := 0

	for i := 0; i < len(brrData); i += 9 {
		block := brrData[i : i+9]

		for len(block) != 9 {
			block = append(block, 0)
		}

		endOfData := block[0]&0x01 == 0x01

		var decodedBlock []int16
		decodedBlock, prev1, prev2 = c.decodeBlock(block, prev1, prev2)
		output = append(output, decodedBlock...)

		if endOfData {
			break
		}
	}

	return output, 32000
}

func (c *nocCodec) EncodingStats() EncodingStats {
	return c.stats
}
