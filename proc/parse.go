// Copyright 2014, David Howden
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package proc

import (
	"bytes"
	"errors"
	"fmt"
	"strconv"
	"unicode"
)

// Special characters used in communication
const (
	newTagChar     byte = 129 // Prefixes every tag line
	runCommandChar byte = 4   // Run command (^D)
)

var newTagSlice = []byte{newTagChar}

// parseTagLine takes a line represented as a byte slice, and returns
// the tag, tag fields, and data (if any).
func parseTagLine(line []byte) (tagName []byte, tagFields [][]byte, data []byte) {
	if len(line) > 1 && line[0] == newTagChar {
		lineSplit := bytes.SplitN(line[1:], newTagSlice, 2)
		fields := bytes.Fields(lineSplit[0])

		if len(fields) > 0 {
			tagName = bytes.TrimSpace(fields[0])
			if len(fields) > 1 {
				tagFields = fields[1:]
			}
			if len(lineSplit) == 2 {
				data = bytes.TrimSpace(lineSplit[1])
			}
		}
	}
	return
}

// Parse the output from the underlying Magma process, and return it
// on one of two channels: `status`, or `output`.  The status channel
// is per-session and gives status flags and other status messages. The
// 'output' channel is per-execution (per call to Execute()) and returns
// the produced output, passed back to the user via the Process.output
// channel.
func (p *Process) parseStdoutLines(ch <-chan []byte) error {
	// Create the response handler for the entire session
	h := &rhandler{}

	// Setup the response object for startup output
	r := newOutput("<startup>")
	h.init(r)
	p.response <- r
	h.start()

	var rch chan *Output
	var done bool

	for {
		output, ok := <-ch
		if !ok {
			return errors.New("waiting for tag line")
		}

		if tagName, tagFields, data := parseTagLine(output); tagName != nil {
			done = true

			// Switch for status tags
			switch tag := statusTag(tagName); tag {
			case TagReady:
				r, err := parseReady(tagFields)
				if err != nil {
					return err
				}
				p.status <- r

				if h.ready() {
					rch = make(chan *Output, 1)
					p.ready <- rch
				}

			case TagInputReceived:
				p.status <- &Status{tag: statusTag(tag)}
				select {
				case r := <-rch:
					h.init(r)
					close(rch)
					rch = nil
					p.response <- r
				default:
					return errors.New("expected Output to be waiting")
				}

			case TagReset:
				p.status <- &Status{tag: statusTag(tag)}

			case TagQuit:
				p.status <- &Status{tag: statusTag(tag)}
				select {
				case qch := <-p.quit:
					close(qch)
				default:
					// Two INT signals sent to the child process can
					// force Magma to quit
				}
				return nil

			case TagInterrupt:
				p.status <- &Status{tag: tag}
				select {
				case ich := <-p.interrupt:
					close(ich)
				default:
					// An INT signal sent directly to the child process
					// can trigger the INT tag...
				}

			case TagRun:
				p.status <- &Status{tag: tag}
				chk, seed, err := parseRun(tagFields)
				if err != nil {
					return err
				}
				h.run(chk, seed)

			case TagErrorParse:
				p.status <- &Status{tag: tag}
				chk, err := parseResponse(tagFields)
				if err != nil {
					return fmt.Errorf("ERP parsing Response: %v", err)
				}
				h.parseError(chk)

			default:
				done = false
			}

			if done {
				continue
			}

			// Switch for output/data tags
			switch tag := tag(tagName); tag {
			case TagErrorHistoryPosition:
				p, err := parseHistoryPosition(tagFields)
				if err != nil {
					return err
				}
				h.send(&p)

			case TagOutput, TagList, TagErrorUser, TagErrorRuntime, TagErrorInternal,
				TagErrorPosition, TagTraceback, TagSignature:
				o, err := parseOutput(tag, tagFields, data)
				if err != nil {
					return err
				}
				if tag == TagErrorInternal {
					h.internalError()
				}
				h.send(o)

			case TagErrorSyntax:
				h.send(&Line{tag: tag})

			case TagReadPrompt, TagReadIntPrompt:
				err := p.parseReadPrompt(tag, tagFields, data, h, ch)
				if err != nil {
					return err
				}
			}
		}
	}
}

