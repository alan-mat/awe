package executor

import "fmt"

type ErrOperatorNotFound struct {
	ExecutorName string
	OperatorName string
}

func (e ErrOperatorNotFound) Error() string {
	return fmt.Sprintf("invalid operator '%s' for executor '%s'", e.ExecutorName, e.OperatorName)
}

type ErrArgMissing struct {
	ArgName string
}

func (e ErrArgMissing) Error() string {
	return fmt.Sprintf("requested argument '%s' does not exist", e.ArgName)
}

type ErrInvalidArgumentType struct {
	Name     string
	Expected string
	Received string
}

func (e ErrInvalidArgumentType) Error() string {
	return fmt.Sprintf("argument '%s' must be of type '%s', but received '%s'",
		e.Name, e.Expected, e.Received)
}

type ErrInvalidParams struct {
	ExecutorName  string
	MissingParams []string
}

func (e ErrInvalidParams) Error() string {
	return fmt.Sprintf("executor '%s' is missing the following params: %v",
		e.ExecutorName, e.MissingParams)
}
