package api

import (
  "github.com/labstack/echo/v4"
  "github.com/arnokay/arnobot-shared/controllers/echo"
)

type Controllers struct {
	ProviderController controllers.Controller
}

func (c Controllers) Routes(group *echo.Group) {
  c.ProviderController.Routes(group)
}
