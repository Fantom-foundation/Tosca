package common

// ConstErr is an error type that can be used to define error constants.
type ConstErr string

func (e ConstErr) Error() string {
	return string(e)
}
