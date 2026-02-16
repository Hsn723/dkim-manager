package dkim

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGenRSA(t *testing.T) {
	t.Parallel()
	priv, pub, err := GenRSA(KeyLength2048)
	assert.NoError(t, err)
	derivedPub, err := DeriveRSAPublicKey(priv, KeyLength2048)
	assert.NoError(t, err)
	assert.Equal(t, pub, derivedPub)
}

func TestGenED25519(t *testing.T) {
	t.Parallel()
	priv, pub, err := GenED25519()
	assert.NoError(t, err)
	derivedPub, err := DeriveED25519PublicKey(priv)
	assert.NoError(t, err)
	assert.Equal(t, pub, derivedPub)
}

func TestDeriveRSAPublicKey(t *testing.T) {
	t.Parallel()

	priv1, pub1, err := GenRSA(KeyLength2048)
	assert.NoError(t, err)
	priv2, _, err := GenED25519()
	assert.NoError(t, err)
	malformedPriv := []byte(`-----BEGIN RSA PRIVATE KEY-----
MIIEpQIBAAKCAQEAsCRgDQW1+V79Kg2WU2ukEEe/xgk9GJJHAQqg2Bh4gMZ8lrpA
QcOjqWCtuJMNzKEnwR4VgurMwoUM2cExrQ0IpKhp9AWuB79ogeCHrivtbA4K8rC0
lgCLJQNBt4XxqWleOWMGnEcbrOevUvYNclmMd4VB+nH+LSei7ZcgBvo=
-----END RSA PRIVATE KEY-----`)

	cases := []struct {
		size        KeyLength
		errFunc     assert.ErrorAssertionFunc
		expectedPub string
		title       string
		priv        []byte
	}{
		{
			title:       "ValidKey",
			size:        KeyLength2048,
			errFunc:     assert.NoError,
			expectedPub: pub1,
			priv:        priv1,
		},
		{
			title:       "InvalidKeyLength",
			size:        KeyLength4096,
			errFunc:     assert.Error,
			expectedPub: "",
			priv:        priv1,
		},
		{
			title:       "InvalidKeyType",
			size:        KeyLength2048,
			errFunc:     assert.Error,
			expectedPub: "",
			priv:        priv2,
		},
		{
			title:       "MalformedKey",
			size:        KeyLength2048,
			errFunc:     assert.Error,
			expectedPub: "",
			priv:        malformedPriv,
		},
	}
	for _, tc := range cases {
		t.Run(tc.title, func(t *testing.T) {
			t.Parallel()
			pub, err := DeriveRSAPublicKey(tc.priv, tc.size)
			tc.errFunc(t, err)
			assert.Equal(t, tc.expectedPub, pub)
		})
	}
}

