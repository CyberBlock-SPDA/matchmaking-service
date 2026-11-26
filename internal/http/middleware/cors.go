package middleware

import (
	"net/http"
	"strings"
)

func CORS(
	allowedOrigins string,
	next http.Handler,
) http.Handler {
	origins := make(map[string]struct{})

	for _, origin := range strings.Split(allowedOrigins, ",") {
		origin = strings.TrimSpace(origin)

		if origin != "" && origin != "*" {
			origins[origin] = struct{}{}
		}
	}

	return http.HandlerFunc(func(
		w http.ResponseWriter,
		r *http.Request,
	) {
		w.Header().Add("Vary", "Origin")

		origin := r.Header.Get("Origin")

		if origin == "" {
			next.ServeHTTP(w, r)
			return
		}

		_, allowed := origins[origin]

		preflight := r.Method == http.MethodOptions &&
			r.Header.Get("Access-Control-Request-Method") != ""

		if preflight {
			w.Header().Add("Vary", "Access-Control-Request-Method")
			w.Header().Add("Vary", "Access-Control-Request-Headers")
		}

		if !allowed {
			if preflight {
				http.Error(
					w,
					"Origin not allowed",
					http.StatusForbidden,
				)
				return
			}

			next.ServeHTTP(w, r)
			return
		}

		w.Header().Set(
			"Access-Control-Allow-Origin",
			origin,
		)

		w.Header().Set(
			"Access-Control-Expose-Headers",
			"Retry-After",
		)

		if preflight {
			w.Header().Set(
				"Access-Control-Allow-Methods",
				"GET, POST, OPTIONS",
			)

			w.Header().Set(
				"Access-Control-Allow-Headers",
				"Content-Type, Accept",
			)

			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}
