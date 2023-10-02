package ct

type Evm interface {
	StepN(State, numSteps int) (State, error)
}
