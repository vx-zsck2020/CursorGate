//go:build !windows

package update

import "fmt"

func Apply(Status) error {
	return fmt.Errorf("当前平台不支持自动替换")
}