func parseReady(tagFields [][]byte) (r *Ready, err error) {
	if len(tagFields) != 5 {
		err = errors.New("parsing RDY: require 5 parameters")
		return
	}
	r = &Ready{
		Ident:   string(tagFields[0]) == "1",
		Frame:   string(tagFields[1]) == "1",
		Verbose: string(tagFields[2]) == "1",
		Set:     string(tagFields[3]) == "1",
	}
	types, err := strconv.Atoi(string(tagFields[4]))
	if err != nil {
		err = fmt.Errorf("RDY parsing number of types: %v", err)
		return
	}
	r.Types = types
	return
}

func parseRun(tagFields [][]byte) (c chunk, s *Seed, err error) {
	if len(tagFields) != 6 {
		err = errors.New("parsing RUN: require 6 parameters")
		return
	}
	seed, err := strconv.Atoi(string(tagFields[0]))
	if err != nil {
		return
	}
	step, err := strconv.Atoi(string(tagFields[1]))
	if err != nil {
		return
	}
	s = &Seed{Seed: uint(seed), Step: uint64(step)}
	c, err = parseResponse(tagFields[2:])
	return
}

func parseResponse(tagFields [][]byte) (c chunk, err error) {
	if len(tagFields) != 4 {
		err = errors.New("response output not of required form")
	}
	c.start, err = parseHistoryPosition(tagFields[:2])
	if err != nil {
		return
	}
	c.end, err = parseHistoryPosition(tagFields[2:])
	if err != nil {
		return
	}
	return
}

func parseHistoryPosition(tagFields [][]byte) (pos Position, err error) {
	// POS row column
	if len(tagFields) != 2 {
		err = errors.New("position output not of required form")
		return
	}
	pos.Row, err = strconv.Atoi(string(tagFields[0]))
	if err != nil {
		return
	}
	pos.Column, err = strconv.Atoi(string(tagFields[1]))
	if err != nil {
		return
	}
	return
}

func parseOutput(tag tag, tagFields [][]byte, data []byte) (output *Line, err error) {
	if len(tagFields) != 1 {
		err = errors.New("output tag not of required form")
		return
	}
	output = &Line{tag: tag}
	if param := string(tagFields[0]); param == "C" {
		output.Continuation = true
	} else {
		output.Indent, err = strconv.Atoi(param)
		if err != nil {
			return
		}
	}
	output.Data = string(bytes.TrimRightFunc(data, unicode.IsSpace))
	return
}

func (p *Process) parseReadPrompt(tag tag, tagFields [][]byte, data []byte, h *rhandler, ch <-chan []byte) error {
	r := &ReadRequest{tag: tag, Output: make(chan string), Err: make(chan error)}
	r.Prompt += string(data)

READ_FORLOOP:
	for {
		output, ok := <-ch
		if !ok {
			return errors.New("expected tag line (RD or RD_END)")
		}
		if tag, tagFields, data := parseTagLine(output); tag != nil {
			switch tag := string(tag); tag {
			case TagReadPrompt, TagReadIntPrompt:
				if string(tagFields[0]) != "C" {
					r.Prompt += "\n"
				}
				r.Prompt += string(data)
				continue READ_FORLOOP
			case TagReadInput, TagReadIntInput:
				// Send the read request
				h.send(r)
				// Listen for the response
				var err error
				select {
				case response := <-r.Output:
					w := <-p.writer
					_, err = w.Write([]byte(response))
					_, err = w.Write([]byte("\n"))
					p.writer <- w
				case err = <-r.Err:
				}
				if err != nil {
					return err
				}
				break READ_FORLOOP
			default:
				return errors.New("expected RD_PR or RD_IN tag")
			}
		} else {
			return errors.New("expected tag line (RD_PR or RD_IN tag line)")
		}
	}
	return nil
}
