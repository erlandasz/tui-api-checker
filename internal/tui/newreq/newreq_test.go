package newreq

import (
	"testing"

	tea "charm.land/bubbletea/v2"
)

func TestNewreq_NotVisibleByDefault(t *testing.T) {
	m := New()
	if m.Visible() {
		t.Error("expected not visible by default")
	}
}

func TestNewreq_ShowMakesVisible(t *testing.T) {
	m := New()
	m.Show("my-collection")
	if !m.Visible() {
		t.Error("expected visible after Show")
	}
}

func TestNewreq_EscCancels(t *testing.T) {
	m := New()
	m.Show("col")
	updated, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	if updated.Visible() {
		t.Error("expected not visible after Esc")
	}
	if cmd == nil {
		t.Fatal("expected a command")
	}
	msg := cmd()
	if _, ok := msg.(CancelledMsg); !ok {
		t.Errorf("expected CancelledMsg, got %T", msg)
	}
}

func TestNewreq_EnterWithEmptyNameDoesNothing(t *testing.T) {
	m := New()
	m.Show("col")
	updated, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if !updated.Visible() {
		t.Error("expected still visible with empty name")
	}
	if cmd != nil {
		t.Error("expected no command with empty name")
	}
}

func TestNewreq_TypeAndEnterCreatesRequest(t *testing.T) {
	m := New()
	m.Show("my-api")

	// Type "Get Users" character by character
	for _, ch := range "Get Users" {
		m, _ = m.Update(tea.KeyPressMsg{Code: ch, Text: string(ch)})
	}

	updated, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if updated.Visible() {
		t.Error("expected not visible after confirm")
	}
	if cmd == nil {
		t.Fatal("expected a command")
	}
	msg := cmd()
	created, ok := msg.(RequestCreatedMsg)
	if !ok {
		t.Fatalf("expected RequestCreatedMsg, got %T", msg)
	}
	if created.Collection != "my-api" {
		t.Errorf("collection = %q, want %q", created.Collection, "my-api")
	}
	if created.Request.Name != "Get Users" {
		t.Errorf("name = %q, want %q", created.Request.Name, "Get Users")
	}
	if created.Request.Method != "GET" {
		t.Errorf("method = %q, want %q", created.Request.Method, "GET")
	}
}

func TestNewreq_BackspaceDeletesChar(t *testing.T) {
	m := New()
	m.Show("col")

	for _, ch := range "abc" {
		m, _ = m.Update(tea.KeyPressMsg{Code: ch, Text: string(ch)})
	}
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyBackspace})
	m, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})

	msg := cmd()
	created := msg.(RequestCreatedMsg)
	if created.Request.Name != "ab" {
		t.Errorf("name = %q, want %q", created.Request.Name, "ab")
	}
	_ = m
}
