package main

import (
	"bytes"
	"context"
	"crypto"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"time"

	"go.cluttr.dev/prebuilt/internal/metaerr"
)

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

func (r *Resolver) Resolve(ctx context.Context, bins []BinarySpec) (Lock, error) {
	// set up workers
	type result struct {
		data BinaryData
		err  error
	}

	numBins := len(bins)
	jobs := make(chan BinarySpec, numBins)
	results := make(chan result, numBins)

	worker := func(specs <-chan BinarySpec, res chan<- result) {
		for spec := range specs {
			data, err := r.resolve(ctx, spec)
			if err != nil {
				res <- result{
					err: metaerr.WithMetadata(err, "name", spec.Name),
				}
			}
			res <- result{
				data: data,
			}
		}
	}

	const concurrency = 8
	for range concurrency {
		go worker(jobs, results)
	}

	// fan out jobs
	for _, spec := range bins {
		jobs <- spec
	}
	close(jobs)

	// fan in results
	var locked []BinaryData
	for range numBins {
		res := <-results
		if res.err != nil {
			return Lock{}, res.err
		}
		locked = append(locked, res.data)
	}
	sort.SliceStable(locked, func(i, j int) bool {
		return locked[i].Name < locked[j].Name
	})

	// calculate checksum
	digest, err := r.hash(locked)
	if err != nil {
		return Lock{}, err
	}

	return Lock{
		Generated: time.Now().UTC(),
		Digest:    digest,
		Binaries:  locked,
	}, nil
}

func (r *Resolver) resolve(ctx context.Context, bin BinarySpec) (BinaryData, error) {
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

func (r *Resolver) hash(bins []BinaryData) (string, error) {
	data, err := json.Marshal(bins)
	if err != nil {
		return "", err
	}
	s, err := digest(bytes.NewBuffer(data))
	if err != nil {
		return "", err
	}
	return "sha256:" + s, nil
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

func digest(in io.Reader) (string, error) {
	hash := crypto.SHA256.New()
	if _, err := io.Copy(hash, in); err != nil {
		return "", err
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}
