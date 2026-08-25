//go:build !darwin

package app

import "fmt"

func isolationProfile(_ isolationRequest) (string, error) {
	return "", fmt.Errorf("OS-enforced agent read isolation is not available on this platform")
}

func isolatedCommand(_, _ string, _ []string) (string, []string, error) {
	return "", nil, fmt.Errorf("OS-enforced agent read isolation is not available on this platform")
}
