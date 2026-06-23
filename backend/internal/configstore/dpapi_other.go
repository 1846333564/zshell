//go:build !windows

package configstore

import "fmt"

func encrypt(data []byte) ([]byte, error) {
	return nil, fmt.Errorf("Windows DPAPI encryption is only available on Windows")
}

func decrypt(data []byte) ([]byte, error) {
	return nil, fmt.Errorf("Windows DPAPI encryption is only available on Windows")
}
