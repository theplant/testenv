package testenv_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/theplant/testenv"
	"gorm.io/gorm"
)

type TestModel struct {
	gorm.Model
	Description string
}

var db *gorm.DB

func TestMain(m *testing.M) {
	env, err := testenv.New().DBEnable(true).SetUp()
	if err != nil {
		panic(err)
	}
	defer env.TearDown()

	// some initialization
	db = env.DB
	if err = db.AutoMigrate(&TestModel{}); err != nil {
		panic(err)
	}

	m.Run()
}

func TestSelectVersion(t *testing.T) {
	var version string
	if err := db.Raw("SELECT version()").Scan(&version).Error; err != nil {
		t.Fatal(err)
	}
	t.Logf("current database version: %q", version)
}

func TestSetupTestEnv(t *testing.T) {
	ctx := context.Background()

	// If you don't want to initialize in TestMain
	env, err := testenv.New().DBEnable(true).RedisEnable(true).SetUpWithT(t)
	if err != nil {
		t.Fatal(err)
	}
	var version string
	if err := env.DB.WithContext(ctx).Raw("SELECT version()").Scan(&version).Error; err != nil {
		t.Fatal(err)
	}
	t.Logf("current database version: %q", version)
	assert.Contains(t, version, "PostgreSQL 16.3")

	{
		cmd := env.Redis.Set(ctx, "test", "test", 0)
		require.NoError(t, cmd.Err())
	}

	{
		cmd := env.Redis.Get(ctx, "test")
		require.NoError(t, cmd.Err())
		assert.Equal(t, "test", cmd.Val())
	}
}
