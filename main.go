package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"

	"go.mukunda.com/snesbrr/brr"
)

// ---------------------------------------------------------------------------------------
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
	Set the pitch rate to the value PITCH. This value is only used
	during decoding. It must be between 0x0001 and 0x3FFF. The default 
	value is 0x1000.

-g, --enable-gauss
	Enable gaussian filtering during decoding. This setting can be
	used to simulate the decoding process of an SNES in the output.`)
}

var ErrBadPitch = errors.New("invalid pitch value")

type programArgs struct {
	Help      bool
	Encode    bool
	LoopStart int
	Pitch     int
	Gauss     bool
}

// ---------------------------------------------------------------------------------------
func parsePitch(pitch string) (int, error) {
	var pitchValue int
	var parsed int
	var err error
	if strings.HasPrefix(pitch, "0x") {
		parsed, err = fmt.Sscanf(pitch, "%x", &pitchValue)
		if parsed == 0 || err != nil {
			return 0, ErrBadPitch
		}
	} else {
		parsed, err = fmt.Sscanf(pitch, "%d", &pitchValue)
	}
	if parsed == 0 || err != nil {
		return 0, ErrBadPitch
	}
	return pitchValue, nil
}

// ---------------------------------------------------------------------------------------
func parseArgs() programArgs {

	args := programArgs{}

	flag.BoolVar(&args.Help, "help", false, "Display this help message")
	flag.BoolVar(&args.Help, "?", false, "Display this help message")

	flag.BoolVar(&args.Encode, "encode", false, "Encode WAV -> BRR")
	flag.BoolVar(&args.Encode, "e", false, "Encode WAV -> BRR")

	var decode bool
	flag.BoolVar(&decode, "decode", false, "Decode BRR -> WAV")
	flag.BoolVar(&decode, "d", false, "Decode BRR -> WAV")

	flag.IntVar(&args.LoopStart, "loop-start", -1, "Set the loop start sample")
	flag.IntVar(&args.LoopStart, "l", -1, "Set the loop start sample")

	var pitch string
	flag.StringVar(&pitch, "pitch", "", "Set the pitch rate for decoding")
	flag.StringVar(&pitch, "p", "", "Set the pitch rate for decoding")

	flag.BoolVar(&args.Gauss, "enable-gauss", false, "Enable gaussian filtering during decoding")
	flag.BoolVar(&args.Gauss, "g", false, "Enable gaussian filtering during decoding")

	flag.Parse()

	if args.Help {
		printUsage(false)
		os.Exit(0)
	}

	if len(flag.Args()) < 1 {
		print("No input supplied.")
		printUsage(true)
		os.Exit(0)
	}

	if !args.Encode && !decode {
		print("Must specify --encode or --decode. See --help for options usage.")
		os.Exit(1)
	}

	if args.Encode && decode {
		print("Error: both --encode and --decode used.")
		os.Exit(1)
	}

	if pitch != "" {
		pitchValue, err := parsePitch(pitch)
		if err != nil {
			fmt.Printf("Error: invalid pitch value %s\n", pitch)
			os.Exit(1)
		}
		args.Pitch = pitchValue
	}

	return args
}

// ---------------------------------------------------------------------------------------
func main() {
	args := parseArgs()

	inputFile := flag.Arg(0)
	outputFile := flag.Arg(1)

	if outputFile == "" {
		if args.Encode {
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

	if args.Pitch != 0 {
		codec.SetPitch(args.Pitch)
	}

	if args.Gauss {
		codec.SetGaussEnabled(true)
	}

	if args.LoopStart >= 0 {
		codec.SetLoop(args.LoopStart)
	}

	if args.Encode {
		if err := codec.ReadWavFile(inputFile); err != nil {
			fmt.Printf("Error loading input. %v\n", err)
			os.Exit(1)
		}

		codec.Encode()
		if err := codec.WriteBrrFile(outputFile); err != nil {
			fmt.Printf("Error creating output file. %v\n", err)
			os.Exit(1)
		}
	} else {
		if err := codec.ReadBrrFile(inputFile); err != nil {
			fmt.Printf("Error loading input file. %v\n", err)
			os.Exit(1)
		}

		codec.Decode()

		if err := codec.WriteWavFile(outputFile); err != nil {
			fmt.Printf("Error writing output. %v\n", err)
			os.Exit(1)
		}
	}
}
