// Copyright (c) 2024 Fantom Foundation
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at fantom.foundation/bsl11.
//
// Change Date: 2028-4-16
//
// On the date above, in accordance with the Business Source License, use of
// this software will be governed by the GNU Lesser General Public License v3.

package lfvm

import (
	"fmt"
	"sort"
	"strings"
	"sync"
)

// statisticRunner is a runner that collects statistics about the instruction
// sequence of the executed code.
type statisticRunner struct {
	mutex sync.Mutex
	stats *statistics
}

func (s *statisticRunner) run(c *context) (status, error) {
	stats := statsCollector{stats: newStatistics()}
	status := statusRunning
	var executionError error
	for status == statusRunning {
		if c.pc < int32(len(c.code)) {
			stats.nextOp(c.code[c.pc].opcode)
		}
		status, executionError = step(c)
		if executionError != nil {
			break
		}
	}
	s.mutex.Lock()
	defer s.mutex.Unlock()
	if s.stats == nil {
		s.stats = newStatistics()
	}
	s.stats.insert(stats.stats)
	return status, executionError
}

// getSummary returns a summary of the collected statistics in a human-readable
// format.
func (s *statisticRunner) getSummary() string {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	if s.stats == nil {
		s.stats = newStatistics()
	}
	return s.stats.print()
}

// reset clears the collected statistics.
func (s *statisticRunner) reset() {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.stats = newStatistics()
}

// statistics contains the instruction sequence statistics of a code execution.
// It counts the number of times each instruction is executed, as well as the
// number of times each pair, triple, and quad of instructions are executed.
type statistics struct {
	count       uint64
	singleCount map[uint64]uint64
	pairCount   map[uint64]uint64
	tripleCount map[uint64]uint64
	quadCount   map[uint64]uint64
}

func newStatistics() *statistics {
	return &statistics{
		singleCount: map[uint64]uint64{},
		pairCount:   map[uint64]uint64{},
		tripleCount: map[uint64]uint64{},
		quadCount:   map[uint64]uint64{},
	}
}

// insert adds the instruction counts of the given statistics to this instance.
func (s *statistics) insert(src *statistics) {
	s.count += src.count
	for k, v := range src.singleCount {
		s.singleCount[k] += v
	}
	for k, v := range src.pairCount {
		s.pairCount[k] += v
	}
	for k, v := range src.tripleCount {
		s.tripleCount[k] += v
	}
	for k, v := range src.quadCount {
		s.quadCount[k] += v
	}
}

// print returns a human-readable summary of the collected statistics.
func (s *statistics) print() string {

	type entry struct {
		value uint64
		count uint64
	}

	getTopN := func(data map[uint64]uint64, n int) []entry {
		list := make([]entry, 0, len(data))
		for k, c := range data {
			list = append(list, entry{k, c})
		}
		sort.Slice(list, func(i, j int) bool {
			return list[i].count > list[j].count
		})
		if len(list) < n {
			return list
		}
		return list[0:n]
	}

	builder := strings.Builder{}
	write := func(format string, args ...interface{}) {
		builder.WriteString(fmt.Sprintf(format, args...))
	}

	write("\n----- Statistics ------\n")
	write("\nSteps: %d\n", s.count)
	write("\nSingles:\n")
	for _, e := range getTopN(s.singleCount, 5) {
		write("\t%-30v: %d (%.2f%%)\n", OpCode(e.value), e.count, float32(e.count*100)/float32(s.count))
	}
	write("\nPairs:\n")
	for _, e := range getTopN(s.pairCount, 5) {
		write("\t%-30v%-30v: %d (%.2f%%)\n", OpCode(e.value>>16), OpCode(e.value), e.count, float32(e.count*100)/float32(s.count))
	}
	write("\nTriples:\n")
	for _, e := range getTopN(s.tripleCount, 5) {
		write("\t%-30v%-30v%-30v: %d (%.2f%%)\n", OpCode(e.value>>32), OpCode(e.value>>16), OpCode(e.value), e.count, float32(e.count*100)/float32(s.count))
	}

	write("\nQuads:\n")
	for _, e := range getTopN(s.quadCount, 5) {
		write("\t%-30v%-30v%-30v%-30v: %d (%.2f%%)\n", OpCode(e.value>>48), OpCode(e.value>>32), OpCode(e.value>>16), OpCode(e.value), e.count, float32(e.count*100)/float32(s.count))
	}
	write("\n")

	return builder.String()
}

// statsCollector is a helper struct that keeps track of the resent history of
// instructions executed by the VM to collect instruction sequence statistics.
type statsCollector struct {
	stats *statistics

	last       uint64
	secondLast uint64
	thirdLast  uint64
}

func (s *statsCollector) nextOp(op OpCode) {
	cur := uint64(op)
	s.stats.count++
	s.stats.singleCount[cur]++
	if s.stats.count == 1 {
		s.last, s.secondLast, s.thirdLast = cur, s.last, s.secondLast
		return
	}
	s.stats.pairCount[s.last<<16|cur]++
	if s.stats.count == 2 {
		s.last, s.secondLast, s.thirdLast = cur, s.last, s.secondLast
		return
	}
	s.stats.tripleCount[s.secondLast<<32|s.last<<16|cur]++
	if s.stats.count == 3 {
		s.last, s.secondLast, s.thirdLast = cur, s.last, s.secondLast
		return
	}
	s.stats.quadCount[s.thirdLast<<48|s.secondLast<<32|s.last<<16|cur]++
	s.last, s.secondLast, s.thirdLast = cur, s.last, s.secondLast
}
