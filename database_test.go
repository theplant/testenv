package testenv

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// Test that SetupDatabase starts a PostgreSQL instance and its DSN is usable.
func TestSetupDatabase_DSNConnects(t *testing.T) {
	ctx := context.Background()

	out, err := SetupDatabase(ctx, &SetupDatabaseInput{})
	require.NoError(t, err)
	require.NotNil(t, out)

	// Ensure resources are cleaned up after the test.
	t.Cleanup(func() { _ = out.Closer() })

	require.NotNil(t, out.DB)
	require.NotEmpty(t, out.DSN)

	// Open a new, independent connection using the returned DSN to validate connectivity.
	newDB, err := gorm.Open(postgres.Open(out.DSN), &gorm.Config{})
	require.NoError(t, err)

	var one int
	require.NoError(t, newDB.WithContext(ctx).Raw("SELECT 1").Scan(&one).Error)
	assert.Equal(t, 1, one)
}
