package main

import (
	"errors"
	"fmt"
	"os"
	"runtime/debug"
	"strings"

	"github.com/genjidb/genji/cmd/genji/shell"
	"github.com/urfave/cli/v2"
)

func main() {
	app := cli.NewApp()
	app.Name = "Genji"
	app.Usage = "Shell for the Genji database"
	app.EnableBashCompletion = true
	app.Flags = []cli.Flag{
		&cli.BoolFlag{
			Name:  "bolt",
			Usage: "use bolt engine",
		},
		&cli.BoolFlag{
			Name:  "badger",
			Usage: "use badger engine",
		},
	}

	app.Commands = []*cli.Command{
		{
			Name:      "insert",
			Usage:     "Insert documents from arguments or standard input",
			UsageText: "genji insert [options] [json...]",
			Description: `
The insert command inserts documents into an existing table.

Insert can take JSON documents as separate arguments:

$ genji insert --db my.db -t foo '{"a": 1}' '{"a": 2}'

It is also possible to pass an array of objects:

$ genji insert --db my.db -t foo '[{"a": 1}, {"a": 2}]'

Also you can use -a flag to create database automatically.
This example will create BoltDB-based database with name 'data_${current unix timestamp}.db'
It can be combined with --db to select an existing database but automatically create the table.

$ genji insert -a -e bolt '[{"a": 1}, {"a": 2}]'

Insert can also insert a stream of objects or an array of objects from standard input:

$ echo '{"a": 1} {"a": 2}' | genji insert --db my.db -t foo
$ echo '[{"a": 1},{"a": 2}]' | genji insert --db my.db -t foo
$ curl https://api.github.com/repos/genjidb/genji/issues | genji insert --db my.db -t foo`,
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:    "engine",
					Aliases: []string{"e"},
					Usage:   "name of the engine to use, options are 'bolt' or 'badger'",
					Value:   "bolt",
				},
				&cli.StringFlag{
					Name:     "db",
					Usage:    "path of the database file",
					Required: false,
				},
				&cli.StringFlag{
					Name:     "table",
					Aliases:  []string{"t"},
					Usage:    "name of the table, it must already exist",
					Required: false,
				},
				&cli.BoolFlag{
					Name:     "auto",
					Aliases:  []string{"a"},
					Usage:    `automatically creates a database and a table whose name is equal to "data_" followed by the current unix timestamp.`,
					Required: false,
					Value:    false,
				},
			},
			Action: func(c *cli.Context) error {
				dbPath := c.String("db")
				table := c.String("table")
				engine := c.String("engine")
				args := c.Args().Slice()

				return runInsertCommand(c.Context, engine, dbPath, table, c.Bool("auto"), args)
			},
		},
		{
			Name:      "restore",
			Usage:     "Restore a database or a table from a file created by genji dump",
			UsageText: "genji restore [options] dumpFile dbPath",
			Description: `
The restore command can restore a database or a table from a valid SQL file.

$ genji restore dump.sql my.db

A specific table can be restored and a different engine can specified as well.

$ genji restore -e badger -t foo dump.sql my.db
			`,
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:    "engine",
					Aliases: []string{"e"},
					Usage:   "name of the engine to use, options are 'bolt' or 'badger'",
					Value:   "bolt",
				},
				&cli.StringFlag{
					Name:    "table",
					Aliases: []string{"t"},
					Usage:   "name of the table to restore. Default to all tables",
				},
			},
			Action: func(c *cli.Context) error {
				engine := c.String("engine")
				table := c.String("table")
				args := c.Args().Slice()

				switch engine {
				case "bolt", "badger":
				default:
					return fmt.Errorf("unsupported engine %q, only 'bolt' or 'badger' can be used", engine)
				}

				switch {
				case len(args) != 2:
					return fmt.Errorf("bad arguments: %v", c.Command.UsageText)
				case !strings.HasSuffix(args[0], ".sql"):
					return errors.New("first argument should be an SQL file")
				case !strings.HasSuffix(args[1], ".db"):
					return errors.New("second argument should be a database file")
				}

				file, err := os.Open(args[0])
				if err != nil {
					return err
				}
				defer file.Close()

				return runRestoreCommand(c.Context, file, engine, table, args[1])
			},
		},
		{
			Name:  "version",
			Usage: "Shows Genji and Genji CLI version",
			Action: func(c *cli.Context) error {
				var cliVersion, genjiVersion string
				info, ok := debug.ReadBuildInfo()

				if !ok {
					fmt.Println(`version not available in GOPATH mode; use "go get" with Go modules enabled`)
					return nil
				}

				cliVersion = info.Main.Version
				for _, mod := range info.Deps {
					if mod.Path != "github.com/genjidb/genji" {
						continue
					}
					genjiVersion = mod.Version
					break
				}
				fmt.Printf("Genji %v\nGenji CLI %v\n", genjiVersion, cliVersion)
				return nil
			},
		},
	}

	// Root command
	app.Action = func(c *cli.Context) error {
		useBolt := c.Bool("bolt")
		useBadger := c.Bool("badger")
		if useBolt && useBadger {
			return cli.NewExitError("cannot use bolt and badger options at the same time", 2)
		}

		dbpath := c.Args().First()

		if (useBolt || useBadger) && dbpath == "" {
			return cli.NewExitError("db path required when using bolt or badger", 2)
		}

		engine := "memory"

		if useBolt || dbpath != "" {
			engine = "bolt"
		}

		if useBadger {
			engine = "badger"
		}

		return shell.Run(c.Context, &shell.Options{
			Engine: engine,
			DBPath: dbpath,
		})
	}

	err := app.Run(os.Args)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stdout, "error: %v\n", err)
		os.Exit(2)
	}
}
