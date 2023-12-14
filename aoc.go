// Package aoc are quick & dirty utilities for helping Brad
// solve Advent of Code problems.
package aoc

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"unicode"

	"golang.org/x/exp/constraints"
)

var flagDay *string

var (
	puzzles      []string
	puzzleByName = map[string]func() any{} // func name -> func
	sampleInput  = map[string]string{}
	sampleWant   = map[string]string{}
)

var (
	curDay   int
	altInput []byte // non-nil to run a sample
)

func Main() {
	flagDay = flag.String("day", "", "func name to run; empty string means latest registered. If it starts with a digit, then \"day\" prefix is assumed.")
	flag.Parse()

	funcName := *flagDay
	if funcName == "" {
		funcName = puzzles[len(puzzles)-1]
	}
	if unicode.IsDigit(rune(funcName[0])) {
		funcName = "day" + funcName
	}

	f, ok := puzzleByName[funcName]
	if !ok {
		log.Fatalf("puzzle func %v not registered", funcName)
	}
	getDay := regexp.MustCompile(`\d+`)
	if m := getDay.FindStringSubmatch(funcName); m == nil {
		log.Fatalf("no digits in func name %q from which to extract day number", *flagDay)
	} else {
		curDay = Int(m[0])
	}
	if want, ok := sampleWant[funcName]; ok {
		altInput = []byte(sampleInput[funcName])
		got := fmt.Sprint(f())
		if got != want {
			fmt.Fprintf(os.Stderr, "❌ for %v sample, got=%v; want %v\n", funcName, got, want)
			os.Exit(1)
		}
		fmt.Fprintf(os.Stderr, "OK sample result.\n")
	} else {
		fmt.Fprintf(os.Stderr, "⚠️ no sample for %v\n", funcName)
	}
	altInput = nil
	v := f()
	fmt.Println(v)
}

func ExtractSamples(src []byte) {
	fs := token.NewFileSet()
	f, err := parser.ParseFile(fs, "aoc.go", src, parser.ParseComments)
	if err != nil {
		log.Fatalf("parsing source to extract samples: %v", err)
	}
	var lastInput string
	wantRx := regexp.MustCompile(`(?sm)^\s*want=([^\n]*)(?:\s+(.+\n))?\s*`)
	for _, d := range f.Decls {
		fd, ok := d.(*ast.FuncDecl)
		if !ok || fd.Doc == nil {
			continue
		}
		funcName := fd.Name.Name
		for _, c := range fd.Doc.List {
			text := strings.TrimPrefix(c.Text, "//")
			if v, ok := strings.CutPrefix(text, "/*"); ok {
				text = strings.TrimSuffix(v, "*/")
			}
			if m := wantRx.FindStringSubmatch(text); m != nil {
				sampleWant[funcName] = m[1]
				in := Or(m[2], lastInput)
				sampleInput[funcName] = in
				lastInput = in
			}
		}
	}
}

func funcName(f func() any) string {
	rv := reflect.ValueOf(f)
	rf := runtime.FuncForPC(rv.Pointer())
	if rf == nil {
		panic("no func found")
	}
	return strings.TrimPrefix(rf.Name(), "main.")
}

func Add(puzFuncs ...func() any) {
	for _, f := range puzFuncs {
		name := funcName(f)
		puzzles = append(puzzles, name)
		puzzleByName[name] = f
	}
}

type Pt2[T constraints.Signed] struct {
	X, Y T
}

type Pt3[T constraints.Signed] struct {
	X, Y, Z T
}

type Pt = Pt2[int]

func (p Pt2[T]) ForNeighbors(f func(Pt2[T]) (keepGoing bool)) {
	for y := T(-1); y <= 1; y++ {
		for x := T(-1); x <= 1; x++ {
			if x == 0 && y == 0 {
				continue
			}
			if !f(Pt2[T]{p.X + x, p.Y + y}) {
				return
			}
		}
	}
}

type Pt3Int = Pt3[int]

func AbsInt[T constraints.Signed](x, y T) T {
	v := x - y
	if v < 0 {
		v = -v
	}
	return v
}

// MDist returns the manhattan distance between a and b.
func (a Pt2[T]) MDist(b Pt2[T]) T {
	return AbsInt[T](a.X, b.X) + AbsInt[T](a.Y, b.Y)
}

// Toward returns a point moving from p to b in max 1 step in the X
// and/or Y direction.
func (p Pt2[T]) Toward(b Pt2[T]) Pt2[T] {
	p1 := p
	if b.X < p.X {
		p1.X--
	} else if b.X > p.X {
		p1.X++
	}
	if b.Y < p.Y {
		p1.Y--
	} else if b.Y > p.Y {
		p1.Y++
	}
	return p1
}

