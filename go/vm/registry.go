//
// Copyright (c) 2024 Fantom Foundation
//
// Use of this software is governed by the Business Source License included
// in the LICENSE.TXT file and at fantom.foundation/bsl11.
//
// Change Date: 2028-4-16
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by the GNU Lesser General Public Licence v3
//

package vm

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
func GetInterpreter(name string) Interpreter {
	interpreterRegistryLock.Lock()
	defer interpreterRegistryLock.Unlock()
	return interpreterRegistry[strings.ToLower(name)]
}

// GetAllRegisteredInterpreters obtains all registered implementations.
func GetAllRegisteredInterpreters() map[string]Interpreter {
	interpreterRegistryLock.Lock()
	defer interpreterRegistryLock.Unlock()
	return maps.Clone(interpreterRegistry)
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
	interpreterRegistryLock.Lock()
	defer interpreterRegistryLock.Unlock()
	if _, found := interpreterRegistry[key]; found {
		panic(fmt.Sprintf("invalid initialization: multiple Interpreters registered for `%s`", key))
	}
	interpreterRegistry[key] = impl
}

// interpreterRegistry is a global registry for Interpreter instances of
// different implementations and configurations.
var interpreterRegistry = map[string]Interpreter{}

// interpreterRegistryLock to protect access to the registry.
var interpreterRegistryLock sync.Mutex