func TestDeriveED25519PublicKey(t *testing.T) {
	t.Parallel()

	priv1, pub1, err := GenED25519()
	assert.NoError(t, err)
	priv2, _, err := GenRSA(KeyLength2048)
	assert.NoError(t, err)
	malformedPriv := []byte(`-----BEGIN PRIVATE KEY-----
MIGHAgEAMBMGByqGSM49AgEGCCqGSM49AwEHBG0wawIBAQQg0sheRg0M5wMUsovd
XAnM+3hWDglLqvKt4vEXzKj80FDKMtpHXzPnT5FER+Pef/oID2jF+tLC
-----END PRIVATE KEY-----`)
	pkcs8RSAPriv := []byte(`-----BEGIN PRIVATE KEY-----
MIIEvwIBADANBgkqhkiG9w0BAQEFAASCBKkwggSlAgEAAoIBAQDsSzZpaoGT4V5n
qKpe9jRkMpF4IFT5sB5smyQSrrxnYnlR/w6FLEGpa7iS4bLOqyPNIv80w2x9L2ZS
fUrvteYhwcxWHC+hkgfHj5U7dUfPP3f8HdZgnvlIp+4d/+e2MB9fAwOvbIEOVIiN
0DtqRl3ny6/rKdRwdcH917mowOO/lx/8SxXV9aBC0ZhhStvO5P2bmhhTeke0DFyZ
5b/A0ri+ZlTR65K8ybGsdjX+6xVgh8G1YnT+Ien2dmc4C/ajJZX501W7/+7RIb/X
VwiVU8nRoPjZ7hJZIguNQanyfbZTOzUABjwyj6eq70KmE4JuiGKxOSOAWb4TNnGe
3iYvVRWBAgMBAAECggEAKUWt2FXREjpgGez87R9F4WZvwDKthPs9JS0n7Nd2cvxh
QnAxYhNr2KLHF2iyqaR82VzOhLHJpkf8MFZQG3SDIMxP246Kx0bRcwriPwNqKonk
dDXl9vRtiEJEthq3pzvajubg4ugp1o3vWA3SAusNheou7P1ebNI0sxjWBeLDJPhz
ozueJ/KFlQvdHr9d8fvlVzFEL2ItQJCHo2Ds0MXTjD5t5Ao7XY9BcirSEIWTdiaD
iASlL/r1P6TEckI31uksCv3lHNgGfimrQaBI2MQl+L8rN49MsPAl4k3kSVHWI4aJ
f7UgeoTQSgOyUPV1rdZIRxWqpk0+NHzXgOIajlIw0QKBgQD3VS+Rz8P5yinWnZCG
MjHtDUYxdkxHsNNG3WI+0I/WcVUnrYH9sekiOfCEa7AOeWTtuPcN3aBHtdt+QuZd
X1Y0dB9UYd0XaITbskg0Hv4p/am5xqM27q5gRroU4RQRF4rGDgpwwyrFUNFkePZE
dRe5jmY0oLhLgZlcaPmjIK6udQKBgQD0kv8gjyKzBvJy349hUq2q7qhzkDef66gB
aNwmZMkPeYE3dsDy9A1VM8NYiWBJIOzNzTpJexO7dyaA8T2diJQ0/fSXCi6TMcXa
QNgd/mWTX6XXq6syAPE/HQb0PoSbKUp5w5B9A0IF6h5OyU8Zcoqq2Y+Z6eXqIeeZ
hY/PVMRBXQKBgQDyL2LWJ5ihxpizQzRag1op4f6ivlCxPm+Ti4IBOh4ugGk+4gJQ
ld5QGmXudLg/ZBU1RhH8bNDehy+3cfC663ixAigPa4ifvEOkEO3sw5BjM7T3aY82
Yf8z3O2nNkJ8/g1wJB2LD0CZV6rB9ERJAlNJ6isgS2RK40t1loEjgAQsZQKBgQC6
sIKC7f/EvKbRPQmLdrsOYYLAQ/PR5OanvM1fmUtIvqz+E24Rhm2u/gY9TQ/sgm+A
YQn/ES3syXTgtEUePSU0li3gJWuL/FBU226c5pXOuxIy4N2bG9ELJjMquZYrgodR
DxD5/ESnkyBzb4Mrn51t8QiGql5QLHVHYQZ3cvMkGQKBgQCCNtH6h3JGE0KLd8p7
PC2WmmFELefPRvKIwSRFw/8HmXs3ozo4FccnjozaPXrQeYzP1a090weEmTFYrv3X
FqSAGKoR3dKJ4dlrXEtZoeFu9p2YpncwjPAVxM6NDZUV+2Bl9LZ+VM7UBGvL2Hsz
BGcscISjwRvQHrE6JigS1SMVHQ==
-----END PRIVATE KEY-----`)

	cases := []struct {
		title       string
		priv        []byte
		expectedPub string
		errFunc     assert.ErrorAssertionFunc
	}{
		{
			title:       "ValidKey",
			priv:        priv1,
			expectedPub: pub1,
			errFunc:     assert.NoError,
		},
		{
			title:       "InvalidKeyType",
			priv:        priv2,
			expectedPub: "",
			errFunc:     assert.Error,
		},
		{
			title:       "MalformedKey",
			priv:        malformedPriv,
			expectedPub: "",
			errFunc:     assert.Error,
		},
		{
			title:       "InvalidPKCS8KeyType",
			priv:        pkcs8RSAPriv,
			expectedPub: "",
			errFunc:     assert.Error,
		},
	}
	for _, tc := range cases {
		t.Run(tc.title, func(t *testing.T) {
			t.Parallel()
			pub, err := DeriveED25519PublicKey(tc.priv)
			tc.errFunc(t, err)
			assert.Equal(t, tc.expectedPub, pub)
		})
	}
}

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
	expectedRecordStart := "\"v=DKIM1; h=sha256; k=rsa;\" \"p="
	actualRecord := GenTXTValue(expectedPub, KeyTypeRSA)
	assert.Contains(t, actualRecord, expectedRecordStart)
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
	expectedRecordStart := "\"v=DKIM1; k=ed25519;\" \"p="
	actualRecord := GenTXTValue(expectedPub, KeyTypeED25519)
	assert.Contains(t, actualRecord, expectedRecordStart)
}

