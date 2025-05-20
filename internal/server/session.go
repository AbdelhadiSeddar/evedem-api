package server

import (
	"context"
	"evedem_api/internal/commons"
	"evedem_api/internal/database"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

func (s *Controller) SessionRegister(email string, password string) (*uuid.UUID, *commons.ApiError) {
	// Handle Req

	db := database.GetDBConn()

	query := `
SELECT * FROM public.UserCheck(
  p_email     := $1,
  p_password  := $2
);
`

	var userExists bool
	var isBanned pgtype.Bool
	var userId pgtype.Int4
	var isAdmin pgtype.Bool
	var adminId pgtype.Int4

	rows, err := db.DB.Query(
		context.Background(),
		query,
		email,
		password,
	)
	if err != nil {
		db.Release()
		return nil, &commons.ApiError{
			Error:     commons.ERR_INTERNAL_DB_FAIL,
			Errorinfo: err,
		}
	}
	defer rows.Close()
	if rows.Next() {
		err := rows.Scan(&userExists, &isBanned, &userId, &isAdmin, &adminId)
		if err != nil {
			db.Release()
			return nil, &commons.ApiError{
				Error:     commons.ERR_INTERNAL_DB_FAIL,
				Errorinfo: err,
			}
		}
	} else {
		db.Release()
		return nil, &commons.ApiError{
			Error:     commons.ERR_INTERNAL_DB_FAIL,
			Errorinfo: rows.Err().Error(),
		}
	}
	db.Release()
	if userExists {
		if isBanned.Bool {
			return nil, &commons.ApiError{
				Error:     commons.ERR_AUTH_DEAD,
				Errorinfo: nil,
			}
		}
		var idses = uuid.New()


    log.Print(" sds ") 
    u, _ := userId.Value()
    log.Println(int(u.(int64)))
		s.Ac.Set(idses, AuthCacheType{
			userid:        int(u.(int64)),
			date_creation: time.Now(),
		})

		return &idses, nil
	} else {
		return nil, nil
	}
}

func (s *Controller) SessionCheck(authkey uuid.UUID) (*CacheStatus, *commons.ApiError) {
	if commons.DebugMode {
		var stat CacheStatus = CACHE_SUCCESS
		return &stat, nil
	}

	var stat CacheStatus = CACHE_SUCCESS
	id, err := s.SessionGetUserUID(authkey)
	if id != nil {
		return &stat, nil
	}
	if err == nil {
		stat = CACHE_MISS
		return &stat, nil
	}
	switch err.Error {
	case commons.ERR_AUTH_DEAD:
		stat = CACHE_EXPIRED
		return &stat, err
	case commons.ERR_AUTH_INVALID:
		stat = CACHE_MISS
		return &stat, err
	default:
		return nil, err
	}
}

func (s *Controller) SessionGetUserUID(authkey uuid.UUID) (*int, *commons.ApiError) {
	re := -1
	if commons.DebugMode {
		return &re, nil
	}

	stat := s.Ac.Get(authkey)

	switch stat {
	case CACHE_EXPIRED:
		return nil, &commons.ApiError{
			Error: commons.ERR_AUTH_DEAD,
		}
	case CACHE_MISS:
		return nil, &commons.ApiError{
			Error: commons.ERR_AUTH_INVALID,
		}
	case CACHE_SUCCESS:
		re := s.Ac.cache[authkey].userid
		return &re, nil
	default:
	}

	return nil, &commons.ApiError{
		Error: commons.ERR_INTERNAL_TRYAGAIN,
	}
}

func (s *Controller) SessionRevoke(authkey uuid.UUID) *commons.ApiError {
	ret := s.Ac.Remove(authkey)
	if ret != CACHE_SUCCESS {
		return &commons.ApiError{
			Error: commons.ERR_AUTH_INVALID,
		}
	}
	return nil
}
