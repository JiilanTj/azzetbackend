package api

import (
	"net/http"

	"codeberg.org/azzet/azzetbe/internal/database"
	rdb "codeberg.org/azzet/azzetbe/internal/redis"
	"codeberg.org/azzet/azzetbe/internal/shared"
)

func HealthCheck(db *database.Database, redis *rdb.Redis) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		status := "ok"
		checks := map[string]string{}

		// Check database
		if err := db.HealthCheck(r.Context()); err != nil {
			status = "degraded"
			checks["database"] = "unhealthy: " + err.Error()
		} else {
			checks["database"] = "healthy"
		}

		// Check redis
		if err := redis.HealthCheck(r.Context()); err != nil {
			status = "degraded"
			checks["redis"] = "unhealthy: " + err.Error()
		} else {
			checks["redis"] = "healthy"
		}

		statusCode := http.StatusOK
		if status != "ok" {
			statusCode = http.StatusServiceUnavailable
		}

		shared.Success(w, statusCode, map[string]any{
			"status": status,
			"checks": checks,
		})
	}
}
