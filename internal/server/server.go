package server

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/btcsuite/btcd/rpcclient"
	"github.com/gin-gonic/gin"
	"github.com/wx-shi/utxo-indexer/internal/config"
	"github.com/wx-shi/utxo-indexer/internal/db"
	"github.com/wx-shi/utxo-indexer/pkg"
	"go.uber.org/zap"
)

const (
	// readTimeout is the maximum duration for reading the entire
	// request, including the body.
	readTimeout = 5 * time.Minute

	// writeTimeout is the maximum duration before timing out
	// writes of the response. It is reset whenever a new
	// request's header is read.
	writeTimeout = 5 * time.Minute

	// idleTimeout is the maximum amount of time to wait for the
	// next request when keep-alives are enabled.
	idleTimeout = 5 * time.Minute
)

type Server struct {
	conf   *config.ServerConfig
	logger *zap.Logger
	db     *db.DB
	rpc    *rpcclient.Client
	engine *gin.Engine
	hs     *http.Server
}

func NewServer(conf *config.ServerConfig, logger *zap.Logger, db *db.DB, rpc *rpcclient.Client) *Server {

	s := &Server{
		conf:   conf,
		logger: logger,
		db:     db,
		rpc:    rpc,
	}

	s.initGin()
	return s
}

func (s *Server) initGin() {
	gin.SetMode(gin.ReleaseMode)
	engine := gin.New()
	engine.Use(pkg.LogMiddleware(s.logger), pkg.CORSMiddleware(), gin.Recovery())

	engine.POST("utxo", s.utxoHandle())
	engine.POST("utxo_info", s.utxoInfoHandle())
	engine.POST("height", s.heightHandle())
	s.engine = engine
}

func (s *Server) Run() {
	addr := fmt.Sprintf("%s:%d", s.conf.Host, s.conf.Port)
	hs := &http.Server{
		Addr:         addr,
		Handler:      s.engine,
		ReadTimeout:  readTimeout,
		WriteTimeout: writeTimeout,
		IdleTimeout:  idleTimeout,
	}
	s.hs = hs

	go func() {
		if err := hs.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			s.logger.Fatal("listen", zap.Error(err))
		}
	}()
	s.logger.Info("listen", zap.String("addr", addr))

}

func (s *Server) Shutdown(ctx context.Context) error {
	return s.hs.Shutdown(ctx)
}
