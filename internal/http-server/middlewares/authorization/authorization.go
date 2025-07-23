package authorization_middleware

import (
	"chat_go/internal/lib/jwts"
	"context"
	"log"
	"net/http"
	"strconv"
)

func AuthorizeJWTToken(next http.Handler) http.Handler {

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("auth_token")
		if err != nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			log.Printf("no cookie found")
			return
		}
						
		tokenString := cookie.Value
						
		tokenValue, err := jwts.VerifyJWTToken(tokenString)
		if err != nil {
			http.Error(w, err.Error(), http.StatusForbidden)
			log.Printf("error: %v", err)
			return
		}
						
		userID, err := strconv.ParseInt(tokenValue, 10, 64)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			log.Printf("error: %v", err)
			return
		}
			
		ctx := context.WithValue(r.Context(), "userid", userID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}