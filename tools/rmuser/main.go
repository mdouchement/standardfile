package main

import (
	"fmt"
	"log"

	"github.com/asdine/storm/v3"
	"github.com/asdine/storm/v3/q"
	"github.com/mdouchement/standardfile/internal/database"
	"github.com/mdouchement/standardfile/internal/model"
	"github.com/muesli/coral"
	"github.com/pkg/errors"
)

func main() {
	c := &coral.Command{
		Use:   "rmuser",
		Short: "Remove a user from the database",
		Args:  coral.ExactArgs(2),
		RunE: func(_ *coral.Command, args []string) error {
			//
			//
			fmt.Println("Opening", args[0])
			db, err := storm.Open(args[0], database.StormCodec)
			if err != nil {
				return errors.Wrap(err, "could not open database")
			}
			defer db.Close()

			// Fetch user
			var user model.User
			err = db.One("Email", args[1], &user)
			if err != nil {
				if err == storm.ErrNotFound {
					fmt.Println("No account for this email")
					return nil
				}
				return errors.Wrap(err, "find user by mail")
			}

			fmt.Println("User found:", user.ID)

			// Deleting user's items
			err = db.Select(q.Eq("UserID", user.ID)).Delete(&model.Item{})
			if err != nil && err != storm.ErrNotFound {
				return errors.Wrap(err, "delete items")
			}
			fmt.Println("Items removed")

			// Delete user
			err = db.DeleteStruct(&user)
			if err != nil && err != storm.ErrNotFound {
				return errors.Wrap(err, "delete user")
			}
			fmt.Println("User removed")

			return nil
		},
	}

	if err := c.Execute(); err != nil {
		log.Fatalf("%+v", err)
	}
}
