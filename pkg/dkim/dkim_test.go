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
		tc := tc
		t.Run(tc.title, func(t *testing.T) {
			t.Parallel()
			actual := splitKey(tc.pub)
			assert.Equal(t, tc.expected, actual)
			assert.Equal(t, strings.ReplaceAll(strings.Join(actual, ""), "\"", ""), tc.pub)
		})
	}
}
