package lfvm


const (
	ErrOutOfGas = ConstError("out of gas")
	ErrInvalidJump = ConstError("invalid jump destination")
	ErrGasUintOverflow = ConstError("gas uint64 overflow")
)


// ConstError is a error type that can be used to define immutable
// error constants.
type ConstError string

func (e ConstError) Error() string {
	return string(e)
}