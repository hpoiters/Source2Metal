//go:build !windows

package main

import "errors"

var errFolderSelectionCancelled = errors.New("folder selection cancelled")

func browseForFolder(title string) (string, error) {
	return "", errFolderSelectionCancelled
}

func isFolderSelectionCancelled(err error) bool {
	return errors.Is(err, errFolderSelectionCancelled)
}
