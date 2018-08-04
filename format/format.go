// Package format is used for inferring the line format
// of a dataset with one line per data point.
package format

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Parse infers the line format of a dataset with one line per data point.
// It requires a separator e.g. `\t` and optionally a date format that time.Parse
// understands. Parse also returns a new io.Reader ready to consume again.
// Parsing strategy is to infer the line format of every line of input separately
// and return the most common line format.
func Parse(r io.Reader, separator rune, dateFormat string) (io.Reader, LineFormat) {
	var (
		lfs = make(map[string]int)
		rd  = bufio.NewReader(r)
		buf bytes.Buffer
		l   string
		err error
	)
	for {
		l, err = rd.ReadString('\n')
		if err == io.EOF {
			break
		}
		l = strings.TrimSpace(l)
		if len(l) == 0 {
			continue
		}
		buf.WriteString(l)
		buf.WriteByte('\n')
		lfs[parseLineFormat(l, separator, dateFormat)]++
	}
	mlf, _ := NewLineFormat(maxLineFormat(lfs), separator, dateFormat) // Can't error when produced with parseLineFormat
	return &buf, mlf
}

// LineFormat represents the format of a line of input
type LineFormat struct {
	ColTypes   []ColType
	Separator  rune
	DateFormat string

	HasFloats     bool // calculated by NewLineFormat once
	HasStrings    bool
	HasDateTimes  bool
	FloatCount    int
	StringCount   int
	DateTimeCount int
}

// ColType represents the type of a column in a data point (i.e. in a line of input)
type ColType int

// A column in a data point can either be a String (e.g. `series1`), a Float (e.g. 1.2) or a DateTime (e.g. 2006-01-02)
const (
	String ColType = iota
	Float
	DateTime
)

func (l LineFormat) String() string {
	var bs = make([]byte, 0)
	for _, c := range l.ColTypes {
		bs = append(bs, c.String()...)
	}
	return fmt.Sprintf("{colTypes: [%v], separator: [%v], dateFormat: [%v]}", string(bs), string(l.Separator), l.DateFormat)
}

func (c ColType) String() string {
	switch c {
	case String:
		return "s"
	case Float:
		return "f"
	case DateTime:
		return "d"
	default:
		return "?"
	}
}

// NewLineFormat creates a LineFormat from a string representing a line format with one rune
// per column, with syntax `[dfs]*` where d=datetime,f=float,s=string
func NewLineFormat(lineFormat string, separator rune, dateFormat string) (LineFormat, error) {
	if ok, err := regexp.Match("[dfs ]*", []byte(lineFormat)); !ok || err != nil {
		return LineFormat{}, fmt.Errorf("format: supplied lineFormat doesn't match syntax `[dfs ]*`")
	}
	var lf = LineFormat{ColTypes: nil, Separator: separator, DateFormat: dateFormat}

	for _, b := range lineFormat {
		switch b {
		case 's':
			lf.ColTypes = append(lf.ColTypes, String)
			lf.StringCount++
		case 'f':
			lf.ColTypes = append(lf.ColTypes, Float)
			lf.FloatCount++
		case 'd':
			lf.ColTypes = append(lf.ColTypes, DateTime)
			lf.DateTimeCount++
		default:
		}
	}
	lf.HasStrings = lf.StringCount > 0
	lf.HasFloats = lf.FloatCount > 0
	lf.HasDateTimes = lf.DateTimeCount > 0
	return lf, nil
}

func maxLineFormat(lfs map[string]int) string {
	max := 0
	lf := ""
	for k, v := range lfs {
		if v > max {
			max = v
			lf = k
		}
	}
	return lf
}

func parseLineFormat(s string, sep rune, df string) string {
	s = string(regexp.MustCompile(string(sep)+"{2,}").ReplaceAll([]byte(s), []byte(string(sep))))
	ss := strings.Split(strings.TrimSpace(s), string(sep))
	lf := ""
	for _, sc := range ss {
		if _, err := strconv.ParseFloat(sc, 64); err == nil {
			lf += "f"
		} else if _, err := time.Parse(df, sc); err == nil && strings.TrimSpace(sc) != "" {
			lf += "d"
		} else {
			lf += "s"
		}
	}
	return lf
}
