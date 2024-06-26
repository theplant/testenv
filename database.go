package testenv

import (
	"cmp"
	"context"
	"fmt"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func (b *TestEnvBuilder) DBEnable(v bool) *TestEnvBuilder {
	b.dbEnable = v
	return b
}

func (b *TestEnvBuilder) DBUser(v string) *TestEnvBuilder {
	b.dbUser = v
	return b
}

func (b *TestEnvBuilder) DBPass(v string) *TestEnvBuilder {
	b.dbPass = v
	return b
}

func (b *TestEnvBuilder) DBName(v string) *TestEnvBuilder {
	b.dbName = v
	return b
}

func setupDatabase(ctx context.Context, dbUser, dbPass, dbName string) (_ *gorm.DB, _ func() error, xerr error) {
	container, err := testcontainers.GenericContainer(ctx,
		testcontainers.GenericContainerRequest{
			ContainerRequest: testcontainers.ContainerRequest{
				Image:        "postgres:16.3-alpine",
				ExposedPorts: []string{"5432/tcp"},
				Env: map[string]string{
					"POSTGRES_USER":     dbUser,
					"POSTGRES_PASSWORD": dbPass,
					"POSTGRES_DB":       dbName,
				},
				WaitingFor: wait.ForLog("database system is ready to accept connections").WithOccurrence(2),
			},
			Started: true,
		},
	)
	if err != nil {
		return nil, nil, fmt.Errorf("fail to start container: %w", err)
	}
	defer func() {
		if xerr != nil {
			container.Terminate(context.Background())
		}
	}()

	endpoint, err := container.Endpoint(ctx, "")
	if err != nil {
		return nil, nil, fmt.Errorf("fail to get endpoint: %w", err)
	}
	dsn := fmt.Sprintf("postgres://%s:%s@%s/%s?sslmode=disable", dbUser, dbPass, endpoint, dbName)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, nil, err
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, nil, fmt.Errorf("no underlying sqlDB: %w", err)
	}

	return db, func() error {
		return cmp.Or(
			sqlDB.Close(),
			container.Terminate(context.Background()),
		)
	}, nil
}
