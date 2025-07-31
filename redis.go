package testenv

import (
	"cmp"
	"context"
	"fmt"
	"strings"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/go-connections/nat"
	"github.com/redis/go-redis/v9"
	"github.com/testcontainers/testcontainers-go"
	testredis "github.com/testcontainers/testcontainers-go/modules/redis"
)

func SetupRedis(ctx context.Context, image, hostPort string) (_ *redis.Client, _ func() error, xerr error) {
	image = cmp.Or(image, "redis:8.0-M04-alpine")

	var opts []testcontainers.ContainerCustomizer
	if hostPort != "" {
		opts = append(opts, testcontainers.WithHostConfigModifier(func(hostConfig *container.HostConfig) {
			hostConfig.PortBindings = map[nat.Port][]nat.PortBinding{
				"6379/tcp": {
					{
						HostIP:   "0.0.0.0",
						HostPort: hostPort,
					},
				},
			}
		}))
	}

	container, err := testredis.Run(ctx,
		image,
		opts...,
	)
	if err != nil {
		return nil, nil, fmt.Errorf("fail to start container: %w", err)
	}
	defer func() {
		if xerr != nil {
			container.Terminate(context.Background())
		}
	}()

	endpoint, err := container.ConnectionString(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("fail to get endpoint: %w", err)
	}

	client := redis.NewClient(&redis.Options{
		Addr: strings.TrimPrefix(endpoint, "redis://"),
	})

	return client, func() error {
		return cmp.Or(
			client.Close(),
			container.Terminate(context.Background()),
		)
	}, nil
}

func (b *Builder) RedisEnable(v bool) *Builder {
	b.redisEnable = v
	return b
}

func (b *Builder) RedisImage(v string) *Builder {
	b.redisImage = v
	return b
}

func (b *Builder) RedisPort(v string) *Builder {
	b.redisPort = v
	return b
}
