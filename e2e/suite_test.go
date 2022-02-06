package e2e

import (
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestE2E(t *testing.T) {
	if !runE2E {
		t.Skip("RUN_E2E is not set")
	}

	RegisterFailHandler(Fail)
	SetDefaultEventuallyTimeout(5 * time.Second)
	SetDefaultEventuallyPollingInterval(100 * time.Millisecond)
	RunSpecs(t, "E2E Suite")
}
