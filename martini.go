// martini

package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	r "github.com/dancannon/gorethink"
	"github.com/go-martini/martini"
	"github.com/martini-contrib/binding"
	"github.com/martini-contrib/render"
	"github.com/martini-contrib/sessions"
)

type User struct {
	Id            int64  `form:"id" db:"id"`
	Email         string `form:"email" db:"email" binding:"required"`
	Password      string `form:"password" db:"password" binding:"required"`
	Name          string `form:"name" db:"name"`
	authenticated bool   `form:"-" db:"-"`
}

func (u User) Validate(errors binding.Errors, req *http.Request) binding.Errors {
	if len(u.Email) < 3 {
		errors = append(errors, binding.Error{
			FieldNames:     []string{"Email"},
			Classification: "ComplaintError",
			Message:        "Email too short",
		})
	}
	return errors
}

func dbinit() *r.Session {
	session, err := r.Connect(r.ConnectOpts{
		Address:     os.Getenv("RETHINKDB_URL"),
		Database:    "test",
		MaxIdle:     10,
		IdleTimeout: time.Second * 10,
	})

	if err != nil {
		log.Println(err)
	}

	err = r.DbCreate("test").Exec(session)
	if err != nil {
		log.Println(err)
	}

	_, err = r.Db("test").TableCreate("todos").RunWrite(session)
	if err != nil {
		log.Println(err)
	}

	return session

}

func main() {

	m := martini.Classic()
	m.Use(render.Renderer())

	store := sessions.NewCookieStore([]byte("secret123"))
	m.Use(sessions.Sessions("martini_shop_session", store))

	m.Get("/", func(r render.Render, session sessions.Session) {

		r.HTML(200, "index", nil)
	})

	m.Get("/admin", func(r render.Render, session sessions.Session) {
		v := session.Get("hello1")
		if v == nil {
			fmt.Println("")
		}
		fmt.Println(v.(string))
		r.HTML(200, "adminlogin", nil)

	})

	m.Post("/admin", binding.Form(User{}), func(postedUser User, r render.Render, ferr binding.Errors) {

		//Example of server side error validation for the client side form
		if ferr.Len() > 0 {
			newmap := map[string]interface{}{"metatitle": "Registration", "errormessage": "Wrong login or password"}
			r.HTML(200, "adminlogin", newmap)

		} else {
			if postedUser.Email == "cotedeazur@gmail.com" {
				if postedUser.Password == "617381" {
					r.HTML(200, "admin", nil)

				} else {
					newmap := map[string]interface{}{"metatitle": "Registration", "errormessage": "Wrong login or password"}
					r.HTML(200, "adminlogin", newmap)
				}
			} else {
				newmap := map[string]interface{}{"metatitle": "Registration", "errormessage": "Wrong login or password"}
				r.HTML(200, "adminlogin", newmap)
			}
		}

	})

	m.Run()

}
