// Copyright 2014, David Howden
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package proc

import (
	"fmt"
	"io"
)

var nl = []byte{'\n'}
var indent = []byte("    ")

// WriteTo writes the raw output equivalent of the Line object
// to the given io.Writer.
// NB: if o.Continuation then a newline preceeds the output
// so when writing a list of Line structs you may need to
// set the first one to have o.Continuation = true.  See
// FlushTaggedToWriter.
func (o *Line) WriteTo(w io.Writer) (n int64, err error) {
	var c int
	if !o.Continuation {
		c, err = w.Write(nl)
		if err != nil {
			n = int64(c)
			return
		}
	}
	if o.Indent > 0 {
		for i := 0; i < o.Indent; i++ {
			c, err = w.Write(indent)
			n += int64(c)
			if err != nil {
				return
			}
		}
	}
	c, err = w.Write([]byte(o.Data))
	n += int64(c)
	return
}

// FlushTaggedToWriter takes the output from the given channel `ch`
// (assumed to come from exec), and writes it out to the given writer w.
// NB: only output values that implement WriterTo behave nicely here,
// anything else will cause an error
func FlushTaggedToWriter(ch <-chan Tagged, w io.Writer) error {
	first := true
	for x := range ch {
		switch x := x.(type) {
		case (*Line):
			if first {
				x.Continuation = true
				first = false
			}
			x.WriteTo(w)
		case (io.WriterTo):
			x.WriteTo(w)
		default:
			return fmt.Errorf("given type does not implement WriterTo: %T (%v)", x, x)
		}
	}
	return nil
}

// LaunchF defines a function prototype used for running a Magma process using Launch.
type LaunchF func(p *Process, st <-chan Tagged, so *Output) error

// Launch starts the given Process and correctly handles errors on Start() and Wait(),
// executing the given function f only if the Process starts correctly.  This function
// is a convenience method which avoids a lot of boilerplate error handling.
func Launch(p *Process, f LaunchF) (err error) {
	st, err := p.StatusTags()
	if err != nil {
		return
	}

	so, err := p.Start()
	ch := make(chan error, 1)
	if err != nil {
		return
	}
	defer func() {
		if err == nil {
			err = <-ch
			close(ch)
			var copyError error
			if err != nil {
				copyError = err
			}
			err = p.Wait()
			if copyError != nil {
				err = copyError
			}
		}
	}()

	go func() {
		ch <- f(p, st, so)
	}()

	return
}
