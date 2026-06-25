package sftpsvc

import (
	"io"
)

type Entry struct {
	Name    string `json:"name"`
	Path    string `json:"path"`
	Size    int64  `json:"size"`
	IsDir   bool   `json:"isDir"`
	Mode    string `json:"mode"`
	Owner   string `json:"owner"`
	ModTime string `json:"modTime"`
}

const MaxTextEditBytes = 64 * 1024 * 1024

type TextFile struct {
	Name    string `json:"name"`
	Path    string `json:"path"`
	Size    int64  `json:"size"`
	Content string `json:"content"`
	ModTime string `json:"modTime"`
}

type TextReadProgressEvent struct {
	Stage       string `json:"stage"`
	Path        string `json:"path,omitempty"`
	FileName    string `json:"fileName,omitempty"`
	LoadedBytes int64  `json:"loadedBytes"`
	TotalBytes  int64  `json:"totalBytes"`
	Message     string `json:"message,omitempty"`
}

type TextReadProgressReporter func(TextReadProgressEvent)

type UploadItem struct {
	FileName     string
	RelativePath string
	Size         int64
	Open         func() (io.ReadCloser, error)
}

type UploadResult struct {
	RemotePath string `json:"remotePath"`
	Size       int64  `json:"size"`
}

type UploadBatchResult struct {
	OK          bool           `json:"ok"`
	Files       []UploadResult `json:"files"`
	Directories []string       `json:"directories"`
	TotalSize   int64          `json:"totalSize"`
}

type UploadProgressEvent struct {
	Stage          string `json:"stage"`
	FileIndex      int    `json:"fileIndex"`
	FileName       string `json:"fileName,omitempty"`
	RelativePath   string `json:"relativePath,omitempty"`
	RemotePath     string `json:"remotePath,omitempty"`
	FileLoaded     int64  `json:"fileLoaded"`
	FileTotal      int64  `json:"fileTotal"`
	LoadedBytes    int64  `json:"loadedBytes"`
	TotalBytes     int64  `json:"totalBytes"`
	CompletedFiles int    `json:"completedFiles"`
	TotalFiles     int    `json:"totalFiles"`
	DirectoryCount int    `json:"directoryCount"`
	Message        string `json:"message,omitempty"`
}

type UploadProgressReporter func(UploadProgressEvent)

type TransferItem struct {
	Path  string `json:"path"`
	IsDir bool   `json:"isDir"`
}

type TransferResult struct {
	RemotePath string `json:"remotePath"`
	IsDir      bool   `json:"isDir"`
	Size       int64  `json:"size"`
}

type TransferBatchResult struct {
	OK          bool             `json:"ok"`
	Action      string           `json:"action"`
	Files       []TransferResult `json:"files"`
	Directories []string         `json:"directories"`
	TotalSize   int64            `json:"totalSize"`
}

type DeleteResult struct {
	RemotePath string `json:"remotePath"`
	IsDir      bool   `json:"isDir"`
	Size       int64  `json:"size"`
}

type DeleteBatchResult struct {
	OK        bool           `json:"ok"`
	Items     []DeleteResult `json:"items"`
	TotalSize int64          `json:"totalSize"`
}
