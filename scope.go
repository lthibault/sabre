package sabre

import (
	"fmt"
	"strings"
	"sync"
)

const nsSeparator = '/'

// NewScope returns an instance of MapScope with no bindings.
func NewScope(parent Scope) *MapScope {
	scope := &MapScope{
		parent:   parent,
		mu:       new(sync.RWMutex),
		bindings: map[nsSymbol]Value{},
	}

	_ = scope.SwitchNS(Symbol{Value: "user"})
	_ = scope.BindGo("ns", scope.SwitchNS)

	return scope
}

// MapScope implements Scope using a Go native hash-map.
type MapScope struct {
	parent   Scope
	mu       *sync.RWMutex
	bindings map[nsSymbol]Value
	curNS    string
}

// Parent returns the parent scope of this scope.
func (scope *MapScope) Parent() Scope {
	return scope.parent
}

// Bind adds the given value to the scope and binds the symbol to it.
func (scope *MapScope) Bind(symbol string, v Value) error {
	scope.mu.Lock()
	defer scope.mu.Unlock()

	nsSym, err := scope.splitSymbol(symbol)
	if err != nil {
		return err
	}

	if nsSym.NS != scope.CurrentNS() {
		return fmt.Errorf("cannot to bind outside current namespace")
	}

	scope.bindings[*nsSym] = v
	return nil
}

// Resolve finds the value bound to the given symbol and returns it if
// found in this scope or parent scope if any.
func (scope *MapScope) Resolve(symbol string) (Value, error) {
	scope.mu.RLock()
	defer scope.mu.RUnlock()

	if symbol == "ns" {
		symbol = "user/ns"
	}

	nsSym, err := scope.splitSymbol(symbol)
	if err != nil {
		return nil, err
	}

	v, found := scope.bindings[*nsSym]
	if !found {
		if scope.parent != nil {
			return scope.parent.Resolve(symbol)
		}

		return nil, fmt.Errorf("unable to resolve symbol: %v", symbol)
	}

	return v, nil
}

// BindGo is similar to Bind but handles convertion of Go value 'v' to
// sabre Val type.
func (scope *MapScope) BindGo(symbol string, v interface{}) error {
	return scope.Bind(symbol, ValueOf(v))
}

// SwitchNS changes the current namespace to the string value of given symbol.
func (scope *MapScope) SwitchNS(sym Symbol) error {
	scope.curNS = sym.String()
	return scope.Bind("*ns*", sym)
}

// CurrentNS returns the current active namespace.
func (scope *MapScope) CurrentNS() string {
	return scope.curNS
}

func (scope *MapScope) splitSymbol(symbol string) (*nsSymbol, error) {
	parts := strings.Split(symbol, string(nsSeparator))
	if len(parts) < 2 {
		return &nsSymbol{
			NS:   scope.curNS,
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
