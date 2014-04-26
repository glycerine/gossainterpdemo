package main

// idemo.go : demonstrate calling the ssa interpreter

import (
	"flag"
	"fmt"
	"go/build"
	"os"

	"code.google.com/p/go.tools/go/loader"
	"code.google.com/p/go.tools/go/ssa"
	"code.google.com/p/go.tools/go/ssa/interp"
	"code.google.com/p/go.tools/go/types"
)

func main() {
	interpDemo()
}

func interpDemo() error {

	const hello = `
package main

import (
"fmt"
"strings"
)
const message = "Hello, World!"

func main() {
   fmt.Printf("%s\n",strings.ToLower(message))
}
`

	flag.Parse()
	args := flag.Args()

	//var conf loader.Config

	conf := loader.Config{
		Build:         &build.Default,
		SourceImports: true,
	}

	var wordSize int64 = 8
	switch conf.Build.GOARCH {
	case "386", "arm":
		wordSize = 4
	}

	wordSize = 8

	conf.TypeChecker.Sizes = &types.StdSizes{
		MaxAlign: 8,
		WordSize: wordSize,
	}

	// Parse the input file.
	file, err := conf.ParseFile("hello.go", hello)
	if err != nil {
		fmt.Print(err) // parse error
		return err
	}

	// Create single-file main package.
	conf.CreateFromFiles("main", file)

	conf.Import("runtime")

	// Load the main package and its dependencies.
	iprog, err := conf.Load()
	if err != nil {
		fmt.Print(err) // type error in some package
		return err
	}

	// Create SSA-form program representation.
	prog := ssa.Create(iprog, ssa.SanityCheckFunctions)
	prog.BuildAll() // creates init() function in fmt?
	mainPkg := prog.Package(iprog.Created[0].Pkg)

	// Print out the package.
	mainPkg.WriteTo(os.Stdout)

	// Build SSA code for bodies of functions in mainPkg.
	mainPkg.Build()

	// Print out the package-level functions.
	mainPkg.Func("init").WriteTo(os.Stdout)
	mainPkg.Func("main").WriteTo(os.Stdout)

	// Run the interpreter.

	var main *ssa.Package
	pkgs := prog.AllPackages()

	// Otherwise, run main.main.
	for _, pkg := range pkgs {
		if pkg.Object.Name() == "main" {
			main = pkg
			if main.Func("main") == nil {
				return fmt.Errorf("no func main() in main package")
			}
			break
		}
	}
	if main == nil {
		return fmt.Errorf("no main package")
	}

	var interpMode interp.Mode

	// either of these will work now:
	//interp.Interpret(main, interpMode, conf.TypeChecker.Sizes, main.Object.Path(), args)
	interp.Interpret(mainPkg, interpMode, conf.TypeChecker.Sizes, mainPkg.Object.Path(), args)

	return nil
}
