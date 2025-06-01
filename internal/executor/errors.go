// Copyright 2025 Alan Matykiewicz
//
// Permission is hereby granted, free of charge, to any person obtaining a copy of
// this software and associated documentation files (the "Software"), to deal in
// the Software without restriction, including without limitation the rights to use,
// copy, modify, merge, publish, distribute, sublicense, and/or sell copies of the
// Software, and to permit persons to whom the Software is furnished to do so,
// subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND,
// EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES
// OF MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND
// NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT
// HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY,
// WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING
// FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR
// OTHER DEALINGS IN THE SOFTWARE.

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
