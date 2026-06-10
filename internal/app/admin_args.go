//go:build windows

package app

import (
	"strings"
	"syscall"
)

const relaunchFlag = "--yyslsplayer-elevated"

func elevatedParamsFromArgs(argv []string) string {
	args := make([]string, 0, len(argv))
	args = append(args, relaunchFlag)
	for _, arg := range argv[1:] {
		if arg == relaunchFlag {
			continue
		}
		args = append(args, syscall.EscapeArg(arg))
	}
	return strings.Join(args, " ")
}

func hasArgIn(argv []string, target string) bool {
	for _, arg := range argv[1:] {
		if arg == target {
			return true
		}
	}
	return false
}

func stripArg(argv []string, target string) []string {
	if len(argv) == 0 {
		return argv
	}
	write := argv[:1]
	for _, arg := range argv[1:] {
		if arg != target {
			write = append(write, arg)
		}
	}
	return write
}
