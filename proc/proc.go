// Copyright 2014, David Howden
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package proc provides a Go API for starting and handling Magma processes.
package proc

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
)

// Default command and arguments for Magma processes.
const (
	DefaultCommand string = "magma"
	DefaultArgs           = "-x -n -b"
)

// Process represents a Magma process being prepared or run.
// Command, Env and Args values are exported to allow for some
// pre-start configuration.
type Process struct {
	// Comand (optional) specifies the Magma command to be run to
	// start a new process.  Can be any command which is on the
	// current PATH.
	//
	// If left blank, Start defaults to DefaultCommand.
	Command string

	// Env specifies the environment of the process.
	//
	// If Env is nil Start defaults to the go exec.Cmd behaviour which
	// uses the environment of the current process.
	Env []string

	// Args gives extra arguments for command. These are appended
	// to the default set of arguments given in DefaultArgs.
	Args []string

	startUp  chan struct{}     // Closed if there is a problem with startup
	ready    chan chan *Output // Notify when process is ready for input
	response chan *Output
	writer   chan io.Writer // Channel for passing around the io.Writer for Magma stdin
	status   chan Tagged    // Channel providing status messages
	errch    chan error     // Channel passing errors back from goroutines
	cmd      *exec.Cmd      // Input used to start process

	interrupt chan chan struct{} // Pass interrupt channel to parser to acknowledge INT tag
	quit      chan chan struct{} // Pass quit channel to parser to acknowledge QUIT tag
}

// StatusTags returns the status channel for this process. Should
// be called before Start(). If not called, then status output is
// discarded.
func (p *Process) StatusTags() (<-chan Tagged, error) {
	if p.status != nil {
		return nil, errors.New("cannot call StatusTags() after Start()")
	}
	p.status = make(chan Tagged)
	return p.status, nil
}

func (p *Process) setupDefaultStatusHandler() {
	p.status = make(chan Tagged)
	go func() {
		for _ = range p.status {
		}
	}()
}

func (p *Process) setupStdoutHandler(stdout io.Reader) {
	ch := make(chan []byte)
	stop := make(chan struct{})

	go func() {
		var err error
		if _, ok := <-p.startUp; ok {
			err = p.parseStdoutLines(ch)
		}
		p.errch <- err
		if err != nil {
			p.Kill()
		}
		close(stop)
	}()

	go func() {
		s := bufio.NewScanner(stdout)
		for s.Scan() {
			b := s.Bytes()
			l := make([]byte, len(b))
			copy(l, b)
			select {
			case ch <- b:
			case <-stop:
				break
			}
		}
		close(ch)
		p.errch <- s.Err()
	}()
}

// Start launches a Magma process using p.Command (or DefaultCommand by default)
// and COMMAND_ARDS + p.Args.  Any enviroment variables set in p.Env are
// set for the process.
// Returns an output channel which passes back any startup output.
func (p *Process) Start() (*Output, error) {
	exe := DefaultCommand
	if p.Command != "" {
		exe = p.Command
	}
	args := strings.Fields(DefaultArgs)
	if p.Args != nil {
		args = append(args, p.Args...)
	}

	p.cmd = exec.Command(exe, args...)
	p.cmd.Env = p.Env

	stdout, err := p.cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("stdoutpipe setup: %v", err)
	}

	stdin, err := p.cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("stdinpipe setup: %v", err)
	}

	p.startUp = make(chan struct{})

	p.ready = make(chan chan *Output, 1)
	p.response = make(chan *Output, 1)

	p.interrupt = make(chan chan struct{}, 1)
	p.quit = make(chan chan struct{}, 1)

	p.errch = make(chan error, 2)
	p.setupStdoutHandler(stdout)

	p.writer = make(chan io.Writer, 1)
	p.writer <- stdin

	if p.status == nil {
		p.setupDefaultStatusHandler()
	}

	if err := p.cmd.Start(); err != nil {
		err := fmt.Errorf("starting command: %v", err)

		<-p.writer
		close(p.writer)

		close(p.startUp)

		close(p.ready)
		close(p.response)
		close(p.status)

		close(p.interrupt)
		close(p.quit)

		p.cmd = nil
		return nil, err
	}

	p.startUp <- struct{}{}
	close(p.startUp)

	return <-p.response, nil
}

