package kv

import (
	"github.com/bulletmys/proceedd/server/kv/internal/delivery"
	"github.com/bulletmys/proceedd/server/kv/internal/repository"
	"github.com/bulletmys/proceedd/server/kv/internal/usecase"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"log"
)

func InitKV() delivery.KVDelivery {
	repo, err := repository.NewKVMapRepository()
	if err != nil {
		log.Fatalf("failed to init repository: %v", err)
	}
	uc := usecase.KVUseCase{Repo: repo}

	return delivery.NewKVDelivery(uc)
}

func Start() error {

	kvDelivery := InitKV()

	e := echo.New()

	e.Use(middleware.Logger())
	e.Use(middleware.Recover())

	e.GET("/config/full", kvDelivery.GetFullConfig)
	//e.GET("/config/diff", kvDelivery.GetDiffConfig)
	e.POST("/config", kvDelivery.UpdConfig)

	// Start server
	// TODO make config
	e.Logger.Fatal(e.Start(":8080"))

	return nil
}
