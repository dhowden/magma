// Copyright 2014, David Howden
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package parse

import (
	"testing"
	"time"

	"github.com/dhowden/magma/proc"
)

func paramsEqual(a, b Param, t *testing.T) {
	if a.Name != b.Name {
		t.Errorf("Param names do not match. Expected: %v, but got: %v", b.Name, a.Name)
	}
	if a.Type != b.Type {
		t.Errorf("Param types do not match.  Expected: %v, but got: %v", b.Type, a.Type)
	}
}

func signaturePositionsEqual(a, b SignatureLocation, t *testing.T) {
	sourcePositionsEqual(a.Location, b.Location, t)

	if a.Column != b.Column {
		t.Errorf("Columns do not match.  Expected %v, but got %v", b.Column, a.Column)
	}
}

func returnsEqual(a, b []string, t *testing.T) {
	if len(a) != len(b) {
		t.Errorf("Number of return values do not match.  Expected %v, but got: %v", len(b), len(a))
	}

	for i := range b {
		if a[i] != b[i] {
			t.Errorf("Return values at index %v do not match.  Expected `%v`, but got: `%v`", i, b[i], a[i])
		}
	}
}

func signaturesEqual(a interface{}, b *Signature, t *testing.T) {
	if a, ok := a.(*Signature); ok {
		signaturePositionsEqual(a.Location, b.Location, t)

		if a.Intrinsic != b.Intrinsic {
			t.Errorf("Intrinsics do not match.  Expected %v, but got: %v", b.Intrinsic, a.Intrinsic)
		}

		if len(a.Params) != len(b.Params) {
			t.Errorf("Params do not match,  Expected %v, but got: %v", len(b.Params), len(a.Params))
		}

		for i := range a.Params {
			paramsEqual(a.Params[i], b.Params[i], t)
		}

		returnsEqual(a.Returns, b.Returns, t)

		if len(a.OptionalParams) != len(b.OptionalParams) {
			t.Errorf("Number of optional params does not match. Expected %v, but got: %v", len(b.OptionalParams), len(a.OptionalParams))
		}

		for i := range a.OptionalParams {
			paramsEqual(a.OptionalParams[i], b.OptionalParams[i], t)
		}
		return
	}
	t.Errorf("Expected *Signature output, got: %v", a)
}

func verifySignature(s *Signature) verifyFn {
	return func(x interface{}, t *testing.T) {
		signaturesEqual(x, s, t)
	}
}

func TestSignatureParserString(t *testing.T) {
	var in = [...]string{
		"Intrinsic 'AutomorphismGroupSolubleGroup'",
		"",
		"Signatures:",
		"",
		"    Defined in file: /magma/Prog/package/Group/GrpPC/aut/aut.m, line 722, column 20:",
		"    (G::GrpPC) -> GrpAuto",
		"    [",
		"        p",
		"    ]",
		"",
		"        Computes the automorphism group of the soluble group G, with the optional parameter 'p' which should be a prime ",
		"        dividing the order of G (the calculation relies on Aut(Syl_p(G))). Default value of p is taken to be the prime ",
		"        diving the order of G which defines the largest Sylow p-subgroup.",
		"",
		"    Defined in file: /magma/Prog/package/Group/GrpPC/aut/aut.m, line 873, column 5:",
		"    (G::GrpPC, p::RngIntElt) -> GrpAuto",
		"",
		"        Computes the automorphism group of the soluble group G using the automorphism group of a Sylow p-subgroup of G. ",
		"        Setting p to 1 is equivalent to calling AutomorphismGroupSolubleGroup(G).",
		"",
	}

	var out1 = &Signature{
		Intrinsic: "AutomorphismGroupSolubleGroup",
		Location: SignatureLocation{
			Location: Location{
				File: "/magma/Prog/package/Group/GrpPC/aut/aut.m",
				Row:  722,
			},
			Column: 20},
		Params: []Param{
			Param{
				Name: "G",
				Type: "GrpPC",
			},
		},
		Returns:        []string{"GrpAuto"},
		OptionalParams: []Param{Param{Name: "p"}},
		Comment: "Computes the automorphism group of the soluble group G, with the optional parameter 'p' which should be a " +
			"prime dividing the order of G (the calculation relies on Aut(Syl_p(G))). Default value of p is taken to be the prime " +
			"diving the order of G which defines the largest Sylow p-subgroup.",
	}

	var out2 = &Signature{
		Intrinsic: "AutomorphismGroupSolubleGroup",
		Location: SignatureLocation{
			Location: Location{
				File: "/magma/Prog/package/Group/GrpPC/aut/aut.m",
				Row:  873,
			},
			Column: 5},
		Params: []Param{
			Param{
				Name: "G",
				Type: "GrpPC",
			},
			Param{
				Name: "p",
				Type: "RngIntElt",
			},
		},
		Returns:        []string{"GrpAuto"},
		OptionalParams: []Param{},
		Comment: "Computes the automorphism group of the soluble group G using the automorphism group of a Sylow p-subgroup of G. " +
			"Setting p to 1 is equivalent to calling AutomorphismGroupSolubleGroup(G).",
	}

	testParser(&SignatureParser{}, in[:], []verifyFn{verifySignature(out1), verifySignature(out2)}, t)
}