func (p Pt2[T]) North() Pt2[T] { return Pt2[T]{p.X, p.Y - 1} }
func (p Pt2[T]) South() Pt2[T] { return Pt2[T]{p.X, p.Y + 1} }
func (p Pt2[T]) West() Pt2[T]  { return Pt2[T]{p.X - 1, p.Y} }
func (p Pt2[T]) East() Pt2[T]  { return Pt2[T]{p.X + 1, p.Y} }

func sliceOf[T any](v ...T) []T { return v }

var NorthCounterClockwise = sliceOf(
	Pt2[int].North,
	Pt2[int].West,
	Pt2[int].South,
	Pt2[int].East,
)

var NorthClockwise = sliceOf(
	Pt2[int].North,
	Pt2[int].East,
	Pt2[int].South,
	Pt2[int].West,
)

func Input() []byte {
	if altInput != nil {
		return altInput
	}
	filename := fmt.Sprintf("%d.input", curDay)
	f, err := os.ReadFile(filename)
	if err == nil {
		return f
	}
	session := MustGet(os.ReadFile(filepath.Join(os.Getenv("HOME"), "keys", "aoc.session")))
	req := MustGet(http.NewRequest("GET", fmt.Sprintf("https://adventofcode.com/2023/day/%d/input", curDay), nil))
	req.AddCookie(&http.Cookie{Name: "session", Value: strings.TrimSpace(string(session))})
	res := MustGet(http.DefaultClient.Do(req))
	if res.StatusCode != 200 {
		log.Fatalf("bad status: %v", res.Status)
	}
	f = MustGet(io.ReadAll(res.Body))
	MustDo(os.WriteFile(filename, f, 0644))
	return f
}

func Scanner() *bufio.Scanner {
	return bufio.NewScanner(bytes.NewReader(Input()))
}

func Int(s string) int {
	return MustGet(strconv.Atoi(s))
}

// MustDo panics if err is non-nil.
func MustDo(err error) {
	if err != nil {
		panic(err)
	}
}

// MustGet returns v as is. It panics if err is non-nil.
func MustGet[T any](v T, err error) T {
	if err != nil {
		panic(err)
	}
	return v
}

// ForLines calls onLine for each line of input.
// The y value is the row number, starting with 0.
func ForLines(onLine func(line string)) {
	ForLinesY(func(_ int, line string) { onLine(line) })
}

// ForLines calls onLine for each line of input.
// The y value is the row number, starting with 0.
func ForLinesY(onLine func(y int, line string)) {
	s := Scanner()
	y := -1
	for s.Scan() {
		y++
		onLine(y, s.Text())
	}
	if err := s.Err(); err != nil {
		log.Fatal(err)
	}
}

func DigVal(b byte) int {
	if b >= '0' && b <= '9' {
		return int(b - '0')
	}
	panic(fmt.Sprintf("bogus digit %q", string(b)))
}

// Or returns the first non-zero element of list, or else returns the zero T.
//
// This is the proposal from
// https://github.com/golang/go/issues/60204#issuecomment-1581245334.
func Or[T comparable](list ...T) T {
	// TODO(bradfitz): remove the comparable constraint so we can use this
	// with funcs too and use reflect to see whether they're non-zero? 🤷‍♂️
	var zero T
	for _, v := range list {
		if v != zero {
			return v
		}
	}
	return zero
}

type Grid map[Pt]rune

func ReadGrid() Grid {
	g := Grid{}
	ForLinesY(func(y int, v string) {
		for x, r := range v {
			if unicode.IsSpace(r) {
				continue
			}
			g[Pt{x, y}] = r
		}
	})
	return g
}

func GridFromString(s string) Grid {
	g := Grid{}
	for y, line := range strings.Split(s, "\n") {
		for x, r := range line {
			if unicode.IsSpace(r) {
				continue
			}
			g[Pt{x, y}] = r
		}
	}
	return g
}

func (g Grid) PosSetWithValue(v rune) map[Pt]bool {
	s := map[Pt]bool{}
	for p, r := range g {
		if r == v {
			s[p] = true
		}
	}
	return s
}

func (g Grid) Bounds() (minX, minY, maxX, maxY int) {
	n := 0
	for p := range g {
		if n == 0 {
			minX = p.X
			maxX = p.X
			minY = p.Y
			maxY = p.Y
		}
		n++
		if p.X < minX {
			minX = p.X
		}
		if p.X > maxX {
			maxX = p.X
		}
		if p.Y < minY {
			minY = p.Y
		}
		if p.Y > maxY {
			maxY = p.Y
		}
	}
	return
}

func (g Grid) Draw() {
	minX, minY, maxX, maxY := g.Bounds()
	for y := minY; y <= maxY; y++ {
		for x := minX; x <= maxX; x++ {
			r := g[Pt{x, y}]
			if r == 0 {
				r = '?'
			}
			fmt.Printf("%c", r)
		}
		fmt.Println()
	}
}
