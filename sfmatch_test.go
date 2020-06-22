package sfmatch

import (
	"fmt"
	"reflect"
	"strings"
	"testing"
)

type opusenc struct {
	Empty         string
	MoreEmpty     string `-`
	EvenMoreEmpty string `sfmatch:"-"`

	Encoded      string  `sfmatch:"Encoded: (.+)"`
	Runtime      string  `sfmatch:"Runtime: (.+)"`
	RealtimeMult float32 `\((.+)x realtime\)`

	// put in the middle; this should be ignored
	nonExported string

	WroteBytes uint64  `Wrote: (\d+) bytes`
	Bitrate    float32 `Bitrate: (.+) kbit/s \(without overhead\)`
	Overhead   float32 `Overhead: (.+)% \(container\+metadata\)`
}

const opusencOutput = `
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
`

func TestMatch(t *testing.T) {
	m, err := Compile((*opusenc)(nil))
	assertShouldErr(t, err, "")

	var enc opusenc
	assertShouldErr(t, m.Unmarshal(opusencOutput, &enc), "")

	expects := opusenc{
		Encoded:      "4",
		Runtime:      "4",
		RealtimeMult: 67.91,
		WroteBytes:   3853633,
		Bitrate:      109.64,
		Overhead:     3.39,
	}

	if !reflect.DeepEqual(expects, enc) {
		t.Fatalf("Unexpected output: %#v", enc)
	}
}

func TestMustCompile(t *testing.T) {
	defer func() {
		if err := recover(); err == nil {
			return
		}
	}()
	MustCompile((*opusenc)(nil))
}

func TestMustCompilePanic(t *testing.T) {
	// This WILL panic.

	defer func() {
		if err := recover(); !strings.Contains(fmt.Sprint(err), "Failed to use field") {
			t.Fatal("Unexpected panic:", err)
		}
	}()

	var panicpls struct {
		UnsupportedType struct{} `sfmatch:"a"`
	}
	MustCompile(&panicpls)
}

func TestAllTypes(t *testing.T) {
	var allTypes struct {
		Bool   bool    `sfmatch:"(\\S+)"`
		Int    int     `sfmatch:"(\\S+)"`
		Uint   uint    `sfmatch:"(\\S+)"`
		Float  float64 `sfmatch:"(\\S+)"`
		String string  `sfmatch:"(\\S+)$"` // $ matches to end of string
	}

	m, err := CompileWithDelimiter(&allTypes, " ?")
	assertShouldErr(t, err, "")

	err = m.Unmarshal("true -0 42 420.69 test", &allTypes)
	assertShouldErr(t, err, "")

	assertTrue(t, allTypes.Bool, "bool")
	assertTrue(t, allTypes.Int == 0, "int")
	assertTrue(t, allTypes.Uint == 42, "uint")
	assertTrue(t, allTypes.Float == 420.69, "float")
	assertTrue(t, allTypes.String == "test", "string")

	assertTrue(t, m.Unmarshal("nope 111 243 .1 string", &allTypes) != nil, "invalid bool")
	assertTrue(t, m.Unmarshal("true not 243 .1 string", &allTypes) != nil, "invalid int")
	assertTrue(t, m.Unmarshal("true 111 -43 .1 string", &allTypes) != nil, "negative uint")
	assertTrue(t, m.Unmarshal("true 111 243 ff string", &allTypes) != nil, "invalid float")
}

func TestMatchFail(t *testing.T) {
	var fail1 struct {
		UnsupportedType struct{} `sfmatch:"valid regex"`
	}

	_, err := Compile(&fail1)
	assertShouldErr(t, err, "Failed to use field")

	var fail2 struct {
		InvalidRegexp string `sfmatch:"["`
	}

	_, err = Compile(&fail2)
	assertShouldErr(t, err, "Failed to compile the regex")

	var fail3 struct {
		NoMatches string `sfmatch:"asdasd"`
	}

	_, err = Compile(&fail3)
	assertShouldErr(t, err, "Mismatch field count and submatch count")

	var fail4 struct {
		TooManyMatches string `sfmatch:"(asdasd)(sadasdasd)"`
	}

	_, err = Compile(&fail4)
	assertShouldErr(t, err, "Mismatch field count and submatch count")
}

func TestUnmarshalFail(t *testing.T) {
	var nomatch struct {
		Field string `sfmatch:"(astolfo)"`
	}

	m, err := Compile(&nomatch)
	assertShouldErr(t, err, "")

	err = m.Unmarshal("himegoto", &nomatch)
	assertShouldErr(t, err, "No matches found")
}

func TestInvalidInput(t *testing.T) {
	var invalid struct {
		Boat float64 `sfmatch:"(.*)"`
	}

	m, err := Compile(&invalid)
	assertShouldErr(t, err, "")

	err = m.Unmarshal("not a float lol", &invalid)
	assertShouldErr(t, err, "Failed to parse field 0")
}

func assertTrue(t *testing.T, cond bool, desc string) {
	t.Helper()
	if !cond {
		t.Fatal("Unexpected false on", desc)
	}
}

func assertShouldErr(t *testing.T, err error, contains string) {
	t.Helper()

	if contains == "" {
		if err == nil {
			return
		}
	} else {
		if err != nil && strings.Contains(err.Error(), contains) {
			return
		}
	}

	t.Fatal("Unexpected error:", err)
}
