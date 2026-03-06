package sync

import (
	"testing"
)

func TestDeriveKey(t *testing.T) {
	ns1, key1 := DeriveKey("my-team-secret")
	ns2, key2 := DeriveKey("my-team-secret")

	if ns1 != ns2 {
		t.Fatalf("namespace not deterministic: %s != %s", ns1, ns2)
	}
	if len(ns1) != 16 {
		t.Fatalf("namespace should be 16 hex chars, got %d", len(ns1))
	}
	if len(key1) != 32 {
		t.Fatalf("key should be 32 bytes, got %d", len(key1))
	}
	for i := range key1 {
		if key1[i] != key2[i] {
			t.Fatal("key not deterministic")
		}
	}

	ns3, _ := DeriveKey("different-secret")
	if ns1 == ns3 {
		t.Fatal("different passphrases should produce different namespaces")
	}
}

func TestEncryptDecrypt(t *testing.T) {
	_, key := DeriveKey("test-pass")
	plaintext := []byte(`{"name":"demo","requests":[]}`)

	ciphertext, err := Encrypt(key, plaintext)
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}
	if string(ciphertext) == string(plaintext) {
		t.Fatal("ciphertext should differ from plaintext")
	}

	decrypted, err := Decrypt(key, ciphertext)
	if err != nil {
		t.Fatalf("decrypt: %v", err)
	}
	if string(decrypted) != string(plaintext) {
		t.Fatalf("roundtrip failed: got %q", decrypted)
	}
}

func TestDecryptWrongKey(t *testing.T) {
	_, key1 := DeriveKey("key-one")
	_, key2 := DeriveKey("key-two")

	ciphertext, _ := Encrypt(key1, []byte("secret"))
	_, err := Decrypt(key2, ciphertext)
	if err == nil {
		t.Fatal("decrypt with wrong key should fail")
	}
}
