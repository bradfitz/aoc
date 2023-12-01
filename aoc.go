// Package aoc are quick & dirty utilities for helping Brad
// solve Advent of Code problems.
package aoc

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"log"
	"os"

	"golang.org/x/exp/constraints"
)

var flagDay *int
var flagSample *bool

var days = map[int]func(){}
var curDay int

func Main() {
	flagDay = flag.Int("day", 0, "day to run")
	flagSample = flag.Bool("sample", false, "use sample input")
	flag.Parse()

	if *flagDay == 0 {
		for k := range days {
			if k > *flagDay {
				*flagDay = k
			}
		}
	}

	f, ok := days[*flagDay]
	if !ok {
		log.Fatalf("day %v not registered", *flagDay)
	}
	curDay = *flagDay
	f()
}

func AddDay(day int, f func()) {
	days[day] = f
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
	var suf string
	if *flagSample {
		suf = ".sample"
	}
	return MustGet(os.ReadFile(fmt.Sprintf("%d.input%s", curDay, suf)))
}

func Scanner() *bufio.Scanner {
	return bufio.NewScanner(bytes.NewReader(Input()))
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
