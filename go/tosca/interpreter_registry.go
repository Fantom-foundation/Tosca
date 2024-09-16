// Copyright (c) 2024 Fantom Foundation
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at fantom.foundation/bsl11.
//
// Change Date: 2028-4-16
//
// On the date above, in accordance with the Business Source License, use of
// this software will be governed by the GNU Lesser General Public License v3.

package tosca

import (
	"fmt"
	"strings"
	"sync"

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
//
// Deprecated: Use NewInterpreter instead.
func GetInterpreter(name string) Interpreter {
	res, err := NewInterpreter(name, nil)
	if err != nil {
		return nil
	}
	return res
}

// NewInterpreter performs a lookup for the given name (case-insensitive) in
// the registry and creates a new Interpreter using the given optional
// configuration. If no configuration is provided, the implementation uses
// its default configuration. An error is returned if no factory was
// registered under the given name.
func NewInterpreter(name string, config ...any) (Interpreter, error) {
	if len(config) > 1 {
		return nil, fmt.Errorf("invalid configuration: too many arguments")
	}
	factory := GetInterpreterFactory(name)
	if factory == nil {
		return nil, fmt.Errorf("interpreter not found: %s", name)
	}
	c := any(nil)
	if len(config) > 0 {
		c = config[0]
	}
	return factory(c)
}

// GetInterpreterFactory performs a lookup for the given name (case-insensitive)
// in the registry. The result is nil if no factory was registered under the
// given name.
func GetInterpreterFactory(name string) InterpreterFactory {
	interpreterRegistryLock.Lock()
	defer interpreterRegistryLock.Unlock()
	return interpreterRegistry[strings.ToLower(name)]
}

// GetAllRegisteredInterpreters obtains all registered implementations.
func GetAllRegisteredInterpreters() map[string]InterpreterFactory {
	interpreterRegistryLock.Lock()
	defer interpreterRegistryLock.Unlock()
	return maps.Clone(interpreterRegistry)
}

// RegisterInterpreter can be used to register a new Interpreter implementation
// to be exported for general use in the binary. The name is not case-sensitive,
// and a panic is triggered if an implementation was bound to the same name
// before, or the implementation is nil. This function is mainly intended to be
// used by package initialization code.
//
// Deprecated: Use RegisterInterpreterFactory instead.
func RegisterInterpreter(name string, impl Interpreter) {
	err := RegisterInterpreterFactory(name, func(any) (Interpreter, error) {
		return impl, nil
	})
	if err != nil {
		panic(err)
	}
}

// RegisterInterpreterFactory registers a new Interpreter implementation
// to be exported for general use in the binary. The name is not case-sensitive,
// and a panic is triggered if a factory was bound to the same name before, or
// the factory is nil. This function is mainly intended to be used by package
// initialization code.
func RegisterInterpreterFactory(name string, factory InterpreterFactory) error {
	key := strings.ToLower(name)
	if factory == nil {
		return fmt.Errorf("invalid initialization: cannot register nil-factory using `%s`", key)
	}
	interpreterRegistryLock.Lock()
	defer interpreterRegistryLock.Unlock()
	if _, found := interpreterRegistry[key]; found {
		return fmt.Errorf("invalid initialization: multiple factories registered for `%s`", key)
	}
	interpreterRegistry[key] = factory
	return nil
}

// InterpreterFactory is the type of a function that creates a new Interpreter
// using a interpreter specific configuration.
type InterpreterFactory func(config any) (Interpreter, error)

// interpreterRegistry is a global registry for Interpreter factories of
// different implementations and configurations.
var interpreterRegistry = map[string]InterpreterFactory{}

// interpreterRegistryLock to protect access to the registry.
var interpreterRegistryLock sync.Mutex
