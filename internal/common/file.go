package common

import (
	"errors"
	"os"
	"path/filepath"
)

// GetAbsPath analyzes the given path string and returns:
// 0 if the string is empty
// 1 if the string is a simple file name or directory name (no path separators)
// 2 if the string is an absolute path
// 3 if the string is a relative path (contains path separators but is not absolute)
func GetAbsPath(path string) (string, error) {
	if path == "" {
		return "", errors.New("path cannot be empty")
	}

	// 1. 判断是否为单纯的文件名/目录名
	// filepath.Base 会返回路径的最后一个元素
	// 如果返回的结果和原字符串完全相同，说明它不包含任何层级目录分隔符
	if filepath.Base(path) == path && path != "." && path != ".." {
		dir, err := os.Getwd()
		if err != nil {
			return path, errors.New("failed to get working directory")
		}
		configPath := filepath.Join(dir, path)
		return configPath, nil
	}

	// 2. 判断是否为绝对路径
	if filepath.IsAbs(path) {
		return path, nil
	}

	// 3. 排除绝对路径和纯文件名后，就是相对路径
	return path, errors.New("relative paths are not allowed")
}
