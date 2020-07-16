package utils

import (
	"bytes"
	"net/http"
	"net/http/httputil"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

type bodyLogWriter struct {
	gin.ResponseWriter
	Body *bytes.Buffer
}

func (w bodyLogWriter) Write(b []byte) (int, error) {
	w.Body.Write(b)
	return w.ResponseWriter.Write(b)
}

func ShimLogger() gin.HandlerFunc {
	log.Logger = log.Output(
		zerolog.ConsoleWriter{
			Out: os.Stderr,
			// NoColor: false,
			NoColor: true,
		},
	)

	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		raw := c.Request.URL.RawQuery
		if raw != "" {
			path = path + "?" + raw
		}

		b, err := httputil.DumpRequest(c.Request, true)
		if err == nil {
			log.Printf("Request: %s\n", b)
		}

		bodyWriter := &bodyLogWriter{Body: bytes.NewBufferString(""), ResponseWriter: c.Writer}
		c.Writer = bodyWriter

		c.Next()

		end := time.Now()
		latency := end.Sub(start)

		msg := "Request"
		if len(c.Errors) > 0 {
			msg = c.Errors.String()
		}

		dumplogger := log.Logger.With().
			Int("status", c.Writer.Status()).
			Str("method", c.Request.Method).
			Str("path", path).
			Str("ip", c.ClientIP()).
			Dur("latency", latency).
			Str("user-agent", c.Request.UserAgent()).
			Logger()

		switch {
		case c.Writer.Status() >= http.StatusBadRequest && c.Writer.Status() < http.StatusInternalServerError:
			{
				dumplogger.Warn().Msg(msg)
			}
		case c.Writer.Status() >= http.StatusInternalServerError:
			{
				dumplogger.Error().Msg(msg)
			}
		default:
			dumplogger.Info().Msg(msg)
			log.Logger.Debug().Msg("Response: " + bodyWriter.Body.String())
		}
	}
}
