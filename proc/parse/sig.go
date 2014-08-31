// Copyright 2014, David Howden
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package parse

import (
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/dhowden/magma/proc"
)

// Param stores intrinsic parameter name/type pairs.
type Param struct {
	Name, Type string
}

// SignatureLocation represents the source location where a signature is defined
type SignatureLocation struct {
	Location
	Column int
}

// Signature represents an intrinsic signature
type Signature struct {
	Location           SignatureLocation
	Intrinsic, Comment string
	Params             []Param
	Returns            []string
	OptionalParams     []Param
}

type signatureParserStateFn func(*SignatureParser) signatureParserStateFn

// SignatureParser is the container associated with parsing signatures.
type SignatureParser struct {
	*lineConsumer
	err     error            // Parse error (if any)
	current *Signature       // Current Signature object (in construction)
	output  chan interface{} // Line channel for delivering completed signatures

	// When parsing a listing of signatures for a given intrinsic, the intrinsic
	// name is given up front.  This also acts as a test for the kind of sig list
	// we are parsing (when a name is not given, then the signatures are for
	// different intrinsic names).
	intrinsic string
}

// Accepts returns true if the signature parser will accept the given
// proc.Tagged, false otherwise.
func (p *SignatureParser) Accepts(x proc.Tagged) bool {
	return x.Tag() == proc.TagSignature
}

// Run creates a lineConsumer for the given channel of Tagged objects and starts
// the parser.  Resulting *Signature structs are passed back on the returned channel.
func (p *SignatureParser) Run(source <-chan proc.Tagged) <-chan interface{} {
	outputSource := taggedOutputSourceForLineConsumer(source, p)
	return p.start(newLineConsumer(outputSource))
}

func (p *SignatureParser) start(consumer *lineConsumer) <-chan interface{} {
	p.lineConsumer = consumer

	p.current = &Signature{}
	p.output = make(chan interface{})

	go p.run()
	return p.output
}

func (p *SignatureParser) run() {
	state := parseSignatureListHeader
	for state != nil {
		state = state(p)
	}
	close(p.output)
}

// Emit a completed Signature struct to the 'output' channel,
// and reset current
func (p *SignatureParser) emit() {
	p.output <- p.current
	p.current = &Signature{}
}

func parseSignatureListHeader(p *SignatureParser) signatureParserStateFn {
	if p.fetchNextLine() {
		if line := strings.TrimPrefix(p.line, "Intrinsic '"); len(p.line) > len(line) {
			if p.intrinsic != "" {
				p.err = fmt.Errorf("new listing, but already have intrisic set")
				return parseSignatureError
			}

			if line[len(line)-1] != '\'' {
				p.err = fmt.Errorf("expected `Intrinsic 'Name'`, got: Intrinsic '%v", line)
				return parseSignatureError
			}

			p.intrinsic = line[:len(line)-1]
			p.consumeLine()
		}

		if line := strings.TrimPrefix(p.line, "Signatures matching "); len(p.line) > len(line) {
			p.intrinsic = ""
			p.consumeLine()
		}
		return parseSignature
	}
	p.err = fmt.Errorf("expected signature list header (`Intrinsic 'Name'` or `Signatures matching...`)")
	return parseSignatureError
}

// Discard output until a signature param statement, and parse it
func parseSignature(p *SignatureParser) signatureParserStateFn {
	for p.fetchNextLine() {
		if p.line != "" && p.line != "Signatures:" {
			if line := strings.TrimPrefix(p.line, "Defined in file: "); len(p.line) > len(line) {
				// Defined in file: /Users/dhowden/etc/file.m, line 123, column 456:
				fields := strings.FieldsFunc(line[:len(line)-1], matchCommaRune)
				if len(fields) < 3 {
					// Expect fields[0] filename, fields[1,2] line, col:
					p.err = errors.New("expected at least 3 chunks from comma split")
					return parseSignatureError
				}

				line, col, err := extractRowColumnFromFieldsSplit(fields[1:])
				if err != nil {
					p.err = err
					return parseSignatureError
				}

				p.current.Location = SignatureLocation{
					Location: Location{
						File: fields[0],
						Row:  int(line),
					},
					Column: int(col),
				}
			} else if strings.HasPrefix(p.line, "Defined in glue: ") {
				// Defined in glue: glue_function_name():
				glue := strings.TrimPrefix(p.line, "Defined in glue: ")
				p.current.Location = SignatureLocation{
					Location: Location{
						Glue: glue[:len(glue)-1],
					},
				}
			} else if p.intrinsic != "" {
				if strings.HasPrefix(p.line, leftParam) {
					p.current.Intrinsic = p.intrinsic
					return parseParams
				}
			} else {
				name := ""
				for {
					if index := strings.Index(p.line, leftParam); index == -1 {
						name += p.line
						p.consumeLine()
						p.fetchNextLine()
					} else {
						name += p.line[:index]
						p.line = p.line[index:]
						break
					}
				}
				p.current.Intrinsic = name
				return parseParams
			}
		}
		p.consumeLine()
	}
	return nil
}

