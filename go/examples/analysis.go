package examples

import (
	"github.com/ethereum/go-ethereum/core/vm"
)

func GenerateAnalysisCode(filler []byte) []byte {
	const MaxCodeLength = 0x6000

	initCode := []byte{
		// Parse the input parameter.
		byte(vm.PUSH1), 4,
		byte(vm.CALLDATALOAD),

		// Store result (input) in memory[0].
		byte(vm.PUSH1), 0,
		byte(vm.MSTORE),

		// Jump over filler code (destination is a placeholder).
		byte(vm.PUSH2), 0xFF, 0xFF,
		byte(vm.JUMP),
	}

	endingCode := []byte{
		// Jumpdest for jumping over filler code.
		byte(vm.JUMPDEST),

		// Return the result from memory[0].
		byte(vm.PUSH1), 32,
		byte(vm.PUSH1), 0,
		byte(vm.RETURN),
	}

	maxFillerCodeLength := MaxCodeLength - len(initCode) - len(endingCode)
	fillerCode := []byte{}

	for i := 0; i < maxFillerCodeLength/len(filler); i++ {
		fillerCode = append(fillerCode, filler...)
	}

	// Fill placeholder destination for jumping over filler code.
	jmpdestPos := len(initCode) + len(fillerCode)
	initCode[7] = byte(jmpdestPos >> 8)
	initCode[8] = byte(jmpdestPos)

	code := append(initCode, fillerCode...)
	code = append(code, endingCode...)

	return code
}

func GetJumpdestAnalysisExample() Example {
	filler := []byte{byte(vm.JUMPDEST)}
	code := GenerateAnalysisCode(filler)

	return exampleSpec{
		Name:      "jumpdest",
		code:      code,
		reference: analysis,
	}.build()
}

func GetStopAnalysisExample() Example {
	filler := []byte{byte(vm.STOP)}
	code := GenerateAnalysisCode(filler)

	return exampleSpec{
		Name:      "stop",
		code:      code,
		reference: analysis,
	}.build()
}

func GetPush1AnalysisExample() Example {
	filler := []byte{byte(vm.PUSH1), 0}
	code := GenerateAnalysisCode(filler)

	return exampleSpec{
		Name:      "push1",
		code:      code,
		reference: analysis,
	}.build()
}

func GetPush32AnalysisExample() Example {
	filler := []byte{byte(vm.PUSH32)}
	filler = append(filler, make([]byte, 32)...)
	code := GenerateAnalysisCode(filler)

	return exampleSpec{
		Name:      "push32",
		code:      code,
		reference: analysis,
	}.build()
}

func analysis(x int) int {
	return x
}