// Wait blocks until the the underlying Magma process has ended,
// and returns any resulting errors.
func (p *Process) Wait() error {
	if p.cmd == nil {
		return errors.New("magma/proc: not started")
	}

	copyError := <-p.errch
	if err := <-p.errch; err != nil && copyError == nil {
		copyError = err
	}

	select {
	case <-p.writer:
	default:
	}
	close(p.writer)

	close(p.ready)
	close(p.response)
	close(p.status)

	close(p.interrupt)
	close(p.quit)

	if err := p.cmd.Wait(); err != nil && copyError == nil {
		return err
	}
	return copyError
}

// Getpid returns the process id of the underlying Magma process.  Returns
// an error if the process isn't running.
func (p *Process) Getpid() (int, error) {
	if p.cmd.Process == nil {
		return 0, errors.New("magma/proc: process not started, cannot get pid")
	}
	return p.cmd.Process.Pid, nil
}

// Checks that the process is running, returns error if not
func (p *Process) checkRunning() error {
	if p.cmd == nil {
		return errors.New("magma/proc: process not started")
	}
	if p.cmd.Process == nil {
		return errors.New("magma/proc: process is not running")
	}
	return nil
}

// Execute passes the given command to the Magma process.
// Subsequent output is given via returned (unbuffered) channel.  The
// channel is closed when the command output is complete (i.e. when
// a RDY tag is received).
func (p *Process) Execute(s string) (*Output, error) {
	err := p.checkRunning()
	if err != nil {
		return nil, err
	}

	// Wait until the process is ready for input
	rch, ok := <-p.ready
	if !ok {
		return nil, errors.New("magma/proc: Execute() called after process has completed")
	}

	// Send the Output struct to the parser
	rch <- newOutput(s)

	// Write the command to the underlying process
	w := <-p.writer
	defer func() {
		p.writer <- w
	}()
	_, err = w.Write([]byte(s))
	if err != nil {
		return nil, err
	}

	// Write the 'execute command' char (^D)
	_, err = w.Write([]byte(string(runCommandChar)))
	if err != nil {
		return nil, err
	}

	// Wait for confirmation of running, and return output channel
	r, ok := <-p.response
	if !ok {
		return nil, errors.New("magma/proc: Execute() response not returned before process completed")
	}
	return r, nil
}

// Quit attempts to gracefully end the current process by sending the
// 'quit;' command to the underlying Magma process.
// Returns a channel which is subsequently closed when a QUIT tag is received.
// NB: Discards any output following the quit; command (should be none).
func (p *Process) Quit() (<-chan struct{}, error) {
	err := p.checkRunning()
	if err != nil {
		return nil, err
	}

	q := make(chan struct{})
	select {
	case p.quit <- q:
	default:
		return nil, errors.New("magma/proc: Quit() has already been called")
	}

	o, err := p.Execute("quit;")
	if err != nil {
		return nil, err
	}

	Discard(o.Output())
	return q, nil
}

// InterruptExecution sends the OS interrupt signal to the underlying Magma process.
// Returns a channel which is subsequently closed when an INT tag is received (i.e.
// the process acknowledges the interrupt).
// Returns an error if the process isn't running, or InterruptExecution has already
// been called and is waiting for an INT tag.
func (p *Process) InterruptExecution() (<-chan struct{}, error) {
	err := p.checkRunning()
	if err != nil {
		return nil, err
	}

	c := make(chan struct{})
	select {
	case p.interrupt <- c:
	default:
		return nil, errors.New("magma/proc: InterruptExecution() has already been called")
	}

	err = p.cmd.Process.Signal(os.Interrupt)
	if err != nil {
		return nil, err
	}
	return c, nil
}

// Kill sends the OS kill signal to the underlying Magma process.  Returns an error
// if the process isn't running, or the attempt fails.
func (p *Process) Kill() error {
	err := p.checkRunning()
	if err != nil {
		return err
	}
	return p.cmd.Process.Signal(os.Kill)
}
