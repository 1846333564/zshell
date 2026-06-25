package sftpsvc

import (
	"fmt"
	"strings"

	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

func createRemoteUploadDirs(sshClient *ssh.Client, sftpClient *sftp.Client, dirs []string) error {
	if len(dirs) == 0 {
		return nil
	}
	if err := createRemoteUploadDirsWithShell(sshClient, dirs); err == nil {
		return nil
	}

	for _, dir := range dirs {
		if err := sftpClient.MkdirAll(dir); err != nil {
			return fmt.Errorf("create remote directory %s: %w", dir, err)
		}
	}
	return nil
}

func createRemoteUploadDirsWithShell(client *ssh.Client, dirs []string) error {
	args := make([]string, 0, uploadMkdirCommandMaxArgs)
	commandLen := len("mkdir -p -- ")
	flush := func() error {
		if len(args) == 0 {
			return nil
		}
		command := "mkdir -p -- " + strings.Join(args, " ")
		args = args[:0]
		commandLen = len("mkdir -p -- ")
		return runRemoteShellCommand(client, command)
	}

	for _, dir := range dirs {
		arg := shellQuote(dir)
		nextLen := commandLen + len(arg) + 1
		if len(args) > 0 && (len(args) >= uploadMkdirCommandMaxArgs || nextLen > uploadMkdirCommandMaxLen) {
			if err := flush(); err != nil {
				return err
			}
			nextLen = commandLen + len(arg) + 1
		}
		args = append(args, arg)
		commandLen = nextLen
	}

	return flush()
}
