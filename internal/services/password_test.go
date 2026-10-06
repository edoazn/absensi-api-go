package services_test

import (
	"strings"
	"testing"

	"github.com/edoazn/absensi-go/internal/services"
)

func TestCheckPasswordRoundtrip(t *testing.T) {
	hash, err := services.HashPassword("rahasia123", 4)
	if err != nil {
		t.Fatalf("hash: %v", err)
	}

	if !services.CheckPassword(hash, "rahasia123") {
		t.Fatal("correct password must verify")
	}
	if services.CheckPassword(hash, "salah") {
		t.Fatal("wrong password must fail")
	}
}

func TestCheckPasswordSupportsPhpBcryptPrefix(t *testing.T) {
	hash, err := services.HashPassword("password", 4)
	if err != nil {
		t.Fatalf("hash: %v", err)
	}

	phpHash := strings.Replace(hash, "$2a$", "$2y$", 1)
	if !strings.HasPrefix(phpHash, "$2y$") {
		t.Fatalf("test setup broken: %s", phpHash)
	}

	if !services.CheckPassword(phpHash, "password") {
		t.Fatal("$2y$ hash from Laravel/PHP must verify")
	}
}
