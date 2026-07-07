package httpapi

import (
	"io"
	"mime/multipart"
	"path"
	"strings"

	"wiShell/backend/internal/sftpsvc"
)

func multipartUploadItems(form *multipart.Form) ([]sftpsvc.UploadItem, []string) {
	if form == nil {
		return nil, nil
	}

	fileHeaders := make([]*multipart.FileHeader, 0)
	fileHeaders = append(fileHeaders, form.File["files"]...)
	fileHeaders = append(fileHeaders, form.File["file"]...)

	relativePaths := form.Value["relativePaths"]
	files := make([]sftpsvc.UploadItem, 0, len(fileHeaders))
	for index, header := range fileHeaders {
		header := header
		relativePath := ""
		if index < len(relativePaths) {
			relativePath = relativePaths[index]
		}
		files = append(files, sftpsvc.UploadItem{
			FileName:     header.Filename,
			RelativePath: relativePath,
			Size:         header.Size,
			Open: func() (io.ReadCloser, error) {
				return header.Open()
			},
		})
	}

	return files, form.Value["directories"]
}

func archiveName(remotePaths []string) string {
	if len(remotePaths) == 1 {
		base := path.Base(strings.TrimSpace(remotePaths[0]))
		if base != "." && base != "/" && base != "" {
			return base + ".zip"
		}
	}
	return "wiShell-download.zip"
}
