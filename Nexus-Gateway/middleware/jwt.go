package middleware

import (
	"context"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"
	"strings"

	"github.com/gogf/gf/v2/net/ghttp"
	"github.com/golang-jwt/jwt/v5"

	"github.com/krustd/nexus-gateway/config"
	"github.com/krustd/nexus-gateway/internal"
)

type contextKeyType string

const userClaimsKey contextKeyType = "user_claims"

// GetUserClaims 从 context 获取 JWT claims
func GetUserClaims(ctx context.Context) jwt.MapClaims {
	if v, ok := ctx.Value(userClaimsKey).(jwt.MapClaims); ok {
		return v
	}
	return nil
}

// JWT 校验中间件，提取用户身份并透传到下游
func JWT(cfg config.JWTConfig) ghttp.HandlerFunc {
	if !cfg.Enabled {
		return passthrough
	}

	skipSet := make(map[string]bool, len(cfg.SkipPaths))
	for _, p := range cfg.SkipPaths {
		skipSet[p] = true
	}

	keyFunc := buildKeyFunc(cfg)

	return func(r *ghttp.Request) {
		// 跳过的路径
		if skipSet[r.URL.Path] {
			r.Middleware.Next()
			return
		}

		// 提取 Bearer token
		tokenStr := extractBearerToken(r)
		if tokenStr == "" {
			internal.GatewayError(r, internal.CodeJWTInvalid, "missing authorization token")
			return
		}

		// 解析并校验
		token, err := jwt.Parse(tokenStr, keyFunc)
		if err != nil || !token.Valid {
			msg := "invalid token"
			if err != nil {
				msg = fmt.Sprintf("invalid token: %v", err)
			}
			internal.GatewayError(r, internal.CodeJWTInvalid, msg)
			return
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			internal.GatewayError(r, internal.CodeJWTInvalid, "invalid token claims")
			return
		}

		// 身份透传：写入请求头供下游读取
		if userID, ok := claims["user_id"].(string); ok {
			r.Request.Header.Set("X-User-Id", userID)
		}
		if role, ok := claims["role"].(string); ok {
			r.Request.Header.Set("X-User-Role", role)
		}

		// 存入 context
		ctx := context.WithValue(r.GetCtx(), userClaimsKey, claims)
		r.SetCtx(ctx)

		r.Middleware.Next()
	}
}

func extractBearerToken(r *ghttp.Request) string {
	auth := r.Header.Get("Authorization")
	if strings.HasPrefix(auth, "Bearer ") {
		return strings.TrimPrefix(auth, "Bearer ")
	}
	return ""
}

func buildKeyFunc(cfg config.JWTConfig) jwt.Keyfunc {
	switch strings.ToUpper(cfg.Algorithm) {
	case "RS256":
		pubKey := loadRSAPublicKey(cfg.PublicKey)
		return func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			return pubKey, nil
		}
	default: // HS256
		secret := []byte(cfg.Secret)
		return func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			return secret, nil
		}
	}
}

func loadRSAPublicKey(path string) *rsa.PublicKey {
	data, err := os.ReadFile(path)
	if err != nil {
		panic(fmt.Sprintf("nexus-gateway: read public key %s: %v", path, err))
	}
	block, _ := pem.Decode(data)
	if block == nil {
		panic("nexus-gateway: failed to decode PEM block")
	}
	pub, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		panic(fmt.Sprintf("nexus-gateway: parse public key: %v", err))
	}
	rsaPub, ok := pub.(*rsa.PublicKey)
	if !ok {
		panic("nexus-gateway: not an RSA public key")
	}
	return rsaPub
}
