package dkim

import (
	"crypto/ed25519"
	"crypto/mldsa"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"strings"
)

const (
	KeyLength1024 KeyLength = 1024
	KeyLength2048 KeyLength = 2048
	KeyLength4096 KeyLength = 4096

	KeyTypeRSA     KeyType = "rsa"
	KeyTypeED25519 KeyType = "ed25519"
	KeyTypeMLDSA44 KeyType = "mldsa44"

	PrivateKeyPEMType = "PRIVATE KEY"
)

type KeyLength uint
type KeyType string

// KeyProvider is an interface for generating DKIM keys and their corresponding public keys.
type KeyProvider interface {
	Name() KeyType
	GenKey() ([]byte, error)
	PublicKey([]byte) (string, error)
}

// RSAKeyProvider is a KeyProvider that generates RSA keys.
type RSAKeyProvider struct {
	Size KeyLength
}

// Name returns the name of the key type.
func (r *RSAKeyProvider) Name() KeyType {
	return KeyTypeRSA
}

// GenKey generates an RSA private key.
func (r *RSAKeyProvider) GenKey() ([]byte, error) {
	privateKey, err := rsa.GenerateKey(rand.Reader, int(r.Size))
	if err != nil {
		return nil, err
	}

	key := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
	})
	return key, err
}

// PublicKey derives the RSA public key from the given private key.
func (r *RSAKeyProvider) PublicKey(priv []byte) (string, error) {
	block, _ := pem.Decode(priv)
	if block == nil || block.Type != "RSA PRIVATE KEY" {
		return "", fmt.Errorf("failed to decode PEM block containing RSA private key")
	}
	privateKey, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		return "", err
	}
	if s := privateKey.N.BitLen(); s != int(r.Size) {
		return "", fmt.Errorf("key size mismatch: expected %d bits, got %d bits", r.Size, s)
	}
	pub, err := x509.MarshalPKIXPublicKey(privateKey.Public())
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(pub), nil
}

type ED25519KeyProvider struct{}

func (e *ED25519KeyProvider) Name() KeyType {
	return KeyTypeED25519
}

func (e *ED25519KeyProvider) GenKey() ([]byte, error) {
	_, privKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, err
	}
	bytes, err := x509.MarshalPKCS8PrivateKey(privKey)
	if err != nil {
		return nil, err
	}
	key := pem.EncodeToMemory(&pem.Block{
		Type:  PrivateKeyPEMType,
		Bytes: bytes,
	})
	return key, nil
}

func (e *ED25519KeyProvider) PublicKey(priv []byte) (string, error) {
	block, _ := pem.Decode(priv)
	if block == nil || block.Type != PrivateKeyPEMType {
		return "", fmt.Errorf("failed to decode PEM block containing ed25519 private key")
	}
	key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return "", err
	}
	privKey, ok := key.(ed25519.PrivateKey)
	if !ok {
		return "", fmt.Errorf("not an ed25519 private key")
	}
	pub, err := x509.MarshalPKIXPublicKey(privKey.Public())
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(pub), nil
}

type MLDSA44KeyProvider struct{}

func (m *MLDSA44KeyProvider) Name() KeyType {
	return KeyTypeMLDSA44
}

func (m *MLDSA44KeyProvider) GenKey() ([]byte, error) {
	privKey, err := mldsa.GenerateKey(mldsa.MLDSA44())
	if err != nil {
		return nil, err
	}
	bytes, err := x509.MarshalPKCS8PrivateKey(privKey)
	if err != nil {
		return nil, err
	}
	key := pem.EncodeToMemory(&pem.Block{
		Type:  PrivateKeyPEMType,
		Bytes: bytes,
	})
	return key, nil
}

func (m *MLDSA44KeyProvider) PublicKey(priv []byte) (string, error) {
	block, _ := pem.Decode(priv)
	if block == nil || block.Type != PrivateKeyPEMType {
		return "", fmt.Errorf("failed to decode PEM block containing MLDSA-44 private key")
	}
	key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return "", err
	}
	privKey, ok := key.(*mldsa.PrivateKey)
	if !ok {
		return "", fmt.Errorf("not an MLDSA-44 private key")
	}
	pub, err := x509.MarshalPKIXPublicKey(privKey.PublicKey())
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(pub), nil
}

func NewKeyProvider(keyType KeyType, size KeyLength) (KeyProvider, error) {
	switch keyType {
	case KeyTypeRSA:
		return &RSAKeyProvider{Size: size}, nil
	case KeyTypeED25519:
		return &ED25519KeyProvider{}, nil
	case KeyTypeMLDSA44:
		return &MLDSA44KeyProvider{}, nil
	default:
		return nil, fmt.Errorf("unsupported key type: %s", keyType)
	}
}

func GenTXTValue(kp KeyProvider, pub string) string {
	res := []string{}
	p := fmt.Sprintf("p=%s", pub)
	res = append(res, fmt.Sprintf("\"v=DKIM1; k=%s;\"", kp.Name()))
	res = append(res, splitKey(p)...)
	return strings.Join(res, " ")
}

func splitKey(pub string) []string {
	var res []string
	parts := len(pub) / 255
	remainder := len(pub) % 255
	if parts < 1 {
		return []string{"\"" + pub + "\""}
	}
	for i := 0; i < parts; i++ {
		res = append(res, "\""+pub[i*255:(i+1)*255]+"\"")
	}
	if remainder > 0 {
		res = append(res, "\""+pub[parts*255:parts*255+remainder]+"\"")
	}
	return res
}
