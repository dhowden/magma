// Copyright 2014, David Howden
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package parse

import "testing"

func sourcePositionsEqual(a, b Location, t *testing.T) {
	if a.File != b.File {
		t.Errorf("Source location file names do not match.  Expected: %v, got: %v", b.File, a.File)
	}

	if a.Row != b.Row {
		t.Errorf("Source location lines do not match.  Expected: %v, got: %v", b.Row, a.Row)
	}

	if a.Glue != b.Glue {
		t.Errorf("Source location glue do not match.  Expected: %v, got: %v", b.Glue, a.Glue)
	}
}

func paramValuesEqual(a, b ParamValue, t *testing.T) {
	if a.Name != b.Name {
		t.Errorf("Param names do not match. Expected: %v, got: %v.", b.Name, a.Name)
	}
	if a.Value != b.Value {
		t.Errorf("Param values do not match.  Expected %v, got: %v.", b.Value, a.Value)
	}
}

func tracebacksEqual(a interface{}, b *Traceback, t *testing.T) {
	if a, ok := a.(*Traceback); ok {
		if a.Index != b.Index {
			t.Errorf("Indexes do not match")
		}

		if a.Current != b.Current {
			t.Errorf("Current flags do not match.  Expected: %v, got %v.", b.Current, a.Current)
		}

		if a.Name != b.Name {
			t.Errorf("Names do not match.  Expected: %v, got: %v.", b.Name, a.Name)
		}

		if len(a.Params) != len(b.Params) {
			t.Errorf("Number of params does not match.  Expected:\n%v, got:\n%v.", b.Params, a.Params)
		} else {
			for i := range a.Params {
				paramValuesEqual(a.Params[i], b.Params[i], t)
			}
		}

		sourcePositionsEqual(a.Location, b.Location, t)
		return
	}
	t.Errorf("Expected *Traceback output, got: %v", a)
}

func verifyTraceback(tb *Traceback) verifyFn {
	return func(x interface{}, t *testing.T) {
		tracebacksEqual(x, tb, t)
	}
}

func TestTracebackParserString(t *testing.T) {
	var in = [...]string{
		"",
		"Test2(",
		"    x: 0",
		")",
		"Test(",
		"    x: 0",
		")",
		"",
	}

	var out1 = &Traceback{Index: NoIndex,
		Name:   "Test2",
		Params: []ParamValue{ParamValue{"x", "0"}},
	}

	var out2 = &Traceback{Index: NoIndex,
		Name:   "Test",
		Params: []ParamValue{ParamValue{"x", "0"}},
	}

	testParser(&TracebackParser{}, in[:], []verifyFn{verifyTraceback(out1), verifyTraceback(out2)}, t)
}

func TestTracebackParserString2(t *testing.T) {
	var in = [...]string{
		"",
		"AutomorphismGroupSolubleGroup(",
		"    G: GrpPC",
		")",
		"FixSubgroup(",
		"    A: A group of automorphisms of GrpPC,",
		"    H: GrpPC : H",
		")",
		"",
	}

	var out1 = &Traceback{Index: NoIndex,
		Name:   "AutomorphismGroupSolubleGroup",
		Params: []ParamValue{ParamValue{"G", "GrpPC"}},
	}

	var out2 = &Traceback{Index: NoIndex,
		Name: "FixSubgroup",
		Params: []ParamValue{
			ParamValue{"A", "A group of automorphisms of GrpPC"},
			ParamValue{"H", "GrpPC : H"}},
	}

	testParser(&TracebackParser{}, in[:], []verifyFn{verifyTraceback(out1), verifyTraceback(out2)}, t)
}

func TestTracebackParserEval(t *testing.T) {
	var in = [...]string{
		"",
		"[<string>:2](",
		")",
	}

	var out = &Traceback{Index: NoIndex,
		Name: "[<string>:2]",
	}

	testParser(&TracebackParser{}, in[:], []verifyFn{verifyTraceback(out)}, t)
}

func TestTracebackParserStringIndexMarkerLocation(t *testing.T) {
	var in = [...]string{
		"#0 *Test(",
		"    x: 0",
		") at <main>:2",
		"#1  Test2(",
		"    x: 0",
		") at <main>:3",
		"",
	}

	var out1 = &Traceback{Index: 0,
		Current:  true,
		Name:     "Test",
		Params:   []ParamValue{ParamValue{"x", "0"}},
		Location: Location{File: "<main>", Row: 2},
	}

	var out2 = &Traceback{Index: 1,
		Current:  false,
		Name:     "Test2",
		Params:   []ParamValue{ParamValue{"x", "0"}},
		Location: Location{File: "<main>", Row: 3},
	}

	testParser(&TracebackParser{}, in[:], []verifyFn{verifyTraceback(out1), verifyTraceback(out2)}, t)
}

func TestTracebackParserStringIndexMarkerLocation2(t *testing.T) {
	var in = [...]string{
		"#0 *FixSubgroup(",
		"    A: A group of automorphisms of GrpPC,",
		"    H: GrpPC : H",
		") at /Users/dave/git/magma/Prog/package/Group/GrpPC/aut/fix-subgroup.m:363",
		"#1 AutomorphismGroupSolubleGroup(",
		"    G: GrpPC",
		") at /Users/dave/git/magma/Prog/package/Group/GrpPC/aut/aut.m:713",
		"",
	}

	var out1 = &Traceback{Index: 0,
		Current: true,
		Name:    "FixSubgroup",
		Params: []ParamValue{
			ParamValue{"A", "A group of automorphisms of GrpPC"},
			ParamValue{"H", "GrpPC : H"}},
		Location: Location{File: "/Users/dave/git/magma/Prog/package/Group/GrpPC/aut/fix-subgroup.m", Row: 363},
	}

	var out2 = &Traceback{Index: 1,
		Current:  false,
		Name:     "AutomorphismGroupSolubleGroup",
		Params:   []ParamValue{ParamValue{"G", "GrpPC"}},
		Location: Location{File: "/Users/dave/git/magma/Prog/package/Group/GrpPC/aut/aut.m", Row: 713},
	}

	testParser(&TracebackParser{}, in[:], []verifyFn{verifyTraceback(out1), verifyTraceback(out2)}, t)
}
