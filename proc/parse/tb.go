// Copyright 2014, David Howden
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package parse

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/dhowden/magma/proc"
)

// ParamValue represents pairs of function parameter names and their values
type ParamValue struct {
	Name, Value string
}

// Location represents a location in a source file, or a C glue function
type Location struct {
	File string
	Row  int
	Glue string
}

// Traceback level information
type Traceback struct {
	Index    int          // index in the back trace (-1 if not set)
	Current  bool         // the current frame
	Name     string       // function name
	Params   []ParamValue // parameters
	Location Location
}

// Index value for unset state
const NoIndex int = -1

type tracebackParserStateFn func(*TracebackParser) tracebackParserStateFn

// TracebackParser is the container associated with the traceback parser
type TracebackParser struct {
	*lineConsumer
	err     error            // Parse error (if any)
	current *Traceback       // Current Traceback object (in construction)
	output  chan interface{} // Line channel for delivering completed tracebacks
}

// Accepts returns true if the traceback parser will accept the given
// proc.Tagged struct, false otherwise.
func (p *TracebackParser) Accepts(x proc.Tagged) bool {
	return x.Tag() == proc.TagTraceback
}

// Run creates a lineConsumer for the given channel of Tagged objects and starts
// the parser.  Resulting *Traceback structs are passed back on the returned channel.
func (p *TracebackParser) Run(source <-chan proc.Tagged) <-chan interface{} {
	outputSource := taggedOutputSourceForLineConsumer(source, p)
	return p.start(newLineConsumer(outputSource))
}

func (p *TracebackParser) start(consumer *lineConsumer) <-chan interface{} {
	p.lineConsumer = consumer

	p.current = &Traceback{}
	p.output = make(chan interface{})

	go p.run()
	return p.output
}

func (p *TracebackParser) run() {
	state := parseTraceback
	for state != nil {
		state = state(p)
	}
	close(p.output)
}

// Emit a completed Traceback struct to the 'output' channel
func (p *TracebackParser) emit() {
	p.output <- p.current
}

// Discard output until a traceback level statement, and parse it
func parseTraceback(p *TracebackParser) tracebackParserStateFn {
	for p.fetchNextLine() {
		if name := strings.TrimSuffix(p.line, leftParam); len(name) < len(p.line) {
			p.current = &Traceback{Name: name, Index: NoIndex}
			if levelName := strings.TrimPrefix(name, "#"); len(levelName) < len(name) {
				levelNameFields := strings.Fields(levelName)
				if len(levelNameFields) != 2 {
					fmt.Errorf("expected split into 2, got %v", levelNameFields)
					return parseTracebackError
				}
				index, err := strconv.Atoi(levelNameFields[0])
				if err != nil {
					p.err = err
					return parseTracebackError
				}
				p.current.Index = index
				nameWithoutMarker := strings.TrimPrefix(levelNameFields[1], "*")
				p.current.Name = nameWithoutMarker
				if len(nameWithoutMarker) < len(levelNameFields[1]) {
					p.current.Current = true
				}
			}
			p.consumeLine()
			return parseTracebackParam
		}
		p.consumeLine()
	}
	return nil
}

// Parse the parmeters
func parseTracebackParam(p *TracebackParser) tracebackParserStateFn {
	for p.fetchNextLine() {
		// Got to the end of the params
		if strings.HasPrefix(p.line, rightParam) {
			return parseTracebackLocation
		}
		fields := strings.SplitN(p.line, ": ", 2)
		if len(fields) != 2 {
			p.err = fmt.Errorf("expected a split of 2, got %v", fields)
			return parseTracebackError
		}
		// Remove the trailing , if there is one...
		fields[1] = strings.TrimSuffix(fields[1], ",")
		p.current.Params = append(p.current.Params, ParamValue{fields[0], fields[1]})
		p.consumeLine()
	}
	panic("should not get here")
}

func parseTracebackLocation(p *TracebackParser) tracebackParserStateFn {
	for p.fetchNextLine() {
		// Double check that we are here correctly...
		if strings.HasPrefix(p.line, rightParam) {
			p.consumeLine()
			locationLine := strings.SplitN(p.line, " at ", 2)
			if len(locationLine) == 2 {
				index := strings.LastIndex(locationLine[1], ":")
				if index == -1 {
					p.err = fmt.Errorf("expected ':', but did not find one in '%v'", locationLine[1])
					return parseTracebackError
				}
				line, err := strconv.Atoi(locationLine[1][index+1:])
				if err != nil {
					p.err = err
					return parseTracebackError
				}
				p.current.Location = Location{File: locationLine[1][:index], Row: line}
			}
			p.emit()
			return parseTraceback
		}
		p.consumeLine()
	}
	panic("should not get here")
}

func parseTracebackError(p *TracebackParser) tracebackParserStateFn {
	if p.err == nil {
		panic("parser error triggered but error value not set")
	}
	p.output <- p.err
	return nil
}
