// Copyright 2014, David Howden
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package parse

import "testing"

func errorPositionsEqual(a interface{}, b *ErrorPosition, t *testing.T) {
	if a, ok := a.(*ErrorPosition); ok {
		if a.File != b.File {
			t.Errorf("ErrorPosision File do not match, expected %v, got: %v", b.File, a.File)
		}

		if a.Eval != b.Eval {
			t.Errorf("ErrorPosition Eval do not match, expected %v, got: %v", b.Eval, a.Eval)
		}

		if a.Eval && len(a.File) > 0 {
			t.Errorf("ErrorPosition Eval set and File non-empty")
		}

		if a.Row != b.Row {
			t.Errorf("ErrorPosition Row do not match, expected %v, got: %v", b.Row, a.Row)
		}

		if a.Column != b.Column {
			t.Errorf("ErrorPosition Column do not match, expected %v, got: %v", b.Column, a.Column)
		}

		if a.SourceFragment != b.SourceFragment {
			t.Errorf("ErrorPosition SourceFragment do not match, expected %v, got: %v", b.SourceFragment, a.SourceFragment)
		}

		if a.LocatedIn != b.LocatedIn {
			errorPositionsEqual(a.LocatedIn, b.LocatedIn, t)
		}
		return
	}
	t.Errorf("Expected *ErrorPosition, got: %v", a)
}

func verifyErrorPosition(ep *ErrorPosition) verifyFn {
	return func(x interface{}, t *testing.T) {
		errorPositionsEqual(x, ep, t)
	}
}

func TestErrorPositionString(t *testing.T) {
	var in = [...]string{
		"In eval expression, line 1, column 3:",
		">> 3 mod 0;",
		"Located in enclosing eval expression, at line 1, column 1:",
		">> eval \"3 mod 0;\";",
		"Located in:",
		">> eval \"eval \\\"3 mod 0;\\\";\";",
	}

	var out = &ErrorPosition{
		Eval:           true,
		Row:            1,
		Column:         3,
		SourceFragment: "3 mod 0;",
		LocatedIn: &ErrorPosition{
			Eval:           true,
			Row:            1,
			Column:         1,
			SourceFragment: "eval \"3 mod 0;\";",
			LocatedIn: &ErrorPosition{
				SourceFragment: "eval \"eval \\\"3 mod 0;\\\";\";",
			},
		},
	}

	testParser(&ErrorPositionParser{}, in[:], []verifyFn{verifyErrorPosition(out)}, t)
}

func TestErrorPositionFileString(t *testing.T) {
	var in = [...]string{
		"In eval expression, line 1, column 3:",
		">> 3 mod 0;",
		"Located in file \"/tmp/2.m\", at line 2, column 5:",
		">> eval \"3 mod 0;\";",
	}

	var out = &ErrorPosition{
		Eval:           true,
		Row:            1,
		Column:         3,
		SourceFragment: "3 mod 0;",
		LocatedIn: &ErrorPosition{
			File:           "/tmp/2.m",
			Row:            2,
			Column:         5,
			SourceFragment: "eval \"3 mod 0;\";",
		},
	}

	testParser(&ErrorPositionParser{}, in[:], []verifyFn{verifyErrorPosition(out)}, t)
}

func TestErrorPositionEvalFileString(t *testing.T) {
	var in = [...]string{
		"In eval expression, line 1, column 3:",
		">> 3 mod 0;",
		"Located in enclosing eval expression, at line 1, column 1:",
		">> eval \"3 mod 0;\";",
		"Located in file \"/tmp/1.m\", at line 2, column 5:",
		">> eval \"eval \\\"3 mod 0;\\\"\";",
	}

	var out = &ErrorPosition{
		Eval:           true,
		Row:            1,
		Column:         3,
		SourceFragment: "3 mod 0;",
		LocatedIn: &ErrorPosition{
			Eval:           true,
			Row:            1,
			Column:         1,
			SourceFragment: "eval \"3 mod 0;\";",
			LocatedIn: &ErrorPosition{
				File:           "/tmp/1.m",
				Row:            2,
				Column:         5,
				SourceFragment: "eval \"eval \\\"3 mod 0;\\\"\";",
			},
		},
	}

	testParser(&ErrorPositionParser{}, in[:], []verifyFn{verifyErrorPosition(out)}, t)
}
