package testenv

import (
	"cmp"
	"context"
	"errors"
	"sync/atomic"
	"testing"

	"golang.org/x/sync/errgroup"
	"gorm.io/gorm"
)

type TestEnvBuilder struct {
	ctx context.Context

	dbEnable               bool
	dbUser, dbPass, dbName string
}

func New() *TestEnvBuilder {
	return &TestEnvBuilder{}
}

func (b *TestEnvBuilder) Context(ctx context.Context) *TestEnvBuilder {
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

func (b *TestEnvBuilder) SetUp() (*TestEnv, error) {
	ctx := cmp.Or(b.ctx, context.Background())

	env := &TestEnv{}

	closers := []func() error{}
	if b.dbEnable {
		db, dbCloser, err := setupDatabase(ctx,
			cmp.Or(b.dbUser, "test_user"),
			cmp.Or(b.dbPass, "test_pass"),
			cmp.Or(b.dbName, "test_db"),
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

func (b *TestEnvBuilder) SetUpWithT(t *testing.T) (*TestEnv, error) {
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
