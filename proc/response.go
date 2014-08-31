// Copyright 2014, David Howden
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package proc

import (
	"strings"
)

// Output represents all the output from Magma which corresponds
// to the execution of command string.
type Output struct {
	cmd string
	ch  chan Response
}

func newOutput(input string) *Output {
	return &Output{cmd: input, ch: make(chan Response)}
}

// Command returns the input command which produced this
// Output instance
func (o Output) Command() string { return o.cmd }

// Responses returns a channel giving a new Response
// for each statement run
func (o Output) Responses() <-chan Response { return o.ch }

// Output returns a channel which provides a single stream of
// Tagged output.
func (o Output) Output() <-chan Tagged { return Combine(o.ch) }

func (o Output) close() { close(o.ch) }

// Combine combines the output of all statements to give a single
func Combine(ch <-chan Response) <-chan Tagged {
	outCh := make(chan Tagged)
	go func() {
		for c := range ch {
			for x := range c.Output() {
				outCh <- x
			}
		}
		close(outCh)
	}()
	return outCh
}

// Discard loops over all values in the given channel and discards the results
func Discard(ch <-chan Tagged) {
	go func() {
		for _ = range ch {
		}
	}()
}

// chunk represents a chunk of the input string, denoted by
// start and end position values
type chunk struct {
	start, end Position
}

func (c chunk) get(input string) string {
	lines := strings.Split(input, "\n")
	if c.start.Row >= len(lines) || c.end.Row >= len(lines) {
		return "Invalid chunk"
	}

	cmd := lines[c.start.Row][c.start.Column:]
	for i := c.start.Row + 1; i < c.end.Row; i++ {
		cmd += "\n" + lines[i]
	}
	cmd += lines[c.end.Row][:c.end.Column]
	return cmd
}

func (o *Output) commandResponse(chk chunk) string {
	return chk.get(o.cmd)
}

// Response represents a chunk of output which corresponds to a portion
// of the input string.
type Response interface {
	Command() string
	Output() <-chan Tagged
}

type responser interface {
	Response
	send(t Tagged)
	close()
}

type response struct {
	cmd string
	ch  chan Tagged
}

func newResponse(cmd string, ch chan Tagged) *response {
	return &response{cmd: cmd, ch: ch}
}

// Command returns the command string that produced this response
func (s response) Command() string { return s.cmd }

// Line returns a <-chan Tagged which passes back the response
// from the underlying process (line-by-line)
func (s response) Output() <-chan Tagged { return s.ch }

func (s response) send(t Tagged) { s.ch <- t }

func (s response) close() { close(s.ch) }

// Seed represents random seed information
type Seed struct {
	Seed uint
	Step uint64
}

// Run represents the result of a command execution
type Run struct {
	response
	Seed *Seed
}

// ParseError represents a parse error in response to invalid input
type ParseError struct {
	response
}

// InternalError represents an internal error which is the result of
// a command
type InternalError struct {
	response
}

func newInternalError() *InternalError {
	return &InternalError{response{ch: make(chan Tagged)}}
}

type rhandler struct {
	r *Output
	c responser
}

func (h *rhandler) ready() bool {
	if h.c != nil {
		h.c.close()
		h.c = nil
	}
	if h.r != nil {
		h.r.close()
		h.r = nil
		return true
	}
	return false
}

func (h *rhandler) init(e *Output) {
	if h.r != nil {
		panic("should not have r set here")
	}
	h.r = e
}

func (h *rhandler) newResponse(chk chunk, r responser) {
	if h.c != nil {
		h.c.close()
	}
	h.c = r
	h.r.ch <- h.c
}

func (h *rhandler) start() {
	h.newResponse(chunk{}, &response{ch: make(chan Tagged)})
}

func (h *rhandler) run(chk chunk, s *Seed) {
	h.newResponse(chk, Run{
		response: response{ch: make(chan Tagged)},
		Seed:     s,
	})
}

func (h *rhandler) parseError(chk chunk) {
	h.newResponse(chk, ParseError{response{ch: make(chan Tagged)}})
}

// NB: we only create a new response if there isn't already one (as it may give a better
// context for the error!)
func (h *rhandler) internalError() {
	if h.c == nil {
		h.newResponse(chunk{}, newInternalError())
	}
}

func (h *rhandler) close() {
	if h.c != nil {
		h.c.close()
	}
	if h.r != nil {
		h.r.close()
	}
}

func (h *rhandler) send(t Tagged) {
	if h.c == nil {
		panic("no current output")
	}
	h.c.send(t)
}
