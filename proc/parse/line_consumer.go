// Copyright 2014, David Howden
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package parse

import "strings"

// lineConsumer allows for easy handling of lines of output
// so that lines can be fetched and discarded when properly
// processed.
type lineConsumer struct {
	source    <-chan string // Source
	line      string        // Current line
	processed bool          // Has the current line been processed?
}

// Construct a new lineConsumer and return with given channel as the
// source of lines to consume.
func newLineConsumer(source <-chan string) *lineConsumer {
	return &lineConsumer{source: source, processed: true}
}

// Fetch the next output line (via source <-chan interface{}).
func (p *lineConsumer) fetchNextLine() bool {
	if p.processed {
		x, ok := <-p.source
		if !ok {
			return false
		}
		p.line = strings.TrimSpace(x)
		p.processed = false
	}
	return true
}

// Mark the current line as processed.
func (p *lineConsumer) consumeLine() {
	p.processed = true
}
