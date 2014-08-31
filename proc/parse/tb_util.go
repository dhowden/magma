// Copyright 2014, David Howden
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package parse

import (
	"io"
	"strconv"
)

// WriteTo writes the raw output equivalent of the ParamValue struct
// to the given io.Writer.
func (pv *ParamValue) WriteTo(w io.Writer) (n int64, err error) {
	c, err := w.Write([]byte("\n" + "    " + pv.Name + " : " + pv.Value))
	n = int64(c)
	return
}

// WriteTo writes the raw output equivalent of the Location struct
// to the given io.Writer.
func (sp *Location) WriteTo(w io.Writer) (n int64, err error) {
	output := ""
	if sp.Glue != "" {
		output += "defined in glue: " + sp.Glue
	} else if sp.File != "" {
		output += "defined in file: " + sp.File + ", line " + strconv.Itoa(sp.Row)
	}
	c, err := w.Write([]byte(output + "\n"))
	n = int64(c)
	return
}

// WriteTo writes the raw output equivalent of the Traceback struct
// to the given io.Writer.
func (tb *Traceback) WriteTo(w io.Writer) (n int64, err error) {
	output := ""
	if tb.Index != NoIndex {
		output += strconv.Itoa(tb.Index)
	}
	output += " "
	if tb.Current {
		output += "*"
	}
	output += tb.Name + "("

	c, err := w.Write([]byte(output))
	n += int64(c)
	if err != nil {
		return
	}

	var c64 int64
	for _, p := range tb.Params {
		c64, err = p.WriteTo(w)
		n += c64
		if err != nil {
			return
		}
	}

	c, err = w.Write([]byte("\n), "))
	n += int64(c)
	if err != nil {
		return
	}
	tb.Location.WriteTo(w)

	return
}
