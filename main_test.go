package e2e_testing

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/alexflint/go-arg"
)

type Cleanup func()

type LocalCmd struct {
	Image string `arg:"required" help:"docker container image to deploy and test"`
}

var args struct {
	Local *LocalCmd `arg:"subcommand:local"`

	GoTestFlags string `help:"go test flags to pass through, e.g. --gotestflags='-test.v'"`
	ProjectID   string `arg:"required,--project-id,env:PROJECT_ID" help:"GCP project id/name"`
}

func TestMain(m *testing.M) {
	p := arg.MustParse(&args)
	if p.Subcommand() == nil {
		p.Fail("missing command")
	}

	// hacky but works
	os.Args = append([]string{os.Args[0]}, strings.Fields(args.GoTestFlags)...)
	flag.Parse()

	var err error
	var cleanup Cleanup
	switch {
	case args.Local != nil:
		cleanup, err = setupLocal(args.Local)
	}

	defer cleanup()
	if err != nil {
		fmt.Printf("%v\n", err)
		panic(err)
	}

	time.Sleep(time.Second * 5)

	// Run tests
	m.Run()
}

/**
 * Set up the instrumented test server for a local run
 */
func setupLocal(local *LocalCmd) (Cleanup, error) {
	runCmd := exec.Command("docker", "run", "--rm", local.Image)
	runCmd.Stdout = os.Stdout
	runCmd.Stderr = os.Stderr
	err := runCmd.Start()

	cleanup := func() {
		fmt.Printf("Cleanup called, killing pid %v\n", runCmd.Process.Pid)
		// For some reason this isn't workign :/
		err = runCmd.Process.Kill()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Could not kill docker process, %v\n", err)
		}
	}

	return cleanup, err
}
