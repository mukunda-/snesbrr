// snesbrr
// Copyright 2025 Mukunda Johnson (mukunda.com)
// Licensed under MIT

/*
A Bit-Rate-Reduction (BRR) codec for the SNES S-DSP.

Basic usage:

	snesbrr --help
	snesbrr --encode input.wav output.brr
	snesbrr --decode output.brr input-transcoded.wav
*/
package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"

	"go.mukunda.com/snesbrr/v2/brr"
)

type returnCode = int

const kUsage = `
Command Line Options
--------------------
-?, --help
  Show this help.

-e, --encode
   Encoding mode. The input (WAV file) will be encoded to
   BRR and saved to the output file in raw BRR format.

-d, --decode
   Decoding mode. The input (raw BRR file) will be decoded
   and saved to the output file in WAV format.

-l START, --loop START
   Specify the starting sample index of the loop with the
   decimal value START. If the loop start or loop length is
   not divisible by 16 (BRR block size), then the loop will
   be unrolled to align, increasing the output size. Looping
   is disabled by default.

--codec noc|dmv
   Sets the codec implementation to "dmv" or "noc". Uses
	"noc" by default, which is a newer implementation. "dmv"
	is a direct port of snesbrr by DMV47 which may contain
	some bugs.

--opt OPT=VALUE
   Set the codec option OPT to VALUE. See below. if =VALUE
	is omitted, it is treated as "1".

Codec Options
-------------

compat = 1 | 0 (default: 0)
   For the dmv codec only. Setting it to "1" will emulate
   the bugs in the original dmv codec.

gauss = 1 | 0 (default: 0)
   For the dmv codec only, setting it to "1" will apply a
   gaussian filter, simulating how the SNES sounds.

pitch = 0x0001-0x3FFF (default: 0x1000)
   For the dmv codec only. This sets a pitch rate for the
   output to the hexadecimal number. This interacts with the
	gaussian filtering.`

func printUsage(short bool) {
	fmt.Println("Usage: [options] input-file output-file")
	if short {
		fmt.Println("Use --help for options help.")
		return
	}

	fmt.Println(kUsage)
}

type codecOptions []string

func (s *codecOptions) String() string {
	return strings.Join(*s, ",")
}

func (s *codecOptions) Set(value string) error {
	*s = append(*s, value)
	return nil
}

type programArgs struct {
	InputFile  string
	OutputFile string
	Help       bool
	Encode     bool
	Decode     bool
	Loop       int
	Opts       codecOptions
	Codec      string
}

var ErrShowHelp = errors.New("show help")
var ErrInvalidArgs = errors.New("invalid arguments")

func parseArgs(argSet []string) (programArgs, error) {
	args := programArgs{}

	flagSet := flag.NewFlagSet(os.Args[0], flag.ContinueOnError)

	flagSet.BoolVar(&args.Help, "help", false, "Display this help message")
	flagSet.BoolVar(&args.Help, "?", false, "Display this help message")

	flagSet.BoolVar(&args.Encode, "encode", false, "Encode WAV -> BRR")
	flagSet.BoolVar(&args.Encode, "e", false, "Encode WAV -> BRR")

	flagSet.BoolVar(&args.Decode, "decode", false, "Decode BRR -> WAV")
	flagSet.BoolVar(&args.Decode, "d", false, "Decode BRR -> WAV")

	flagSet.IntVar(&args.Loop, "loop", -1, "Set the loop start sample")
	flagSet.IntVar(&args.Loop, "l", -1, "Set the loop start sample")

	flagSet.StringVar(&args.Codec, "codec", "", "Set the codec implementation")

	flagSet.Var(&args.Opts, "opt", "Set codec options in the form OPT=VALUE")

	err := flagSet.Parse(argSet)

	if err == nil {
		args.InputFile = flagSet.Arg(0)
		args.OutputFile = flagSet.Arg(1)
	}

	return args, err
}

func parseCodecOpt(opt string) (string, string) {
	key, value, found := strings.Cut(opt, "=")
	key = strings.TrimSpace(key)
	key = strings.ToLower(key)

	if !found {
		value = "1"
	} else {
		value = strings.TrimSpace(value)
	}

	return key, value
}

func run(cliArgs []string) returnCode {
	args, argsErr := parseArgs(cliArgs)

	if args.Help || errors.Is(argsErr, flag.ErrHelp) {
		printUsage(false)
		return 0
	}

	if argsErr != nil {
		fmt.Printf("Error: %v\n", argsErr)
	}

	if args.InputFile == "" {
		fmt.Println("No input supplied.")
		printUsage(true)
		return 0
	}

	if !args.Encode && !args.Decode {
		fmt.Println("Error: Must specify --encode or --decode. See --help for options usage.")
		return 1
	}

	if args.Encode && args.Decode {
		fmt.Println("Error: both --encode and --decode used.")
		return 1
	}

	if args.OutputFile == "" {
		if args.Encode {
			args.OutputFile = args.InputFile + ".brr"
		} else {
			args.OutputFile = args.InputFile + ".wav"
		}

		if _, err := os.Stat(args.OutputFile); err == nil {
			fmt.Printf("Error: output file %s already exists.\n", args.OutputFile)
			return 1
		}
	}

	codec := brr.NewCodec()

	if args.Codec != "" {
		err := codec.SetCodecImplementation(args.Codec)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			return 1
		}
	}

	if args.Loop >= 0 {
		codec.SetLoop(args.Loop)
	}

	for _, opt := range args.Opts {
		key, value := parseCodecOpt(opt)
		if err := codec.SetCodecOption(key, value); err != nil {
			fmt.Printf("Error: %v\n", err)
			return 1
		}
	}

	if args.Encode {
		if err := codec.ReadWavFile(args.InputFile); err != nil {
			fmt.Printf("Error loading input. %v\n", err)
			return 1
		}

		codec.Encode()

		if err := codec.WriteBrrFile(args.OutputFile); err != nil {
			fmt.Printf("Error creating output file. %v\n", err)
			return 1
		}
	} else {
		if err := codec.ReadBrrFile(args.InputFile); err != nil {
			fmt.Printf("Error loading input file. %v\n", err)
			return 1
		}

		codec.Decode()

		if err := codec.WriteWavFile(args.OutputFile); err != nil {
			fmt.Printf("Error writing output. %v\n", err)
			return 1
		}
	}

	return 0
}

func main() {
	os.Exit(run(os.Args[1:]))
}
