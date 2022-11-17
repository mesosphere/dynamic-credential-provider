// Copyright 2022 D2iQ, Inc. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package shim

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/mesosphere/kubelet-image-credential-provider-shim/apis/config/v1alpha1"
	"github.com/mesosphere/kubelet-image-credential-provider-shim/pkg/urlglobber"
	"golang.org/x/sync/singleflight"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer/json"
	"k8s.io/klog/v2"
	"k8s.io/kubelet/config/v1beta1"
	credentialproviderapi "k8s.io/kubelet/pkg/apis/credentialprovider"
)

// newPluginProvider returns a new pluginProvider based on the credential provider config.
func newPluginProvider(
	pluginBinDir string,
	provider v1beta1.CredentialProvider,
) (*pluginProvider, error) {
	mediaType := "application/json"
	info, ok := runtime.SerializerInfoForMediaType(codecs.SupportedMediaTypes(), mediaType)
	if !ok {
		return nil, fmt.Errorf("unsupported media type %q", mediaType)
	}

	gv, ok := v1alpha1.APIVersions[provider.APIVersion]
	if !ok {
		return nil, fmt.Errorf("invalid apiVersion: %q", provider.APIVersion)
	}

	return &pluginProvider{
		matchImages: provider.MatchImages,
		plugin: &execPlugin{
			name:         provider.Name,
			apiVersion:   provider.APIVersion,
			encoder:      codecs.EncoderForVersion(info.Serializer, gv),
			pluginBinDir: pluginBinDir,
			args:         provider.Args,
			envVars:      provider.Env,
			environ:      os.Environ,
		},
	}, nil
}

type Provider interface {
	Provide(image string) (*credentialproviderapi.CredentialProviderResponse, error)
}

// pluginProvider is the plugin-based implementation of the DockerConfigProvider interface.
type pluginProvider struct {
	sync.Mutex

	group singleflight.Group

	// matchImages defines the matching image URLs this plugin should operate against.
	// The plugin provider will not return any credentials for images that do not match
	// against this list of match URLs.
	matchImages []string

	// plugin is the exec implementation of the credential providing plugin.
	plugin Plugin
}

var ErrInvalidCredentialProviderResponse = errors.New(
	"invalid response type returned by external credential provider",
)

// Provide returns a *credentialproviderapi.CredentialProviderResponse based on the credentials returned
// by executing the plugin.
func (p *pluginProvider) Provide(
	image string,
) (*credentialproviderapi.CredentialProviderResponse, error) {
	if !p.isImageAllowed(image) {
		return nil, nil
	}

	// ExecPlugin is wrapped in single flight to exec plugin once for concurrent same image request.
	// The caveat here is we don't know cacheKeyType yet, so if cacheKeyType is registry/global and credentials saved in
	// cache on per registry/global basis then exec will be called for all requests if requests are made concurrently.
	// foo.bar.registry
	// foo.bar.registry/image1
	// foo.bar.registry/image2
	res, err, _ := p.group.Do(image, func() (interface{}, error) {
		return p.plugin.ExecPlugin(context.Background(), image)
	})

	if err != nil {
		return nil, fmt.Errorf("error calling external credential provider: %w", err)
	}

	response, ok := res.(*credentialproviderapi.CredentialProviderResponse)
	if !ok {
		return nil, fmt.Errorf("%w: %T", ErrInvalidCredentialProviderResponse, res)
	}

	return response, nil
}

// isImageAllowed returns true if the image matches against the list of allowed matches by the plugin.
func (p *pluginProvider) isImageAllowed(image string) bool {
	for _, matchImage := range p.matchImages {
		if matched, _ := urlglobber.URLsMatchStr(matchImage, image); matched {
			return true
		}
	}

	return false
}

// Plugin is the interface calling ExecPlugin. This is mainly for testability
// so tests don't have to actually exec any processes.
type Plugin interface {
	ExecPlugin(
		ctx context.Context,
		image string,
	) (*credentialproviderapi.CredentialProviderResponse, error)
}

// execPlugin is the implementation of the Plugin interface that execs a credential provider plugin based
// on it's name provided in CredentialProviderConfig. It is assumed that the executable is available in the
// plugin directory provided by the kubelet.
type execPlugin struct {
	name         string
	apiVersion   string
	encoder      runtime.Encoder
	args         []string
	envVars      []v1beta1.ExecEnvVar
	pluginBinDir string
	environ      func() []string
}

