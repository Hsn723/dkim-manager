package dkim

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"strings"
)

type KeyLength uint
type KeyType string

const (
	KeyLength1024 KeyLength = 1024
	KeyLength2048 KeyLength = 2048
	KeyLength4096 KeyLength = 4096

	KeyTypeRSA     KeyType = "rsa"
	KeyTypeED25519 KeyType = "ed25519"
)

// GenRSA generates an RSA key pair.
func GenRSA(size KeyLength) ([]byte, string, error) {
	privateKey, err := rsa.GenerateKey(rand.Reader, int(size))
	if err != nil {
		return nil, "", err
	}

	key := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
	})

	pub, err := x509.MarshalPKIXPublicKey(privateKey.Public().(*rsa.PublicKey))
	if err != nil {
		return nil, "", err
	}
	return key, base64.StdEncoding.EncodeToString(pub), nil
}

// GenED25519 generates an ed25519 key pair.
func GenED25519() ([]byte, string, error) {
	pubKey, privKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, "", err
	}
	bytes, err := x509.MarshalPKCS8PrivateKey(privKey)
	if err != nil {
		return nil, "", err
	}
	key := pem.EncodeToMemory(&pem.Block{
		Type:  "PRIVATE KEY",
		Bytes: bytes,
	})
	pub, err := x509.MarshalPKIXPublicKey(pubKey)
	if err != nil {
		return nil, "", err
	}
	return key, base64.StdEncoding.EncodeToString(pub), nil
}

// DeriveRSAPublicKey computes the RSA public key from the private key.
func DeriveRSAPublicKey(priv []byte, size KeyLength) (string, error) {
	block, _ := pem.Decode(priv)
	if block == nil || block.Type != "RSA PRIVATE KEY" {
		return "", fmt.Errorf("failed to decode PEM block containing RSA private key")
	}
	privateKey, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		return "", err
	}
	if s := privateKey.N.BitLen(); s != int(size) {
		return "", fmt.Errorf("key size mismatch: expected %d bits, got %d bits", size, s)
	}
	pub, err := x509.MarshalPKIXPublicKey(privateKey.Public().(*rsa.PublicKey))
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(pub), nil
}

// DeriveED25519PublicKey computes the ed25519 public key from the private key.
func DeriveED25519PublicKey(priv []byte) (string, error) {
	block, _ := pem.Decode(priv)
	if block == nil || block.Type != "PRIVATE KEY" {
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
	pub, err := x509.MarshalPKIXPublicKey(privKey.Public().(ed25519.PublicKey))
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(pub), nil
}

// GenTXTValue generates the DKIM record for the given public key.
func GenTXTValue(pub string, keyType KeyType) string {
	res := []string{}
	p := fmt.Sprintf("p=%s", pub)
	switch keyType {
	case KeyTypeRSA:
		res = append(res, fmt.Sprintf("\"v=DKIM1; h=sha256; k=%s;\"", keyType))
	case KeyTypeED25519:
		res = append(res, fmt.Sprintf("\"v=DKIM1; k=%s;\"", keyType))
	default:
		return ""
	}

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
