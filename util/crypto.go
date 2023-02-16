package util

import (
	"bytes"
	"crypto/aes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"errors"
)

// Padding is the padding type for encryption and decryption
type Padding string

// Definition of padding type
const (
	PKCS5Padding Padding = "PKCS5"
	PKCS7Padding Padding = "PKCS7"
	ZerosPadding Padding = "Zeros"
	NoPadding    Padding = "None"
)

// GenerateRandomBytes uses to generate a random bytes by given length
func GenerateRandomBytes(length int) ([]byte, error) {
	buffer := make([]byte, length)
	_, err := rand.Read(buffer)
	if err != nil {
		return nil, err
	}
	return buffer, nil
}

func EncryptAES(secretKey string, padding Padding, data string) (string, error) {
	// if len(secretKey) == 16 {
	// 	secretKey += secretKey[:8]
	// }
	block, err := aes.NewCipher([]byte(secretKey))
	if err != nil {
		return "", err
	}
	blockSize := block.BlockSize()
	src := []byte(data)
	switch padding {
	case PKCS5Padding:
		src = pkcs5Padding(src, blockSize)
	case PKCS7Padding:
		src = pkcs7Padding(src, blockSize)
	case ZerosPadding:
		src = zerosPadding(src, blockSize)
	}
	if len(src)%blockSize != 0 {
		return "", errors.New("crypto/cipher: input not full blocks")
	}
	dst := make([]byte, len(src))
	tmp := dst
	for len(src) > 0 {
		block.Encrypt(tmp, src[:blockSize])
		src = src[blockSize:]
		tmp = tmp[blockSize:]
	}
	return base64.StdEncoding.EncodeToString(dst), nil
}

func DecryptAES(secretKey string, padding Padding, data string) (string, error) {
	// if len(secretKey) == 16 {
	// 	secretKey += secretKey[:8]
	// }
	block, err := aes.NewCipher([]byte(secretKey))
	if err != nil {
		return "", err
	}
	blockSize := block.BlockSize()
	src, err := base64.StdEncoding.DecodeString(data)
	if err != nil {
		return "", err
	}
	if len(src)%blockSize != 0 {
		return "", errors.New("crypto/cipher: input not full blocks")
	}
	dst := make([]byte, len(src))
	tmp := dst
	for len(src) > 0 {
		block.Decrypt(tmp, src[:blockSize])
		src = src[blockSize:]
		tmp = tmp[blockSize:]
	}
	switch padding {
	case PKCS5Padding:
		dst = pkcs5Unpadding(dst)
	case PKCS7Padding:
		dst = pkcs7Unpadding(dst)
	case ZerosPadding:
		dst = zerosUnpadding(dst)
	}
	return string(dst), nil
}

func pkcs5Padding(src []byte, blockSize int) []byte {
	return pkcs7Padding(src, blockSize)
}

func pkcs5Unpadding(src []byte) []byte {
	return pkcs7Unpadding(src)
}

func pkcs7Padding(src []byte, blockSize int) []byte {
	padding := blockSize - len(src)%blockSize
	padtext := bytes.Repeat([]byte{byte(padding)}, padding)
	return append(src, padtext...)
}

func pkcs7Unpadding(src []byte) []byte {
	length := len(src)
	unpadding := int(src[length-1])
	return src[:(length - unpadding)]
}

func zerosPadding(src []byte, blockSize int) []byte {
	paddingCount := blockSize - len(src)%blockSize
	if paddingCount == 0 {
		return src
	}
	return append(src, bytes.Repeat([]byte{byte(0)}, paddingCount)...)
}

func zerosUnpadding(src []byte) []byte {
	for i := len(src) - 1; ; i-- {
		if src[i] != 0 {
			return src[:i+1]
		}
	}
}

func ParseRsaPublicKeyFromPemBase64Str(pubPEM string) (*rsa.PublicKey, error) {
	block, _ := pem.Decode([]byte(pubPEM))
	if block == nil {
		return nil, errors.New("failed to parse PEM block containing the key")
	}

	pub, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, err
	}

	switch pub := pub.(type) {
	case *rsa.PublicKey:
		return pub, nil
	default:
		break // fall through
	}
	return nil, errors.New("key type is not RSA")
}

func ParseRsaPrivateKeyFromPemBase64Str(privPEM, password string) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode([]byte(privPEM))
	if block == nil {
		return nil, errors.New("failed to parse PEM block containing the key")
	}

	var (
		decryptedPrivateKey []byte
		privKey             *rsa.PrivateKey
		err                 error
	)
	if password != "" {
		decryptedPrivateKey, err = x509.DecryptPEMBlock(block, []byte(password))
		if err != nil {
			return nil, err
		}
		privKey, err = x509.ParsePKCS1PrivateKey(decryptedPrivateKey)
		if err != nil {
			return nil, err
		}
	} else {
		privKey, err = x509.ParsePKCS1PrivateKey(block.Bytes)
		if err != nil {
			return nil, err
		}
	}

	return privKey, nil
}
