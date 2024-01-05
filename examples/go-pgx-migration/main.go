package main

import (
	"database/sql"
	"embed"
	"errors"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pressly/goose/v3"
)

type Migration struct {
	db *sql.DB
}

//go:embed migrations/*.sql
var embedMigrations embed.FS

func NewMigration(pool *pgxpool.Pool) (*Migration, error) {
	if pool == nil {
		return &Migration{}, errors.New("pool is nil")
	}

	goose.SetBaseFS(embedMigrations)

	if err := goose.SetDialect("postgres"); err != nil {
		return &Migration{}, err
	}

	cp := pool.Config().ConnConfig.ConnString()
	db, err := sql.Open("pgx/v5", cp)
	if err != nil {
		return &Migration{}, err
	}

	return &Migration{db: db}, nil
}

func (m *Migration) Up() error {
	if err := goose.Up(m.db, "migrations"); err != nil {
		return err
	}
	return nil
}

func (m *Migration) Down() error {
	if err := goose.Down(m.db, "migrations"); err != nil {
		return err
	}
	return nil
}

//func TestMain(m *testing.M) {
//	dockerPool, err := dockertest.NewPool("")
//	if err != nil {
//		log.Fatalf("Could not connect to docker %v", err)
//	}
//
//	resource, err := dockerPool.RunWithOptions(&dockertest.RunOptions{
//		Repository: "postgres",
//		Tag:        "16",
//		Env: []string{
//			"POSTGRES_PASSWORD=secret",
//			"POSTGRES_USER=postgres",
//			"POSTGRES_DB=postgres",
//			"listen_addresses = '*'",
//		},
//	}, func(config *docker.HostConfig) {
//		config.AutoRemove = true
//		config.RestartPolicy = docker.RestartPolicy{
//			Name: "no",
//		}
//	})
//	if err != nil {
//		log.Fatalf("Could not start resource %v", err)
//	}
//
//	hostAndPort := resource.GetHostPort("5432/tcp")
//	databaseUrl := fmt.Sprintf("postgres://postgres:secret@%s/postgres?sslmode=disable", hostAndPort)
//
//	if err := resource.Expire(120); err != nil {
//		log.Fatal().Err(err).Msg("Could not set expire")
//	} // Tell docker to hard kill the container in 120 seconds
//
//	dockerPool.MaxWait = 120 * time.Second
//	if err := dockerPool.Retry(func() error {
//		var err error
//		pool, err = pgxpool.New(context.Background(), databaseUrl)
//		if err != nil {
//			return err
//		}
//		return nil
//	}); err != nil {
//		log.Fatal().Err(err).Msgf("Could not connect to docker %v", err)
//	}
//
//	//resource, err := dockerPool.Run("postgres", "16", []string{
//	//	"POSTGRES_PASSWORD=secret", "POSTGRES_DB=postgres", "POSTGRES_USER=postgres",
//	//})
//	//if err != nil {
//	//	log.Fatal().Err(err).Msg("Could not start resource")
//	//}
//	//
//	//// set AutoRemove to true so that stopped container goes away by itself
//	//
//	//resource.Expire(120) // Tell docker to hard kill the container in 120 seconds
//	//
//	//var connStr string
//	//if err := dockerPool.Retry(func() error {
//	//	var err error
//	//	connStr = "host=localhost port=" + resource.GetPort("5432/tcp") + " user=postgres password=secret dbname=postgres sslmode=disable"
//	//	pool, err = pgxpool.New(context.Background(), connStr)
//	//	if err != nil {
//	//		return err
//	//	}
//	//	return nil
//	//},
//	//); err != nil {
//	//	log.Fatal().Err(err).Msg("Could not connect to docker")
//	//}
//	os.Exit(m.Run())
//}
