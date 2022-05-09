package main

import (
	"flag"
	"fmt"
	"go/build"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"golang.org/x/tools/go/buildutil"
)

var ignoreDirs []string

func main() {
	d := flag.String("ignore-dirs", ".checkout_git", "Dir patterns (static string) to ignore, comma-separated")
	flag.Parse()
	if len(*d) > 0 {
		ignoreDirs = strings.Split(*d, ",")
	}

	args := flag.Args()
	if len(args) != 1 {
		die("Usage: %s commit..commit\n", os.Args[0])
	}

	commitRange := args[0]
	files := changedFiles(commitRange)

	cwd, err := os.Getwd()
	if err != nil {
		die("Could not get CWD: %s", err)
	}

	repo := gitRoot()

	pkgSeen := make(map[string]bool)
	var modifiedPackages []*build.Package
	buildContext := build.Default
	for _, f := range files {
		if isIgnored(f) {
			continue
		}

		pkg, err := buildutil.ContainingPackage(&buildContext, cwd, f)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error finding package for file %s: %s\n", f, err)
			continue
		}
		if pkg.Goroot {
			fmt.Fprintf(os.Stderr, "Ignoring STDLIB file %s\n", f)
			continue
		}

		if pkgSeen[pkg.ImportPath] {
			continue
		}

		pkgSeen[pkg.ImportPath] = true
		modifiedPackages = append(modifiedPackages, pkg)
	}

	// TODO: list all packages in the repo, determine dependency tree and filter package list to those that transitively import affected packages
	// build the import tree
	var (
		imports = make(map[string][]string)
		scanErr error
	)
	buildutil.ForEachPackage(&buildContext, func(importPath string, err error) {
		if err != nil {
			fmt.Fprintf(os.Stderr, "Could not read package %s: %s\n", importPath, err)
			scanErr = err
			return
		}

		if isIgnored(importPath) {
			return
		}

		pkg, err := buildContext.Import(importPath, repo, build.AllowBinary)
		if err, ok := err.(*build.NoGoError); err != nil && ok {
			return
		}
		if err != nil {
			fmt.Fprintf(os.Stderr, "Could not import dir %s: %s\n", importPath, err)
			scanErr = err
			return
		}

		for _, imp := range pkg.Imports {
			imports[imp] = append(imports[imp], importPath)
		}

	})
	if scanErr != nil {
		die("Package scan incomplete, aborting")
	}

	// filter the package list to those affected
	var affectedPackages []string
	for _, p := range modifiedPackages {
		affectedPackages = append(affectedPackages, p.ImportPath)
	}
	addedMore := true
	for addedMore {
		addedMore = false
		var newAdditions []string
		for _, p := range affectedPackages {
			for _, imp := range imports[p] {
				if isIgnored(imp) {
					continue
				}
				var found bool
				for _, p := range affectedPackages {
					if imp == p {
						found = true
						break
					}
				}
				for _, p := range newAdditions {
					if imp == p {
						found = true
						break
					}
				}
				if found {
					continue
				}

				newAdditions = append(newAdditions, imp)
			}
		}

		addedMore = len(newAdditions) > 0
		affectedPackages = append(affectedPackages, newAdditions...)
	}

	fmt.Println(strings.Join(affectedPackages, "\n"))
}

func die(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, format, args...)
	os.Exit(1)
}

func isIgnored(f string) bool {
	for _, d := range ignoreDirs {
		if strings.Contains(f, d) {
			return true
		}
	}
	return false
}

func gitRoot() string {
	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	dat, err := cmd.Output()
	if err != nil {
		die("Could not find git root: %s", err)
	}
	return strings.TrimSpace(string(dat))
}

func changedFiles(commitRange string) []string {
	root := gitRoot()

	cmd := exec.Command("git", "diff", "--name-only", commitRange)
	dat, err := cmd.Output()
	if err != nil {
		die("Could not run git diff-tree: %v", err)
	}
	files := strings.Split(string(dat), "\n")
	var res []string
	for _, f := range files {
		f = strings.TrimSpace(f)
		if len(f) == 0 {
			continue
		}
		res = append(res, filepath.Join(root, f))
	}
	return res
}
