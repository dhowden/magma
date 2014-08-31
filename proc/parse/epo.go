// Copyright 2014, David Howden
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package parse

import (
	"errors"
	"fmt"
	"strings"

	"github.com/dhowden/magma/proc"
)

// ErrorPosition represents the position of an error as reported by the magma EPO
// (TagErrorPosition) tag.
type ErrorPosition struct {
	File           string         // The file (if any)
	Eval           bool           // Eval == true iff File == ""
	Row, Column    int            // The line and column of the error
	SourceFragment string         // A string containing the problem
	LocatedIn      *ErrorPosition // Further location information
}

type errorPositionParserStateFn func(*ErrorPositionParser) errorPositionParserStateFn

// ErrorPositionParser is the container associated with the error position parser
type ErrorPositionParser struct {
	*lineConsumer
	err        error            // Parse error (if any)
	current    *ErrorPosition   // Current ErrorPosition object (in construction)
	currentSub *ErrorPosition   // Current `LocatedIn` ErrorPosition object that is being built
	output     chan interface{} // Line channel for delivering completed signatures
}

// Accepts returns true if the error position parser will accept the given
// proc.Tagged, false otherwise.
func (p *ErrorPositionParser) Accepts(x proc.Tagged) bool {
	return x.Tag() == proc.TagErrorPosition
}

// Run creates a lineConsumer for the given channel of Tagged objects and starts
// the parser.  Resulting *Signature structs are passed back on the returned channel.
func (p *ErrorPositionParser) Run(source <-chan proc.Tagged) <-chan interface{} {
	outputSource := taggedOutputSourceForLineConsumer(source, p)
	return p.start(newLineConsumer(outputSource))
}

func (p *ErrorPositionParser) start(consumer *lineConsumer) <-chan interface{} {
	p.lineConsumer = consumer
	p.output = make(chan interface{})

	go p.run()
	return p.output
}

func (p *ErrorPositionParser) run() {
	state := parseTopLevel
	for state != nil {
		state = state(p)
	}

	if p.current != nil && p.err == nil {
		p.output <- p.current
	}
	close(p.output)
}

// Extract file, row, column, from a string with format:
// `<file>, [at] line <row>, column <column>:`
func extractFileRowColumn(input string) (file string, row, col int, err error) {
	commaSplit := strings.FieldsFunc(input, matchCommaRune)
	if len(commaSplit) < 3 {
		err = errors.New("expected at least 3 (file, line, column) in comma split")
	}
	file = commaSplit[0]
	row, col, err = extractRowColumnFromFieldsSplit(commaSplit[len(commaSplit)-2 : len(commaSplit)])
	return
}

func parseTopLevel(p *ErrorPositionParser) errorPositionParserStateFn {
	if p.fetchNextLine() {
		if line := strings.TrimPrefix(p.line, "In eval expression, "); len(p.line) > len(line) {
			// `In eval expression, line <x>, column <y>:`
			row, col, err := extractRowColumnFromString(line[:len(line)-1])
			if err != nil {
				p.err = err
				return parseErrorPositionError
			}

			p.current = &ErrorPosition{
				Eval:   true,
				Row:    row,
				Column: col,
			}
		} else if line := strings.TrimPrefix(p.line, "In file "); len(p.line) > len(line) {
			// `In file "<path-to-file>", line <x>, column <y>:`
			file, row, col, err := extractFileRowColumn(line)
			if err != nil {
				p.err = err
				return parseErrorPositionError
			}

			p.current = &ErrorPosition{
				File:   file[1 : len(file)-1],
				Row:    row,
				Column: col,
			}
		} else {
			return nil
		}
		p.consumeLine()
		return parseSourceFragment
	}
	return nil
}

func parseSourceFragment(p *ErrorPositionParser) errorPositionParserStateFn {
	if p.fetchNextLine() {
		if line := strings.TrimPrefix(p.line, ">> "); len(p.line) > len(line) {
			if p.currentSub != nil {
				p.currentSub.SourceFragment = line
			} else {
				p.current.SourceFragment = line
			}
			p.consumeLine()
			return parseLocatedInExpression
		}
	}
	p.err = fmt.Errorf("expected source fragment line, got %v", p.line)
	return parseErrorPositionError
}

func parseLocatedInExpression(p *ErrorPositionParser) errorPositionParserStateFn {
	if p.fetchNextLine() {
		if line := strings.TrimPrefix(p.line, "Located in"); len(p.line) > len(line) {
			var s *ErrorPosition
			if line2 := strings.TrimPrefix(line, " enclosing eval expression, at "); len(line2) < len(line) {
				// Located in enclosing eval expression, at line x, column y:
				row, col, err := extractRowColumnFromString(line2[:len(line2)-1])
				if err != nil {
					p.err = err
					return parseErrorPositionError
				}

				s = &ErrorPosition{
					Eval:   true,
					Row:    row,
					Column: col,
				}
			} else if line2 := strings.TrimPrefix(line, " file "); len(line) > len(line2) {
				// Located in file "<path-to-file>", at line <x>, column <y>:
				file, row, col, err := extractFileRowColumn(line2[:len(line2)-1])
				if err != nil {
					p.err = err
					return parseErrorPositionError
				}

				s = &ErrorPosition{
					File:   file[1 : len(file)-1],
					Row:    row,
					Column: col,
				}
			} else if line == ":" {
				// Located in:
				s = &ErrorPosition{}
			} else {
				p.err = errors.New("`Located in` line with unrecognised suffix")
				return parseErrorPositionError
			}

			if p.current.LocatedIn == nil {
				p.current.LocatedIn = s
			} else {
				p.currentSub.LocatedIn = s
			}
			p.currentSub = s
			p.consumeLine()
			return parseSourceFragment
		}
	}
	return nil
}

func parseErrorPositionError(p *ErrorPositionParser) errorPositionParserStateFn {
	if p.err == nil {
		panic("parser error triggered but error value not set")
	}
	p.output <- p.err
	return nil
}
