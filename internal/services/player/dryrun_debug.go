//go:build !production

package player

func resolveDryRun(value bool) bool {
	return value
}
