package main

import (
	"context"
	"fmt"
	"net/http"

	"go.cluttr.dev/prebuilt/internal/metaerr"
)

type BinaryData struct {
	Provider    string
	Name        string
	Version     string
	DownloadURL string
	ExtractPath string
}

type Resolver struct {
	Providers map[string]*Provider
}

func (r *Resolver) Init(ps []ProviderSpec) error {
	r.Providers = make(map[string]*Provider)

	builtinProviderSpecs := []ProviderSpec{
		githubProviderSpec,
		gitlabProviderSpec,
		httpsProviderSpec,
		httpProviderSpec,
	}
	for _, spec := range builtinProviderSpecs {
		if err := r.initProvider(spec); err != nil {
			return metaerr.WithMetadata(fmt.Errorf("init provider: %w", err), "name", spec.Name)
		}
	}

	for _, spec := range ps {
		if err := r.initProvider(spec); err != nil {
			return metaerr.WithMetadata(fmt.Errorf("init provider: %w", err), "name", spec.Name)
		}
	}

	return nil
}

func (r *Resolver) Client(name string) *http.Client {
	if c, ok := r.Providers[name]; ok {
		return c.Client
	}
	return defaultClient()
}

func (r *Resolver) initProvider(spec ProviderSpec) error {
	if spec.Name == "" {
		return fmt.Errorf("missing provider name")
	}
	if _, ok := r.Providers[spec.Name]; ok {
		return fmt.Errorf("provider already initialized: %s", spec.Name)
	}

	r.Providers[spec.Name] = NewProvider(spec)
	return nil
}

func (r *Resolver) Resolve(ctx context.Context, bin BinarySpec) (BinaryData, error) {
	prov, data, err := r.resolveProvider(bin.Provider)
	if err != nil {
		return BinaryData{}, err
	}

	// Name
	name := getBinName(bin, prov.Spec)

	// Version
	var versionSpec VersionSpec
	if bin.Version.String != nil {
		versionSpec.Constraints = *bin.Version.String
	} else if bin.Version.Spec != nil {
		versionSpec = *bin.Version.Spec
	}
	versionsUrl, err := renderTemplate(prov.Spec.VersionsURL, map[string]any{
		"Provider": data,
	})
	if err != nil {
		return BinaryData{}, err
	}
	version, err := ResolveVersion(ctx, prov.Client, versionsUrl, prov.Spec.VersionsJSONPath, versionSpec.Constraints, versionSpec.Prefix)
	if err != nil {
		return BinaryData{}, metaerr.WithMetadata(fmt.Errorf("resolve version: %w", err), "url", versionsUrl)
	}

	// DownloadURL
	downloadURL, err := renderTemplate(prov.Spec.DownloadURL, map[string]any{
		"Provider": data,
		"Version":  version,
	})
	if err != nil {
		return BinaryData{}, metaerr.WithMetadata(fmt.Errorf("render download url: %w", err), "template", prov.Spec.DownloadURL)
	}

	// ExtractPath
	var extractPath string
	if bin.ExtractPath != "" {
		extractPath, err = renderTemplate(bin.ExtractPath, map[string]any{
			"Provider": data,
			"Version":  version,
		})
		if err != nil {
			return BinaryData{}, metaerr.WithMetadata(fmt.Errorf("render extract path: %w", err), "template", bin.ExtractPath)
		}
	}

	return BinaryData{
		Provider:    prov.Spec.Name,
		Name:        name,
		Version:     version,
		DownloadURL: downloadURL,
		ExtractPath: extractPath,
	}, nil
}

func (r *Resolver) resolveProvider(cfg ProviderConfig) (*Provider, ProviderData, error) {
	var (
		prov *Provider
		data ProviderData
		err  error
	)
	if cfg.Spec != nil {
		prov = NewProvider(*cfg.Spec)
		data.Scheme = cfg.Spec.Name
	} else if cfg.DSN != nil {
		data, err = parseDSN(*cfg.DSN)
		if err != nil {
			return nil, ProviderData{}, err
		}
		var ok bool
		prov, ok = r.Providers[data.Scheme]
		if !ok {
			return nil, ProviderData{}, fmt.Errorf("provider unknown: %s", data.Scheme)
		}
	} else {
		return nil, ProviderData{}, fmt.Errorf("invalid provider config")
	}

	return prov, data, nil
}
