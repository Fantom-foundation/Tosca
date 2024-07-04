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
	"slices"
	"testing"

	gomock "go.uber.org/mock/gomock"
	"golang.org/x/exp/maps"
)

func TestProcessorRegistry_CanListContent(t *testing.T) {
	myFactory := func(Interpreter) Processor {
		return nil
	}

	name := "test1"
	RegisterProcessorFactory(name, myFactory)

	factories := maps.Keys(GetAllRegisteredProcessorFactories())
	if !slices.Contains(factories, name) {
		t.Errorf("%v not found in list of factories, found %v", name, factories)
	}
}

func TestProcessorRegistry_RegisteredFactoryCanBeUsed(t *testing.T) {
	counter := 0
	name := "test2"
	myFactory := func(Interpreter) Processor {
		counter++
		return nil
	}
	RegisterProcessorFactory(name, myFactory)

	got := GetProcessorFactory(name)
	if got == nil {
		t.Fatalf("expected factory, got nil")
	}
	got(nil)
	if counter != 1 {
		t.Errorf("expected factory to be called once, got %d", counter)
	}
}

func TestProcessorRegistry_RegisteredFactoryIsUsedByGetProcessor(t *testing.T) {
	ctrl := gomock.NewController(t)
	interpreter := NewMockInterpreter(ctrl)

	name := "test3"
	myFactory := func(i Interpreter) Processor {
		if i != interpreter {
			t.Fatalf("unexpected interpreter passed to factory")
		}
		return nil
	}
	RegisterProcessorFactory(name, myFactory)

	GetProcessor(name, interpreter)
}

func TestProcessorRegistry_GetProcessorReturnsNilForUnknownProcessor(t *testing.T) {
	if processor := GetProcessor("something odd", nil); processor != nil {
		t.Errorf("expected nil processor, got %v", processor)
	}
}

func TestProcessorRegistry_FailToRegisterNilFactory(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("expected panic, got nil")
		}
	}()
	RegisterProcessorFactory("nil", nil)
}

func TestProcessorRegistry_FailToRegisterSameNameMultipleTimes(t *testing.T) {
	name := "test4"
	myFactory := func(Interpreter) Processor { return nil }

	// The first time it is fine.
	RegisterProcessorFactory(name, myFactory)

	// The second time it should panic.
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("expected panic, got nil")
		}
	}()
	RegisterProcessorFactory(name, myFactory)
}
