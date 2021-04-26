package e2e_testing

import (
	"flag"
	"os"
	"strings"
	"testing"

	"github.com/alexflint/go-arg"
)

type LocalCmd struct {
	Image string `arg:"required" help:"docker container image to deploy and test"`
}

var args struct {
	Local       *LocalCmd `arg:"subcommand:local"`
	GoTestFlags string    `help:"go test flags to pass through, e.g. --gotestflags='-test.v'"`
}

func TestMain(m *testing.M) {
	p := arg.MustParse(&args)
	if p.Subcommand() == nil {
		p.Fail("missing command")
	}

	// hacky but works
	os.Args = append([]string{os.Args[0]}, strings.Fields(args.GoTestFlags)...)
	flag.Parse()

	m.Run()
}
