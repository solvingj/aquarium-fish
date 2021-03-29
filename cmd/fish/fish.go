package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"time"

	dqlite_app "github.com/canonical/go-dqlite/app"
	dqlite_client "github.com/canonical/go-dqlite/client"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"golang.org/x/sys/unix"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/adobe/aquarium-fish/lib/core"
	"github.com/adobe/aquarium-fish/lib/fish"
)

func main() {
	var api_address string
	var db_address string
	var join *[]string
	var drivers *[]string
	var cfg string
	var dir string
	var verbose bool

	cmd := &cobra.Command{
		Use:   "aquarium-fish",
		Short: "Aquarium fish",
		Long:  `Part of the Aquarium suite - a distributed resources manager`,
		RunE: func(cmd *cobra.Command, args []string) error {
			log.Println("Fish running...")
			dir := filepath.Join(dir, db_address)
			if err := os.MkdirAll(dir, 0o755); err != nil {
				return errors.Wrapf(err, "can't create %s", dir)
			}
			logFunc := func(l dqlite_client.LogLevel, format string, a ...interface{}) {
				if !verbose {
					return
				}
				log.Printf(fmt.Sprintf("%s: %s: %s\n", api_address, l.String(), format), a...)
			}

			log.Println("Fish starting dqlite...")
			dqlite, err := dqlite_app.New(dir, dqlite_app.WithAddress(db_address), dqlite_app.WithCluster(*join), dqlite_app.WithLogFunc(logFunc))
			if err != nil {
				return err
			}

			if err := dqlite.Ready(context.Background()); err != nil {
				return err
			}

			dqlite_db, err := dqlite.Open(context.Background(), "aquarium-fish")
			if err != nil {
				return err
			}

			log.Println("Fish starting orm...")
			db, err := gorm.Open(&sqlite.Dialector{Conn: dqlite_db}, &gorm.Config{
				Logger: logger.Default.LogMode(logger.Warn),
			})
			if err != nil {
				return err
			}

			log.Println("Fish starting server...")
			fish, err := fish.New(db, cfg, *drivers)
			if err != nil {
				return err
			}

			srv, err := core.Init(fish, api_address)
			if err != nil {
				return err
			}

			log.Println("Fish initialized")
			quit := make(chan os.Signal)
			signal.Notify(quit, unix.SIGINT)
			signal.Notify(quit, unix.SIGQUIT)
			signal.Notify(quit, unix.SIGTERM)

			<-quit

			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			if err := srv.Shutdown(ctx); err != nil {
				log.Fatal("Fish forced to shutdown:", err)
			}

			fish.Close()

			log.Println("Fish exiting...")
			dqlite_db.Close()

			dqlite.Handover(context.Background())
			dqlite.Close()

			return nil
		},
	}

	flags := cmd.Flags()
	flags.StringVarP(&api_address, "api", "a", "", "address used to expose the fish API")
	flags.StringVarP(&db_address, "db", "d", "", "address used for internal database replication")
	join = flags.StringSliceP("join", "j", nil, "database addresses of existing nodes, comma separated")
	drivers = flags.StringSliceP("drivers", "r", nil, "enabled resource drivers, comma separated")
	flags.StringVarP(&cfg, "cfg", "c", "", "yaml configuration file")
	flags.StringVarP(&dir, "dir", "D", "fish_db", "database directory")
	flags.BoolVarP(&verbose, "verbose", "v", false, "verbose logging")

	cmd.MarkFlagRequired("api")
	cmd.MarkFlagRequired("db")

	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
