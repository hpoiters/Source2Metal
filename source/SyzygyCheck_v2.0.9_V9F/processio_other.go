//go:build !windows

package main

func processReadBytes(pid int) (uint64, bool) { return 0, false }
