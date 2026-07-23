package main

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
)

func secureDownloadPath(downloadDir, filePath string) (string, error) {
	root, err := filepath.EvalSymlinks(downloadDir)
	if err != nil {
		return "", err
	}
	path, err := filepath.EvalSymlinks(filePath)
	if err != nil {
		return "", err
	}
	relative, err := filepath.Rel(root, path)
	if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return "", errors.New("file is outside download directory")
	}
	info, err := os.Stat(path)
	if err != nil || !info.Mode().IsRegular() {
		return "", errors.New("file is not regular")
	}
	return path, nil
}

func removeDownloadFile(downloadDir, filePath string) error {
	if filePath == "" {
		return nil
	}
	if _, err := os.Lstat(filePath); errors.Is(err, os.ErrNotExist) {
		return nil
	} else if err != nil {
		return err
	}
	path, err := secureDownloadPath(downloadDir, filePath)
	if err != nil {
		return err
	}
	return os.Remove(path)
}
