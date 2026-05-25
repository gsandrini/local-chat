//go:build !windows

package main

import "os/exec"

func setSysProcAttr(cmd *exec.Cmd) {
    // noop su Linux/macOS
}