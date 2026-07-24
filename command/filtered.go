package command

import (
	"strings"

	"github.com/luchrv/lazyncu/detect"
	"github.com/luchrv/lazyncu/scanner"
)

// ProjectUpdateFiltered builds `cd <dir> && ncu -u <pkg…> && <install>` using
// ncu's positional package filters, so only the given packages are upgraded.
// No packages yields an empty string.
func ProjectUpdateFiltered(dir string, pm detect.PackageManager, pkgs []string) string {
	if len(pkgs) == 0 {
		return ""
	}
	install := installStep(pm)
	return "cd " + dir + " && ncu -u " + strings.Join(pkgs, " ") + " && " + install
}

// GlobalUpdateFiltered builds `npm install -g pkg@ver ...` restricted to the
// marked packages, preserving scan order. Marks that no longer match a
// scanned package are ignored; an empty result yields an empty string.
func GlobalUpdateFiltered(pkgs []scanner.Package, marked map[string]bool) string {
	subset := make([]scanner.Package, 0, len(pkgs))
	for _, p := range pkgs {
		if marked[p.Name] {
			subset = append(subset, p)
		}
	}
	return GlobalUpdate(subset)
}

func installStep(pm detect.PackageManager) string {
	install := map[detect.PackageManager]string{
		detect.Npm:  "npm install",
		detect.Pnpm: "pnpm install",
		detect.Yarn: "yarn",
	}[pm]
	if install == "" {
		install = "npm install"
	}
	return install
}
