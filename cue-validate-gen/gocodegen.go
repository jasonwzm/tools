package main

import (
	"log"
	"os"
	"path/filepath"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/load"
	"cuelang.org/go/encoding/gocode"
)

func main() {

	cwd, _ := os.Getwd()

	cfg := &load.Config{
		Dir:        cwd,
		Module:     "istio.io/jasonwzm/api",
		ModuleRoot: cwd,
	}

	instances := load.Instances([]string{"./" + "networking/v1alpha3"}, cfg)
	inst := cue.Build(instances)[0]
	if inst.Err != nil {
		log.Fatalf("Instance failed: %v", inst.Err)
	}

	b, err := gocode.Generate("", inst, nil)
	if err != nil {
		log.Fatalf("Error generating gocode: %v", err)
	}

	gofile := filepath.Join(cwd, "networking/v1alpha3", "generated_cue.go")

	f, err := os.Create(gofile)
	if err != nil {
		log.Fatalf("error creating file: %v", err)
	}
	defer f.Close()

	if _, err = f.Write(b); err != nil {
		log.Fatalf("error writing to file: %v", err)
	}

}