func TestSignatureParserStringMultipleReturn(t *testing.T) {
	var in = [...]string{
		"Intrinsic 'IsIsomorphicSolubleGroup'",
		"",
		"Signatures:",
		"",
		"    (G1::GrpPC, G2::GrpPC) -> BoolElt, Map",
		"    [",
		"        p",
		"    ]",
		"",
		"        Performs an isomorphism test between the soluble groups G1 and G2, with ",
		"        the optional parameter 'p' which should be a prime dividing the order of",
		"        G (the calculation relies on IsIsomorphic(Syl_p(G_i)) and ",
		"        Aut(Syl_p(G_i)) for i = 1,2. Default value of p is taken to be one which",
		"        defines the largest Sylow p-subgroup.",
		"",
		"",
	}

	var out = &Signature{
		Intrinsic: "IsIsomorphicSolubleGroup",
		Params: []Param{
			Param{
				Name: "G1",
				Type: "GrpPC",
			},
			Param{
				Name: "G2",
				Type: "GrpPC",
			},
		},
		Returns: []string{"BoolElt", "Map"},
		OptionalParams: []Param{
			Param{Name: "p"},
		},
		Comment: "Performs an isomorphism test between the soluble groups G1 and G2, with the optional parameter 'p' " +
			"which should be a prime dividing the order of G (the calculation relies on IsIsomorphic(Syl_p(G_i)) and " +
			"Aut(Syl_p(G_i)) for i = 1,2. Default value of p is taken to be one which defines the largest Sylow p-subgroup.",
	}

	testParser(&SignatureParser{}, in[:], []verifyFn{verifySignature(out)}, t)
}

func TestSignatureParserMapInputString(t *testing.T) {
	var in = [...]string{
		"Intrinsic 'HighestWeights'",
		"",
		"Signatures:",
		"",
		"    (rho::Map[AlgLie, AlgMatLie]) -> SeqEnum, SeqEnum",
		"    [",
		"        Basis",
		"    ]",
		"",
		"    The highest weights of rho.",
		"",
		"",
	}

	var out = &Signature{
		Intrinsic: "HighestWeights",
		Params: []Param{
			Param{
				Name: "rho",
				Type: "Map[AlgLie, AlgMatLie]",
			},
		},
		Returns: []string{"SeqEnum", "SeqEnum"},
		OptionalParams: []Param{
			Param{
				Name: "Basis",
			},
		},
		Comment: "The highest weights of rho.",
	}

	testParser(&SignatureParser{}, in[:], []verifyFn{verifySignature(out)}, t)
}

