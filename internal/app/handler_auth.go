package app

import (
	"log"
	"net/http"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v5"
)

type LoginPayload struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func (h *Handler) LoginHandler(c *echo.Context) error {
	var loginVals LoginPayload
	if err := c.Bind(&loginVals); err != nil {
		return echo.ErrBadRequest
	}

	username := loginVals.Username
	password := loginVals.Password

	cfgUser, err := h.sm.ValidateUserAndPassword(username, password)
	if err != nil {
		return echo.ErrUnauthorized
	}

	isAdmin := cfgUser.Role == "admin"
	isMod := cfgUser.Role == "moderator" || isAdmin
	isRO := cfgUser.Role == "read_only" || isMod

	cfg := h.sm.GetAuthConfig()

	to := time.Duration(0)
	if cfg.Timeout != nil {
		to = *cfg.Timeout
	}
	expireAt := time.Now().Add(to)

	u := &User{
		UserName: cfgUser.Username,
		Role:     cfgUser.Role,
		Admin:    isAdmin,
		Mod:      isMod,
		ReadOnly: isRO,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expireAt),
		},
	}

	privateKey, err := os.ReadFile(cfg.PrivateKeyPath)
	if err != nil {
		log.Fatal("Could not read private key:", err)
	}

	signingKey, err := jwt.ParseRSAPrivateKeyFromPEM(privateKey)
	if err != nil {
		return err
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, u)
	t, err := token.SignedString(signingKey)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, map[string]any{
		"code":      http.StatusOK,
		"token":     t,
		"expire":    expireAt.Format(time.RFC3339),
		"user_name": u.UserName,
		"role":      u.Role,
		"admin":     u.Admin,
		"mod":       u.Mod,
		"read_only": u.ReadOnly,
	})
}