// ExecPlugin executes the plugin binary with arguments and environment variables specified in CredentialProviderConfig:
//
//	$ ENV_NAME=ENV_VALUE <plugin-name> args[0] args[1] <<<request
//
// The plugin is expected to receive the CredentialProviderRequest API via stdin from the kubelet and
// return CredentialProviderResponse via stdout.
func (e *execPlugin) ExecPlugin(
	ctx context.Context,
	image string,
) (*credentialproviderapi.CredentialProviderResponse, error) {
	klog.V(5).Infof("Getting image %s credentials from external exec plugin %s", image, e.name)

	authRequest := &credentialproviderapi.CredentialProviderRequest{Image: image}
	data, err := e.encodeRequest(authRequest)
	if err != nil {
		return nil, fmt.Errorf("failed to encode auth request: %w", err)
	}

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	stdin := bytes.NewBuffer(data)

	// Use a catch-all timeout of 1 minute for all exec-based plugins, this should leave enough
	// head room in case a plugin needs to retry a failed request while ensuring an exec plugin
	// does not run forever. In the future we may want this timeout to be tweakable from the plugin
	// config file.
	ctx, cancel := context.WithTimeout(ctx, 1*time.Minute)
	defer cancel()

	cmd := exec.CommandContext(ctx, filepath.Join(e.pluginBinDir, e.name), e.args...)
	cmd.Stdout, cmd.Stderr, cmd.Stdin = stdout, stderr, stdin

	var configEnvVars []string
	for _, v := range e.envVars {
		configEnvVars = append(configEnvVars, fmt.Sprintf("%s=%s", v.Name, v.Value))
	}

	// Append current system environment variables, to the ones configured in the
	// credential provider file. Failing to do so may result in unsuccessful execution
	// of the provider binary, see https://github.com/kubernetes/kubernetes/issues/102750
	// also, this behaviour is inline with Credential Provider Config spec
	cmd.Env = mergeEnvVars(e.environ(), configEnvVars)

	if err = e.runPlugin(ctx, cmd, image); err != nil {
		return nil, err
	}

	data = stdout.Bytes()
	// check that the response apiVersion matches what is expected
	gvk, err := json.DefaultMetaFactory.Interpret(data)
	if err != nil {
		return nil, fmt.Errorf("error reading GVK from response: %w", err)
	}

	if gvk.GroupVersion().String() != e.apiVersion {
		return nil, fmt.Errorf(
			"apiVersion from credential plugin response did not match expected apiVersion:%s, actual apiVersion:%s",
			e.apiVersion,
			gvk.GroupVersion().String(),
		)
	}

	response, err := e.decodeResponse(data)
	if err != nil {
		// err is explicitly not wrapped since it may contain credentials in the response.
		return nil, errors.New("error decoding credential provider plugin response from stdout")
	}

	return response, nil
}

func (e *execPlugin) runPlugin(ctx context.Context, cmd *exec.Cmd, image string) error {
	err := cmd.Run()
	if ctx.Err() != nil {
		return fmt.Errorf(
			"error execing credential provider plugin %s for image %s: %w",
			e.name,
			image,
			ctx.Err(),
		)
	}
	if err != nil {
		return fmt.Errorf(
			"error execing credential provider plugin %s for image %s: %w",
			e.name,
			image,
			err,
		)
	}
	return nil
}

// encodeRequest encodes the internal CredentialProviderRequest type into the v1alpha1 version in json
func (e *execPlugin) encodeRequest(
	request *credentialproviderapi.CredentialProviderRequest,
) ([]byte, error) {
	data, err := runtime.Encode(e.encoder, request)
	if err != nil {
		return nil, fmt.Errorf("error encoding request: %w", err)
	}

	return data, nil
}

// decodeResponse decodes data into the internal CredentialProviderResponse type.
func (e *execPlugin) decodeResponse(
	data []byte,
) (*credentialproviderapi.CredentialProviderResponse, error) {
	obj, gvk, err := codecs.UniversalDecoder().Decode(data, nil, nil)
	if err != nil {
		return nil, err
	}

	if gvk.Kind != "CredentialProviderResponse" {
		return nil, fmt.Errorf(
			"failed to decode CredentialProviderResponse, unexpected Kind: %q",
			gvk.Kind,
		)
	}

	if gvk.Group != credentialproviderapi.GroupName {
		return nil, fmt.Errorf(
			"failed to decode CredentialProviderResponse, unexpected Group: %s",
			gvk.Group,
		)
	}

	if internalResponse, ok := obj.(*credentialproviderapi.CredentialProviderResponse); ok {
		return internalResponse, nil
	}

	return nil, fmt.Errorf("unable to convert %T to *CredentialProviderResponse", obj)
}

// parseRegistry extracts the registry hostname of an image (including port if specified).
func parseRegistry(image string) string {
	imageParts := strings.Split(image, "/")
	return imageParts[0]
}

// mergedEnvVars overlays system defined env vars with credential provider env vars,
// it gives priority to the credential provider vars allowing user to override system
// env vars
func mergeEnvVars(sysEnvVars, credProviderVars []string) []string {
	mergedEnvVars := sysEnvVars
	for _, credProviderVar := range credProviderVars {
		mergedEnvVars = append(mergedEnvVars, credProviderVar)
	}
	return mergedEnvVars
}
