package app

import (
	"database/sql"

	store "flyteam-website/cmd/flyteam-server/internal/database"
)

func openDatabase(cfg Config) (*sql.DB, error) {
	return store.Open(cfg.DatabaseFile)
}

func (s *Server) loadJSONFromDB(key string, dst any) bool {
	return store.LoadJSON(s.db, key, dst)
}

func (s *Server) saveJSONToDB(key string, data any) error {
	return store.SaveJSON(s.db, key, data, nowISO())
}
