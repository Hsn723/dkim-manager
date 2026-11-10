package dkim

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestKeyProvider(t *testing.T) {
	t.Parallel()
	cases := []struct {
		title     string
		keyType   KeyType
		keyLength KeyLength
		errFunc   assert.ErrorAssertionFunc
	}{
		{
			title:     "RSA2048",
			keyType:   KeyTypeRSA,
			keyLength: KeyLength2048,
			errFunc:   assert.NoError,
		},
		{
			title:     "RSA4096",
			keyType:   KeyTypeRSA,
			keyLength: KeyLength4096,
			errFunc:   assert.NoError,
		},
		{
			title:   "ED25519",
			keyType: KeyTypeED25519,
			errFunc: assert.NoError,
		},
		{
			title:   "MLDSA44",
			keyType: KeyTypeMLDSA44,
			errFunc: assert.NoError,
		},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.title, func(t *testing.T) {
			t.Parallel()
			kp, err := NewKeyProvider(tc.keyType, tc.keyLength)
			assert.NoError(t, err)
			priv, err := kp.GenKey()
			tc.errFunc(t, err)
			if err != nil {
				t.Skip()
			}
			pub, err := kp.PublicKey(priv)
			assert.NoError(t, err)
			assert.NotEmpty(t, pub)
			name := kp.Name()
			assert.Equal(t, tc.keyType, name)
			txt := GenTXTValue(kp, pub)
			expectContents := []string{
				"v=DKIM1",
				"k=" + string(tc.keyType),
				"p=",
			}
			for _, content := range expectContents {
				assert.Contains(t, txt, content)
			}
		})
	}
}

func TestNewKeyProviderInvalidType(t *testing.T) {
	t.Parallel()
	_, err := NewKeyProvider("invalid", KeyLength2048)
	assert.Error(t, err)
}

func TestPublicKeyMismatch(t *testing.T) {
	t.Parallel()
	cases := []struct {
		title     string
		source    KeyType
		target    KeyType
		sourceLen KeyLength
		targetLen KeyLength
	}{
		{
			title:  "MLDSA44 with ED25519",
			source: KeyTypeMLDSA44,
			target: KeyTypeED25519,
		},
		{
			title:  "ED25519 with MLDSA44",
			source: KeyTypeED25519,
			target: KeyTypeMLDSA44,
		},
		{
			title:     "MLDSA44 with RSA",
			source:    KeyTypeMLDSA44,
			target:    KeyTypeRSA,
			targetLen: KeyLength2048,
		},
		{
			title:     "RSA with mismatched length",
			source:    KeyTypeRSA,
			sourceLen: KeyLength2048,
			target:    KeyTypeRSA,
			targetLen: KeyLength4096,
		},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.title, func(t *testing.T) {
			t.Parallel()
			source, err := NewKeyProvider(tc.source, tc.sourceLen)
			assert.NoError(t, err)
			priv, err := source.GenKey()
			assert.NoError(t, err)
			target, err := NewKeyProvider(tc.target, tc.targetLen)
			assert.NoError(t, err)
			res, err := target.PublicKey(priv)
			assert.Empty(t, res)
			assert.Error(t, err)
		})
	}
}

func TestPublicKeyMalformed(t *testing.T) {
	t.Parallel()
	tc := []KeyType{KeyTypeRSA, KeyTypeED25519, KeyTypeMLDSA44}
	for _, keyType := range tc {
		keyType := keyType
		t.Run(string(keyType), func(t *testing.T) {
			t.Parallel()
			kp, err := NewKeyProvider(keyType, KeyLength2048)
			assert.NoError(t, err)
			res, err := kp.PublicKey([]byte("malformed"))
			assert.Empty(t, res)
			assert.Error(t, err)
		})
	}
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
