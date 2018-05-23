package runtime

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"regexp"
	"strconv"
	"syscall"
	"time"

	"github.com/pkg/errors"

	"github.com/Southclaws/sampctl/print"
	"github.com/Southclaws/sampctl/types"
)

var (
	matchPreamble = regexp.MustCompile(`Loaded [0-9]{1,2} filterscripts\.`)
	matchMainEnd  = regexp.MustCompile(`Number of vehicle models\: [0-9]*`)
	matchTestEnd  = regexp.MustCompile(`\*\*\* Tests: (\d+), Fails: (\d+)`)
)

type testResults struct {
	Tests int
	Fails int
}

// Run handles the actual running of the server process - it collects log output too
func Run(ctx context.Context, cfg types.Runtime, cacheDir string) (err error) {
	if cfg.Container != nil {
		return RunContainer(cfg, cacheDir)
	}

	binary := "./" + getServerBinary(cfg.Platform)
	fullPath := filepath.Join(cfg.WorkingDir, binary)
	print.Verb("starting", binary, "in", cfg.WorkingDir)

	return run(ctx, fullPath, cfg.Mode)
}

func run(ctx context.Context, binary string, runType types.RunMode) (err error) {
	outputReader, outputWriter := io.Pipe()
	cmd := exec.CommandContext(ctx, binary)
	cmd.Dir = filepath.Dir(binary)
	cmd.Stdout = outputWriter
	cmd.Stderr = outputWriter
	errChan := make(chan error)        // channel for sending runtime errors to watchdog
	endChan := make(chan struct{})     // channel for internal end signals
	sigChan := make(chan os.Signal, 1) // channel for capturing host signals

	defer func() {
		err = outputWriter.Close()
		if err != nil {
			print.Erro("Compiler output read error:", err)
		}
	}()

	switch runType {
	case types.MainOnly:
		go func() {
			preamble := true
			preambleSpace := false
			scanner := bufio.NewScanner(outputReader)
			for scanner.Scan() {
				line := scanner.Text()

				if matchPreamble.MatchString(line) {
					preamble = false
					preambleSpace = true
					continue
				}
				if preambleSpace {
					preambleSpace = false
					continue
				}

				if matchMainEnd.MatchString(line) {
					endChan <- struct{}{} // end the server process
					break
				}

				if !preamble {
					fmt.Println(line)
				}
			}
		}()
	case types.YTesting:
		go func() {
			preamble := true
			preambleSpace := false
			scanner := bufio.NewScanner(outputReader)
			for scanner.Scan() {
				line := scanner.Text()

				if matchPreamble.MatchString(line) {
					preamble = false
					preambleSpace = true
					continue
				}
				if preambleSpace {
					preambleSpace = false
					continue
				}

				if matchTestEnd.MatchString(line) {
					testResults := testResultsFromLine(line)
					if testResults.Fails > 0 {
						print.Erro(testResults.Tests, "tests, with:", testResults.Fails, "failures.")
						errChan <- errors.New("tests failed")
					} else {
						print.Info(testResults.Tests, "tests passed!")
						endChan <- struct{}{} // end the server process
					}

					break
				}

				if !preamble {
					fmt.Println(line)
				}
			}
		}()
	default:
		go func() {
			scanner := bufio.NewScanner(outputReader)
			for scanner.Scan() {
				line := scanner.Text()
				fmt.Println(line)
			}
		}()
	}

	print.Verb("running with mode", runType)

	go func() {
		var (
			startTime          time.Time     // time of most recent start/restart
			exponentialBackoff = time.Second // exponential backoff cooldown
		)
		for {
			err = cmd.Start()
			if err != nil {
				errChan <- err
				break
			}
			startTime = time.Now()
			cmdError := cmd.Wait()

			if cmdError.Error() == "exit status 1" {
				break
			}

			runTime := time.Since(startTime)
			if runTime < time.Minute {
				exponentialBackoff *= 2
			} else {
				exponentialBackoff = time.Second
			}

			if exponentialBackoff > time.Second*15 {
				errChan <- errors.Errorf("too many crashloops, last error: %v", cmdError)
				break
			}

			print.Warn("crash loop backoff for", exponentialBackoff, "reason:", cmdError)
			time.Sleep(exponentialBackoff)
		}
	}()

	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	select {
	case s := <-sigChan:
		err = errors.Errorf("received signal: %v", s)
	case err = <-errChan:
		err = errors.Wrap(err, "received runtime error")
	case <-endChan:
		err = nil
	}

	if cmd.Process != nil {
		killErr := cmd.Process.Kill()
		if killErr != nil {
			print.Erro(killErr)
		}
	}

	return
}

func testResultsFromLine(line string) (results testResults) {
	match := matchTestEnd.FindStringSubmatch(line)
	results.Tests, _ = strconv.Atoi(match[1])
	results.Fails, _ = strconv.Atoi(match[2])
	return
}
