package setup

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/jjcheng/wawa-go/internal/cfg"
	"github.com/jjcheng/wawa-go/internal/repository"
	"github.com/jjcheng/wawa-go/internal/repository/gormdb"
	"github.com/jjcheng/wawa-go/internal/service"
	"github.com/jjcheng/wawa-go/internal/types"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// SetupDatabase initializes database connections and returns a UnitOfWork
func SetupDatabase(dsn string, loggerService *service.Logger) (repository.UnitOfWork, error) {
	timeOut := 200 * time.Millisecond
	if cfg.Default().Site.Environment == types.EnvironmentDevelop {
		timeOut = 500 * time.Second
	}
	dbLogger := logger.New(
		log.New(os.Stdout, "\r\n", log.LstdFlags),
		logger.Config{
			SlowThreshold:             timeOut,
			LogLevel:                  logger.Warn, // only log warnings & errors
			IgnoreRecordNotFoundError: true,
			Colorful:                  true,
		},
	)
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger:                 dbLogger,
		SkipDefaultTransaction: true,
	})
	if err == nil {
		log.Println("successfully connected to DB")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to connect to DB with connection %v: %v", dsn, err)
	}
	// Set the session timezone to UTC for display
	db.Exec("SET timezone = 'UTC'")
	return gormdb.NewUnitOfWork(db, loggerService), nil
}
