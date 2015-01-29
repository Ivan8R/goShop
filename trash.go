// trash
package main

import (
	"fmt"
	"log"
	"net/http"
	"reflect"
)

func main() {
	fmt.Println("Hello World!")
}

func checkIfstaffloggedIn(w http.ResponseWriter, q *http.Request, arg ...renderStruct) {
	if len(arg) == 1 {
		session, _ := store.Get(q, sessionname)
		if session.Values["userLogedIn"] != nil {
			x := reflect.Indirect(reflect.ValueOf(session.Values["userLogedIn"]))

			db := dbinit()
			defer db.Close()

			res, err := r.Table("staff").Filter(map[string]interface{}{"Login": x.Field(0).String()}).Run(db)
			if err != nil {
				log.Println(err)
			}
			if res.IsNil() {
				http.Redirect(w, q, "/admin", http.StatusFound)

			} else {

				render(w, arg[0].templatename, arg[0].parametrs)

			}
		} else {
			http.Redirect(w, q, "/admin", http.StatusFound)
		}
	} else if len(arg) == 2 {

		session, _ := store.Get(q, sessionname)
		if session.Values["userLogedIn"] != nil {
			x := reflect.Indirect(reflect.ValueOf(session.Values["userLogedIn"]))

			db := dbinit()
			defer db.Close()

			res, err := r.Table("staff").Filter(map[string]interface{}{"Login": x.Field(0).String()}).Run(db)
			if err != nil {
				log.Println(err)
			}
			if res.IsNil() {
				render(w, arg[1].templatename, arg[1].parametrs)

			} else {

				render(w, arg[0].templatename, arg[0].parametrs)

			}
		} else {
			render(w, arg[1].templatename, arg[1].parametrs)
		}
	}

}

type renderStruct struct {
	templatename string
	parametrs    map[string]interface{}
}

func checkIfStaffExist(q *http.Request) bool {

	session, _ := store.Get(q, sessionname)
	if session.Values["userLogedIn"] != nil {
		x := reflect.Indirect(reflect.ValueOf(session.Values["userLogedIn"]))

		db := dbinit()
		defer db.Close()

		res, err := r.Table("staff").Filter(map[string]interface{}{"Login": x.Field(0).String()}).Run(db)
		if err != nil {
			log.Println(err)
		}
		if res.IsNil() {
			return false

		} else {

			return true

		}
	} else {
		return false
	}

}

type userLogedIn struct {
	UserLogin     string
	IsUserLogedIn bool
}

$(".itemadd").click(function() {
   $.ajax({
          type: "POST",
          url: "/addtocart",
          data: { productadd: this.classList[0],quantity: $("input[name=prodcutQuantity"+this.classList[0]+"]").val()}
          })
    .done(function( msg ) {
     //alert( "Data Saved: " + msg );
  });
});



	//get value from fields
	//put product to order
	////переписать
	////
	////
	////
	////
	SKU := q.FormValue("SKU")
	Quantity, err := strconv.Atoi(q.FormValue("prodcutQuantity" + SKU))
	if err != nil {
		log.Println(err)
	}

	//Add product to session
	session, _ := store.Get(q, sessionname)

	if session.Values["Order"] != nil {

		cart := reflect.ValueOf(session.Values["Order"]).Elem().Interface().(Order)

		p := findproductCart{
			CartArray: cart.CartOrderd,
			SKU:       SKU,
		}

		if p.checkProductinCart() == true {
			fmt.Println("Productincart")

			//x.CartOrderd[p.Position].Quantity = x.CartOrderd[p.Position].Quantity + Quantity

			//session.Values["Order"] = x
			//session.Save(q, w)
			//fmt.Println(session.Values["Order"])

		} else {
			fmt.Println("New product in cart")
			//item := ProductInCart{
			//	SKU:      SKU,
			//	Quantity: Quantity,
			//}
			//d := append(x.CartOrderd, item)
			//session.Values["Order"] = d
			//session.Save(q, w)
			//fmt.Println(session.Values["Order"])

		}

	} else {
		fmt.Println("New cart")
		item := ProductInCart{
			SKU:      SKU,
			Quantity: Quantity,
		}

		order := Order{
			CartOrderd: ProductsInCart{item},
		}

		session.Values["Order"] = order
		session.Save(q, w)
		fmt.Println(session.Values["Order"])

	}
