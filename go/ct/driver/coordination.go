package main

import (
	"regexp"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Fantom-foundation/Tosca/go/ct/rlz"
	"github.com/Fantom-foundation/Tosca/go/ct/spc"
	"github.com/Fantom-foundation/Tosca/go/ct/st"
	"pgregory.net/rand"
)

func ForEachState(
	opFunction func(state *st.State) rlz.ConsumerResult,
	printIssueCounts func(relativeTime time.Duration, rate float64, current int64),
	numJobs int,
	seed uint64,
	fullMode bool,
	filter *regexp.Regexp,
) error {
	// The execution of test cases is distributed to parallel goroutines in a three-step
	// process:
	//   - this goroutine writes the list of rules to be tested into a channel
	//   - a team of goroutines fetches rules from the first channel, runs the
	//     test state enumeration for the retrieved rule, and forward those states
	//     into a second channel
	//   - another team of goroutines fetches test-input states from the second
	//     channel and processes the actual tests.
	// Additionally, a goroutine periodically reporting progress information to the
	// console is started.
	// To avoid dead-locks in this goroutine, consuming goroutines are started before
	// producing routines. Thus, the order in which goroutines and teams of goroutines
	// are started below is in the reverse order as listed above.

	var stateWaitGroup sync.WaitGroup
	var testCounter atomic.Int64
	var abortTests atomic.Bool
	abortTests.Store(false)

	done := make(chan bool)
	printerDone := make(chan bool)
	go func() {
		defer close(printerDone)
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()
		startTime := time.Now()
		lastTime := startTime
		lastTestCounter := int64(0)
		for {
			select {
			case <-done:
				return
			case curTime := <-ticker.C:
				cur := testCounter.Load()

				diffCounter := cur - lastTestCounter
				diffTime := curTime.Sub(lastTime)

				lastTime = curTime
				lastTestCounter = cur

				relativeTime := curTime.Sub(startTime)
				rate := float64(diffCounter) / diffTime.Seconds()
				printIssueCounts(relativeTime, rate, cur)
			}
		}
	}()

	// Run goroutines processing the actual tests.
	stateWaitGroup.Add(numJobs)
	stateChannel := make(chan *st.State, 10*numJobs)
	for i := 0; i < numJobs; i++ {
		go func() {
			defer stateWaitGroup.Done()
			for state := range stateChannel {
				testCounter.Add(1)
				consumeStatus := opFunction(state)
				if consumeStatus == rlz.ConsumeAbort {
					abortTests.Store(true)
				}
				state.Release()
			}
		}()
	}

	// Generate test states in parallel (generation can be the bottleneck if there
	// are many workers processing test cases in parallel).
	ruleChannel := make(chan rlz.Rule, 10*numJobs)
	var rulesWaitGroup sync.WaitGroup
	rulesWaitGroup.Add(numJobs)

	var errorMutex sync.Mutex
	var returnError error

	for i := 0; i < numJobs; i++ {
		go func() {
			defer rulesWaitGroup.Done()
			for rule := range ruleChannel {
				if abortTests.Load() {
					continue // keep consume rules in the ruleChannel
				}
				// random is re-seeded for each rule to be reproducible.
				rand := rand.New(seed)
				err := rule.EnumerateTestCases(rand, func(state *st.State) rlz.ConsumerResult {
					if abortTests.Load() {
						return rlz.ConsumeAbort
					}
					if !fullMode {
						if applies, err := rule.Condition.Check(state); !applies || err != nil {
							return rlz.ConsumeContinue
						}
					}

					stateChannel <- state.Clone()
					return rlz.ConsumeContinue
				})
				if err != nil {
					abortTests.Store(true)
					errorMutex.Lock()
					returnError = err
					errorMutex.Unlock()
					continue
				}
			}
		}()
	}

	// Feed the rule generator workers with rules.
	rules := spc.Spec.GetRules()
	rules = FilterRules(rules, filter)
	for _, rule := range rules {
		ruleChannel <- rule
	}

	close(ruleChannel)
	rulesWaitGroup.Wait()

	close(stateChannel)
	stateWaitGroup.Wait() // < releases when all test cases are processed

	// Wait for the printer to be finished.
	close(done)   // < signals progress printer to stop
	<-printerDone // < blocks until channel is closed by progress printer

	return returnError
}

func FilterRules(rules []rlz.Rule, filter *regexp.Regexp) []rlz.Rule {
	if filter == nil {
		return rules
	}
	res := make([]rlz.Rule, 0, len(rules))
	for _, rule := range rules {
		if filter.MatchString(rule.Name) {
			res = append(res, rule)
		}
	}
	return res
}