func TestSignatureParserNoParamName(t *testing.T) {
	var in = [...]string{
		"Intrinsic 'Order'",
		"",
		"Signatures:",
		"",
		"    (::RngFunOrd, l::SeqEnum[FldFunElt]) -> RngFunOrd",
		"    [",
		"        Verify: BoolElt,",
		"        Order: BoolElt",
		"    ]",
		"",
		"    The minimal order containing all elements of l.",
		"",
		"",
	}

	var out = &Signature{
		Intrinsic: "Order",
		Params: []Param{
			Param{
				Name: "",
				Type: "RngFunOrd",
			},
			Param{
				Name: "l",
				Type: "SeqEnum[FldFunElt]",
			},
		},
		Returns: []string{"RngFunOrd"},
		OptionalParams: []Param{
			Param{
				Name: "Verify",
				Type: "BoolElt",
			},
			Param{
				Name: "Order",
				Type: "BoolElt",
			},
		},
		Comment: "The minimal order containing all elements of l.",
	}

	testParser(&SignatureParser{}, in[:], []verifyFn{verifySignature(out)}, t)
}

func TestSignatureParserNoReturn(t *testing.T) {
	var in = [...]string{
		"Intrinsic 'ListSignatures'",
		"",
		"Signatures:",
		"",
		"    (C::Cat)",
		"    [",
		"        Search: \"Arguments\" | \"Both\" | \"ReturnValues\", ",
		"        Isa: BoolElt, ",
		"        ShowSrc: BoolElt",
		"    ]",
		"",
		"        List the signatures of all intrinsics which have an argument belonging to category C.",
		"",
		"",
	}

	var out = &Signature{
		Intrinsic: "ListSignatures",
		Params: []Param{
			Param{
				Name: "C",
				Type: "Cat",
			},
		},
		OptionalParams: []Param{
			Param{
				Name: "Search",
				Type: "\"Arguments\" | \"Both\" | \"ReturnValues\"",
			},
			Param{
				Name: "Isa",
				Type: "BoolElt",
			},
			Param{
				Name: "ShowSrc",
				Type: "BoolElt",
			},
		},
		Comment: "List the signatures of all intrinsics which have an argument belonging to category C.",
	}

	testParser(&SignatureParser{}, in[:], []verifyFn{verifySignature(out)}, t)
}

func TestSignatureParserNoParamNoReturn(t *testing.T) {
	var in = [...]string{
		"Intrinsic 'ShowIdentifiers'",
		"",
		"Signatures:",
		"",
		"    ()",
		"",
		"        List all the currently assigned identifiers.",
		"",
		"",
	}

	var out = &Signature{
		Intrinsic: "ShowIdentifiers",
		Comment:   "List all the currently assigned identifiers.",
	}

	testParser(&SignatureParser{}, in[:], []verifyFn{verifySignature(out)}, t)
}

func TestListSignatures(t *testing.T) {
	var in = [...]string{
		"Signatures matching (GrpPC) -> RngIntElt:",
		"",
		"    '#'(G::GrpPC) -> RngIntElt",
		"",
		"        No comment assigned.",
		"",
		"    AbelianSection(G::GrpPC) -> RngIntElt, RngIntElt",
		"",
		"        Returns the minimal index i s.t. G_i := <g_i,...g_n> is abelian, and exp(G_i).",
		"",
	}

	var out1 = &Signature{
		Intrinsic: "'#'",
		Params: []Param{
			Param{
				Name: "G",
				Type: "GrpPC",
			},
		},
		Returns: []string{"RngIntElt"},
		Comment: "No comment assigned.",
	}

	var out2 = &Signature{
		Intrinsic: "AbelianSection",
		Params: []Param{
			Param{
				Name: "G",
				Type: "GrpPC",
			},
		},
		Returns: []string{"RngIntElt", "RngIntElt"},
		Comment: "Returns the minimal index i s.t. G_i := <g_i,...g_n> is abelian, and exp(G_i).",
	}

	testParser(&SignatureParser{}, in[:], []verifyFn{verifySignature(out1), verifySignature(out2)}, t)
}

