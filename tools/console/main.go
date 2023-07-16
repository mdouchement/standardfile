package main

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/asdine/storm/v3"
	"github.com/mdouchement/standardfile/internal/database"
	"github.com/mdouchement/standardfile/internal/model"
	"github.com/mdouchement/standardfile/pkg/stormsql"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

// go run tools/console/main.go standardfile.db " SELECT count(*) FROM items WHERE UserID = 'f2a98ab0-2c40-42b4-be08-da3b771be935' AND UpdatedAt > '2019-02-16 20:52:55';  "

func main() {
	c := &cobra.Command{
		Use:   "console",
		Short: "SQL console for standardfile database",
		Args:  cobra.ExactArgs(2),
		RunE: func(_ *cobra.Command, args []string) error {
			//
			//
			sc, err := stormsql.ParseSelect(args[1])
			if err != nil {
				return err
			}

			//
			//
			fmt.Println("Opening", args[0])
			db, err := storm.Open(args[0], database.StormCodec)
			if err != nil {
				return errors.Wrap(err, "could not open database")
			}
			defer db.Close()

			//
			// Prepare request
			//

			query := db.Select(sc.Matcher)
			if sc.Skip > 0 {
				query.Skip(sc.Skip)
			}
			if sc.Limit > 0 {
				query.Limit(sc.Limit)
			}
			if len(sc.OrderBy) > 0 {
				query.OrderBy(sc.OrderBy...)
				if sc.OrderByReversed {
					query.Reverse()
				}
			}

			// Execute

			if sc.Count {
				return count(sc, query)
			}

			return list(sc, query)
		},
	}

	if err := c.Execute(); err != nil {
		log.Fatalf("%+v", err)
	}
}

func count(sc *stormsql.SelectClause, query storm.Query) error {
	var records any
	switch sc.Tablename {
	case "users":
		records = &model.User{}
	case "items":
		records = &model.Item{}
	default:
		return errors.Errorf("unknown tablename: %s", sc.Tablename)
	}

	n, err := query.Count(records)

	if err != nil {
		return errors.Wrap(err, "could not perform query")
	}

	fmt.Println("Count:", n)

	return nil
}

func list(sc *stormsql.SelectClause, query storm.Query) error {
	var records any
	switch sc.Tablename {
	case "users":
		records = &[]*model.User{}
	case "items":
		records = &[]*model.Item{}
	default:
		return errors.Errorf("unknown tablename: %s", sc.Tablename)
	}

	err := query.Find(records)
	if err == storm.ErrNotFound {
		fmt.Println("[]")
		return nil
	}

	if err != nil {
		return errors.Wrap(err, "could not perform query")
	}

	jsondump(records)

	return nil
}

func jsondump(v any) {
	d, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		panic(err)
	}
	fmt.Println(string(d))
}
