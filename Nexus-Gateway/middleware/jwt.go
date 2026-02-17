package middleware

import (
	"context"
	"crypto"
	"crypto/ed25519"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"log"
	"sync"

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

// KeyManager JWKS 密钥管理器，支持多 kid 并发安全查找
type KeyManager struct {
	mu   sync.RWMutex
	keys map[string]keyEntry // kid → parsed key + algorithm
}

type keyEntry struct {
	pubKey    crypto.PublicKey
	algorithm string // RS256 / EdDSA
}

// NewKeyManager 创建密钥管理器
func NewKeyManager() *KeyManager {
	return &KeyManager{
		keys: make(map[string]keyEntry),
	}
}

// UpdateKeys 从配置更新密钥集合（原子替换）
func (km *KeyManager) UpdateKeys(items []config.JWKItem) {
	newKeys := make(map[string]keyEntry, len(items))
	for _, item := range items {
		pub, err := parsePublicKey(item.PublicKey, item.Algorithm)
		if err != nil {
			log.Printf("[nexus-gateway] skip key kid=%s: %v", item.KID, err)
			continue
		}
		newKeys[item.KID] = keyEntry{pubKey: pub, algorithm: item.Algorithm}
	}

	km.mu.Lock()
	km.keys = newKeys
	km.mu.Unlock()

	log.Printf("[nexus-gateway] JWT keys updated, %d key(s) loaded", len(newKeys))
}

// keyFunc 返回 jwt.Keyfunc，根据 token header 中的 kid 查找公钥
func (km *KeyManager) keyFunc(token *jwt.Token) (interface{}, error) {
	// 从 token header 获取 kid
	kidRaw, ok := token.Header["kid"]
	if !ok {
		return nil, fmt.Errorf("token missing kid in header")
	}
	kid, ok := kidRaw.(string)
	if !ok {
		return nil, fmt.Errorf("token kid is not a string")
	}

	km.mu.RLock()
	entry, exists := km.keys[kid]
	km.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("unknown kid: %s", kid)
	}

	// 验证签名方法与密钥算法匹配
	switch entry.algorithm {
	case "RS256":
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("kid %s expects RS256, got %v", kid, token.Header["alg"])
		}
	case "EdDSA":
		if token.Method.Alg() != "EdDSA" {
			return nil, fmt.Errorf("kid %s expects EdDSA, got %v", kid, token.Header["alg"])
		}
	default:
		return nil, fmt.Errorf("unsupported algorithm for kid %s: %s", kid, entry.algorithm)
	}

	return entry.pubKey, nil
}

// JWT 校验中间件，支持 JWKS 多密钥 + 动态配置热更新
func JWT(holder *config.DynamicConfigHolder, km *KeyManager) ghttp.HandlerFunc {
	return func(r *ghttp.Request) {
		cfg := holder.Load().JWT

		// JWT 未启用
		if !cfg.Enabled {
			r.Middleware.Next()
			return
		}

		// 跳过的路径
		for _, p := range cfg.SkipPaths {
			if r.URL.Path == p {
				r.Middleware.Next()
				return
			}
		}

		// 提取 Bearer token
		tokenStr := extractBearerToken(r)
		if tokenStr == "" {
			internal.GatewayError(r, internal.CodeJWTInvalid, "missing authorization token")
			return
		}

		// 解析并校验（通过 kid 查找公钥）
		token, err := jwt.Parse(tokenStr, km.keyFunc)
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

// parsePublicKey 解析 PEM 编码的公钥（支持 RSA 和 Ed25519）
func parsePublicKey(pemStr string, algorithm string) (crypto.PublicKey, error) {
	block, _ := pem.Decode([]byte(pemStr))
	if block == nil {
		return nil, fmt.Errorf("failed to decode PEM block")
	}

	pub, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("parse public key: %w", err)
	}

	switch algorithm {
	case "RS256":
		rsaPub, ok := pub.(*rsa.PublicKey)
		if !ok {
			return nil, fmt.Errorf("key is not RSA public key")
		}
		return rsaPub, nil
	case "EdDSA":
		edPub, ok := pub.(ed25519.PublicKey)
		if !ok {
			return nil, fmt.Errorf("key is not Ed25519 public key")
		}
		return edPub, nil
	default:
		return nil, fmt.Errorf("unsupported algorithm: %s", algorithm)
	}
}

func extractBearerToken(r *ghttp.Request) string {
	auth := r.Header.Get("Authorization")
	const prefix = "Bearer "
	if len(auth) > len(prefix) && auth[:len(prefix)] == prefix {
		return auth[len(prefix):]
	}
	return ""
}
