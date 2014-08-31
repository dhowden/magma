// Copyright 2014, David Howden
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package parse

import (
	"bytes"
	"testing"
)

func TestTracebackWriter(t *testing.T) {
	var in1 = &Traceback{Index: 0,
		Current: true,
		Name:    "FixSubgroup",
		Params: []ParamValue{
			ParamValue{"A", "A group of automorphisms of GrpPC"},
			ParamValue{"H", "GrpPC : H"}},
		Location: Location{File: "/Users/dave/git/magma/Prog/package/Group/GrpPC/aut/fix-subgroup.m", Row: 363},
	}
	var in2 = &Traceback{Index: 1,
		Current:  false,
		Name:     "AutomorphismGroupSolubleGroup",
		Params:   []ParamValue{ParamValue{"G", "GrpPC"}},
		Location: Location{File: "/Users/dave/git/magma/Prog/package/Group/GrpPC/aut/aut.m", Row: 713},
	}

	var out = `0 *FixSubgroup(
    A : A group of automorphisms of GrpPC
    H : GrpPC : H
), defined in file: /Users/dave/git/magma/Prog/package/Group/GrpPC/aut/fix-subgroup.m, line 363
1 AutomorphismGroupSolubleGroup(
    G : GrpPC
), defined in file: /Users/dave/git/magma/Prog/package/Group/GrpPC/aut/aut.m, line 713
`

	var buf bytes.Buffer
	in1.WriteTo(&buf)
	in2.WriteTo(&buf)

	testOut := buf.String()

	if out != testOut {
		t.Errorf("Expected %v, but got %v.", out, testOut)
	}
}
