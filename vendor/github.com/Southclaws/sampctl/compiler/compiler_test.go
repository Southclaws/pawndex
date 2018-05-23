package compiler

import (
	"context"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/Southclaws/sampctl/types"
	"github.com/Southclaws/sampctl/util"
)

func TestCompileSource(t *testing.T) {
	type args struct {
		cacheDir string
		config   types.BuildConfig
	}
	tests := []struct {
		name         string
		args         args
		wantProblems types.BuildProblems
		wantResult   types.BuildResult
		wantErr      bool
		wantOutput   bool
	}{
		{"simple-pass", args{
			util.FullPath("./tests/cache-compile"),
			types.BuildConfig{
				WorkingDir: "./tests/build-simple-pass",
				Input:      "./tests/build-simple-pass/script.pwn",
				Output:     "./tests/build-simple-pass/script.amx",
				Includes:   []string{},
				Version:    "3.10.4",
			}},
			nil,
			types.BuildResult{},
			false, true},
		{"simple-pass-d3", args{
			util.FullPath("./tests/cache-compile"),
			types.BuildConfig{
				WorkingDir: "./tests/build-simple-pass",
				Input:      "./tests/build-simple-pass/script.pwn",
				Output:     "./tests/build-simple-pass/script.amx",
				Args:       []string{"-d3"},
				Includes:   []string{},
				Version:    "3.10.4",
			}},
			nil,
			types.BuildResult{
				Header:    60,
				Code:      184,
				Data:      0,
				StackHeap: 16384,
				Estimate:  20,
				Total:     16628,
			},
			false, true},
		{"simple-fail", args{
			util.FullPath("./tests/cache-compile"),
			types.BuildConfig{
				WorkingDir: "./tests/build-simple-fail",
				Input:      "./tests/build-simple-fail/script.pwn",
				Output:     "./tests/build-simple-fail/script.amx",
				Includes:   []string{},
				Version:    "3.10.4",
			}},
			types.BuildProblems{
				{"script.pwn", 1, types.ProblemError, `invalid function or declaration`},
				{"script.pwn", 3, types.ProblemError, `invalid function or declaration`},
				{"script.pwn", 6, types.ProblemWarning, `symbol is never used: "a"`},
				{"script.pwn", 6, types.ProblemError, `no entry point (no public functions)`},
			},
			types.BuildResult{},
			false, false},
		{"local-include-pass", args{
			util.FullPath("./tests/cache-compile"),
			types.BuildConfig{
				WorkingDir: "./tests/build-local-include-pass",
				Input:      "./tests/build-local-include-pass/script.pwn",
				Output:     "./tests/build-local-include-pass/script.amx",
				Args:       []string{"-d3", "-;+", "-(+", "-\\+", "-Z+"},
				Includes:   []string{},
				Version:    "3.10.4",
			}},
			nil,
			types.BuildResult{
				Header:    60,
				Code:      220,
				Data:      0,
				StackHeap: 16384,
				Estimate:  32,
				Total:     16664,
			},
			false, true},
		{"local-include-warn", args{
			util.FullPath("./tests/cache-compile"),
			types.BuildConfig{
				WorkingDir: "./tests/build-local-include-warn",
				Input:      "./tests/build-local-include-warn/script.pwn",
				Output:     "./tests/build-local-include-warn/script.amx",
				Args:       []string{"-d3", "-;+", "-(+", "-\\+", "-Z+"},
				Includes:   []string{},
				Version:    "3.10.4",
			}},
			types.BuildProblems{
				{"library.inc", 6, types.ProblemWarning, `symbol is never used: "b"`},
				{"script.pwn", 5, types.ProblemWarning, `symbol is never used: "a"`},
			},
			types.BuildResult{
				Header:    60,
				Code:      276,
				Data:      0,
				StackHeap: 16384,
				Estimate:  32,
				Total:     16720,
			},
			false, true},
		{"fatal", args{
			util.FullPath("./tests/cache-compile"),
			types.BuildConfig{
				WorkingDir: "./tests/build-fatal",
				Input:      "./tests/build-fatal/script.pwn",
				Output:     "./tests/build-fatal/script.amx",
				Args:       []string{"-d3", "-;+", "-(+", "-\\+", "-Z+"},
				Includes:   []string{},
				Version:    "3.10.4",
			}},
			types.BuildProblems{
				{"script.pwn", 1, types.ProblemFatal, `cannot read from file: "idonotexist"`},
			},
			types.BuildResult{},
			false, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotProblems, gotResult, err := CompileSource(context.Background(), ".", tt.args.cacheDir, runtime.GOOS, tt.args.config)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			for i, p := range tt.wantProblems {
				tt.wantProblems[i].File = util.FullPath(filepath.Join(tt.args.config.WorkingDir, p.File))
			}

			assert.Equal(t, tt.wantProblems, gotProblems)
			assert.Equal(t, tt.wantResult, gotResult)

			if tt.wantOutput {
				assert.True(t, util.Exists(tt.args.config.Output))
			}
		})
	}
}
