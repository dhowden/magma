// Copyright 2014, David Howden
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package proc

import (
	"fmt"
	"log"
	"os"
	"testing"
	"time"
)

// errorfer allows testing.T and testing.B to be passed to helper functions
type errorfer interface {
	Errorf(string, ...interface{})
	Fatalf(string, ...interface{})
	Error(...interface{})
}

func checkError(t errorfer, err error) {
	if err != nil {
		t.Error(err)
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

func positionsEqual(a interface{}, b Position, t errorfer) {
	if a, ok := a.(*Position); ok {
		if a.Tag() != TagErrorHistoryPosition {
			t.Errorf("expected Tag() of Position to be TagErrorHistoryPosition, got: %v", a.Tag())
		}
		if a.Row != b.Row {
			t.Errorf("Position.Row do not match expected %v, got: %v", b.Row, a.Row)
		}
		if a.Column != b.Column {
			t.Errorf("Position Column do not match expected %v, got: %v", b.Column, a.Column)
		}
		return
	}
	t.Errorf("Expected Position, got %v", a)
}

func outputsEqual(a interface{}, b *Line, t errorfer) {
	if a, ok := a.(*Line); ok {
		if a.Continuation != b.Continuation {
			t.Errorf("Expected Continuation %v, got: %v", b.Continuation, a.Continuation)
		}
		if a.Indent != b.Indent {
			t.Errorf("Expected Indent %v, got: %v", b.Indent, a.Indent)
		}
		if a.Data != b.Data {
			t.Errorf("Expected Data %v, got: %v", b.Data, a.Data)
		}

		return
	}
	t.Errorf("Expected *Line, got: %v", a)
}

type tagged interface {
	Tag() tag
}

func emptyTaggedChToLogPrintf(format string, ch <-chan Tagged) {
	for x := range ch {
		log.Printf(format, x)
	}
}

func NewTestOutput(t tag, data string) *Line {
	return &Line{tag: t, Data: data}
}

func checkSeedOutput(t errorfer, o *Run) {
	if o.Seed == nil {
		t.Error("Expected seed value to be set on *Run struct")
	}
}

type processFn func(p *Process, t errorfer)
type processWithStatusFn func(p *Process, t errorfer, ch <-chan Tagged)

func runProcess(f processFn, t errorfer) error {
	fst := func(p *Process, t errorfer, ch <-chan Tagged) {
		go emptyTaggedChToLogPrintf("Status tag received: %v", ch)
		go f(p, t)
	}
	err := runCustomProcess(&Process{}, fst, t)
	checkError(t, err)
	return err
}

func runProcessWithStatus(f processWithStatusFn, t errorfer) error {
	// Create a new process and start it
	err := runCustomProcess(&Process{}, f, t)
	checkError(t, err)
	return err
}

func runCustomProcess(p *Process, f processWithStatusFn, t errorfer) (err error) {
	st, _ := p.StatusTags()

	so, err := p.Start()
	if err != nil {
		err = fmt.Errorf("unexpected Start() error: %v", err)
		return
	}
	defer func() {
		if err == nil {
			err = p.Wait()
			if err != nil {
				err = fmt.Errorf("unexpected Wait() error: %v", err)
			}
		}
	}()
	go emptyTaggedChToLogPrintf("Startup output: %v", so.Output())

	go f(p, t, st)
	return
}

func testQuitAndWait(p *Process, t errorfer) {
	// Send quit
	log.Print("Sending quit...")
	qch, err := p.Quit()
	checkErrorf(t, "Quit() error: %v", err)

	// Wait for a quit
	select {
	case <-qch:
		// Now will quit
	case <-time.After(5 * time.Second):
		t.Errorf("Quit command timed out")
	}
}

// Test submission of commands and retreival of output
func TestProcess(t *testing.T) {
	const in, out = "a := 1; print a;", "1"

	test := func(p *Process, t errorfer) {
		// Send command
		log.Printf("Sending command: %v", in)
		o, err := p.Execute(in)
		checkFatalf(t, "Execute() error: %v", err)

		ch := o.Output()

		select {
		case x := <-ch:
			if x, ok := x.(*Line); ok {
				if x.Data != out {
					t.Errorf("Process sent: %v, got %v, want: %v", in, x.Data, out)
				}
			} else {
				t.Errorf("Process returned: %v, expected output of type: exec.Line", x)
			}
		case <-time.After(5 * time.Second):
			t.Errorf("Evaluation timed out")
		}

		testQuitAndWait(p, t)
	}
	runProcess(test, t)
}

// Test submission of commands and retreival of output
func TestProcessInvalidExecutable(t *testing.T) {
	const in, out = "a := 1; print a;", "1"

	test := func(p *Process, t errorfer, st <-chan Tagged) {
		go emptyTaggedChToLogPrintf("Status tag received: %v", st)

		// Send command
		log.Printf("Sending command: %v", in)
		o, err := p.Execute(in)
		checkErrorf(t, "Execute() error: %v", err)

		ch := o.Output()

		select {
		case x := <-ch:
			if x, ok := x.(*Line); ok {
				if x.Data != out {
					t.Errorf("Process sent: %v, got %v, want: %v", in, x.Data, out)
				}
			} else {
				t.Errorf("Process returned: %v, expected output of type: exec.Line", x)
			}
		case <-time.After(5 * time.Second):
			t.Errorf("Evaluation timed out")
		}

		testQuitAndWait(p, t)
	}
	p := &Process{Command: "vboptwrong"}
	err := runCustomProcess(p, test, t)
	if err == nil {
		t.Errorf("expected an error")
	}
}

// Test handling of output continuation and indents
func TestContinuedLinesAndIndent(t *testing.T) {
	// NB. we're relying on the internal Magma output buffer being 1024 characters
	// to force a continuation
	const in = `
		procedure p()
			IndentPush();
			t := "";
			for i in [1..1025] do
				t cat:= "X";
			end for;
			print t;
			IndentPop();
		end procedure;
		p(); print "Y";`
	var out1 = &Line{
		Indent: 1,
	}
	for i := 0; i < 1024; i++ {
		out1.Data += "X"
	}
	var out2 = &Line{
		Data:         "X",
		Continuation: true,
	}
	var out3 = &Line{
		Data: "Y",
	}

	test := func(p *Process, t errorfer) {
		// Send command
		log.Printf("Sending command: %v", in)
		o, err := p.Execute(in)
		checkFatalf(t, "Execute() error: %v", err)

		ch := o.Output()

		// checkSeedOutput(t, o)

		select {
		case x := <-ch:
			outputsEqual(x, out1, t)
		case <-time.After(5 * time.Second):
			t.Errorf("Evaluation timed out")
		}
		select {
		case x := <-ch:
			outputsEqual(x, out2, t)
		case <-time.After(5 * time.Second):
			t.Errorf("Timed out waiting for second line of output")
		}
		select {
		case x := <-ch:
			outputsEqual(x, out3, t)
		case <-time.After(5 * time.Second):
			t.Errorf("Timed out waiting for second line of output")
		}

		testQuitAndWait(p, t)
	}
	runProcess(test, t)
}

// Test trivial statement (i.e. empty, or commented input, which doesn't produce
// a RUN tag, or any other output tag.)
func TestTrivialStmt(t *testing.T) {
	const in = "/* Comment! */"

	test := func(p *Process, t errorfer) {
		// Send command
		log.Printf("Sending command: %v", in)
		rch := make(chan (<-chan Tagged), 1)
		go func() {
			o, err := p.Execute(in)
			checkFatalf(t, "Execute() error: %v", err)
			rch <- o.Output()
		}()

		select {
		case ch := <-rch:
			if x, ok := <-ch; ok {
				t.Errorf("Process returned: %v, expected output channel to be closed with no output", x)
			}
		case <-time.After(5 * time.Second):
			t.Errorf("Execute timed out")
		}

		testQuitAndWait(p, t)
	}
	runProcess(test, t)
}

// Test trivial input (i.e. empty, or commented input, which doesn't produce
// a RUN tag, or any other output tag.)
func TestErrorPositionParsing(t *testing.T) {
	const in = "1 mod 0;"
	pos := Position{Row: 0, Column: 2}

	test := func(p *Process, t errorfer) {
		// Send command
		log.Printf("Sending command: %v", in)
		rch := make(chan (<-chan Tagged), 1)
		go func() {
			o, err := p.Execute(in)
			checkFatalf(t, "Execute() error: %v", err)
			rch <- o.Output()
		}()

		select {
		case ch := <-rch:
			// First output is an empty TB
			if x, ok := <-ch; ok {
				if tx, txok := x.(Tagged); txok {
					if tx.Tag() != TagTraceback {
						t.Errorf("Expected traceback tagged output but got: %v", tx)
					}
				} else {
					t.Errorf("Expected taggedoutput but got: %v", x)
				}
			} else {
				t.Error("expected Position from <-ch, but ch was closed")
			}

			// Second output is the POS
			if x, ok := <-ch; ok {
				positionsEqual(x, pos, t)
			} else {
				t.Error("expected Position from <-ch, but ch was closed")
			}

			// Don't care about the rest of the output
			for _ = range ch {
			}
		case <-time.After(1 * time.Second):
			t.Errorf("execute timed out")
		}

		testQuitAndWait(p, t)
	}
	runProcess(test, t)
}

// Test interrupting
func TestProcessInterrupt(t *testing.T) {
	const in = "i := 0; while i lt 1 do print i; end while;"

	test := func(p *Process, t errorfer) {
		log.Printf("Sending command: %v", in)
		o, err := p.Execute(in)
		checkFatalf(t, "Execute() error: %v", err)

		go func() {
			for x := range o.Output() {
				log.Printf("Line: %v", x)
			}
		}()

		// Send interrupt
		log.Print("Sending interrupt...")
		ich, err := p.InterruptExecution()
		checkErrorf(t, "Interrupt() error: %v", err)

		// Wait for an interrupt
		select {
		case <-ich:
			// Now will quit
		case <-time.After(5 * time.Second):
			t.Errorf("Interrupt command timed out")
		}

		testQuitAndWait(p, t)
	}
	runProcess(test, t)
}

func TestProcessInterruptNoRun(t *testing.T) {
	// Create a new process and start it
	test := func(p *Process, t errorfer) {
		o, err := p.Execute("1;")
		checkFatalf(t, "Execute error: %v", err)
		emptyTaggedChToLogPrintf("Line: %v", o.Output())

		// Send interrupt
		log.Print("Sending interrupt...")
		ich, err := p.InterruptExecution()
		checkErrorf(t, "Interrupt() error: %v", err)

		// Wait for an interrupt
		select {
		case <-ich:
			// Now will quit
		case <-time.After(5 * time.Second):
			t.Errorf("Interrupt command timed out")
		}

		o, err = p.Execute("1;")
		checkErrorf(t, "Execute error: %v", err)
		emptyTaggedChToLogPrintf("Line: %v", o.Output())

		testQuitAndWait(p, t)
	}
	runProcess(test, t)
}

func TestExternalProcessInterrupt(t *testing.T) {
	const in = "i := 0; while i lt 1 do print i; end while;"

	test := func(p *Process, t errorfer, st <-chan Tagged) {
		ich := make(chan struct{})
		go func() {
			for x := range st {
				if x, ok := x.(*Status); ok {
					if x.Tag() == TagInterrupt {
						close(ich)
					}
				}
				log.Print("Status tag received: ", x)
			}
		}()

		log.Printf("Sending command: %v", in)
		o, err := p.Execute(in)
		checkFatalf(t, "Execute() error: %v", err)

		go emptyTaggedChToLogPrintf("Line: %v", o.Output())

		// Fetch the pid of the underlying magma process
		pid, err := p.Getpid()
		checkErrorf(t, "Error retrieving pid: %v", err)

		// Find it
		ep, err := os.FindProcess(pid)
		checkErrorf(t, "Error in FindProcess(): %v", err)

		// Send an interrupt signal to the external process
		err = ep.Signal(os.Interrupt)
		checkErrorf(t, "Error sending os.Interrupt", err)

		// Wait for an interrupt (via status channel)
		select {
		case <-ich:
			// Now will quit
		case <-time.After(5 * time.Second):
			t.Errorf("Interrupt command timed out")
		}

		testQuitAndWait(p, t)
	}
	runProcessWithStatus(test, t)
}

// Test submission of commands and retreival of output
func TestReadStatement(t *testing.T) {
	const promptIn = "x\\n:"
	const promptOut = "x\n:"
	const read = `read x`
	const in = `1`
	const in2 = `print x;`
	const out2 = `1`

	test := func(p *Process, t errorfer) {
		// Send command
		command := read + ", \"" + promptIn + "\";"
		log.Printf("Sending command: %v", command)
		o, err := p.Execute(command)
		checkFatalf(t, "Execute() error: %v", err)

		// checkSeedOutput(t, o)
		ch := o.Output()

		select {
		case x := <-ch:
			if x, ok := x.(*ReadRequest); ok {
				log.Printf("ReadRequest received")
				if x.Prompt != promptOut {
					t.Errorf("Process sent prompt: %v, want: %v", x.Prompt, promptOut)
				}
				log.Printf("ReadRequest prompt: %v", x.Prompt)

				select {
				case x.Output <- in:
					log.Printf("Output sent")
				case <-time.After(5 * time.Second):
					t.Errorf("Output posting timed out")
				}
			} else {
				t.Errorf("Process returned: %v, expected output of type: exec.ReadRequest", x)
			}
		case <-time.After(5 * time.Second):
			t.Errorf("Evaluation timed out")
		}

		o, err = p.Execute(in2)
		checkErrorf(t, "Execute() error: %v", err)

		// checkErrorSeedOutput(t, o)
		ch = o.Output()

		select {
		case x := <-ch:
			if x, ok := x.(*Line); ok {
				if x.Data != out2 {
					t.Errorf("Process sent: %v, got %v, want: %v", in2, x.Data, out2)
				}
			} else {
				t.Errorf("Process returned: %v, expected output of type: exec.Line", x)
			}
		case <-time.After(5 * time.Second):
			t.Errorf("Evaluation timed out")
		}

		testQuitAndWait(p, t)
	}
	runProcess(test, t)
}

// Test submission of commands and retreival of output
func TestReadStatementContinuedPrompt(t *testing.T) {
	// NB. we're relying on the internal Magma output buffer being 1024 characters
	// to force a continuation
	const command = `Prompt := "";
for i in [1..1025] do
	Prompt cat:= "X";
end for;
Prompt cat:= "\nY";`
	const readCommand = `read x, Prompt;`
	const in = `1`
	const in2 = `print x;`
	const out2 = `1`
	var promptOut = ""
	for i := 0; i < 1025; i++ {
		promptOut += "X"
	}
	promptOut += "\nY"

	test := func(p *Process, t errorfer) {
		// Send command
		cmd := command + "\n" + readCommand
		log.Printf("Sending command: %v\n", cmd)
		o, err := p.Execute(cmd)
		checkFatalf(t, "Execute() error: %v", err)

		// checkSeedOutput(t, o)

		ch := o.Output()

		select {
		case x := <-ch:
			if x, ok := x.(*ReadRequest); ok {
				log.Printf("ReadRequest received")
				if x.Prompt != promptOut {
					t.Errorf("Process sent prompt: %v, want: %v", x.Prompt, promptOut)
				}
				log.Printf("ReadRequest prompt: %v", x.Prompt)

				select {
				case x.Output <- in:
					log.Printf("Output sent")
				case <-time.After(5 * time.Second):
					t.Errorf("Output posting timed out")
				}
			} else {
				t.Errorf("Process returned: %v, expected output of type: exec.ReadRequest", x)
			}
		case <-time.After(5 * time.Second):
			t.Errorf("Evaluation timed out")
		}

		o, err = p.Execute(in2)
		checkErrorf(t, "Execute() error: %v", err)

		ch = o.Output()
		// checkSeedOutput(t, o)

		select {
		case x := <-ch:
			if x, ok := x.(*Line); ok {
				if x.Data != out2 {
					t.Errorf("Process sent: %v, got %v, want: %v", in2, x.Data, out2)
				}
			} else {
				t.Errorf("Process returned: %v, expected output of type: exec.Line", x)
			}
		case <-time.After(5 * time.Second):
			t.Errorf("Evaluation timed out")
		}

		testQuitAndWait(p, t)
	}
	runProcess(test, t)
}

func BenchmarkBasicThroughput(b *testing.B) {
	const in = `for i in [1..%v+1] do print "XXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX"; end for;`

	// Stop the timer whilst we start the process...
	b.StopTimer()

	// Create a new process and start it
	p := &Process{}
	st, _ := p.StatusTags()
	go emptyTaggedChToLogPrintf("Status tag received: %v", st)

	so, err := p.Start()
	checkErrorf(b, "Start() error: %v", err)
	defer func() {
		err := p.Wait()
		checkErrorf(b, "Wait() error: %v", err)
	}()
	go emptyTaggedChToLogPrintf("Startup output: %v", so.Output())

	go func() {
		b.StartTimer()

		cmd := fmt.Sprintf(in, b.N)

		log.Printf("Sending command: %v", cmd)
		o, err := p.Execute(cmd)
		checkErrorf(b, "Execute() error: %v", err)

		// checkSeedOutput(b, o)

		for x := range o.Output() {
			if _, ok := x.(*Line); ok {
				continue
			}
			b.Errorf("Process returned: %v, expected output of type: exec.Line", x)
		}

		b.StopTimer()

		testQuitAndWait(p, b)
	}()
}
