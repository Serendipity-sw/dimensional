package main

import "os"

// +build windows
const recordPath = "./logs/"
const pidFile = "./dimensional.pid"

func isProcessExist(pid int) bool {
	// 进程是否存在
	_, err := os.FindProcess(pid)
	if err == nil {
		return true
	}
	return false
}
