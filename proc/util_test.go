// Copyright 2014, David Howden
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package proc

import (
	"bytes"
	"testing"
)

func TestFlushOutputToWriter(t *testing.T) {
	var in = `print "Hello";
for i in [1..10] do printf "X"; end for;
printf "Y\n";
IndentPush();
print "Z";
printf "Z";
printf "Z";
IndentPop();
print "Z";
print "Z";`
	var out = `Hello
XXXXXXXXXXY
    Z
    ZZZ
Z`
	test := func(p *Process, t errorfer) {
		o, err := p.Execute(in)
		checkFatalf(t, "Execute() error: %v", err)

		var buf bytes.Buffer
		err = FlushTaggedToWriter(o.Output(), &buf)
		checkErrorf(t, "FlushOutputToWriter() error: %v", err)

		output := string(buf.Bytes())

		if output != out {
			t.Errorf("Expected %v, got %v.", out, output)
		}

		testQuitAndWait(p, t)
	}

	runProcess(test, t)
}
