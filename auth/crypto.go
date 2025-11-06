package auth

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
)

// EncryptionKey 用于加密敏感数据的密钥（从 JWT Secret 派生）
var EncryptionKey []byte

// SetEncryptionKey 设置加密密钥
func SetEncryptionKey(key []byte) {
	EncryptionKey = key
}

// EncryptSensitiveData 加密敏感数据（API Key、Secret Key 等）
// 使用 AES-256-GCM 加密
func EncryptSensitiveData(plaintext string) (string, error) {
	if len(EncryptionKey) == 0 {
		return "", fmt.Errorf("加密密钥未设置")
	}

	// 确保密钥长度为 32 字节（AES-256）
	key := make([]byte, 32)
	copy(key, EncryptionKey)

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// DecryptSensitiveData 解密敏感数据
func DecryptSensitiveData(ciphertext string) (string, error) {
	if len(EncryptionKey) == 0 {
		return "", fmt.Errorf("加密密钥未设置")
	}

	// 确保密钥长度为 32 字节（AES-256）
	key := make([]byte, 32)
	copy(key, EncryptionKey)

	data, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return "", fmt.Errorf("密文数据无效")
	}

	nonce, ciphertextBytes := data[:nonceSize], data[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertextBytes, nil)
	if err != nil {
		return "", err
	}

	return string(plaintext), nil
}
