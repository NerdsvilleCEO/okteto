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

package build

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/moby/buildkit/client"
	"github.com/moby/buildkit/session"
	"github.com/moby/buildkit/session/auth/authprovider"
	"github.com/okteto/okteto/pkg/buildkit"
	"github.com/okteto/okteto/pkg/okteto"
)

const (
	frontend = "dockerfile.v0"
)

//GetBuildKitHost returns thee buildkit url
func GetBuildKitHost() (string, error) {
	buildKitHost := os.Getenv("BUILDKIT_HOST")
	if buildKitHost != "" {
		return buildKitHost, nil
	}
	return okteto.GetBuildKit()
}

//GetSolveOpt returns the buildkit solve options
func GetSolveOpt(buildCtx, file, imageTag, target string, noCache bool) (*client.SolveOpt, error) {
	if file == "" {
		file = filepath.Join(buildCtx, "Dockerfile")
	}
	if _, err := os.Stat(file); os.IsNotExist(err) {
		return nil, fmt.Errorf("Dockerfile '%s' does not exist", file)
	}
	localDirs := map[string]string{
		"context":    buildCtx,
		"dockerfile": filepath.Dir(file),
	}

	frontendAttrs := map[string]string{
		"filename": filepath.Base(file),
	}
	if target != "" {
		frontendAttrs["target"] = target
	}
	if noCache {
		frontendAttrs["no-cache"] = ""
	}

	attachable := []session.Attachable{}
	if strings.HasPrefix(imageTag, okteto.GetRegistry()) {
		// set Okteto Cloud credentials
		token, err := okteto.GetToken()
		if err != nil {
			return nil, fmt.Errorf("failed to read okteto token. Did you run 'okteto login'?")
		}
		attachable = append(attachable, buildkit.NewDockerAndOktetoAuthProvider(okteto.GetUserID(), token.Token, os.Stderr))
	} else {
		// read docker credentials from `.docker/config.json`
		attachable = append(attachable, authprovider.NewDockerAuthProvider(os.Stderr))
	}
	opt := &client.SolveOpt{
		LocalDirs:     localDirs,
		Frontend:      frontend,
		FrontendAttrs: frontendAttrs,
		Session:       attachable,
	}

	if imageTag != "" {
		opt.Exports = []client.ExportEntry{
			{
				Type: "image",
				Attrs: map[string]string{
					"name": imageTag,
					"push": "true",
				},
			},
		}
		opt.CacheExports = []client.CacheOptionsEntry{
			{
				Type: "inline",
			},
		}
		opt.CacheImports = []client.CacheOptionsEntry{
			{
				Type:  "registry",
				Attrs: map[string]string{"ref": imageTag},
			},
		}
	}

	return opt, nil
}
