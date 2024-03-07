package vm

import (
	"fmt"
	"strings"

	"golang.org/x/exp/maps"
)

// This file provides a registry for Interpreter instances in Tosca.
//
// The registry is intended to be used by all client applications that would
// like to use interpreter services. For an implementation to be available
// it needs to be registered. Typically, this registration is part of the
// init code of the package providing an implementation. Thus, by including
// the implementation package, interpreter implementations become available
// in this central registry.

// GetInterpreter performs a lookup for the given name (case-insensitive) in
// the registry. The result is nil if no interpreter was registered under the
// given name.
func GetInterpreter(name string) Interpreter {
	return interpreter_registry[strings.ToLower(name)]
}

// GetAllRegisteredInterpreters obtains all registered implementations.
func GetAllRegisteredInterpreters() map[string]Interpreter {
	return maps.Clone(interpreter_registry)
}

// RegisterInterpreter can be used to register a new Interpreter implementation
// to be exported for general use in the binary. The name is not case-sensitive,
// and a panic is triggered if an implementation was bound to the same name
// before, or the implementation is nil. This function is mainly intended to be
// used by package initialization code.
func RegisterInterpreter(name string, impl Interpreter) {
	key := strings.ToLower(name)
	if impl == nil {
		panic(fmt.Sprintf("invalid initialization: cannot register nil-interpreter using `%s`", key))
	}
	if _, found := interpreter_registry[key]; found {
		panic(fmt.Sprintf("invalid initialization: multiple Interpreters registered for `%s`", key))
	}
	interpreter_registry[key] = impl
}

// interpreter_registry is a global registry for Interpreter instances of
// different implementations and configurations.
var interpreter_registry = map[string]Interpreter{}
