package sabre

import (
	"fmt"
	"io"
	"reflect"
	"strings"
)

// Eval consumes data from reader 'r' till EOF, parses into forms and
// evaluates all the forms obtained and returns the result.
func Eval(scope Scope, r io.Reader) (Value, error) {
	mod, err := NewReader(r).All()
	if err != nil {
		return nil, err
	}

	return mod.Eval(scope)
}

// EvalStr is a convinience wrapper for Eval that reads forms from string
// and evaluates for result.
func EvalStr(scope Scope, src string) (Value, error) {
	return Eval(scope, strings.NewReader(src))
}

// Scope implementation is responsible for managing value bindings.
type Scope interface {
	Bind(symbol string, v Value) error
	Resolve(symbol string) (Value, error)
}

// Value represents data/forms in sabre. This includes those emitted by
// Reader, values obtained as result of an evaluation etc.
type Value interface {
	// Eval should evaluate this value against the scope and return
	// the resultant value or an evaluation error.
	Eval(scope Scope) (Value, error)

	// String should return the LISP representation of the value.
	String() string
}

// Invokable represents any value that supports invocation. Vector, Fn
// etc support invocation.
type Invokable interface {
	Invoke(argVals ...Value) (Value, error)
}

// SpecialFn implementations receive unevaluated list of forms during invoke.
// This allows such values to define their own evaluation rules.
type SpecialFn func(scope Scope, args []Value) (Value, error)

// Eval simply returns the special fn.
func (fn SpecialFn) Eval(_ Scope) (Value, error) {
	return fn, nil
}

func (fn SpecialFn) String() string {
	return fmt.Sprintf("SpecialFn{%v}", reflect.ValueOf(fn))
}

// Invoke simply dispaatches the call to the wrapped function.
func (fn SpecialFn) Invoke(scope Scope, args []Value) (Value, error) {
	return fn(scope, args)
}
