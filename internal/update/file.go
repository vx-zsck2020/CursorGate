package update

import "os"

func createFile(path string) (*os.File, error) {
	return os.OpenFile(path, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0644)
}

func removeFile(path string) error {
	return os.Remove(path)
}

func replaceFile(src, dst string) error {
	_ = os.Remove(dst)
	return os.Rename(src, dst)
}
