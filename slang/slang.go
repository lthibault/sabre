package slang

import (
	"fmt"
	"io"
	"reflect"
	"strings"
	"sync"

	"github.com/spy16/sabre"
)

const (
	nsSeparator = '/'
	defaultNS   = "user"
)

// New returns a new instance of Slang interpreter.
func New() *Slang {
	sl := &Slang{
		mu:       &sync.RWMutex{},
		bindings: map[nsSymbol]sabre.Value{},
	}

	_ = sl.SwitchNS(sabre.Symbol{Value: defaultNS})
	_ = sl.BindGo("ns", sl.SwitchNS)
	_ = BindAll(sl)

	return sl
}

// Slang represents an instance of slang interpreter.
type Slang struct {
	mu        *sync.RWMutex
	currentNS string
	bindings  map[nsSymbol]sabre.Value
}

// Eval evalautes the given value in Slang context.
func (slang *Slang) Eval(v sabre.Value) (sabre.Value, error) {
	return sabre.Eval(slang, v)
}

// ReadEval reads from the given reader and evaluates all the forms
// obtained in Slang context.
func (slang *Slang) ReadEval(r io.Reader) (sabre.Value, error) {
	return sabre.ReadEval(slang, r)
}

// ReadEvalStr reads the source and evalautes it in Slang context.
func (slang *Slang) ReadEvalStr(src string) (sabre.Value, error) {
	return sabre.ReadEvalStr(slang, src)
}

// Bind binds the given name to the given Value into the slang interpreter
// context.
func (slang *Slang) Bind(symbol string, v sabre.Value) error {
	slang.mu.Lock()
	defer slang.mu.Unlock()

	nsSym, err := slang.splitSymbol(symbol)
	if err != nil {
		return err
	}

	if nsSym.NS != slang.currentNS {
		return fmt.Errorf("cannot to bind outside current namespace")
	}

	slang.bindings[*nsSym] = v
	return nil
}

// Resolve finds the value bound to the given symbol and returns it if
// found in the Slang context and returns it.
func (slang *Slang) Resolve(symbol string) (sabre.Value, error) {
	slang.mu.RLock()
	defer slang.mu.RUnlock()

	if symbol == "ns" {
		symbol = "user/ns"
	}

	nsSym, err := slang.splitSymbol(symbol)
	if err != nil {
		return nil, err
	}

	v, found := slang.bindings[*nsSym]
	if !found {
		v, found = slang.bindings[*nsSym]
		if !found {
			return nil, fmt.Errorf("unable to resolve symbol: %v", symbol)
		}

		return nil, fmt.Errorf("unable to resolve symbol: %v", symbol)
	}

	return v, nil
}

// BindGo is similar to Bind but handles convertion of Go value 'v' to
// sabre Value type.
func (slang *Slang) BindGo(symbol string, v interface{}) error {
	return slang.Bind(symbol, sabre.ValueOf(v))
}

// SwitchNS changes the current namespace to the string value of given symbol.
func (slang *Slang) SwitchNS(sym sabre.Symbol) error {
	slang.mu.Lock()
	slang.currentNS = sym.String()
	slang.mu.Unlock()

	return slang.Bind("*ns*", sym)
}

// Parent always returns nil to represent this is the root scope.
func (slang *Slang) Parent() sabre.Scope {
	return nil
}

// CurrentNS returns the current active namespace.
func (slang *Slang) CurrentNS() string {
	slang.mu.RLock()
	defer slang.mu.RUnlock()

	return slang.currentNS
}

func (slang *Slang) splitSymbol(symbol string) (*nsSymbol, error) {
	parts := strings.Split(symbol, string(nsSeparator))
	if len(parts) < 2 {
		return &nsSymbol{
			NS:   slang.currentNS,
			Name: symbol,
		}, nil
	} else if len(parts) > 2 {
		return nil, fmt.Errorf("invalid qualified symbol: '%s'", symbol)
	}

	return &nsSymbol{
		NS:   parts[0],
		Name: parts[1],
	}, nil
}

type nsSymbol struct {
	NS   string
	Name string
}

// BindAll binds all core functions into the given scope.
func BindAll(scope sabre.Scope) error {
	core := map[string]sabre.Value{
		"core/eval":     sabre.GoFunc(Eval),
		"core/not":      Fn(Not),
		"core/boolean":  Fn(MakeBool),
		"core/str":      Fn(MakeString),
		"core/type":     Fn(TypeOf),
		"core/set":      makeContainer(sabre.Set{}),
		"core/list":     makeContainer(&sabre.List{}),
		"core/vector":   makeContainer(sabre.Vector{}),
		"core/nil?":     IsType(reflect.TypeOf(sabre.Nil{})),
		"core/int?":     IsType(reflect.TypeOf(sabre.Int64(0))),
		"core/set?":     IsType(reflect.TypeOf(sabre.Set{})),
		"core/boolean?": IsType(reflect.TypeOf(sabre.Bool(false))),
		"core/list?":    IsType(reflect.TypeOf(sabre.List{})),
		"core/string?":  IsType(reflect.TypeOf(sabre.String(""))),
		"core/float?":   IsType(reflect.TypeOf(sabre.Float64(0))),
		"core/vector?":  IsType(reflect.TypeOf(sabre.Vector{})),
		"core/keyword?": IsType(reflect.TypeOf(sabre.Keyword(""))),
		"core/symbol?":  IsType(reflect.TypeOf(sabre.Symbol{})),
	}

	for sym, val := range core {
		if err := scope.Bind(sym, val); err != nil {
			return err
		}
	}

	return nil
}