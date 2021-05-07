package usecase

import (
	"github.com/bulletmys/proceedd/server/kv/internal/models"
	"github.com/bulletmys/proceedd/server/kv/internal/repository"
)

type KVUseCase struct {
	Repo repository.Repository
}

func (uc KVUseCase) GetFullConfig(timestamp int64) (models.KVFull, error) {
	return uc.Repo.GetFullConfig(timestamp)
}

func (uc KVUseCase) GetDiffConfig(version uint64) (models.KVDiff, error) {
	return uc.Repo.GetDiffConfig(version)
}

func (uc KVUseCase) UpdConfig(data map[string][]string, jsonData models.KVDiff) error {
	return uc.Repo.UpdConfig(data, jsonData)
}
