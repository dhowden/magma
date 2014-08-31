// Copyright 2014, David Howden
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package proc

// tag specifies a type of output, or status flag given by the Magma process.
type tag string

// Line tags that are produced by Magma
const (
	TagOutput               tag = "OUT"    // Normal output
	TagList                     = "LST"    // List output
	TagSignature                = "SIG"    // Signature output
	TagErrorSyntax              = "ENE"    // Input finished before statement was complete
	TagErrorInternal            = "EI"     // Internal error
	TagErrorUser                = "EU"     // User error
	TagErrorRuntime             = "ER"     // Runtime error
	TagTraceback                = "TB"     // Traceback output
	TagErrorHistoryPosition     = "POS"    // Error position (line/column from history)
	TagErrorPosition            = "EPO"    // Error position (file/eval/other values)
	TagReadPrompt               = "RD_PR"  // Read prompt
	TagReadInput                = "RD_IN"  // Expecting input (ends on newline)
	TagReadIntPrompt            = "RDI_PR" // Integer read prompt
	TagReadIntInput             = "RDI_IN" // Expecting integer input (ends on newline)
	TagReadIntError             = "RDI_ER" // Error in interactive integer read
)

// statusTag is a special tag which indicates a change of status of the underlying
// Magma process
type statusTag tag

// Status tags that are produced by Magma
const (
	TagReady         statusTag = "RDY"  // Ready (Ready, with flags)
	TagInputReceived           = "IR"   // Input received
	TagRun                     = "RUN"  // Running statement
	TagErrorParse              = "ERP"  // Error occurred in parsing statement
	TagInterrupt               = "INT"  // Process interrupted
	TagQuit                    = "QUIT" // End session
	TagReset                   = "RES"  // Reset of frame variables
	TagDebugReady              = "DRDY" // Debugger ready
)

// Tagged is an interface which is implemented by types which represent
// data received directly from Magma (and thus have an associated output
// tag).
type Tagged interface {
	Tag() tag
}

// IsError returns true if the given tag is part of error
// output; false otherwise.
func IsError(t Tagged) bool {
	switch t.Tag() {
	case TagErrorSyntax, TagTraceback, TagErrorHistoryPosition, TagErrorPosition,
		TagErrorInternal, TagErrorRuntime, TagErrorUser, TagReadIntError:
		return true
	}
	return false
}

// Tag returns the underlying tag for the given tag!
func (t tag) Tag() tag {
	return t
}

// Status output
type Status struct {
	tag statusTag
}

// Tag returns the statusTag associated with this Status instance
func (s *Status) Tag() tag {
	return tag(s.tag)
}

// Ready represents the ready state and gives more detailed status output
type Ready struct {
	Ident, Frame, Verbose, Set bool // Change flags
	Types                      int  // Diff of type count
}

// Tag returns the statusTag associated with the Ready instance
func (r *Ready) Tag() tag {
	return tag(TagReady)
}

// Line represents the standard data output, which contains an indent level
// and a continuation flag indicating if the output data should begin with
// a new line.
type Line struct {
	tag
	Continuation bool   // Should this start a new line of output?
	Indent       int    // Indentation level
	Data         string // Captured output line (following tag line)
}

// Position tag output, commonly precedes error messages/traceback, and gives
// the source location of an error.
type Position struct {
	Row    int // Source input row
	Column int // Source input column
}

// Tag returns the tag corresponding to Position
func (p *Position) Tag() tag {
	return TagErrorHistoryPosition
}

// ReadRequest is used in interactions need for `read`/`readi` statements
type ReadRequest struct {
	tag
	Prompt string      // Tag and prompt to show to user
	Output chan string // Channel to allow for pass-back
	Err    chan error  // Error if not fullfilled
}
