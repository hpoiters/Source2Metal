package main

import (
	"io/fs"
	"os"
)

func osStat(name string) (fs.FileInfo, error) { return os.Stat(name) }
