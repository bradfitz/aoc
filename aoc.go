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
	wantRx := regexp.MustCompile(`(?sm)/\*\s*want=([^\n]*)(?:\s+(.+\n))?\s*\*/`)
	for _, d := range f.Decls {
		fd, ok := d.(*ast.FuncDecl)
		if !ok || fd.Doc == nil {
			continue
		}
		funcName := fd.Name.Name
		for _, c := range fd.Doc.List {
			if m := wantRx.FindStringSubmatch(c.Text); m != nil {
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

type PtInt = Pt2[int]
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

func ForLines(onLine func(string)) {
	s := Scanner()
	for s.Scan() {
		onLine(s.Text())
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
