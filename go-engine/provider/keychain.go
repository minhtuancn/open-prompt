package provider

import (
	"errors"

	keyring "github.com/zalando/go-keyring"
)

// KeychainServiceName là service name dùng để nhóm tất cả tokens trong system keychain
const KeychainServiceName = "open-prompt"

// ErrNotFound là lỗi khi key không tìm thấy trong keychain
var ErrNotFound = errors.New("key not found in keychain")

// Keychain là wrapper quanh system keychain
type Keychain struct {
	service string // service name dùng để nhóm các keys
}

// NewKeychain tạo Keychain mới với service name
func NewKeychain(service string) *Keychain {
	return &Keychain{service: service}
}

// Set lưu token vào system keychain
func (k *Keychain) Set(providerID, token string) error {
	return keyring.Set(k.service, providerID, token)
}

// Get lấy token từ system keychain
// Trả về ("", nil) nếu không tìm thấy
func (k *Keychain) Get(providerID string) (string, error) {
	val, err := keyring.Get(k.service, providerID)
	if err != nil {
		// go-keyring trả về error "secret not found" khi không tìm thấy
		// Normalize về empty string + nil error
		if isNotFoundError(err) {
			return "", nil
		}
		return "", err
	}
	return val, nil
}

// Delete xóa token khỏi system keychain
func (k *Keychain) Delete(providerID string) error {
	err := keyring.Delete(k.service, providerID)
	if err != nil && isNotFoundError(err) {
		return nil // không có gì để xóa — OK
	}
	return err
}

// isNotFoundError kiểm tra error có phải "not found" không
func isNotFoundError(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return msg == "secret not found" ||
		msg == "The specified item could not be found in the keychain." ||
		msg == keyring.ErrNotFound.Error()
}
