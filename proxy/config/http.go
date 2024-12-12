package config

import (
	"context"
	"fmt"
	"io"

	"github.com/caarlos0/env/v11"
	"oras.land/oras-go/v2/registry/remote"
	"oras.land/oras-go/v2/registry/remote/auth"
	"oras.land/oras-go/v2/registry/remote/retry"
)

const (
	tag       = "latest"
	chunkSize = 1024 * 1024
)

type ChunkPayload struct {
	AppName     string `json:"app_name"`
	ChunkIdx    int    `json:"chunk_idx"`
	TotalChunks int    `json:"total_chunks"`
	Data        []byte `json:"data"`
}

type HTTPProxyConfig struct {
	RegistryURL  string `env:"REGISTRY_URL" envDefault:"localhost:5000"`
	Authenticate bool   `env:"AUTHENTICATE" envDefault:"false"`
	Username     string `env:"USERNAME"     envDefault:""`
	Password     string `env:"PASSWORD"     envDefault:""`
}

func LoadHTTPConfig(opts env.Options) (*HTTPProxyConfig, error) {
	config := HTTPProxyConfig{}
	if err := env.ParseWithOptions(&config, opts); err != nil {
		return nil, err
	}

	return &config, nil
}

func (c *HTTPProxyConfig) FetchFromReg(ctx context.Context, containerName string) ([]ChunkPayload, error) {
	fullPath := fmt.Sprintf("%s/%s", c.RegistryURL, containerName)

	repo, err := remote.NewRepository(fullPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create repository for %s: %w", containerName, err)
	}

	if c.Authenticate {
		repo.Client = &auth.Client{
			Client: retry.DefaultClient,
			Cache:  auth.NewCache(),
			Credential: auth.StaticCredential(c.RegistryURL, auth.Credential{
				Username: c.Username,
				Password: c.Password,
			}),
		}
	}

	descriptor, err := repo.Resolve(ctx, tag)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve manifest for %s: %w", containerName, err)
	}

	reader, err := repo.Fetch(ctx, descriptor)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch blob for %s: %w", containerName, err)
	}
	defer reader.Close()

	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to read blob for %s: %w", containerName, err)
	}

	totalChunks := (len(data) + chunkSize - 1) / chunkSize

	chunks := make([]ChunkPayload, 0, totalChunks)
	for i := 0; i < totalChunks; i++ {
		start := i * chunkSize
		end := start + chunkSize
		if end > len(data) {
			end = len(data)
		}

		chunk := ChunkPayload{
			AppName:     containerName,
			ChunkIdx:    i,
			TotalChunks: totalChunks,
			Data:        data[start:end],
		}
		chunks = append(chunks, chunk)
	}

	return chunks, nil
}
