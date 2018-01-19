package main

import (
	"github.com/google/go-github/github"
)

// scrapeRepo is responsible for taking a repo and checking its contents for the qualifying
// properties of a Pawn Package. This includes the presence of one or more .inc files and optionally
// a pawn.json or pawn.yaml file. If one of these files exists, additional information is extracted.
// This function pushes to the `toIndex` channel if the repo is valid.
func (app *App) scrapeRepo(repo github.Repository) (err error) {
	// pkg := types.Package{}
	// runtime.GetPluginRemotePackage

	return
}
