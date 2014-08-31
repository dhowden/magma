// Copyright 2014, David Howden
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package parse

import (
	"io"
	"strconv"
)

// WriteTo writes the raw output equivalent of the Param value
// to the given io.Writer (note that this is optional param, not
// a full param).
func (pv *Param) WriteTo(w io.Writer) (n int64, err error) {
	output := "\n" + "    " + pv.Name
	if pv.Type != "" {
		output += " : " + pv.Type
	}
	c, err := w.Write([]byte(output))
	n = int64(c)
	return
}

// WriteTo writes the raw output equivalent of the SignatureLocation
// to the given io.Writer
func (sp *SignatureLocation) WriteTo(w io.Writer) (n int64, err error) {
	output := ""
	if sp.Glue != "" {
		output += "Defined in glue: " + sp.Glue
	} else if sp.File != "" {
		output += "Defined in file: " + sp.File + ", line " + strconv.Itoa(sp.Row) +
			", column " + strconv.Itoa(sp.Column)
	}
	c, err := w.Write([]byte(output + ":\n"))
	n = int64(c)
	return
}

// WriteTo writes the raw output equivalent of the Signature struct
// to the given io.Writer.
func (s *Signature) WriteTo(w io.Writer) (n int64, err error) {
	var c int
	n, err = s.Location.WriteTo(w)
	if err != nil {
		return
	}

	c, err = w.Write([]byte(s.Intrinsic + "("))
	n += int64(c)
	if err != nil {
		return
	}

	paramsOutput := ""
	for _, p := range s.Params {
		paramsOutput += p.Name + "::" + p.Type + ", "
	}
	paramsOutput = paramsOutput[:len(paramsOutput)-2] + ") -> "
	for _, r := range s.Returns {
		paramsOutput += r + ", "
	}
	paramsOutput = paramsOutput[:len(paramsOutput)-2] + "\n"
	w.Write([]byte(paramsOutput))
	n += int64(c)
	if err != nil {
		return
	}

	if len(s.OptionalParams) > 0 {
		c, err = w.Write([]byte("["))
		n += int64(c)
		if err != nil {
			return
		}
		for _, op := range s.OptionalParams {
			op.WriteTo(w)
		}
		c, err = w.Write([]byte("\n]\n"))
		n += int64(c)
		if err != nil {
			return
		}
	}

	c, err = w.Write([]byte(s.Comment + "\n"))
	n += int64(c)
	return
}
