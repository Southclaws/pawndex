package main

import (
	"fmt"
	"path/filepath"
	"runtime"

	"github.com/pkg/errors"
	"gopkg.in/urfave/cli.v1"

	"github.com/Southclaws/sampctl/download"
	"github.com/Southclaws/sampctl/print"
	"github.com/Southclaws/sampctl/rook"
	"github.com/Southclaws/sampctl/types"
	"github.com/Southclaws/sampctl/util"
)

var packageTemplateRunFlags = []cli.Flag{
	cli.StringFlag{
		Name:  "version",
		Value: "0.3.7",
		Usage: "the SA:MP server version to use",
	},
	cli.StringFlag{
		Name:  "endpoint",
		Value: "http://files.sa-mp.com",
		Usage: "endpoint to download packages from",
	},
	cli.StringFlag{
		Name:  "mode",
		Value: "main",
		Usage: "runtime mode, one of: server, main, y_testing",
	},
}

func packageTemplateRun(c *cli.Context) (err error) {
	if c.Bool("verbose") {
		print.SetVerbose()
	}

	version := c.String("version")
	endpoint := c.String("endpoint")
	mode := c.String("mode")

	if len(c.Args()) != 2 {
		cli.ShowCommandHelpAndExit(c, "run", 0)
		return nil
	}

	cacheDir, err := download.GetCacheDir()
	if err != nil {
		return
	}
	template := c.Args().Get(0)
	filename := c.Args().Get(1)

	templatePath := filepath.Join(cacheDir, "templates", template)
	if !util.Exists(templatePath) {
		return errors.Errorf("template '%s' does not exist", template)
	}

	if !util.Exists(filename) {
		return errors.Errorf("no such file or directory: %s", filename)
	}

	pkg, err := rook.PackageFromDir(true, templatePath, "")
	if err != nil {
		return errors.Wrap(err, "template package is invalid")
	}

	err = util.CopyFile(filename, filepath.Join(templatePath, "tmpl.pwn"))
	if err != nil {
		return errors.Wrap(err, "failed to copy target script to template package directory")
	}

	problems, result, err := rook.Build(&pkg, "", cacheDir, runtime.GOOS, false, false, "")
	if err != nil {
		return
	}

	print.Info("Build complete with", len(problems), "problems")
	print.Info(fmt.Sprintf("Results, in bytes: Header: %d, Code: %d, Data: %d, Stack/Heap: %d, Estimated usage: %d, Total: %d\n",
		result.Header,
		result.Code,
		result.Data,
		result.StackHeap,
		result.Estimate,
		result.Total))

	if !problems.IsValid() {
		return errors.New("cannot run with build errors")
	}

	cfg := types.Runtime{
		Platform:   runtime.GOOS,
		AppVersion: c.App.Version,
		Version:    version,
		Endpoint:   endpoint,
	}
	pkg.Runtime = new(types.Runtime)
	pkg.Runtime.Mode = types.RunMode(mode)

	err = rook.Run(pkg, cfg, cacheDir, "", false, false, false, "")
	if err != nil {
		return
	}

	return
}
