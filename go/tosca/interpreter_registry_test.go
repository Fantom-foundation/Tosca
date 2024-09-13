package tosca

import "testing"

func TestInterpreterRegistry_NameCollisionsAreDetected(t *testing.T) {
	const name = "something-just-for-this-test"
	factory := func(any) (Interpreter, error) {
		return nil, nil
	}
	if err := RegisterInterpreterFactory(name, factory); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := RegisterInterpreterFactory(name, factory); err == nil {
		t.Fatalf("expected error, got nil")
	}
}

func TestInterpreterRegistry_NilFactoriesAreRejected(t *testing.T) {
	const name = "something"
	if err := RegisterInterpreterFactory(name, nil); err == nil {
		t.Fatalf("expected error, got nil")
	}
}
