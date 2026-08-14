package service

import "testing"

func TestExtractSlots(t *testing.T) {
	s := newExtractSlots(1)
	if !s.try() {
		t.Fatal("first acquire")
	}
	if s.try() {
		t.Fatal("second acquire should fail at cap 1")
	}
	s.release()
	if !s.try() {
		t.Fatal("after release")
	}
	s.release()
}

func TestExtractSlots_CapTwo(t *testing.T) {
	s := newExtractSlots(2)
	if !s.try() || !s.try() {
		t.Fatal("two slots")
	}
	if s.try() {
		t.Fatal("third should fail")
	}
	s.release()
	s.release()
}
