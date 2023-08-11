package profaastinate

import (
	"context"
	"fmt"
	"github.com/jackc/pgx/v5"
	"github.com/nuclio/logger"
	"net/http"
	"os"
)

type Procastinator struct {
	conn   *pgx.Conn
	logger logger.Logger
}

func NewProcastinator(logger logger.Logger) Procastinator {
	// create DB connection
	conn, err := pgx.Connect(context.Background(), "postgres:1234@postgres:5432")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to connect to database: %v\n", err)
		os.Exit(1)
	}
	err = ensureTablesExist(conn)
	return Procastinator{
		conn, logger,
	}
}

func ensureTablesExist(conn *pgx.Conn) error {
	_, err := conn.Exec(context.Background(), "CREATE TABLE IF NOT EXISTS function_metadata(function_name VARCHAR ( 50 ) NOT NULL PRIMARY KEY, max_latency INT NOT NULL);")
	if err != nil {
		return err
	}
	_, err = conn.Exec(context.Background(), "CREATE TABLE IF NOT EXISTS delayed_calls(id SERIAL PRIMARY KEY, function_name VARCHAR ( 50 ) NOT NULL, call_time TIMESTAMP NOT NULL, request TEXT NOT NULL);")
	if err != nil {
		return err
	}
	return nil
}

func serializeRequest(request http.Request) string {

}

func deserializeRequest(request string) http.Request {

}

func (*Procastinator) Procrastinate(request *http.Request) error {

	return nil
}
