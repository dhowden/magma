// Copyright 2014, David Howden
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package parse

import (
	"bytes"
	"testing"
)

func TestSignatureWriter(t *testing.T) {
	var in1 = &Signature{
		Intrinsic: "",
		Location: SignatureLocation{
			Location: Location{Glue: "glue_grp_pc_automorphism_group()"},
		},
		Params:  []Param{Param{Name: "G", Type: "GrpPC"}},
		Returns: []string{"GrpAuto"},
		OptionalParams: []Param{
			Param{"CharacteristicSubgroups", "SeqEnum"},
			Param{"Algorithm", "\"Default\" | \"PermGrp\" | \"SolGrp\" | \"pGrp\""}},
		Comment: "The automorphism group of the group G.",
	}

	var in2 = &Signature{
		Intrinsic: "",
		Location: SignatureLocation{
			Location: Location{
				File: "/magma/Prog/package/Lattice/Lat/auto.m",
				Row:  1070,
			},
			Column: 5,
		},
		Params: []Param{
			Param{Name: "L", Type: "Lat"},
			Param{Name: "F", Type: "SeqEnum[Mtrx]"},
			Param{Name: "S", Type: "SeqEnum[SetEnum[Mtrx]]"}},
		Returns: []string{"GrpMat"},
		OptionalParams: []Param{
			Param{Name: "Depth"},
			Param{Name: "BacherDepth"},
			Param{Name: "BacherSCP"},
			Param{Name: "Stabilizer"},
			Param{Name: "Generators"},
			Param{Name: "NaturalAction"},
			Param{Name: "Decomposition"},
			Param{Name: "VectorsLimit"},
			Param{Name: "Vectors"},
		},
		Comment: "The subgroup of the automorphism group of the lattice L fixing the forms in F individually and the forms in S setwise.",
	}

	var out = `Defined in glue: glue_grp_pc_automorphism_group():
(G::GrpPC) -> GrpAuto
[
    CharacteristicSubgroups : SeqEnum
    Algorithm : "Default" | "PermGrp" | "SolGrp" | "pGrp"
]
The automorphism group of the group G.
Defined in file: /magma/Prog/package/Lattice/Lat/auto.m, line 1070, column 5:
(L::Lat, F::SeqEnum[Mtrx], S::SeqEnum[SetEnum[Mtrx]]) -> GrpMat
[
    Depth
    BacherDepth
    BacherSCP
    Stabilizer
    Generators
    NaturalAction
    Decomposition
    VectorsLimit
    Vectors
]
The subgroup of the automorphism group of the lattice L fixing the forms in F individually and the forms in S setwise.
`
	var buf bytes.Buffer
	in1.WriteTo(&buf)
	in2.WriteTo(&buf)

	testOut := buf.String()

	if out != testOut {
		t.Errorf("Expected %v, but got %v.", out, testOut)
	}
}
