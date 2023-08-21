package database

import (
	"context"
	"time"
	"os"
	"path/filepath"

	"github.com/ethereum/go-ethereum/builder/database/vars"
	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
)

type DatabaseService struct {
	DB *sqlx.DB
}

func NewDatabaseService(dirPath string, reset bool) (*DatabaseService, error) {

	db_file := filepath.Join(dirPath, vars.DBName)

	// Check if the database path exists and if the file exist if not create it	
	if _, err := os.Stat(dirPath); os.IsNotExist(err) {
		err = os.MkdirAll(dirPath, 0755)
		if err != nil {
			return nil, err
		}
	}

	db, err := sqlx.Connect("sqlite3", db_file)
	if err != nil {
		return nil, err
	}

	if reset {
		db.MustExec(vars.ForceDropSchema)
	} else {
		// Drop tables if they are not of right configuration
		db.MustExec(vars.DropSchema)
	}

	// Migrate the schema
	db.MustExec(vars.CreateSchema)

	dbService := &DatabaseService{DB: db}

	go dbService.CleanOldData()

	return dbService, err
}


func (s *DatabaseService) Close() error {
	return s.DB.Close()
}

func (s *DatabaseService) CleanOldData() error {

	ticker := time.NewTicker(24 * time.Hour)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	
	for {
		select {
		case <-ticker.C:
			s.DB.MustExec(vars.CleanOldData)
		case <-ctx.Done():
			ticker.Stop()
			return ctx.Err()
		}
	}
}


