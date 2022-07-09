package main

import (
	"os"
	"os/exec"
	"strings"
)

const (
	readme = "README.md"
	begin  = "<!-- RESULT:BEGIN -->"
	end    = "<!-- RESULT:END -->"
	sep    = "\n```\n"
)

func main() {
	stat, err := os.Stat(readme)
	if err != nil {
		panic(err)
	}

	file, err := os.ReadFile(readme)
	if err != nil {
		panic(err)
	}

	x, y, found := strings.Cut(string(file), begin)
	if !found {
		panic("not found")
	}

	_, z, found := strings.Cut(y, end)
	if !found {
		panic("not found")
	}

	out, err := exec.Command("go", "test").CombinedOutput()
	if err != nil {
		panic(err)
	}

	err = os.WriteFile(readme, []byte(x+begin+sep+string(out)+sep+end+z), stat.Mode())
	if err != nil {
		panic(err)
	}
}
