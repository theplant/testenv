package testenv

import (
	"cmp"
	"context"
	"errors"
	"os"
	"sync/atomic"
	"testing"

	"golang.org/x/sync/errgroup"
	"gorm.io/gorm"
)

type Builder struct {
	ctx context.Context

	dbEnable               bool
	dbUser, dbPass, dbName string
	// dbPort is the dbPort that the database listens on.
	// this is the parameter to be passed to the `docker run -p dbPort:5432` command.
	// if not set or set to "0", a random dbPort will be assigned.
	dbPort string
}

func New() *Builder {
	return &Builder{}
}

func (b *Builder) Context(ctx context.Context) *Builder {
	b.ctx = ctx
	return b
}

type TestEnv struct {
	DB       *gorm.DB
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
	EnvDBPort = "THEPLANT_TEST_ENV_DB_PORT"
)

func (b *Builder) SetUp() (*TestEnv, error) {
	ctx := cmp.Or(b.ctx, context.Background())

	env := &TestEnv{}

	var closers []func() error
	if b.dbEnable {
		db, dbCloser, err := setupDatabase(ctx,
			cmp.Or(b.dbUser, "test_user"),
			cmp.Or(b.dbPass, "test_pass"),
			cmp.Or(b.dbName, "test_db"),
			cmp.Or(b.dbPort, os.Getenv(EnvDBPort), "0"),
		)
		if err != nil {
			return nil, err
		}
		env.DB = db
		closers = append(closers, dbCloser)
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
