// Copyright 2014, David Howden
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package parse

import (
	"errors"
	"strconv"
	"strings"

	"github.com/dhowden/magma/proc"
)

// General constants used in parsing Magma output
const (
	leftParam          string = "("
	rightParam         string = ")"
	leftOptionalParam  string = "["
	rightOptionalParam string = "]"
)

// parser is the common interface shared by all parsers
type parser interface {
	// start the parser with the given lineConsumer, and return the parsed objects
	start(*lineConsumer) <-chan interface{}
}

// TaggedParser is the (public) common interface implemented by all proc parsers
type TaggedParser interface {
	parser
	Run(<-chan proc.Tagged) <-chan interface{}
	Accepts(proc.Tagged) bool
}

// taggedOutputSourceForLineConsumer reads a channel of proc.Tagged and puts structs
// which are accepted by the TaggedParser p onto the returned *proc.Line channel.  Anything not
// accepted is discarded.
func taggedOutputSourceForLineConsumer(source <-chan proc.Tagged, p TaggedParser) <-chan string {
	outputSource := make(chan string)
	go func() {
		for x := range source {
			if x, ok := x.(*proc.Line); ok && p.Accepts(x) {
				outputSource <- x.Data
			}
		}
		close(outputSource)
	}()
	return outputSource
}

// ParseTagged takes an input channel and passes back the parsed
// structs on the out channel using the given list of parsers. If
// no parser accepts a struct, then it is passed back.
func ParseTagged(in <-chan proc.Tagged, out chan<- interface{}, parsers ...TaggedParser) {
	var cur TaggedParser
	var src chan proc.Tagged
	var done chan struct{}

IN_LOOP:
	for x := range in {
		if cur != nil {
			if cur.Accepts(x) {
				src <- x
				continue
			}
			close(src)
			cur = nil
			// wait for current parser to finish-up
			<-done
		}

		for _, p := range parsers {
			if p.Accepts(x) {
				src = make(chan proc.Tagged)
				done = make(chan struct{})
				parsedOutput := p.Run(src)
				go func() {
					for y := range parsedOutput {
						out <- y
					}
					close(done)
				}()
				cur = p
				continue IN_LOOP
			}
		}
		out <- x
	}

	if cur != nil {
		close(src)
		cur = nil
		// wait for current parser to finish-up
		<-done
	}
	close(out)
}

// Splitting function for commas
func matchCommaRune(r rune) bool {
	return r == ','
}

// Extract the line and column where fields are:
// `[at] line x` and `column y`
func extractRowColumnFromFieldsSplit(fields []string) (row, col int, err error) {
	if len(fields) != 2 {
		err = errors.New("expect 2 elements in expansion of line/column location data")
		return
	}

	lineSplit := strings.Fields(fields[0])
	row, err = strconv.Atoi(lineSplit[len(lineSplit)-1])
	if err != nil {
		return
	}

	col, err = strconv.Atoi(strings.Fields(fields[1])[1])
	return
}

// Extract the line and column numbers from a string with format:
// `[at] line x, column y` where x and y are integers
func extractRowColumnFromString(input string) (row, column int, err error) {
	fields := strings.FieldsFunc(input, matchCommaRune)
	row, column, err = extractRowColumnFromFieldsSplit(fields)
	return
}
