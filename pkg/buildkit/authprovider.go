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

package buildkit

import (
	"context"
	"io"
	"sync"

	"github.com/docker/cli/cli/config"
	"github.com/docker/cli/cli/config/configfile"
	"github.com/docker/cli/cli/config/types"
	"github.com/moby/buildkit/session"
	"github.com/moby/buildkit/session/auth"
	"github.com/okteto/okteto/pkg/okteto"
	"google.golang.org/grpc"
)

//NewDockerAndOktetoAuthProvider reads local docker credentials and auto-injects okteto registry credentials
func NewDockerAndOktetoAuthProvider(username, password string, stderr io.Writer) session.Attachable {
	result := &authProvider{
		config: config.LoadDefaultConfigFile(stderr),
	}
	result.config.AuthConfigs[okteto.GetRegistry()] = types.AuthConfig{
		ServerAddress: okteto.GetRegistry(),
		Username:      username,
		Password:      password,
	}
	return result
}

type authProvider struct {
	config *configfile.ConfigFile

	// The need for this mutex is not well understood.
	// Without it, the docker cli on OS X hangs when
	// reading credentials from docker-credential-osxkeychain.
	// See issue https://github.com/docker/cli/issues/1862
	mu sync.Mutex
}

func (ap *authProvider) Register(server *grpc.Server) {
	auth.RegisterAuthServer(server, ap)
}

func (ap *authProvider) Credentials(ctx context.Context, req *auth.CredentialsRequest) (*auth.CredentialsResponse, error) {
	res := &auth.CredentialsResponse{}
	if req.Host == okteto.GetRegistry() {
		res.Username = ap.config.AuthConfigs[okteto.GetRegistry()].Username
		res.Secret = ap.config.AuthConfigs[okteto.GetRegistry()].Password
		return res, nil
	}

	ap.mu.Lock()
	defer ap.mu.Unlock()
	if req.Host == "registry-1.docker.io" {
		req.Host = "https://index.docker.io/v1/"
	}
	ac, err := ap.config.GetAuthConfig(req.Host)
	if err != nil {
		return nil, err
	}
	if ac.IdentityToken != "" {
		res.Secret = ac.IdentityToken
	} else {
		res.Username = ac.Username
		res.Secret = ac.Password
	}
	return res, nil
}
