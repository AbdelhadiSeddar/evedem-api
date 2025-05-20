package server

import (
	"evedem_api/internal/commons"

	"github.com/google/uuid"
)

// / Possible returns
// / nil -> Valid Authkey
// / ApiError -> Error where ApiError.Indefier is returned
func (c *Controller) AuthVerif(authkey uuid.UUID) *commons.ApiError {

	if commons.DebugMode {
		return nil
	}
	stat, err := c.SessionCheck(authkey)
  if *stat == CACHE_SUCCESS {
    return nil
  } else 
  {
    if err == nil {
      return &commons.ApiError{
        Error: commons.ERR_INTERNAL_TRYAGAIN,
      }
    }
    return err
  }
}
