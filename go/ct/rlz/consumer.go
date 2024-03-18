package rlz

// ConsumerResult is the return type of callback functions consuming
// partial or fully generated test case inputs.
type ConsumerResult bool

const (
	ConsumeContinue ConsumerResult = true
	ConsumeAbort    ConsumerResult = false
)
