package sftpsvc

import "errors"

var (
	ErrTargetExists        = errors.New("remote target path already exists")
	ErrInvalidRenameName   = errors.New("invalid remote rename name")
	ErrInvalidRenamePath   = errors.New("invalid remote rename path")
	ErrProtectedRenamePath = errors.New("refuse to rename protected remote path")
)
