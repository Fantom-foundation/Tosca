package lfvm

const (
	ErrGasUintOverflow       = ConstError("gas uint64 overflow")
	ErrInvalidCode           = ConstError("invalid code")
	ErrInvalidJump           = ConstError("invalid jump destination")
	ErrOutOfGas              = ConstError("out of gas")
	ErrReturnDataOutOfBounds = ConstError("return data out of bounds")
	ErrStackOverflow         = ConstError("stack overflow")
	ErrStackUnderflow        = ConstError("stack underflow")
	ErrWriteProtection       = ConstError("write protection")
)

// ConstError is an error type that can be used to define immutable
// error constants.
type ConstError string

func (e ConstError) Error() string {
	return string(e)
}
