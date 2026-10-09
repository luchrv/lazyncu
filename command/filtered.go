package command

import "github.com/luchrv/lazyncu/scanner"

// GlobalUpdateFiltered builds `npm install -g pkg@ver ...` restricted to the
// marked packages, preserving scan order. Marks that no longer match a scanned
// package are ignored; an empty result yields an empty string.
func GlobalUpdateFiltered(pkgs []scanner.Package, marked map[string]bool) string {
	subset := make([]scanner.Package, 0, len(pkgs))
	for _, p := range pkgs {
		if marked[p.Name] {
			subset = append(subset, p)
		}
	}
	return GlobalUpdate(subset)
}