func TestSignatureParserOptionalParamTypesString(t *testing.T) {
	var in = [...]string{
		"Intrinsic 'AutomorphismGroup'",
		"",
		"Signatures:",
		"",
		"   (G::GrpPC) -> GrpAuto",
		"   [",
		"       CharacteristicSubgroups: SeqEnum, ",
		"       Algorithm: \"Default\" | \"PermGrp\" | \"SolGrp\" | \"pGrp\"",
		"   ]",
		"   ",
		"       The automorphism group of the group G.",
		"",
		"   (L::Lat, F::SeqEnum[Mtrx], S::SeqEnum[SetEnum[Mtrx]]) -> GrpMat",
		"   [",
		"       Depth,",
		"       BacherDepth,",
		"       BacherSCP,",
		"       Stabilizer,",
		"       Generators,",
		"       NaturalAction,",
		"       Decomposition,",
		"       VectorsLimit,",
		"       Vectors",
		"   ]",
		"",
		"       The subgroup of the automorphism group of the lattice L fixing the forms in F individually and",
		"       the forms in S setwise.",
		"",
	}

	var out1 = &Signature{
		Intrinsic: "AutomorphismGroup",
		Params: []Param{
			Param{
				Name: "G",
				Type: "GrpPC",
			},
		},
		Returns: []string{"GrpAuto"},
		OptionalParams: []Param{
			Param{"CharacteristicSubgroups", "SeqEnum"},
			Param{"Algorithm", "\"Default\" | \"PermGrp\" | \"SolGrp\" | \"pGrp\""}},
		Comment: "The automorphism group of the group G.",
	}

	var out2 = &Signature{
		Intrinsic: "AutomorphismGroup",
		Params: []Param{
			Param{
				Name: "L",
				Type: "Lat",
			},
			Param{
				Name: "F",
				Type: "SeqEnum[Mtrx]",
			},
			Param{
				Name: "S",
				Type: "SeqEnum[SetEnum[Mtrx]]"},
		},
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

	testParser(&SignatureParser{}, in[:], []verifyFn{verifySignature(out1), verifySignature(out2)}, t)
}

func TestSignatureParserUsingProcess(t *testing.T) {
	var in = "AutomorphismGroupSolubleGroup;"

	var out1 = &Signature{
		Intrinsic: "AutomorphismGroupSolubleGroup",
		Params: []Param{
			Param{
				Name: "G",
				Type: "GrpPC",
			},
		},
		Returns:        []string{"GrpAuto"},
		OptionalParams: []Param{Param{Name: "p"}},
		Comment: "Computes the automorphism group of the soluble group G, with the optional parameter 'p' which should be a " +
			"prime dividing the order of G (the calculation relies on Aut(Syl_p(G))). Default value of p is taken to be the prime " +
			"diving the order of G which defines the largest Sylow p-subgroup.",
	}

	var out2 = &Signature{
		Intrinsic: "AutomorphismGroupSolubleGroup",
		Params: []Param{
			Param{
				Name: "G",
				Type: "GrpPC",
			},
			Param{
				Name: "p",
				Type: "RngIntElt",
			},
		},
		Returns:        []string{"GrpAuto"},
		OptionalParams: []Param{},
		Comment: "Computes the automorphism group of the soluble group G using the automorphism group of a Sylow p-subgroup of G. " +
			"Setting p to 1 is equivalent to calling AutomorphismGroupSolubleGroup(G).",
	}

	// Create a new process and start it
	testSignatureParser := func(p *proc.Process, st <-chan proc.Tagged, so *proc.Output) error {
		go emptyTaggedChToLogPrintf("Status tag received: %v", st)
		go emptyTaggedChToLogPrintf("Startup output: %v", so.Output())

		// Execute the command
		c, err := p.Execute(in)
		checkErrorf(t, "Execute() error: %v", err)

		out := make(chan proc.Tagged)
		go func() {
			for x := range c.Output() {
				if x, ok := x.(*proc.Line); ok {
					out <- x
					continue
				}
				t.Errorf("Expected *proc.Line, got %v", x)
			}
			close(out)
		}()

		// Feed the output into the signature parser
		sp := &SignatureParser{}
		ch := sp.Run(out)

		testChannelOutput(ch, []verifyFn{verifySignature(out1), verifySignature(out2)}, t)

		qch, err := p.Quit()
		checkErrorf(t, "Quit() error: %v", err)

		// Wait for a quit
		select {
		case <-qch:
			// Now will quit
		case <-time.After(5 * time.Second):
			t.Errorf("Quit command timed out")
		}
		return nil
	}

	err := proc.Launch(&proc.Process{}, testSignatureParser)
	checkErrorf(t, "Launch() error: %v", err)
}
