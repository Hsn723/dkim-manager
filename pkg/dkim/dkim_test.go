package dkim

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"fmt"
	"testing"
)

func TestGenTXTValueRSA(t *testing.T) {
	t.Parallel()
	key, err := rsa.GenerateKey(rand.Reader, int(KeyLength2048))
	if err != nil {
		t.Fatal(err)
	}
	expectedPubBytes, err := x509.MarshalPKIXPublicKey(key.Public().(*rsa.PublicKey))
	if err != nil {
		t.Fatal(err)
	}
	expectedPub := base64.StdEncoding.EncodeToString(expectedPubBytes)
	expectedRecord := fmt.Sprintf("v=DKIM1; h=sha256; k=rsa; p=%s", string(expectedPub))
	actualRecord := GenTXTValue(expectedPub, KeyTypeRSA)
	if actualRecord != expectedRecord {
		t.Errorf("expected %s, got %s", expectedRecord, actualRecord)
	}
}

func TestGenTXTValueED25519(t *testing.T) {
	t.Parallel()
	pub, _, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	expectedPubBytes, err := x509.MarshalPKIXPublicKey(pub)
	if err != nil {
		t.Fatal(err)
	}
	expectedPub := base64.StdEncoding.EncodeToString(expectedPubBytes)
	expectedRecord := fmt.Sprintf("v=DKIM1; k=ed25519; p=%s", string(expectedPub))
	actualRecord := GenTXTValue(expectedPub, KeyTypeED25519)
	if actualRecord != expectedRecord {
		t.Errorf("expected %s, got %s", expectedRecord, actualRecord)
	}
}
