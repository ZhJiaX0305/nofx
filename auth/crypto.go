package auth

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"strings"
)

type CryptoManager struct {
	encryptionKey []byte
}

func NewCryptoManager() (*CryptoManager, error) {
	keyHex := os.Getenv("ENCRYPTION_KEY")
	if keyHex == "" {
		return nil, errors.New("ENCRYPTION_KEY environment variable is required")
	}

	// 解码十六进制密钥
	keyBytes, err := hex.DecodeString(keyHex)
	if err != nil {
		return nil, fmt.Errorf("failed to decode encryption key: %v", err)
	}

	if len(keyBytes) != 32 {
		return nil, errors.New("encryption key must be 32 bytes for AES-256")
	}
	return &CryptoManager{
		encryptionKey: keyBytes,
	}, nil
}

func (c *CryptoManager) GetEncryptionKey() []byte {
	return c.encryptionKey
}

func (c *CryptoManager) SetEncryptionKey(key []byte) {
	c.encryptionKey = key
}

func (c *CryptoManager) DecryptSensitiveData(ciphertext string) (string, error) {
	// 分离 IV 和加密数据
	parts := strings.Split(ciphertext, ":")
	if len(parts) != 2 {
		return "", errors.New("invalid encrypted data format: expected IV:data")
	}

	ivHex := parts[0]
	encryptedDataB64 := parts[1]

	// 解码 IV
	iv, err := hex.DecodeString(ivHex)
	if err != nil {
		return "", fmt.Errorf("failed to decode IV: %v", err)
	}

	// 解码加密数据
	encryptedData, err := base64.StdEncoding.DecodeString(encryptedDataB64)
	if err != nil {
		return "", fmt.Errorf("failed to decode encrypted data: %v", err)
	}

	// 使用 CTR 模式解密
	return c.decryptCTR(iv, encryptedData)
}

// decryptCTR 使用 CTR 模式解密
func (c *CryptoManager) decryptCTR(iv, encryptedData []byte) (string, error) {
	block, err := aes.NewCipher(c.encryptionKey)
	if err != nil {
		return "", err
	}

	stream := cipher.NewCTR(block, iv)
	plaintext := make([]byte, len(encryptedData))
	stream.XORKeyStream(plaintext, encryptedData)

	return string(plaintext), nil
}

func (c *CryptoManager) EncryptSensitiveData(plaintext string) (string, error) {
	// 生成随机 IV
	iv := make([]byte, 16)
	if _, err := rand.Read(iv); err != nil {
		return "", fmt.Errorf("failed to generate IV: %v", err)
	}

	// 使用 CTR 模式加密
	block, err := aes.NewCipher(c.encryptionKey)
	if err != nil {
		return "", err
	}

	stream := cipher.NewCTR(block, iv)
	plaintextBytes := []byte(plaintext)
	ciphertext := make([]byte, len(plaintext))
	stream.XORKeyStream(ciphertext, plaintextBytes)

	// 编码为字符串格式
	ivHex := hex.EncodeToString(iv)
	encryptedDataB64 := base64.StdEncoding.EncodeToString(ciphertext)

	return ivHex + ":" + encryptedDataB64, nil
}
