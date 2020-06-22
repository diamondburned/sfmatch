package sfmatch

import (
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"strings"

	"github.com/pkg/errors"
)

type primitiveParser = func(string, reflect.Value) error

var ErrUnsupportedKind = errors.New("Unsupported kind")

// primitives only
func typeParser(kind reflect.Kind, input string, v reflect.Value) error {
	var canSet = v.CanSet()

	switch kind {
	case reflect.Bool:
		// Ignore if we can't set the value. Likely this is just a construction.
		if !canSet {
			return nil
		}

		b, err := strconv.ParseBool(input)
		if err != nil {
			return err
		}
		v.SetBool(b)

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if !canSet {
			return nil
		}

		i, err := strconv.ParseInt(input, 10, 64)
		if err != nil {
			return err
		}
		v.SetInt(i)

	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		if !canSet {
			return nil
		}

		u, err := strconv.ParseUint(input, 10, 64)
		if err != nil {
			return err
		}
		v.SetUint(u)

	case reflect.Float32, reflect.Float64:
		if !canSet {
			return nil
		}

		f, err := strconv.ParseFloat(input, 64)
		if err != nil {
			return err
		}
		v.SetFloat(f)

	case reflect.String:
		if !canSet {
			return nil
		}
		v.SetString(input)

	default:
		return ErrUnsupportedKind
	}

	return nil
}

type Match struct {
	regex   *regexp.Regexp
	indices []int
	kinds   []reflect.Kind
	vtype   reflect.Type
}

// Compile compiles the structure into a regex delimited with [\s\S]*.
func Compile(structure interface{}) (*Match, error) {
	return CompileWithDelimiter(structure, "[\\s\\S]*")
}

func MustCompile(structure interface{}) *Match {
	m, err := Compile(structure)
	if err != nil {
		panic(err)
	}
	return m
}

func CompileWithDelimiter(structure interface{}, delim string) (*Match, error) {
	t := reflect.TypeOf(structure)

	// If the given type is a pointer, then we should dereference that and the
	// value.
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	n := t.NumField()

	var fields = make([]int, 0, n)
	var kinds = make([]reflect.Kind, 0, n)

	regex := strings.Builder{}
	regex.WriteString("(?mU)") // non-greedy

	for i := 0; i < n; i++ {
		ft := t.Field(i)

		// Check if the field is exported, which it is if PkgPath is empty.
		if ft.PkgPath != "" {
			continue
		}

		// Write the regex.
		tg, ok := ft.Tag.Lookup("sfmatch")
		if !ok {
			tg = string(ft.Tag)
		}

		// Should we skip this field? Yes if it's a dash or is nothing.
		if tg == "-" || tg == "" {
			continue
		}

		// Test if the kind is supported.
		fk := ft.Type.Kind()

		// Test against the function. We can ignore all other errors, as it's
		// most likely reflect being unable to set the field.
		if err := typeParser(fk, "", reflect.Value{}); err == ErrUnsupportedKind {
			return nil, fmt.Errorf("Failed to use field %s: %w", ft.Name, err)
		}

		// Write the regex separator.
		regex.WriteString(delim)
		// Write the actual specified regex.
		regex.WriteString(tg)
		// Recognize the field.
		fields = append(fields, i)
		kinds = append(kinds, fk)
	}

	// Stringify the regex and try compiling it.
	r, err := regexp.Compile(regex.String())
	if err != nil {
		return nil, errors.Wrap(err, "Failed to compile the regex")
	}

	// Confirm that we have enough matching groups.
	if r.NumSubexp() != len(fields) {
		return nil, errors.New("Mismatch field count and submatch count")
	}

	return &Match{
		regex:   r,
		indices: fields,
		kinds:   kinds,
		vtype:   t,
	}, nil
}

// Unmarshal regex-matches the given data and unmarshals it into value. It does
// NOT type-check value, thus reflect will panic if the type mismatches.
func (m *Match) Unmarshal(data string, value interface{}) error {
	s := m.regex.FindStringSubmatch(data)
	if s == nil {
		return errors.New("No matches found")
	}

	v := reflect.ValueOf(value).Elem()

	for i, j := range m.indices {
		// add 1 to i because match 0 is the entire match
		if err := typeParser(m.kinds[i], s[i+1], v.Field(j)); err != nil {
			return errors.Wrapf(err, "Failed to parse field %d", j)
		}
	}

	return nil
}
