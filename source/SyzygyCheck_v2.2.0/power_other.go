//go:build !windows

package main

func preventSystemSleep() func() { return func() {} }
