package sdkmeta

import "slices"

const (
	MigrationVersionBaseTables          int64 = 1
	MigrationVersionIndexes             int64 = 2
	MigrationVersionDropRedundantIndex  int64 = 3
	MigrationVersionMetadataFilterIndex int64 = 4

	LatestSchemaMigrationVersion = MigrationVersionMetadataFilterIndex
)

var requiredExtensionNames = []string{"pg_search", "vector"}

func RequiredExtensionNames() []string {
	return slices.Clone(requiredExtensionNames)
}
