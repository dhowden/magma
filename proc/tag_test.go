// Copyright 2014, David Howden
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package proc

import "testing"

func pos(t tag, s []tag) int {
	for p, v := range s {
		if v == t {
			return p
		}
	}
	return -1
}

func TestIsError(t *testing.T) {
	var allOutputTags = [...]tag{
		TagOutput,
		TagList,
		TagSignature,
		TagErrorSyntax,
		TagErrorInternal,
		TagErrorUser,
		TagErrorRuntime,
		TagTraceback,
		TagErrorHistoryPosition,
		TagErrorPosition,
		TagReadPrompt,
		TagReadInput,
		TagReadIntPrompt,
		TagReadIntInput,
		TagReadIntError,
	}

	var errorTags = [...]tag{
		TagErrorSyntax,
		TagErrorInternal,
		TagErrorUser,
		TagErrorRuntime,
		TagTraceback,
		TagErrorHistoryPosition,
		TagErrorPosition,
		TagReadIntError,
	}

	for _, tag := range allOutputTags {
		if !IsError(tag) && pos(tag, errorTags[:]) != -1 {
			t.Errorf("IsError fails on %v", tag)
		}
	}
}
