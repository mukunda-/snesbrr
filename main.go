package main

import (
	"flag"
	"fmt"
	"os"

	"go.mukunda.com/snesbrr/brr"
)

func printUsage(short bool) {
	fmt.Println("Usage: [options] input-file output-file")
	if short {
		fmt.Println("Use --help for options help.")
		return
	}

	fmt.Println(`
Options
-------
-?, --help
	Print a summary of the command line options and exit.

-e, --encode
	Encode WAV (input-file) to BRR (output-file).
	Valid options: '--loop-start'.

-d, --decode
	Decode BRR (input-file) to WAV (output-file).
	Valid options: '--pitch', '--enable-gauss'.

-l START, --loop-start START
	Set the loop start sample to the decimal value START. If this value
	or the size of the loop is not a multiple of 16, the looped samples
	will be repeated until both the start and end points are multiples
	of 16. Looping is disabled by default.

-p PITCH, --pitch PITCH
	Set the pitch rate to the hexadecimal value PITCH. This value is
	only used during decoding. It must be between 0x0001 and 0x3FFF.
	The default value is 0x1000.

-g, --enable-gauss
	Enable gaussian filtering during decoding. This setting can be
	used to simulate the decoding process of an SNES in the output.
`)
}

func main() {
	help := flag.Bool("help", false, "Display this help message")
	flag.BoolVar(help, "?", false, "Display this help message")

	encode := flag.Bool("encode", false, "Encode WAV -> BRR")
	flag.BoolVar(encode, "e", false, "Encode WAV -> BRR")

	decode := flag.Bool("decode", false, "Decode BRR -> WAV")
	flag.BoolVar(decode, "d", false, "Decode BRR -> WAV")

	loopStart := flag.Int("loop-start", 0, "Set the loop start sample")
	flag.IntVar(loopStart, "l", 0, "Set the loop start sample")

	pitch := flag.Int("pitch", 0.0, "Set the pitch rate for decoding")
	flag.IntVar(pitch, "p", 0.0, "Set the pitch rate for decoding")

	enableGauss := flag.Bool("enable-gauss", false, "Enable gaussian filtering during decoding")
	flag.BoolVar(enableGauss, "g", false, "Enable gaussian filtering during decoding")

	flag.Parse()

	if help != nil {
		printUsage(false)
		os.Exit(0)
	}

	if len(flag.Args()) < 1 {
		print("No input supplied.")
		printUsage(true)
		os.Exit(1)
	}

	if encode == nil && decode == nil {
		print("Must specify --encode or --decode. See --help for options usage.")
		os.Exit(1)
	}

	if encode != nil && decode != nil {
		print("Error: both --encode and --decode used.")
		os.Exit(1)
	}

	encoding := *encode

	inputFile := flag.Arg(0)
	outputFile := flag.Arg(1)

	if outputFile == "" {
		if encoding {
			outputFile = inputFile + ".brr"
		} else {
			outputFile = inputFile + ".wav"
		}

		if _, err := os.Stat(outputFile); err == nil {
			fmt.Printf("Error: output file %s already exists.\n", outputFile)
			os.Exit(1)
		}
	}

	codec := brr.NewBrrCodec()

	if encoding {

	}

	fmt.Printf("Input File: %s\n", inputFile)
	fmt.Printf("Output File: %s\n", outputFile)
	fmt.Printf("Encode: %v\n", *encode)
	fmt.Printf("Decode: %v\n", *decode)
	fmt.Printf("Loop Start: %d\n", *loopStart)
	fmt.Printf("Pitch: %f\n", *pitch)
	fmt.Printf("Enable Gauss: %v\n", *enableGauss)
}
