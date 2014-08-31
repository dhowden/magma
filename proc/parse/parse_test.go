// Copyright 2014, David Howden
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package parse

import (
	"log"
	"testing"
	"time"

	"github.com/dhowden/magma/proc"
)

func outputLineConsumerProvider(arr []string) *lineConsumer {
	ch := make(chan string)
	go func() {
		for _, x := range arr {
			ch <- x
		}
		close(ch)
	}()
	return newLineConsumer(ch)
}

type verifyFn func(interface{}, *testing.T)

func testParser(p parser, in []string, v []verifyFn, t *testing.T) {
	src := outputLineConsumerProvider(in)
	ch := p.start(src)
	testChannelOutput(ch, v, t)
}

func testChannelOutput(ch <-chan interface{}, v []verifyFn, t *testing.T) {
	for _, f := range v {
		if x, ok := <-ch; ok {
			f(x, t)
			continue
		}
		t.Errorf("expected output but channel was closed")
	}

	if x, ok := <-ch; ok {
		t.Errorf("expected output channel to be closed, but got: %v", x)
	}
}

// errorfer allows testing.T and testing.B to be passed to helper functions
type errorfer interface {
	Errorf(string, ...interface{})
	Fatalf(string, ...interface{})
	Error(...interface{})
}

func checkError(t errorfer, msg string, err error) {
	if err != nil {
		t.Error(msg, err)
	}
}

func checkErrorf(t errorfer, fmt string, err error) {
	if err != nil {
		t.Errorf(fmt, err)
	}
}

func checkFatalf(t errorfer, fmt string, err error) {
	if err != nil {
		t.Fatalf(fmt, err)
	}
}

func emptyTaggedChToLogPrintf(format string, ch <-chan proc.Tagged) {
	for x := range ch {
		log.Printf(format, x)
	}
}

func TestParseTagged(t *testing.T) {
	var in = "AutomorphismGroupSolubleGroup;"

	var out1 = &Signature{
		Intrinsic:      "",
		Params:         []Param{Param{Type: "GrpPC", Name: "G"}},
		Returns:        []string{"GrpAuto"},
		OptionalParams: []Param{Param{Name: "p"}},
		Comment: "Computes the automorphism group of the soluble group G, with the optional parameter 'p' which should be a " +
			"prime dividing the order of G (the calculation relies on Aut(Syl_p(G))). Default value of p is taken to be the prime " +
			"diving the order of G which defines the largest Sylow p-subgroup.",
	}

	var out2 = &Signature{
		Intrinsic:      "",
		Params:         []Param{Param{Type: "GrpPC", Name: "G"}, Param{Type: "RngIntElt", Name: "p"}},
		Returns:        []string{"GrpAuto"},
		OptionalParams: []Param{},
		Comment: "Computes the automorphism group of the soluble group G using the automorphism group of a Sylow p-subgroup of G. " +
			"Setting p to 1 is equivalent to calling AutomorphismGroupSolubleGroup(G).",
	}

	// Create a new process and start it
	m := &proc.Process{}
	st, _ := m.StatusTags()
	go emptyTaggedChToLogPrintf("Status tag received: %v", st)

	so, err := m.Start()
	checkErrorf(t, "Start() error: %v", err)
	if err != nil {
		return
	}
	defer func() {
		if err != nil {
			err := m.Wait()
			checkErrorf(t, "Wait() error: %v", err)
		}
	}()
	go emptyTaggedChToLogPrintf("Startup output: %v", so.Output())

	// Execute the command
	c, err := m.Execute(in)
	checkFatalf(t, "Execute() error: %v", err)

	out := make(chan interface{})
	go ParseTagged(c.Output(), out, &SignatureParser{})

	testChannelOutput(out, []verifyFn{verifySignature(out1), verifySignature(out2)}, t)

	qch, err := m.Quit()
	checkErrorf(t, "Quit() error: %v", err)

	// Wait for a quit
	select {
	case <-qch:
		// Now will quit
	case <-time.After(5 * time.Second):
		t.Errorf("Quit command timed out")
	}
}
