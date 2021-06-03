package kv

import (
	"fmt"
	"github.com/bulletmys/proceedd/server/kv/internal/delivery"
	"github.com/bulletmys/proceedd/server/kv/internal/repository"
	"github.com/bulletmys/proceedd/server/kv/internal/usecase"
	"github.com/golobby/config/v2"
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

func Start(c *config.Config) error {
	kvDelivery := InitKV()

	port, err := c.GetInt("kv.port")
	if err != nil {
		log.Fatalf("failed to parse config: %v", err)
	}

	e := echo.New()

	e.Use(middleware.Logger())
	e.Use(middleware.Recover())

	e.GET("/config/full", kvDelivery.GetFullConfig)
	//e.GET("/config/diff", kvDelivery.GetDiffConfig)
	e.POST("/config", kvDelivery.UpdConfig)

	// Start server
	e.Logger.Fatal(e.Start(fmt.Sprintf(":%v", port)))

	return nil
}
