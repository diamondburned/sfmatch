# sfmatch

![](https://gitlab.com/diamondburned/sfmatch/badges/himegoto/pipeline.svg?style=flat-square)
![](https://gitlab.com/diamondburned/sfmatch/badges/himegoto/coverage.svg?style=flat-square)


A library that allows matching input strings into structs using regex matches.

## Example

Assume this output from the `opusenc` program set into the `output` variable:

```
Encoding complete
-----------------------------------------------------
       Encoded: 4 minutes and 31.64 seconds
       Runtime: 4 seconds
                (67.91x realtime)
         Wrote: 3853633 bytes, 13582 packets, 275 pages
       Bitrate: 109.64 kbit/s (without overhead)
 Instant rates: 1.2 to 193.2 kbit/s
                (3 to 483 bytes per packet)
      Overhead: 3.39% (container+metadata)
```

To parse the above output:

```go
type opusenc struct {
	// sfmatch accepts the normal tag syntax.
	Encoded      string  `sfmatch:"Encoded: (.+)"`
	Runtime      string  `sfmatch:"Runtime: (.+)"`

	// You can also elide the tag key directly.
	RealtimeMult float32 `\((.+)x realtime\)`
	WroteBytes   uint64  `Wrote: (\d+) bytes`
	Bitrate      float32 `Bitrate: (.+) kbit/s \(without overhead\)`
	Overhead     float32 `Overhead: (.+)% \(container\+metadata\)`
}

// Compile once and reuse this.
m, err := sfmatch.Compile(&opusenc{})
if err != nil { return err }

var enc opusenc //    v above output
if err := m.Unmarshal(output, &enc); err != nil { return err }
```

## Important Details

Unmarshal does **not** type-check, thus the user should always make sure
whatever type goes into `Unmarshal` is the same as whatever was compiled.

Actually, you shouldn't even use this library in production.

## Supported types

The following types are supported:

- bool
- int, int8, int16, int32, int64
- uint, uint8, uint16, uint32, uint64
- float32, float64
- string
