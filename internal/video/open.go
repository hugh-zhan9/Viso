package video

import (
	"os/exec"
	"runtime"
)

// OpenFile 使用系统默认程序打开文件
func OpenFile(path string) error {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("cmd", "/c", "start", "", path)
	case "darwin":
		cmd = exec.Command("open", path)
	default: // linux and others
		cmd = exec.Command("xdg-open", path)
	}

	return cmd.Start()
}

// OpenEditor 使用系统默认文本编辑器打开文件
func OpenEditor(path string) error {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("cmd", "/c", "start", "", path)
	case "darwin":
		cmd = exec.Command("open", "-e", path) // -e 使用 TextEdit
	default:
		cmd = exec.Command("xdg-open", path)
	}

	return cmd.Start()
}