// Parse the params line
func parseParams(p *SignatureParser) signatureParserStateFn {
	l := p.line
	p.consumeLine()
	for {
		p.fetchNextLine()
		// stop when we get an empty line (before comment), or the beginning of
		// optional params
		if p.line == "" || p.line == leftOptionalParam {
			break
		}
		l += p.line
		p.consumeLine()
	}

	// Avoid the "()" case
	if index := strings.Index(l, ")"); index != 1 {
		params := l[1:index]

		// signatureArg := `\<(?P<arg_type>[A-Za-z0-9\ \[\],]+)\>(?:\s(?P<arg_name>[A-Za-z0-9]+))?`
		signatureArg := `(?:(?P<arg_name>[A-Za-z0-9]+)::)?(?P<arg_type>[A-Za-z0-9]+(?:\[[^]]+\]+)?)`
		argRegex := regexp.MustCompile(signatureArg)

		matches := argRegex.FindAllStringSubmatch(l[1:index], -1)
		currentParams := make([]Param, 0, len(params))

		for _, match := range matches {
			currentParams = append(currentParams, Param{
				Name: match[1],
				Type: match[2],
			})
		}
		p.current.Params = currentParams

		index = strings.Index(l, "->")
		if index != -1 {
			returns := strings.Split(l[index+3:len(l)], ", ")
			p.current.Returns = returns
		}
	}

	if p.line == "" { // blank line preceeds comment
		p.consumeLine()
		return parseComment
	} else if p.line == leftOptionalParam {
		return parseOptionalParams
	}
	return nil
}

// Parse optional param statement
func parseOptionalParams(p *SignatureParser) signatureParserStateFn {
	p.fetchNextLine()
	optionalParams := ""
	if strings.HasPrefix(p.line, leftOptionalParam) {
		// Optional params appear
		p.consumeLine()
		for p.fetchNextLine() {
			// Parse until a ] line
			if strings.HasPrefix(p.line, rightOptionalParam) {
				p.consumeLine()
				// Get the next line, it should be empty
				p.fetchNextLine()
				p.consumeLine()
				if p.line != "" {
					p.err = fmt.Errorf("expected an empty line to follow optional params, got: %v", p.line)
					return parseSignatureError
				}
				if optionalParams != "" {
					params := strings.Split(optionalParams, ",")
					p.current.OptionalParams = make([]Param, len(params))
					for i, x := range params {
						paramTypeNameArr := strings.Split(x, ": ")
						if len(paramTypeNameArr) == 2 {
							p.current.OptionalParams[i] = Param{Name: paramTypeNameArr[0], Type: paramTypeNameArr[1]}
						} else {
							p.current.OptionalParams[i] = Param{Name: paramTypeNameArr[0]}
						}
					}
				}
				return parseComment
			}
			optionalParams += p.line
			p.consumeLine()
		}
		panic("should not get here")
	}
	return parseComment
}

// Parse signature comment (starts with a non-empty line)
func parseComment(p *SignatureParser) signatureParserStateFn {
	for p.fetchNextLine() {
		// The end of the comment
		if p.line == "" {
			p.consumeLine()
			p.emit()
			return parseSignature
		}

		if p.current.Comment == "" {
			p.current.Comment = p.line
		} else {
			p.current.Comment += " " + p.line
		}
		p.consumeLine()
	}
	return nil
}

func parseSignatureError(p *SignatureParser) signatureParserStateFn {
	if p.err == nil {
		panic("parser error triggered but error value not set")
	}
	p.output <- p.err
	return nil
}
