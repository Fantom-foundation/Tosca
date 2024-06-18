// Copyright (c) 2024 Fantom Foundation
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at fantom.foundation/bsl11.
//
// Change Date: 2028-4-16
//
// On the date above, in accordance with the Business Source License, use of
// this software will be governed by the GNU Lesser General Public License v3.

package cliUtils

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/Fantom-foundation/Tosca/go/ct/st"
)

type issue struct {
	input *st.State
	err   error
}

func (i *issue) Error() error {
	return i.err
}

func (i *issue) Input() *st.State {
	return i.input
}

type IssuesCollector struct {
	issues []issue
	mu     sync.Mutex
}

func (c *IssuesCollector) AddIssue(state *st.State, err error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	var clone *st.State
	if state != nil {
		clone = state.Clone()
	}
	c.issues = append(c.issues, issue{clone, err})
}

func (c *IssuesCollector) NumIssues() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.issues)
}

func (c *IssuesCollector) GetIssues() []issue {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.issues
}

func (c *IssuesCollector) ExportIssues() error {
	if len(c.issues) == 0 {
		return nil
	}
	jsonDir, err := os.MkdirTemp("", "ct_issues_*")
	if err != nil {
		return fmt.Errorf("failed to create output directory for %d issues", len(c.issues))
	}
	for i, issue := range c.issues {
		fmt.Printf("----------------------------\n")
		fmt.Printf("%s\n", issue.err)

		// If there is an input state for this issue, it is exported into a file
		// to aid its debugging using the regression test infrastructure.
		if issue.input != nil {
			path := filepath.Join(jsonDir, fmt.Sprintf("issue_%06d.json", i))
			if err := st.ExportStateJSON(issue.input, path); err == nil {
				fmt.Printf("Input state dumped to %s\n", path)
			} else {
				fmt.Printf("failed to dump state: %v\n", err)
			}
		}
	}
	return nil
}
