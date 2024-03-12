package lfvm

const (
	errGasUintOverflow       = ConstError("gas uint64 overflow")
	errInvalidCode           = ConstError("invalid code")
	errInvalidJump           = ConstError("invalid jump destination")
	errOutOfGas              = ConstError("out of gas")
	errReturnDataOutOfBounds = ConstError("return data out of bounds")
	errStackOverflow         = ConstError("stack overflow")
	errStackUnderflow        = ConstError("stack underflow")
	errWriteProtection       = ConstError("write protection")
)

// ConstError is an error type that can be used to define immutable
// error constants.
type ConstError string

func (e ConstError) Error() string {
	return string(e)
}
