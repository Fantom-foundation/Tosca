// Copyright (c) 2024 Fantom Foundation
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at fantom.foundation/bsl11.
//
// Change Date: 2028-4-16
//
// On the date above, in accordance with the Business Source License, use of
// this software will be governed by the GNU Lesser General Public License v3.

package vm

import (
	"fmt"
	"strings"
	"sync"

	"golang.org/x/exp/maps"
)

// This file provides a registry for Processor factories in Tosca.
//
// The registry is intended to be used by all client applications that would
// like to use processor services. For an implementation to be available
// it needs to be registered. Typically, this registration is part of the
// init code of the package providing an implementation. Thus, by including
// the implementation package, processor implementations become available
// in this central registry.

// GetProcessor performs a lookup for the given name (case-insensitive) and
// creates a processor instance using the given interpreter. The result is
// nil if no factory was registered under the given name.
func GetProcessor(name string, interpreter Interpreter) Processor {
	factory := GetProcessorFactory(name)
	if factory == nil {
		return nil
	}
	return factory(interpreter)
}

// GetProcessorFactory performs a lookup for the given name (case-insensitive)
// in the registry. The result is nil if no factory was registered under the
// given name.
func GetProcessorFactory(name string) ProcessorFactory {
	processorRegistryLock.Lock()
	defer processorRegistryLock.Unlock()
	return processorRegistry[strings.ToLower(name)]
}

// GetAllRegisteredProcessorFactories obtains all registered implementations.
func GetAllRegisteredProcessorFactories() map[string]ProcessorFactory {
	processorRegistryLock.Lock()
	defer processorRegistryLock.Unlock()
	return maps.Clone(processorRegistry)
}

// RegisterProcessorFactory can be used to register a new Processor implementation
// to be exported for general use in the binary. The name is not case-sensitive,
// and a panic is triggered if an implementation was bound to the same name
// before, or the implementation is nil. This function is mainly intended to be
// used by package initialization code.
func RegisterProcessorFactory(name string, impl ProcessorFactory) {
	key := strings.ToLower(name)
	if impl == nil {
		panic(fmt.Sprintf("invalid initialization: cannot register nil-processor using `%s`", key))
	}
	processorRegistryLock.Lock()
	defer processorRegistryLock.Unlock()
	if _, found := processorRegistry[key]; found {
		panic(fmt.Sprintf("invalid initialization: multiple Processors registered for `%s`", key))
	}
	processorRegistry[key] = impl
}

// ProcessorFactory is the type of a function that creates a new Processor
// using a given interpreter.
type ProcessorFactory func(Interpreter) Processor

// processorRegistry is a global registry for Processor instances of
// different implementations and configurations.
var processorRegistry = map[string]ProcessorFactory{}

// processorRegistryLock to protect access to the registry.
var processorRegistryLock sync.Mutex
