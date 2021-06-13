package repository

import "github.com/bulletmys/proceedd/server/kv/internal/models"

type Repository interface {
	FullConfig(timestamp int64) (models.KVFull, error)
	//GetDiffConfig(version uint64) (models.KVDiff, error)
	UpdConfig(data map[string][]string, jsonData models.KVDiff) error
	//test()
}