func TestSplitKey(t *testing.T) {
	t.Parallel()
	cases := []struct {
		title    string
		pub      string
		expected []string
	}{
		{
			title:    "SmallKey",
			pub:      "p=MCowBQYDK2VwAyEAmkOfvsQ8hWUtwl1GoRCWDRColuZVF/W9mX9uVocYXHE=",
			expected: []string{"\"p=MCowBQYDK2VwAyEAmkOfvsQ8hWUtwl1GoRCWDRColuZVF/W9mX9uVocYXHE=\""},
		},
		{
			title:    "ExactFit",
			pub:      "p=MIIBITANBgkqhkiG9w0BAQEFAAOCAQ4AMIIBCQKCAQBcIjyCd+/fUqlYzi3ELmmWGJtEGiIpwi6qWXygfbarGkmHEYPsHTFBZj6Sy+r7wEA80cbcvWTcP41lETNR/k4NuaA88U6LHbDhBOhW4lVNWUQQxOA+EPq99blCaiDsAJU2MnEEA7iGryehPxRe5YU/I4MY1DJPOTerDGjJnlmp11n2Wd8XJnxXkVhgDZ6pgmUk2KiDOxNuT6iGHRFkK",
			expected: []string{"\"p=MIIBITANBgkqhkiG9w0BAQEFAAOCAQ4AMIIBCQKCAQBcIjyCd+/fUqlYzi3ELmmWGJtEGiIpwi6qWXygfbarGkmHEYPsHTFBZj6Sy+r7wEA80cbcvWTcP41lETNR/k4NuaA88U6LHbDhBOhW4lVNWUQQxOA+EPq99blCaiDsAJU2MnEEA7iGryehPxRe5YU/I4MY1DJPOTerDGjJnlmp11n2Wd8XJnxXkVhgDZ6pgmUk2KiDOxNuT6iGHRFkK\""},
		},
		{
			title: "RSA2048",
			pub:   "p=MIIBITANBgkqhkiG9w0BAQEFAAOCAQ4AMIIBCQKCAQBcIjyCd+/fUqlYzi3ELmmWGJtEGiIpwi6qWXygfbarGkmHEYPsHTFBZj6Sy+r7wEA80cbcvWTcP41lETNR/k4NuaA88U6LHbDhBOhW4lVNWUQQxOA+EPq99blCaiDsAJU2MnEEA7iGryehPxRe5YU/I4MY1DJPOTerDGjJnlmp11n2Wd8XJnxXkVhgDZ6pgmUk2KiDOxNuT6iGHRFkKKXD+hdRFXobnsHq10HPy8jkfef60FeUwvr9bUTrzjVG+3BjdfZUp6bBScamjPaY2qmUSIU2wc86vGj+t96FL/maPzzBTrjFdjh7YKPZUlU5BP3xvb5ogdIBqNHHE3RXabyJAgMBAAE=",
			expected: []string{
				"\"p=MIIBITANBgkqhkiG9w0BAQEFAAOCAQ4AMIIBCQKCAQBcIjyCd+/fUqlYzi3ELmmWGJtEGiIpwi6qWXygfbarGkmHEYPsHTFBZj6Sy+r7wEA80cbcvWTcP41lETNR/k4NuaA88U6LHbDhBOhW4lVNWUQQxOA+EPq99blCaiDsAJU2MnEEA7iGryehPxRe5YU/I4MY1DJPOTerDGjJnlmp11n2Wd8XJnxXkVhgDZ6pgmUk2KiDOxNuT6iGHRFkK\"",
				"\"KXD+hdRFXobnsHq10HPy8jkfef60FeUwvr9bUTrzjVG+3BjdfZUp6bBScamjPaY2qmUSIU2wc86vGj+t96FL/maPzzBTrjFdjh7YKPZUlU5BP3xvb5ogdIBqNHHE3RXabyJAgMBAAE=\"",
			},
		},
		{
			title: "RSA4096",
			pub:   "p=MIICIjANBgkqhkiG9w0BAQEFAAOCAg8AMIICCgKCAgEAl69aPHwhXzuXRDMYG9Dt0dtrCrPjBiI8l9RQRouWAQLRh5KompSw8PLSyM0aLWwh7dQCILrLN4oz8SL8AWUdrs5D3KDfJuRFlAu3RyBwVgPPqPQwWhajy+3HWPd+EMlY2cifhf32xOvNmsBC/K7e7xGkBEA8ZTnQx4i4oAP62El1r1jyHZ52FyKxK+jP7KPgAIo8TEmHZK2jWxeGBR0V2h8ST1+XYlWoIzaaFyrNe04Yn+mbGp5DV0uF60o+dIdw59LRdTWJsYqfFbkRG9+9MiXKnzbaHDXE/tsdIwsipALtmxojw2PbqMaKON2PeQ2FeXZJJVOOm3UHqtsa9tD/u6bYpBwlmhKm/DRpthwXaeCGAOfyRH6NGpvH/gvvWf+XEGT68mLSiCKQQDvsXu3lxeVWylz7EML1jpOpaYqXWQ2zvERnH0UQocYgKCzbC3OAll8+3hYFhJsAlNYLCn6/gHJk24YnM/NwSFejgpYvqNQoRRAkP+OqUD0gBT+24RT698/cyaJI8enRApoVJlwsr536xysimuyAOGDR3X1XLFqUA0rDMI373lyEOKYasXRu7FtULC6Io51rhuIvS9ZRp7MbruKjAfL80vj8HSytVMu/vd5HhPXcwxIfd/fUa10zLkW6EGwL6FBOrLnM/dTnyru/oN11A3HGnE4EtXRIvpcCAwEAAQ==",
			expected: []string{
				"\"p=MIICIjANBgkqhkiG9w0BAQEFAAOCAg8AMIICCgKCAgEAl69aPHwhXzuXRDMYG9Dt0dtrCrPjBiI8l9RQRouWAQLRh5KompSw8PLSyM0aLWwh7dQCILrLN4oz8SL8AWUdrs5D3KDfJuRFlAu3RyBwVgPPqPQwWhajy+3HWPd+EMlY2cifhf32xOvNmsBC/K7e7xGkBEA8ZTnQx4i4oAP62El1r1jyHZ52FyKxK+jP7KPgAIo8TEmHZK2jWxeGB\"",
				"\"R0V2h8ST1+XYlWoIzaaFyrNe04Yn+mbGp5DV0uF60o+dIdw59LRdTWJsYqfFbkRG9+9MiXKnzbaHDXE/tsdIwsipALtmxojw2PbqMaKON2PeQ2FeXZJJVOOm3UHqtsa9tD/u6bYpBwlmhKm/DRpthwXaeCGAOfyRH6NGpvH/gvvWf+XEGT68mLSiCKQQDvsXu3lxeVWylz7EML1jpOpaYqXWQ2zvERnH0UQocYgKCzbC3OAll8+3hYFhJsAlNYL\"",
				"\"Cn6/gHJk24YnM/NwSFejgpYvqNQoRRAkP+OqUD0gBT+24RT698/cyaJI8enRApoVJlwsr536xysimuyAOGDR3X1XLFqUA0rDMI373lyEOKYasXRu7FtULC6Io51rhuIvS9ZRp7MbruKjAfL80vj8HSytVMu/vd5HhPXcwxIfd/fUa10zLkW6EGwL6FBOrLnM/dTnyru/oN11A3HGnE4EtXRIvpcCAwEAAQ==\"",
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.title, func(t *testing.T) {
			t.Parallel()
			actual := splitKey(tc.pub)
			assert.Equal(t, tc.expected, actual)
			assert.Equal(t, strings.ReplaceAll(strings.Join(actual, ""), "\"", ""), tc.pub)
		})
	}
}
