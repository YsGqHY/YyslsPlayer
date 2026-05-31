//go:build production

package player

func resolveDryRun(bool) bool {
	return false
}
