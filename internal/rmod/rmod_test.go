package rmod_test

import (
	"os"
	"testing"

	"github.com/emilekm/demos-hub/internal/rmod"
	"github.com/stretchr/testify/require"
)

var (
	licenseContent []byte
)

func init() {
	licenseFile, ok := os.LookupEnv("PRBF2_LICENSE")
	if !ok {
		return
	}

	var err error
	licenseContent, err = os.ReadFile(licenseFile)
	if err != nil {
		panic(err)
	}
}

func TestValidateLicense(t *testing.T) {
	if len(licenseContent) == 0 {
		t.Skip("no license file provided")
	}
	ok, err := rmod.ValidateLicense(
		"45.63.41.122",
		"16567",
		string(licenseContent),
	)

	require.NoError(t, err)
	require.True(t, ok)
}
