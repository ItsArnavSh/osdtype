//go:build integration

// This file is compiled only for integration-tagged builds. It gives the
// integration tests a way to seed and inspect rows without widening the
// production API surface.
package postgresql

import (
	"context"

	"gorm.io/gorm"
)

// RawDB exposes the underlying gorm handle for assertions and fixtures.
func (d *Database) RawDB() *gorm.DB { return d.db }

// CountRows returns the number of rows matching a model and a where clause.
// It reports an error rather than a zero count so a broken query in a test is
// not mistaken for an empty table.
func (d *Database) CountRows(ctx context.Context, model any, query string, args ...any) (int64, error) {
	var n int64
	err := d.db.WithContext(ctx).Model(model).Where(query, args...).Count(&n).Error
	return n, err
}
