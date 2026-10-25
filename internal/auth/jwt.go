package auth

import (
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v5"
	echojwt "github.com/labstack/echo-jwt/v5"
	"github.com/quoihv-3696/realworld-echo/internal/apperr"

	"github.com/labstack/echo/v5"
)

const (
	DefaultTokenTTL = 72 * time.Hour
	MinSecretBytes  = 32
	userIDKey       = "userID"
)

type JWT struct {
	secret []byte
	ttl    time.Duration
}

func NewJWT(secret string, ttl time.Duration) (*JWT, error) {
	if len(secret) < MinSecretBytes {
		return nil, fmt.Errorf("JWT secret phải dài ít nhất %d byte, hiện có %d", MinSecretBytes, len(secret))
	}
	return &JWT{secret: []byte(secret), ttl: ttl}, nil
}

func (j *JWT) Sign(userID uint) (string, error) {
	now := time.Now()
	claims := jwt.RegisteredClaims{
		Subject:   strconv.FormatUint(uint64(userID), 10),
		IssuedAt:  jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(now.Add(j.ttl)),
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(j.secret)
}

func (j *JWT) Required() echo.MiddlewareFunc { return j.middleware(false) }

func (j *JWT) Optional() echo.MiddlewareFunc { return j.middleware(true) }

func (j *JWT) middleware(optional bool) echo.MiddlewareFunc {
	return echojwt.WithConfig(echojwt.Config{
		SigningKey:    j.secret,
		TokenLookup:   "header:Authorization:Token ",
		NewClaimsFunc: func(c *echo.Context) jwt.Claims { return new(jwt.RegisteredClaims) },
		SuccessHandler: func(c *echo.Context) error {
			token, ok := c.Get("user").(*jwt.Token)
			if !ok {
				return apperr.TokenInvalid()
			}
			claims, ok := token.Claims.(*jwt.RegisteredClaims)
			if !ok {
				return apperr.TokenInvalid()
			}
			id, err := strconv.ParseUint(claims.Subject, 10, 64)
			if err != nil || id == 0 {
				return apperr.TokenInvalid()
			}
			SetUserID(c, uint(id))
			return nil
		},
		ErrorHandler: func(c *echo.Context, err error) error {
			if optional && errors.Is(err, echojwt.ErrJWTMissing) {
				return nil
			}
			if errors.Is(err, echojwt.ErrJWTMissing) {
				return apperr.TokenMissing()
			}
			return apperr.TokenInvalid()
		},
		ContinueOnIgnoredError: optional,
	})
}

func SetUserID(c *echo.Context, id uint) {
	c.Set(userIDKey, id)
}

func UserID(c *echo.Context) (uint, bool) {
	id, err := echo.ContextGet[uint](c, userIDKey)
	return id, err == nil
}
