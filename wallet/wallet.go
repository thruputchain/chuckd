// Package wallet implements a wallet API provided by the validator.
package wallet

import (
	"context"
	"crypto/sha256"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/groundhogchain/chuckd/kit"
	"github.com/groundhogchain/chuckd/lib/liblog"
	"github.com/groundhogchain/chuckd/util/xdrutil"
	proto "github.com/groundhogchain/protocol-go"
)

type WalletGateway struct {
	svc    *WalletService
	jwtKey proto.JWTSecretKey
	r      *gin.Engine
	lk     sync.Mutex
	user   map[string]proto.WalletUser
}

func init() {
	gin.SetMode(gin.ReleaseMode)
}

func NewWalletGateway(
	ctx context.Context,
	submitter kit.SignedTxSubmitter,
	jwtKey proto.JWTSecretKey,
	users []proto.WalletUser,
) *WalletGateway {

	r := gin.Default()
	g := &WalletGateway{
		svc:    NewWalletService(ctx, submitter),
		jwtKey: jwtKey,
		r:      r,
		user:   map[string]proto.WalletUser{},
	}
	for _, u := range users {
		g.user[u.Login] = u
	}

	// public routes
	r.POST("/login", g.login)

	// authenticated routes
	auth := r.Group("/wallet")
	auth.Use(g.authMiddleware)
	auth.POST("/submit", g.submitEndpoint)

	return g
}

func (g *WalletGateway) Service() proto.Wallet1 {
	return g.svc
}

func (g *WalletGateway) StartHTTP(_ context.Context, address string) {
	go func() {
		if err := g.r.Run(address); err != nil {
			liblog.Fatalf("start wallet gateway http server: %v", err)
		}
	}()
}

// User structure passed over HTTP on login
type User struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

func (g *WalletGateway) login(c *gin.Context) {
	var user User
	if err := c.ShouldBindJSON(&user); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	// Check if user exists and password is correct
	walletUser, exists := g.user[user.Login]
	if !exists || walletUser.Password_hash != sha256.Sum256([]byte(user.Password)) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
		return
	}

	// Create token
	expirationTime := time.Now().Add(24 * time.Hour)
	claims := &JWTClaim{
		Login: user.Login,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
			Issuer:    "chuckd-wallet-jwt-auth",
			Subject:   user.Login,
		},
	}

	// Generate token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(g.jwtKey)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate token"})
		return
	}

	// Return token
	c.JSON(http.StatusOK, gin.H{
		"token": tokenString,
	})
}

func (g *WalletGateway) authMiddleware(c *gin.Context) {

	// Get token from Authorization header
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "authorization header required"})
		return
	}

	// Extract token (remove "Bearer " prefix if present)
	tokenString := authHeader
	if strings.HasPrefix(tokenString, "Bearer ") {
		tokenString = strings.TrimPrefix(tokenString, "Bearer ")
	}

	// Parse and validate token
	token, err := jwt.ParseWithClaims(tokenString, &JWTClaim{},
		func(token *jwt.Token) (interface{}, error) {
			return g.jwtKey, nil
		},
	)

	if err != nil || !token.Valid {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired token"})
		return
	}

	// Get claims
	claims, ok := token.Claims.(*JWTClaim)
	if !ok {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "failed to parse claims"})
		return
	}

	// Add username to context
	c.Set("login", claims.Login)

	// Continue to the next handler
	c.Next()
}

func (g *WalletGateway) submitEndpoint(c *gin.Context) {

	// Get username from context
	_, ok := c.Get("login")
	if !ok {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "not logged in"})
		return
	}

	var walletSubmit proto.WalletSubmit

	postData, err := c.GetRawData()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to read request body"})
		return
	}

	// We use JsonToXdr, because xdr unions cannot unmarshal using the standard golang JSON mechanism.
	err = xdrutil.JsonToXdr(&walletSubmit, postData)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid json/xdr format"})
		return
	}

	ack, err := g.Service().Wallet_submit(c.Request.Context(), walletSubmit)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "submission error"})
		return
	}

	// We use XdrToJson, because xdr unions cannot serialize using the standard golang JSON mechanism.
	postReply, err := xdrutil.XdrToJson(&ack)
	if err != nil {
		c.Status(http.StatusInternalServerError)
		return
	}

	c.Header("Content-Type", "application/json")
	c.Data(http.StatusOK, "application/json", postReply)
}
