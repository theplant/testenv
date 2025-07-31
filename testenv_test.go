package testenv_test

import (
	"context"
	"testing"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/theplant/testenv"
	"gorm.io/gorm"
)

type TestModel struct {
	gorm.Model
	Description string
}

var (
	db          *gorm.DB
	redisClient *redis.Client
)

func TestMain(m *testing.M) {
	env, err := testenv.New().DBEnable(true).RedisEnable(true).SetUp()
	if err != nil {
		panic(err)
	}
	defer env.TearDown()

	db = env.DB
	if err = db.AutoMigrate(&TestModel{}); err != nil {
		panic(err)
	}

	redisClient = env.Redis
	ctx := context.Background()
	if err = redisClient.Ping(ctx).Err(); err != nil {
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

func TestRedisInfo(t *testing.T) {
	ctx := context.Background()

	pong := redisClient.Ping(ctx)
	require.NoError(t, pong.Err())
	assert.Equal(t, "PONG", pong.Val())

	info := redisClient.Info(ctx, "server")
	require.NoError(t, info.Err())
	assert.Contains(t, info.Val(), "redis_version")
	t.Logf("Redis server info: %s", info.Val())
}

func TestRedisBasicOperations(t *testing.T) {
	ctx := context.Background()

	key := "test_key_from_main"
	value := "test_value_from_main"

	setCmd := redisClient.Set(ctx, key, value, 0)
	require.NoError(t, setCmd.Err())

	getCmd := redisClient.Get(ctx, key)
	require.NoError(t, getCmd.Err())
	assert.Equal(t, value, getCmd.Val())

	delCmd := redisClient.Del(ctx, key)
	require.NoError(t, delCmd.Err())
	assert.Equal(t, int64(1), delCmd.Val())

	t.Logf("Redis basic operations test passed")
}
