//go:build windows

package configstore

import (
	"unsafe"

	"golang.org/x/sys/windows"
)

const cryptProtectUiForbidden = 0x1

func encrypt(data []byte) ([]byte, error) {
	if len(data) == 0 {
		data = []byte("{}")
	}

	in := windows.DataBlob{
		Size: uint32(len(data)),
		Data: &data[0],
	}
	var out windows.DataBlob
	if err := windows.CryptProtectData(&in, nil, nil, 0, nil, cryptProtectUiForbidden, &out); err != nil {
		return nil, err
	}
	defer windows.LocalFree(windows.Handle(unsafe.Pointer(out.Data)))

	return dataFromBlob(out), nil
}

func decrypt(data []byte) ([]byte, error) {
	if len(data) == 0 {
		return nil, nil
	}

	in := windows.DataBlob{
		Size: uint32(len(data)),
		Data: &data[0],
	}
	var out windows.DataBlob
	if err := windows.CryptUnprotectData(&in, nil, nil, 0, nil, cryptProtectUiForbidden, &out); err != nil {
		return nil, err
	}
	defer windows.LocalFree(windows.Handle(unsafe.Pointer(out.Data)))

	return dataFromBlob(out), nil
}

func dataFromBlob(blob windows.DataBlob) []byte {
	if blob.Size == 0 || blob.Data == nil {
		return nil
	}
	source := unsafe.Slice(blob.Data, blob.Size)
	result := make([]byte, len(source))
	copy(result, source)
	return result
}
