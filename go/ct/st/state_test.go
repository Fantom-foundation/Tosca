package st

import (
	"fmt"
	"regexp"
	"testing"
)

func TestState_Eq(t *testing.T) {
	s1 := NewState(NewCode([]byte{}))
	s2 := NewState(NewCode([]byte{}))
	if !s1.Eq(s2) {
		t.Fail()
	}

	s1.Status = Running
	s2.Status = Stopped
	if s1.Eq(s2) {
		t.Fail()
	}
	s2.Status = Running

	s1.Revision = Istanbul
	s2.Revision = London
	if s1.Eq(s2) {
		t.Fail()
	}
	s2.Revision = Istanbul

	s1.Pc = 1
	s2.Pc = 2
	if s1.Eq(s2) {
		t.Fail()
	}
	s2.Pc = 1

	s1.Gas = 1
	s2.Gas = 2
	if s1.Eq(s2) {
		t.Fail()
	}
	s2.Gas = 1

	s1 = NewState(NewCode([]byte{byte(ADD), byte(STOP)}))
	s2 = NewState(NewCode([]byte{byte(ADD), byte(ADD)}))
	if s1.Eq(s2) {
		t.Fail()
	}
}

func TestState_EqFailureStates(t *testing.T) {
	s1 := NewState(NewCode([]byte{}))
	s2 := NewState(NewCode([]byte{}))

	s1.Status = Failed
	s1.Gas = 1

	s2.Status = Failed
	s2.Gas = 2

	if !s1.Eq(s2) {
		t.Fail()
	}
}

func TestState_PrinterStatus(t *testing.T) {
	s := NewState(NewCode([]byte{}))
	s.Status = Running

	r := regexp.MustCompile("Status: ([[:alpha:]]+)")
	match := r.FindStringSubmatch(s.String())

	if len(match) != 2 {
		t.Fatal("invalid print, did not find 'Status' text")
	}

	want := s.Status.String()
	got := match[1]

	if got != want {
		t.Errorf("invalid print, wanted %s, got %s", want, got)
	}
}

func TestState_PrinterRevision(t *testing.T) {
	s := NewState(NewCode([]byte{}))
	s.Revision = London

	r := regexp.MustCompile("Revision: ([[:alpha:]]+)")
	match := r.FindStringSubmatch(s.String())

	if len(match) != 2 {
		t.Fatal("invalid print, did not find 'Revision' text")
	}

	want := s.Revision.String()
	got := match[1]

	if got != want {
		t.Errorf("invalid print, wanted %s, got %s", want, got)
	}
}

func TestState_PrinterPc(t *testing.T) {
	s := NewState(NewCode([]byte{byte(STOP)}))
	s.Pc = 1

	r := regexp.MustCompile(`Pc: ([[:digit:]]+) \(0x([0-9a-f]{4})\)`)
	match := r.FindStringSubmatch(s.String())

	if len(match) != 3 {
		t.Fatal("invalid print, did not find 'Pc' text")
	}

	want := fmt.Sprintf("%d", s.Pc)
	got := match[1]
	if got != want {
		t.Errorf("invalid print, wanted %s, got %s", want, got)
	}

	want = fmt.Sprintf("%04x", s.Pc)
	got = match[2]
	if got != want {
		t.Errorf("invalid print, wanted %s, got %s", want, got)
	}
}

func TestState_PrinterPcData(t *testing.T) {
	s := NewState(NewCode([]byte{byte(PUSH1), 7}))
	s.Pc = 1

	r := regexp.MustCompile(`\(points to data\)`)
	match := r.MatchString(s.String())

	if !match {
		t.Error("invalid print, did not find 'points to data' text")
	}
}

func TestState_PrinterPcOperation(t *testing.T) {
	s := NewState(NewCode([]byte{byte(ADD)}))
	s.Pc = 0

	r := regexp.MustCompile(`\(operation: ([[:alpha:]]+)\)`)
	match := r.FindStringSubmatch(s.String())

	if len(match) != 2 {
		t.Fatal("invalid print, did not find 'operation' text")
	}

	want := OpCode(s.Code.code[s.Pc]).String()
	got := match[1]
	if want != got {
		t.Errorf("invalid print, wanted %s, got %s", want, got)
	}
}

func TestState_PrinterPcOutOfBounds(t *testing.T) {
	s := NewState(NewCode([]byte{byte(STOP)}))
	s.Pc = 2

	r := regexp.MustCompile(`\(out of bounds\)`)
	match := r.MatchString(s.String())

	if !match {
		t.Error("invalid print, did not find 'out of bounds' text")
	}
}

func TestState_PrinterGas(t *testing.T) {
	s := NewState(NewCode([]byte{byte(STOP)}))
	s.Gas = 42

	r := regexp.MustCompile("Gas: ([[:digit:]]+)")
	match := r.FindStringSubmatch(s.String())

	if len(match) != 2 {
		t.Fatal("invalid print, did not find 'Gas' text")
	}

	want := fmt.Sprintf("%d", s.Gas)
	got := match[1]
	if want != got {
		t.Errorf("invalid print, wanted %s, got %s", want, got)
	}
}

func TestState_PrinterCode(t *testing.T) {
	s := NewState(NewCode([]byte{byte(PUSH2), 42, 42, byte(ADD), byte(STOP)}))

	r := regexp.MustCompile("Code: ([0-9a-f]+)")
	match := r.FindStringSubmatch(s.String())

	if len(match) != 2 {
		t.Fatal("invalid print, did not find 'Code' text")
	}

	want := s.Code.String()
	got := match[1]
	if want != got {
		t.Errorf("invalid print, wanted %s, got %s", want, got)
	}
}

func TestState_PrinterAbbreviatedCode(t *testing.T) {
	var longCode []byte
	for i := 0; i < codeCutoffLength+1; i++ {
		longCode = append(longCode, byte(INVALID))
	}

	s := NewState(NewCode(longCode))

	r := regexp.MustCompile(`Code: ([0-9a-f]+)... \(size: ([[:digit:]]+)\)`)
	match := r.FindStringSubmatch(s.String())

	if len(match) != 3 {
		t.Fatal("invalid print, did not find 'Code' text")
	}

	want := fmt.Sprintf("%x", s.Code.code[:codeCutoffLength])
	got := match[1]
	if want != got {
		t.Errorf("invalid print, wanted %s, got %s", want, got)
	}

	want = fmt.Sprintf("%d", len(s.Code.code))
	got = match[2]
	if want != got {
		t.Errorf("invalid print, wanted %s, got %s", want, got)
	}
}
