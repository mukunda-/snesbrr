## snesbrr

A Bit-Rate-Reduction (BRR) codec for the SNES S-DSP.

### Overview

snesbrr encodes standard PCM wave files into BRR files that can be used by the S-DSP of
the SNES. It supports all valid combinations of BRR ranges and filters as well as optional
sample looping.

### Quickstart

```
snesbrr --help
snesbrr --encode input.wav output.brr
snesbrr --decode output.brr input-transcoded.wav
```

### Usage

```
snesbrr [options] input-file output-file

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
	gaussian filtering.
```

### Additional notes

* There are two included codecs. One is a port of DMV47's work. The other is based off of
  Fullsnes.

  * See [https://problemkaputt.de/fullsnes.htm#snesapudspbrrsamples](Fullsnes) for more information.

* The loop start point and the loop size should both be multiples of 16 in order to
  produce the smallest possible BRR files. Otherwise unrolling will take effect.

* Gaussian filtering and pitch shifting is supported by the DMV codec only. Choosing a
  higher sampling rate than needed for the source sample will improve the quality of the
  encoded sound. It will also help to offset the effects of gaussian filtering which can
  reduce the volume of the high frequencies (relative to the sampling rate). For example,
  a 4000 Hz square wave needs a minimum sampling rate of 8000 Hz. However, at this rate
  the decoded square wave will only reach about 27% of its original volume level due to
  the gaussian filtering. If the rate is increased to 16000 Hz, it will reach 89% volume.
  At 32000 Hz, 99% volume.

  * See [SNES APU DSP BRR Pitch](https://problemkaputt.de/fullsnes.htm#snesapudspbrrpitch)
    on Fullsnes for more info, especially about the gaussian filtering.
  
  * Simple waves can easily be manually encoded for improved quality and
    size. For example, at a pitch rate of 0x1000 the waves listed below
    can be generated using the provided BRR block data.

```
  1000 Hz Square Wave:
  B0 77 77 77 77 77 77 77 77
  B3 99 99 99 99 99 99 99 99
  1000 Hz Triangle Wave:
  B0 01 23 45 67 76 54 32 10
  B3 FE DC BA 98 89 AB CD EF
  1000 Hz Sawtooth Wave:
  B0 00 11 22 33 44 55 66 77
  B3 88 99 AA BB CC DD EE FF
```

* See the issues page for pending features and bugs.

### License/Credits

This project is licensed under MIT.

Based on the original snesbrr program by DMV47 which is licensed under "Common Development
and Distribution License, Version 1.0 only".
