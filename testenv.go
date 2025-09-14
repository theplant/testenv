package testenv

import (
	"cmp"
	"context"
	"errors"
	"os"
	"sync/atomic"
	"testing"

	"github.com/redis/go-redis/v9"
	"golang.org/x/sync/errgroup"
	"gorm.io/gorm"
)

type Builder struct {
	ctx context.Context

	dbEnable                                bool
	dbImage, dbUser, dbPass, dbName, dbPort string

	redisEnable           bool
	redisImage, redisPort string
}

func New() *Builder {
	return &Builder{}
}

func (b *Builder) Context(ctx context.Context) *Builder {
	b.ctx = ctx
	return b
}

type DBEnv struct {
	DB  *gorm.DB
	DSN string
}

type TestEnv struct {
	DBEnv    *DBEnv
	Redis    *redis.Client
	tearDown func() error
	tornDown atomic.Bool
}

func (env *TestEnv) TearDown() error {
	if !env.tornDown.CompareAndSwap(false, true) {
		return errors.New("torn down")
	}
	return env.tearDown()
}

const (
	EnvDBPort    = "THEPLANT_TEST_ENV_DB_PORT"
	EnvRedisPort = "THEPLANT_TEST_ENV_REDIS_PORT"
)

func (b *Builder) SetUp() (*TestEnv, error) {
	ctx := cmp.Or(b.ctx, context.Background())

	env := &TestEnv{}

	var closers []func() error
	if b.dbEnable {
		output, err := SetupDatabase(ctx, &SetupDatabaseInput{
			Image:        b.dbImage,
			User:         cmp.Or(b.dbUser, "test_user"),
			Pass:         cmp.Or(b.dbPass, "test_pass"),
			DatabaseName: cmp.Or(b.dbName, "test_db"),
			HostPort:     cmp.Or(b.dbPort, os.Getenv(EnvDBPort), ""),
		})
		if err != nil {
			return nil, err
		}
		env.DBEnv = &DBEnv{
			DB:  output.DB,
			DSN: output.DSN,
		}
		closers = append(closers, output.Closer)
	}

	if b.redisEnable {
		redis, redisCloser, err := SetupRedis(ctx,
			cmp.Or(b.redisImage, ""),
			cmp.Or(b.redisPort, os.Getenv(EnvRedisPort), ""),
		)
		if err != nil {
			return nil, err
		}
		env.Redis = redis
		closers = append(closers, redisCloser)
	}

	env.tearDown = func() error {
		var errG errgroup.Group
		for _, f := range closers {
			errG.Go(f)
		}
		return errG.Wait()
	}
	return env, nil
}

func (b *Builder) SetUpWithT(t *testing.T) (*TestEnv, error) {
	env, err := b.SetUp()
	if err != nil {
		return nil, err
	}
	t.Cleanup(func() {
		if err := env.TearDown(); err != nil {
			t.Logf("fail to tear down: %v", err)
		}
	})
	return env, nil
}
