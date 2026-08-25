package middleware

import (
	"github.com/jjcheng/wawa-go/internal/service"

	"github.com/gin-gonic/gin"
)

// 2 ways to authenticate, JWT (client web) and API Key (server-to-server)
func Authenticate(dependencies *service.Dependencies) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		// var accessToken string
		// // Check for JWT token in Authorization header (format: "Bearer <token>")
		// authHeader := ctx.GetHeader("Authorization")
		// if authHeader != "" {
		// 	if len(authHeader) > 7 && authHeader[:7] == "Bearer " {
		// 		accessToken = authHeader[7:]
		// 	}
		// }
		// // jwt authentication is used for web chatbot
		// if accessToken != "" {
		// 	// Try to validate JWT token from Authorization header
		// 	token, parseErr := jwt.Parse(accessToken, func(token *jwt.Token) (any, error) {
		// 		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
		// 			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		// 		}
		// 		return []byte(cfg.Default().Site.ServerKey), nil
		// 	})
		// 	if parseErr != nil || !token.Valid {
		// 		dependencies.Logger.Warnf("AUTH: invalid token: %v", parseErr)
		// 		responseObject := dto.NewFailedResponse[any](http.StatusUnauthorized, "invalid authorization token")
		// 		ctx.AbortWithStatusJSON(responseObject.StatusCode, responseObject)
		// 		return
		// 	}
		// 	// JWT token is valid, extract claims
		// 	claims, ok := token.Claims.(jwt.MapClaims)
		// 	if !ok {
		// 		dependencies.Logger.Warnln("AUTH: invalid token claims")
		// 		responseObject := dto.NewFailedResponse[any](http.StatusUnauthorized, "invalid token claims")
		// 		ctx.AbortWithStatusJSON(responseObject.StatusCode, responseObject)
		// 		return
		// 	}
		// 	// 1. Try to get api key
		// 	if apiKey, ok := claims["api_key"].(string); ok && apiKey != "" {
		// 		// Get user and app directly from api key
		// 		user, app, ex := dependencies.UnitOfWork.AccountUserRepository().GetByAPIKey(ctx.Request.Context(), apiKey)
		// 		if ex != nil {
		// 			dependencies.Logger.Warnf("AUTH: error getting user by api key %s: %v", apiKey, ex)
		// 		}
		// 		if user != nil && app != nil {
		// 			if !user.InUse {
		// 				dependencies.Logger.Warnf("AUTH: user %d is disabled: %s", user.Id, user.Identifier)
		// 				responseObject := dto.NewFailedResponse[any](http.StatusUnauthorized, "user is disabled")
		// 				ctx.AbortWithStatusJSON(responseObject.StatusCode, responseObject)
		// 				return
		// 			}
		// 			userDTO := dto_ai.NewUser(*user, app)
		// 			ctx.Set(cfg.Default().Site.HTTPRequestUserKey, &userDTO)
		// 			ctx.Next()
		// 			return
		// 		}
		// 		dependencies.Logger.Warnf("AUTH: user or app not found for api key: %s", apiKey)
		// 		responseObject := dto.NewFailedResponse[any](http.StatusUnauthorized, "unable to authenticate")
		// 		ctx.AbortWithStatusJSON(responseObject.StatusCode, responseObject)
		// 		return
		// 	}
		// 	// 2. Fallback: Invalid claims
		// 	dependencies.Logger.Warnln("AUTH: missing user_identifier or api_key in claims")
		// 	responseObject := dto.NewFailedResponse[any](http.StatusUnauthorized, "invalid JWT claims")
		// 	ctx.AbortWithStatusJSON(responseObject.StatusCode, responseObject)
		// 	return
		// }
		// // api key authentication is for server-to-server communication (WhatsApp)
		// apiKey := ctx.GetHeader(cfg.Default().Site.HTTPHeaderAPIKey)
		// if helper.IsEmpty(apiKey) {
		// 	responseObject := dto.NewFailedResponse[any](http.StatusUnauthorized, fmt.Sprintf("missing %s header or Authorization Bearer token", cfg.Default().Site.HTTPHeaderAPIKey))
		// 	ctx.AbortWithStatusJSON(responseObject.StatusCode, responseObject)
		// 	return
		// }
		// // Get user from database
		// user, app, ex := dependencies.UnitOfWork.AIUserRepository().GetByAPIKey(ctx.Request.Context(), apiKey)
		// if ex != nil {
		// 	if ex.StatusCode == http.StatusNotFound {
		// 		responseObject := dto.NewFailedResponse[any](http.StatusUnauthorized, fmt.Sprintf("invalid %s", cfg.Default().Site.HTTPHeaderAPIKey))
		// 		ctx.AbortWithStatusJSON(responseObject.StatusCode, responseObject)
		// 		return
		// 	}
		// 	responseObject := dto.NewFailedResponse[any](ex.StatusCode, ex.Message)
		// 	ctx.AbortWithStatusJSON(responseObject.StatusCode, responseObject)
		// 	return
		// }
		// if !user.InUse {
		// 	dependencies.Logger.Warnf("AUTH: user %d is disabled for API key: %s", user.Id, apiKey)
		// 	responseObject := dto.NewFailedResponse[any](http.StatusUnauthorized, "user is disabled")
		// 	ctx.AbortWithStatusJSON(responseObject.StatusCode, responseObject)
		// 	return
		// }
		// // set user info in context
		// userDTO := dto_ai.NewUser(*user, app)
		// ctx.Set(cfg.Default().Site.HTTPRequestUserKey, &userDTO)
		ctx.Next()
	}
}
