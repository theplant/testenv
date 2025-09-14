package testenv

import (
	"cmp"
	"context"
	"fmt"
	"github.com/pkg/errors"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/go-connections/nat"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func (b *Builder) DBEnable(v bool) *Builder {
	b.dbEnable = v
	return b
}

func (b *Builder) DBImage(v string) *Builder {
	b.dbImage = v
	return b
}

func (b *Builder) DBUser(v string) *Builder {
	b.dbUser = v
	return b
}

func (b *Builder) DBPass(v string) *Builder {
	b.dbPass = v
	return b
}

func (b *Builder) DBName(v string) *Builder {
	b.dbName = v
	return b
}

func (b *Builder) DBPort(v string) *Builder {
	b.dbPort = v
	return b
}

type SetupDatabaseInput struct {
	Image        string
	User         string
	Pass         string
	DatabaseName string
	HostPort     string
}

type SetupDatabaseOutput struct {
	DB     *gorm.DB
	DSN    string
	Closer func() error
}

func SetupDatabase(ctx context.Context, input *SetupDatabaseInput) (_ *SetupDatabaseOutput, xerr error) {
	image := cmp.Or(input.Image, "postgres:17.4-alpine3.21")
	dbUser := cmp.Or(input.User, "postgres")
	dbPass := cmp.Or(input.Pass, "postgres")
	dbName := cmp.Or(input.DatabaseName, "postgres")
	hostPort := cmp.Or(input.HostPort, "")
	req := testcontainers.ContainerRequest{
		Image:        image,
		ExposedPorts: []string{"5432/tcp"},
		Env: map[string]string{
			"POSTGRES_USER":     dbUser,
			"POSTGRES_PASSWORD": dbPass,
			"POSTGRES_DB":       dbName,
		},
		Cmd:        []string{"postgres", "-c", "fsync=off"},
		WaitingFor: wait.ForLog("database system is ready to accept connections").WithOccurrence(2),
	}
	if hostPort != "" {
		req.HostConfigModifier = func(hostConfig *container.HostConfig) {
			hostConfig.PortBindings = map[nat.Port][]nat.PortBinding{
				"5432/tcp": {
					{
						HostIP:   "0.0.0.0",
						HostPort: hostPort,
					},
				},
			}
		}
	}
	_container, err := testcontainers.GenericContainer(ctx,
		testcontainers.GenericContainerRequest{
			ContainerRequest: req,
			Started:          true,
		},
	)
	if err != nil {
		return nil, errors.Wrap(err, "fail to start container")
	}
	defer func() {
		if xerr != nil {
			_ = _container.Terminate(context.Background())
		}
	}()

	endpoint, err := _container.Endpoint(ctx, "")
	if err != nil {
		return nil, errors.Wrap(err, "fail to get endpoint")
	}
	dsn := fmt.Sprintf("postgres://%s:%s@%s/%s?sslmode=disable", dbUser, dbPass, endpoint, dbName)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, errors.Wrap(err, "no underlying sqlDB")
	}

	return &SetupDatabaseOutput{
		DB:  db,
		DSN: dsn,
		Closer: func() error {
			return cmp.Or(
				sqlDB.Close(),
				_container.Terminate(context.Background()),
			)
		},
	}, nil
}
