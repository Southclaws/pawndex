package types

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"path/filepath"

	"github.com/google/go-github/github"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"

	"github.com/Southclaws/sampctl/util"
	"github.com/Southclaws/sampctl/versioning"
)

// Package represents a definition for a Pawn package and can either be used to define a build or
// as a description of a package in a repository. This is akin to npm's package.json and combines
// a project's dependencies with a description of that project.
//
// For example, a gamemode that includes a library does not need to define the User, Repo, Version,
// Contributors and Include fields at all, it can just define the Dependencies list in order to
// build correctly.
//
// On the flip side, a library written in pure Pawn should define some contributors and a web URL
// but, being written in pure Pawn, has no dependencies.
//
// Finally, if a repository stores its package source files in a subdirectory, that directory should
// be specified in the Include field. This is common practice for plugins that store the plugin
// source code in the root and the Pawn source in a subdirectory called 'include'.
type Package struct {
	// Parent indicates that this package is a "working" package that the user has explicitly
	// created and is developing. The opposite of this would be packages that exist in the
	// `dependencies` directory that have been downloaded as a result of an Ensure.
	Parent bool `json:"-" yaml:"-"`
	// Local path, this indicates the Package object represents a local copy which is a directory
	// containing a `samp.json`/`samp.yaml` file and a set of Pawn source code files.
	// If this field is not set, then the Package is just an in-memory pointer to a remote package.
	Local string `json:"-" yaml:"-"`
	// The vendor directory - for simple packages with no sub-dependencies, this is simply
	// `<local>/dependencies` but for nested dependencies, this needs to be set.
	Vendor string `json:"-" yaml:"-"`
	// format stores the original format of the package definition file, either `json` or `yaml`
	Format string `json:"-" yaml:"-"`
	// allDependencies stores a list of all dependency meta from this package and sub packages
	// this field is only used if `parent` is true.
	AllDependencies []versioning.DependencyMeta `json:"-" yaml:"-"`
	// allPlugins stores a list of all plugin dependency meta from this package and sub packages
	// this field is only used if `parent` is true.
	AllPlugins []versioning.DependencyMeta `json:"-" yaml:"-"`

	// Inferred metadata, not always explicitly set via JSON/YAML but inferred from the dependency path
	versioning.DependencyMeta

	// Metadata, set by the package author to describe the package
	Contributors []string `json:"contributors,omitempty" yaml:"contributors,omitempty"` // list of contributors
	Website      string   `json:"website,omitempty" yaml:"website,omitempty"`           // website or forum topic associated with the package

	// Functional, set by the package author to declare relevant files and dependencies
	Entry        string                        `json:"entry"`                                                        // entry point script to compile the project
	Output       string                        `json:"output"`                                                       // output amx file
	Dependencies []versioning.DependencyString `json:"dependencies,omitempty" yaml:"dependencies,omitempty"`         // list of packages that the package depends on
	Development  []versioning.DependencyString `json:"dev_dependencies,omitempty" yaml:"dev_dependencies,omitempty"` // list of packages that only the package builds depend on
	Builds       []BuildConfig                 `json:"builds,omitempty" yaml:"builds,omitempty"`                     // list of build configurations
	Runtime      *Runtime                      `json:"runtime,omitempty" yaml:"runtime,omitempty"`                   // runtime configuration for executing the package code
	IncludePath  string                        `json:"include_path,omitempty" yaml:"include_path,omitempty"`         // include path within the repository, so users don't need to specify the path explicitly
	Resources    []Resource                    `json:"resources,omitempty" yaml:"resources,omitempty"`               // list of additional resources associated with the package
}

func (pkg Package) String() string {
	return fmt.Sprintf("%s/%s:%s", pkg.User, pkg.Repo, pkg.Version)
}

// Validate checks a package for missing fields
func (pkg Package) Validate() (err error) {
	if pkg.Entry == pkg.Output && pkg.Entry != "" && pkg.Output != "" {
		return errors.New("package entry and output point to the same file")
	}

	return
}

// GetAllDependencies returns the Dependencies and the Development dependencies in one list
func (pkg Package) GetAllDependencies() (result []versioning.DependencyString) {
	result = append(result, pkg.Dependencies...)
	result = append(result, pkg.Development...)
	return
}

// PackageFromDep creates a Package object from a Dependency String
func PackageFromDep(depString versioning.DependencyString) (pkg Package, err error) {
	dep, err := depString.Explode()
	pkg.User, pkg.Repo, pkg.Path, pkg.Version = dep.User, dep.Repo, dep.Path, dep.Version
	return
}

