package delivery

import (
	"fmt"
	"github.com/bulletmys/proceedd/server/kv/internal/models"
	"github.com/bulletmys/proceedd/server/kv/internal/usecase"
	jsoniter "github.com/json-iterator/go"
	"github.com/labstack/echo/v4"
	"net/http"
	"strconv"
)

var json = jsoniter.ConfigCompatibleWithStandardLibrary

type KVDelivery struct {
	uc usecase.UseCase
}

func NewKVDelivery(uc usecase.UseCase) KVDelivery {
	return KVDelivery{uc: uc}
}

func (d KVDelivery) UpdConfig(c echo.Context) error {
	var data models.KVDiff
	// TODO bug with interpreting data ({"data":11111111} == 1.1111111+e7)
	if err := c.Bind(&data); err != nil {
		return c.JSON(http.StatusBadRequest, models.Error{Err: fmt.Sprintf("failed to parse json: %v", err)})
	}
	err := d.uc.UpdConfig(c.QueryParams(), data)
	switch err {
	case models.VersionNotModified:
		return c.NoContent(http.StatusNotModified)
	case nil:
	default:
		return c.JSON(http.StatusBadRequest, models.Error{Err: err.Error()})
	}

	return c.NoContent(http.StatusOK)
}

func (d KVDelivery) GetFullConfig(c echo.Context) error {
	timestampParam := c.QueryParam("ts")
	timestamp, err := strconv.ParseInt(timestampParam, 10, 64)
	if err != nil {
		return c.JSON(http.StatusBadRequest, models.Error{Err: "bad timestamp param"})
	}

	fullConfig, err := d.uc.GetFullConfig(timestamp)
	switch err {
	case models.VersionNotModified:
		return c.NoContent(http.StatusNotModified)
	case nil:
	default:
		return c.JSON(http.StatusBadRequest, models.Error{Err: err.Error()})
	}

	c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
	c.Response().WriteHeader(http.StatusOK)
	return json.NewEncoder(c.Response()).Encode(fullConfig)
}
//
//// Not implemented yet
//func (d KVDelivery) GetDiffConfig(c echo.Context) error {
//	versionParam := c.QueryParam("version")
//	version, err := strconv.ParseUint(versionParam, 10, 64)
//	if err != nil {
//		return c.JSON(http.StatusBadRequest, models.Error{Err: "bad version param"})
//	}
//
//	diffConfig, err := d.uc.GetDiffConfig(version)
//	switch err {
//	case models.VersionNotModified:
//		return c.NoContent(http.StatusNotModified)
//	case models.OldVersionRequested:
//		return c.JSON(http.StatusNotFound, models.Error{Err: "requested version is too old"})
//	case nil:
//	default:
//		return c.JSON(http.StatusBadRequest, models.Error{Err: err.Error()})
//	}
//
//	c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
//	c.Response().WriteHeader(http.StatusOK)
//	return json.NewEncoder(c.Response()).Encode(diffConfig)
//}
