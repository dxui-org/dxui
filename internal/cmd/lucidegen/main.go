// Command lucidegen converts the pinned Lucide SVG release into immutable Go data.
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
)

func main() {
	source := flag.String("source", filepath.FromSlash("third_party/lucide/lucide-static-"+lucideVersion+".tgz"), "pinned lucide-static npm archive")
	output := flag.String("out", "icon", "generated icon package directory")
	catalog := flag.String("catalog", filepath.FromSlash("icon/catalog"), "generated full catalog package directory")
	fixtures := flag.String("fixtures", ".", "directory for generated renderer reference fixtures")
	flag.Parse()
	icons, license, err := loadSource(*source)
	if err == nil {
		err = generate(*output, *catalog, *fixtures, icons, license)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "lucidegen:", err)
		os.Exit(1)
	}
}