// PackageFromDir attempts to parse a pawn.json or pawn.yaml file from a directory
func PackageFromDir(dir string) (pkg Package, err error) {
	jsonFile := filepath.Join(dir, "pawn.json")
	if util.Exists(jsonFile) {
		return PackageFromJSON(jsonFile)
	}

	yamlFile := filepath.Join(dir, "pawn.yaml")
	if util.Exists(yamlFile) {
		return PackageFromYAML(yamlFile)
	}

	err = errors.New("no pawn.json/pawn.yaml present")

	return
}

// PackageFromJSON creates a config from a JSON file
func PackageFromJSON(file string) (pkg Package, err error) {
	var contents []byte
	contents, err = ioutil.ReadFile(file)
	if err != nil {
		err = errors.Wrap(err, "failed to read pawn.json")
		return
	}

	err = json.Unmarshal(contents, &pkg)
	if err != nil {
		err = errors.Wrap(err, "failed to unmarshal pawn.json")
		return
	}

	pkg.Format = "json"

	return
}

// PackageFromYAML creates a config from a YAML file
func PackageFromYAML(file string) (pkg Package, err error) {
	var contents []byte
	contents, err = ioutil.ReadFile(file)
	if err != nil {
		err = errors.Wrap(err, "failed to read pawn.yaml")
		return
	}

	err = yaml.Unmarshal(contents, &pkg)
	if err != nil {
		err = errors.Wrap(err, "failed to unmarshal pawn.yaml")
		return
	}

	pkg.Format = "yaml"

	return
}

// WriteDefinition creates a JSON or YAML file for a package object, the format depends
// on the `Format` field of the package.
func (pkg Package) WriteDefinition() (err error) {
	switch pkg.Format {
	case "json":
		var contents []byte
		contents, err = json.MarshalIndent(pkg, "", "\t")
		if err != nil {
			return errors.Wrap(err, "failed to encode package metadata")
		}
		err = ioutil.WriteFile(filepath.Join(pkg.Local, "pawn.json"), contents, 0755)
		if err != nil {
			return errors.Wrap(err, "failed to write pawn.json")
		}
	case "yaml":
		var contents []byte
		contents, err = yaml.Marshal(pkg)
		if err != nil {
			return errors.Wrap(err, "failed to encode package metadata")
		}
		err = ioutil.WriteFile(filepath.Join(pkg.Local, "pawn.yaml"), contents, 0755)
		if err != nil {
			return errors.Wrap(err, "failed to write pawn.yaml")
		}
	default:
		err = errors.New("package has no format associated with it")
	}

	return
}

// GetPluginRemotePackage attempts to get a package definition for the given dependency meta
// it first checks the repository itself, if that fails it falls back to using the sampctl central
// plugin metadata repository
func GetPluginRemotePackage(client *github.Client, meta versioning.DependencyMeta) (pkg Package, err error) {
	repo, _, err := client.Repositories.Get(context.Background(), meta.User, meta.Repo)
	if err == nil {
		var resp *http.Response

		resp, err = http.Get(fmt.Sprintf("https://raw.githubusercontent.com/%s/%s/%s/pawn.json", meta.User, meta.Repo, *repo.DefaultBranch))
		if err != nil {
			return
		}

		if resp.StatusCode == 200 {
			var contents []byte
			contents, err = ioutil.ReadAll(resp.Body)
			if err != nil {
				return
			}
			err = json.Unmarshal(contents, &pkg)
			return
		}

		resp, err = http.Get(fmt.Sprintf("https://raw.githubusercontent.com/%s/%s/%s/pawn.yaml", meta.User, meta.Repo, *repo.DefaultBranch))
		if err != nil {
			return
		}

		if resp.StatusCode == 200 {
			var contents []byte
			contents, err = ioutil.ReadAll(resp.Body)
			if err != nil {
				return
			}
			err = yaml.Unmarshal(contents, &pkg)
			return
		}
	}

	resp, err := http.Get(fmt.Sprintf("https://raw.githubusercontent.com/sampctl/plugins/master/%s-%s.json", meta.User, meta.Repo))
	if err != nil {
		return
	}

	if resp.StatusCode == 200 {
		dec := json.NewDecoder(resp.Body)
		err = dec.Decode(&pkg)
		return
	}

	err = errors.New("could not find plugin package definition")

	return
}
