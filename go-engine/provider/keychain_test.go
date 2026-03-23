package provider_test

import (
	"testing"

	"github.com/minhtuancn/open-prompt/go-engine/provider"
)

func TestKeychain_SetAndGet(t *testing.T) {
	kc := provider.NewKeychain("open-prompt-test")

	// Set token
	if err := kc.Set("test-provider", "test-api-key-value"); err != nil {
		t.Skipf("keychain không khả dụng trong môi trường này: %v", err)
	}

	// Get token
	val, err := kc.Get("test-provider")
	if err != nil {
		t.Fatalf("Get thất bại: %v", err)
	}
	if val != "test-api-key-value" {
		t.Errorf("Get = %q, muốn %q", val, "test-api-key-value")
	}

	// Delete token
	if err := kc.Delete("test-provider"); err != nil {
		t.Fatalf("Delete thất bại: %v", err)
	}

	// Get sau delete phải trả về empty
	val2, _ := kc.Get("test-provider")
	if val2 != "" {
		t.Error("Get sau Delete phải trả về rỗng")
	}
}

func TestKeychain_GetNotFound(t *testing.T) {
	kc := provider.NewKeychain("open-prompt-test")
	val, err := kc.Get("nonexistent-provider-xyz")
	// Không có error, chỉ trả về empty string
	if err != nil {
		t.Skipf("keychain không khả dụng: %v", err)
	}
	if val != "" {
		t.Errorf("Get không tìm thấy phải trả về rỗng, got %q", val)
	}
}
