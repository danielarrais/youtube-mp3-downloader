//go:build !windows

package main

import "os"

func replaceQueueFile(source, destination string) error {
	return os.Rename(source, destination)
}
