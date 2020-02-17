// Copyright 2020 The Okteto Authors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/containerd/console"
	"github.com/moby/buildkit/client"
	"github.com/moby/buildkit/util/progress/progressui"
	"github.com/okteto/okteto/pkg/analytics"
	"github.com/okteto/okteto/pkg/cmd/build"
	"github.com/okteto/okteto/pkg/log"
	"github.com/okteto/okteto/pkg/okteto"
	"golang.org/x/sync/errgroup"

	"github.com/spf13/cobra"
)

//Build build and optionally push a Docker image
func Build() *cobra.Command {
	var file string
	var tag string
	var target string
	var noCache bool

	cmd := &cobra.Command{
		Use:   "build",
		Short: "Build (and optionally push) a Docker image",
		RunE: func(cmd *cobra.Command, args []string) error {
			log.Debug("starting build command")
			if _, err := RunBuild(args[0], file, tag, target, noCache); err != nil {
				analytics.TrackBuild(false)
				return err
			}
			analytics.TrackBuild(true)
			return nil
		},
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) < 1 {
				return fmt.Errorf("build requires the PATH context argument (e.g. '.' for the current directory)")
			}
			return nil
		},
	}

	cmd.Flags().StringVarP(&file, "file", "f", "", "name of the Dockerfile (Default is 'PATH/Dockerfile')")
	cmd.Flags().StringVarP(&tag, "tag", "t", "", "name and optionally a tag in the 'name:tag' format (it is automatically pushed)")
	cmd.Flags().StringVarP(&target, "target", "", "", "set the target build stage to build")
	cmd.Flags().BoolVarP(&noCache, "no-cache", "", false, "do not use cache when building the image")
	return cmd
}

//RunBuild starts the build sequence
func RunBuild(path, file, tag, target string, noCache bool) (string, error) {
	ctx := context.Background()

	buildKitHost, err := build.GetBuildKitHost()
	if err != nil {
		return "", fmt.Errorf("'BUILDKIT_HOST' not set. You can run 'okteto login' to run your builds in Okteto Cloud for free")
	}
	oktetoBuildKit, _ := okteto.GetBuildKit()
	if oktetoBuildKit != buildKitHost {
		log.Information("Running your build in %s...", buildKitHost)
	} else {
		if strings.Contains(buildKitHost, okteto.CloudBuildKitURL) {
			log.Information("Running your build in Okteto Cloud...")
		} else {
			log.Information("Running your build in Okteto Enterprise...")
		}
	}

	c, err := client.New(ctx, buildKitHost, client.WithFailFast())
	if err != nil {
		return "", err
	}

	ch := make(chan *client.SolveStatus)
	eg, ctx := errgroup.WithContext(ctx)

	if file == "" {
		file = filepath.Join(path, "Dockerfile")
	}
	if buildKitHost == oktetoBuildKit {
		fileWithCacheHandler, err := build.GetDockerfileWithCacheHandler(file)
		if err != nil {
			return "", fmt.Errorf("failed to create temporary build folder: %s", err)
		}

		defer os.Remove(fileWithCacheHandler)
		file = fileWithCacheHandler
	}

	solveOpt, err := build.GetSolveOpt(path, file, tag, target, noCache)
	if err != nil {
		return "", err
	}

	if tag == "" {
		log.Information("Your image won't be pushed. To push your image specify the flag '-t'.")
	}

	var solveResp *client.SolveResponse
	eg.Go(func() error {
		var err error
		solveResp, err = c.Solve(ctx, nil, *solveOpt, ch)
		return err
	})

	eg.Go(func() error {
		var c console.Console
		if cn, err := console.ConsoleFromFile(os.Stderr); err == nil {
			c = cn
		}
		// not using shared context to not disrupt display but let it finish reporting errors
		return progressui.DisplaySolveStatus(context.TODO(), "", c, os.Stdout, ch)
	})

	if err := eg.Wait(); err != nil {
		return "", err
	}

	return solveResp.ExporterResponse["containerimage.digest"], nil
}
