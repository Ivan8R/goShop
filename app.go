package main

import (
	"crypto/rand"
	"encoding/gob"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"net/textproto"
	"os"
	"reflect"
	"strconv"
	"strings"
	"time"

	r "github.com/dancannon/gorethink"
	"github.com/gorilla/mux"
	"github.com/gorilla/schema"
	"github.com/gorilla/sessions"
)

var sessionname string = "GOSHOPSESSION"
var store = sessions.NewCookieStore([]byte("something-very-secret"))

var templatesA = template.Must(template.ParseGlob("./templates/admin/*"))

//var templatesU = template.Must(template.ParseGlob("./templates/theme/*"))

func render(w http.ResponseWriter, tmpl string, context map[string]interface{}) {
	tmpl_list := []string{fmt.Sprintf("templates/%s.html", tmpl)}
	t, err := template.ParseFiles(tmpl_list...)
	if err != nil {
		log.Print("template parsing error: ", err)
	}
	err = t.Execute(w, context)
	if err != nil {
		log.Print("template executing error: ", err)
	}
}

//returnt full strin for order id

func getorderId(t string, o string, s string) string {

	return t + "-" + o + "-" + s
}

//GENERATE STRING FOR ORDER
func randStr(strSize int, randType string) string {

	var dictionary string

	if randType == "alphanum" {
		dictionary = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"
	}

	if randType == "alpha" {
		dictionary = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"
	}

	if randType == "number" {
		dictionary = "0123456789"
	}

	var bytes = make([]byte, strSize)
	rand.Read(bytes)
	for k, v := range bytes {
		bytes[k] = dictionary[v%byte(len(dictionary))]
	}
	return string(bytes)
}

func statuses() (string, string, string, string) {
	return "Processing", "Prepare for shipping", "Shipped", "Delivered"

}

func FloatToString(input_num float64) string {

	// to convert a float number to a string
	return strconv.FormatFloat(input_num, 'f', 2, 64)
}

func getOrderCartP(q *http.Request) Order {
	session, _ := store.Get(q, sessionname)
	order := *session.Values["Order"].(*Order)
	return order
}

func getOrderCart(q *http.Request) Order {
	session, _ := store.Get(q, sessionname)
	order := session.Values["Order"].(Order)
	return order
}

//STRUCTERS FOR INSERTING IN DB

type adminRegister struct {
	Login    string `schema:"Login"`
	Password string `schema:"Password"`
}

func dbinit() *r.Session {
	session, err := r.Connect(r.ConnectOpts{
		Address:     "localhost:28015",
		Database:    "goShop",
		MaxIdle:     10,
		IdleTimeout: time.Second * 10,
	})

	if err != nil {
		log.Println(err)
	}

	err = r.DbCreate("goShop").Exec(session)
	if err != nil {
		log.Println(err)
	}

	_, err = r.Db("goShop").TableCreate("staff").RunWrite(session)
	if err != nil {
		log.Println(err)
	}

	newAdmin := adminRegister{
		Login:    "Admin",
		Password: "Admin",
	}
	_, err = r.Db("goShop").Table("staff").Insert(newAdmin).RunWrite(session)
	if err != nil {
		log.Println(err)
	}

	_, err = r.Db("goShop").TableCreate("clients").RunWrite(session)
	if err != nil {
		log.Println(err)
	}

	_, err = r.Db("goShop").TableCreate("items").RunWrite(session)
	if err != nil {
		log.Println(err)
	}

	_, err = r.Db("goShop").TableCreate("orders").RunWrite(session)
	if err != nil {
		log.Println(err)
	}

	_, err = r.Db("goShop").TableCreate("options").RunWrite(session)
	if err != nil {
		log.Println(err)
	}

	return session

}

type Context struct {
	Db      *r.Session
	Session *sessions.Session
}

func NewContext(q *http.Request) (*Context, error) {
	sess, err := store.Get(q, sessionname)
	db := dbinit()

	return &Context{
		Db:      db,
		Session: sess,
	}, err
}

type userLogedIn struct {
	UserLogin     string
	IsUserLogedIn bool
}

type staffLogedIn struct {
	UserLogin     string
	IsUserLogedIn bool
}

type User struct {
	Name  string
	Email string
	Phone string
}

type UserRegistration struct {
	User     User
	Password string
	Adresses []Adress
	Orders   []string
}

type Adress struct {
	FullName            string
	AddressLine1        string
	AddressLine2        string
	City                string
	StateProvinceRegion string
	ZIP                 string
	Country             string
	PhoneNumber         string
}

type Shipping struct {
	ShippingMethod string
	ShippingCost   float64
}

type Order struct {
	Customer    User
	OrderNumber string
	Date        string
	//показывать при пподробном просмотре
	CartOrderd     ProductsInCart
	ShippingAdress Adress
	BillingAdress  Adress
	Shipping       Shipping
	PaymentMethod  string
	OrdersNote     []string
	Status         string
	Total          string
}

type authUser struct {
	Login    string `schema:"Login"`
	Password string `schema:"Password"`
}

func (a *authUser) checkstaffloginpassword() bool {

	db := dbinit()
	defer db.Close()

	res, err := r.Table("staff").Filter(map[string]interface{}{"Login": a.Login, "Password": a.Password}).Run(db)
	if err != nil {
		log.Println(err)
	}
	if res.IsNil() {
		return false
	} else {
		return true
	}

}

type Product struct {
	Title       string `schema:"Title"`
	Price       string `schema:"Price"`
	SKU         string `schema:"SKU"`
	Image       []string
	Description string `schema:"Description"`
}

type ProductInCart struct {
	Product
	SKU      string
	Quantity int `json:",string"`
}

type ProductsInCart []ProductInCart

type FileHeader struct {
	Filename string
	Header   textproto.MIMEHeader
}

func uploadImage(q *http.Request, nameInForm string, SKU string, i int) {

	file, _, err := q.FormFile(nameInForm)

	if err != nil {
		fmt.Println(err)

	}
	defer file.Close()
	//handler.Filename

	err = os.Mkdir("./static/uploadimages/"+SKU, 0777)
	if err != nil {
		fmt.Println(err)
	}

	f, err := os.OpenFile("./static/uploadimages/"+SKU+"/"+SKU+"-"+strconv.Itoa(i)+".jpg", os.O_WRONLY|os.O_CREATE, 0666)
	if err != nil {
		fmt.Println(err)

	}
	defer f.Close()
	io.Copy(f, file)
}

func checkifImageneedUpload(q *http.Request, slice []string, SKU string) []string {
	imageList := []string{}
	for i, s := range slice {
		file, _, err := q.FormFile(s)

		if err != nil {
			fmt.Println(err)

		}
		if file != nil {
			uploadImage(q, s, SKU, i)
			imageList = append(imageList, SKU+"-"+strconv.Itoa(i))
		}

	}

	return imageList
}

//Get one Product with SKU
type getProductInfo struct {
	SKU string
}

func (p *getProductInfo) getProduct() *Product {
	db := dbinit()
	defer db.Close()

	rows, err := r.Db("goShop").Table("items").Filter(map[string]interface{}{"SKU": p.SKU}).Run(db)

	if err != nil {
		log.Println(err)
	}

	t := new(Product)

	rows.One(&t)
	return t

}

type findproductCart struct {
	Position  int
	CartArray ProductsInCart
	SKU       string
}

func (p *findproductCart) total() (int, float64) {
	var sum float64
	var quantity int
	quantity = len(p.CartArray)
	for _, v := range p.CartArray {
		oneItem := reflect.ValueOf(&v).Elem().Interface().(ProductInCart)

		p := getProductInfo{
			SKU: oneItem.SKU,
		}
		d := p.getProduct()
		i, _ := strconv.ParseFloat(d.Price, 64)
		sum = sum + (i * float64(oneItem.Quantity))

	}

	return quantity, sum
}

func (p *findproductCart) getProductsInCart() []ProductInCart {
	var productsInfo []ProductInCart
	for _, v := range p.CartArray {
		oneItem := reflect.ValueOf(&v).Elem().Interface().(ProductInCart)
		pr := getProductInfo{
			SKU: oneItem.SKU,
		}
		p := ProductInCart{
			Product:  *pr.getProduct(),
			Quantity: oneItem.Quantity,
		}
		productsInfo = append(productsInfo, p)

	}

	return productsInfo
}

type mainhandler func(w http.ResponseWriter, q *http.Request)

func authuser(m mainhandler) mainhandler {
	return func(w http.ResponseWriter, q *http.Request) {
		session, _ := store.Get(q, sessionname)
		if session.Values["userLogedIn"] != nil {
			x := *session.Values["userLogedIn"].(*userLogedIn)

			db := dbinit()
			defer db.Close()
			//check in userdb

			res, err := r.Table("clients").Filter(r.Row.Field("User").Field("Email").Eq(x.UserLogin)).Run(db)

			if err != nil {
				log.Println(err)
			}
			if res.IsNil() {
				render(w, "userlogin", nil)

			} else {

				m(w, q)
			}
		} else {
			render(w, "userlogin", nil)
		}
	}
}

func authstaff(m mainhandler) mainhandler {
	return func(w http.ResponseWriter, q *http.Request) {
		session, _ := store.Get(q, sessionname)
		if session.Values["staffLogedIn"] != nil {
			x := *session.Values["staffLogedIn"].(*staffLogedIn)

			db := dbinit()
			defer db.Close()

			res, err := r.Table("staff").Filter(map[string]interface{}{"Login": x.UserLogin}).Run(db)
			if err != nil {
				log.Println(err)
			}

			if res.IsNil() {
				render(w, "adminlogin", nil)

			} else {

				m(w, q)

			}
		} else {
			render(w, "adminlogin", nil)
		}
	}
}

func main() {
	gob.Register(&userLogedIn{})
	gob.Register(&staffLogedIn{})
	gob.Register(&ProductsInCart{})
	gob.Register(&Order{})

	rtr := mux.NewRouter()

	//CUSTOMER SECTION
	rtr.HandleFunc("/", index).Methods("GET")
	rtr.HandleFunc("/product/{SKU}", productDetail).Methods("GET")
	rtr.HandleFunc("/cart", cart).Methods("GET")
	rtr.HandleFunc("/clearcart", clearCart).Methods("GET")
	rtr.HandleFunc("/buy-box", buybox).Methods("GET")
	rtr.HandleFunc("/newitem/{SKU}", newitem).Methods("GET")

	//CHECKOUT SECTION
	rtr.HandleFunc("/checkoutcart", checkoutcart).Methods("POST")

	rtr.HandleFunc("/loginuser", loginuser)

	rtr.HandleFunc("/registrationuser", registrationuser)

	rtr.HandleFunc("/addressselect", authuser(addressselect))
	rtr.HandleFunc("/addressselect/{command}/{index}", authuser(addressselect)).Methods("GET")

	rtr.HandleFunc("/shipoptionselect", authuser(shipoptionselect))

	rtr.HandleFunc("/payselect", authuser(payselect))

	rtr.HandleFunc("/buy", authuser(buy))

	rtr.HandleFunc("/youraccount", authuser(youraccount))
	rtr.HandleFunc("/logoutuser", authuser(logoutuser)).Methods("GET")

	//ADMIN SECTION
	rtr.HandleFunc("/admin", adminlogin)
	//rtr.HandleFunc("/admin", adminornot).Methods("POST")

	rtr.HandleFunc("/items", authstaff(items)).Methods("GET")
	rtr.HandleFunc("/items", authstaff(addItem)).Methods("POST")

	rtr.HandleFunc("/productdelete", authstaff(deleteProduct)).Methods("POST")

	rtr.HandleFunc("/orders", authstaff(orders)).Methods("GET")
	rtr.HandleFunc("/orders/{orderId}", authstaff(getorder)).Methods("GET")
	rtr.HandleFunc("/clients", authstaff(clients)).Methods("GET")
	rtr.HandleFunc("/options", authstaff(options)).Methods("GET")
	rtr.HandleFunc("/logout", authstaff(logout)).Methods("GET")

	//ADMIN TRIGER SECTION

	rtr.PathPrefix("/").Handler(http.FileServer(http.Dir("./static/")))

	http.Handle("/", rtr)

	log.Println("Listening...")
	http.ListenAndServe(":3000", nil)
}

func index(w http.ResponseWriter, q *http.Request) {

	ctx, err := NewContext(q)
	if err != nil {
		log.Println(err)
	}

	rows, err := r.Db("goShop").Table("items").OrderBy("SKU").Run(ctx.Db)

	if err != nil {
		log.Println(err)
	}
	var t []Product
	err = rows.All(&t)
	if err != nil {
		log.Println(err)
	}

	newmap := map[string]interface{}{"Products": t}

	render(w, "index", newmap)

}

func productDetail(w http.ResponseWriter, q *http.Request) {

	SKU := mux.Vars(q)["SKU"]
	fmt.Println(SKU)
}

func buybox(w http.ResponseWriter, q *http.Request) {
	SKU := q.FormValue("SKU")
	Quantity, err := strconv.Atoi(q.FormValue("prodcutQuantity" + SKU))
	if err != nil {
		log.Println(err)
	}

	ctx, err := NewContext(q)
	if err != nil {
		log.Println(err)
	}

	product := ProductInCart{
		SKU:      q.FormValue("SKU"),
		Quantity: Quantity,
	}

	if ctx.Session.Values["Order"] != nil {
		order := ctx.Session.Values["Order"].(*Order)
		cart := order.CartOrderd
		found := false
		for i, x := range cart {
			if x.SKU == product.SKU {
				cart[i].Quantity = x.Quantity + product.Quantity
				order.CartOrderd = cart
				ctx.Session.Values["Order"] = order
				ctx.Session.Save(q, w)
				found = true
			} else {
				continue
			}
		}
		if found == false {
			updatedorder := append(order.CartOrderd, product)
			order.CartOrderd = updatedorder
			ctx.Session.Values["Order"] = order
			ctx.Session.Save(q, w)
		}

	} else {

		order := Order{
			CartOrderd: ProductsInCart{product},
		}

		ctx.Session.Values["Order"] = order
		ctx.Session.Save(q, w)
	}
	http.Redirect(w, q, "/newitem/"+SKU, http.StatusFound)

}

func newitem(w http.ResponseWriter, q *http.Request) {
	//here add sku to get info
	SKU := mux.Vars(q)["SKU"]
	ctx, err := NewContext(q)
	if err != nil {
		log.Println(err)
	}

	cart := ctx.Session.Values["Order"].(*Order).CartOrderd
	t := getProductInfo{
		SKU: SKU,
	}
	p := findproductCart{
		CartArray: cart,
	}

	qua, total := p.total()
	newmap := map[string]interface{}{"Product": t.getProduct(), "Total": total, "Quantity": qua}
	render(w, "editcartorcheckout", newmap)
}

func cart(w http.ResponseWriter, q *http.Request) {
	ctx, err := NewContext(q)
	if err != nil {
		log.Println(err)
	}
	if ctx.Session.Values["Order"] != nil {
		p := findproductCart{
			CartArray: ctx.Session.Values["Order"].(*Order).CartOrderd,
		}
		qua, total := p.total()

		newmap := map[string]interface{}{"Products": p.getProductsInCart(), "Total": total, "Quantity": qua}
		render(w, "cart", newmap)
	} else {
		render(w, "emptycart", nil)
	}

}

func checkoutcart(w http.ResponseWriter, q *http.Request) {

	session, _ := store.Get(q, sessionname)
	err := q.ParseForm()

	if err != nil {
		// Handle error
		fmt.Println(err)
	}
	a := []byte(q.FormValue("array"))
	cart := make(ProductsInCart, 0)
	json.Unmarshal(a, &cart)

	order := Order{
		CartOrderd: cart,
	}
	session.Values["Cart"] = cart

	session.Values["Order"] = order
	session.Save(q, w)

}

func loginuser(w http.ResponseWriter, q *http.Request) {
	session, _ := store.Get(q, sessionname)
	if q.Method == "POST" {
		err := q.ParseForm()

		if err != nil {
			// Handle error
			fmt.Println(err)
		}

		if q.FormValue("RegistredUserorNot") == "YES" {
			err := q.ParseForm()

			if err != nil {
				// Handle error
				fmt.Println(err)
			}
			db := dbinit()
			defer db.Close()

			res, err := r.Table("clients").Filter(r.Row.Field("User").Field("Email").Eq(q.FormValue("email"))).Run(db)

			if err != nil {
				log.Println(err)
			}
			if res.IsNil() {
				newmap := map[string]interface{}{"errormessage": "email not registred"}
				render(w, "userlogin", newmap)

			} else {
				var u UserRegistration
				res.One(&u)
				if u.Password != q.FormValue("Password") {
					newmap := map[string]interface{}{"errormessage": "wrong password"}
					render(w, "userlogin", newmap)
				} else {
					ulin := userLogedIn{
						UserLogin:     q.FormValue("email"),
						IsUserLogedIn: true,
					}

					session.Values["userLogedIn"] = ulin
					session.Save(q, w)
					http.Redirect(w, q, "/addressselect", http.StatusFound)

				}
			}

		} else {

			session.Values["emailforregistration"] = q.FormValue("email")
			session.Save(q, w)
			http.Redirect(w, q, "/registrationuser", http.StatusFound)

		}
	} else {
		//прописать если вошел то на свой аккаунт
		render(w, "userlogin", nil)

	}

}

func registrationuser(w http.ResponseWriter, q *http.Request) {
	session, _ := store.Get(q, sessionname)
	if q.Method == "POST" {
		err := q.ParseForm()

		if err != nil {
			// Handle error
			fmt.Println(err)
		}
		//set uppercase in lowercase

		if (q.FormValue("Email") == q.FormValue("Repeatemail")) && (q.FormValue("Password") == q.FormValue("Repeatpassword")) {
			db := dbinit()
			defer db.Close()

			res, err := r.Table("clients").Filter(r.Row.Field("User").Field("Email").Eq(strings.ToLower(q.FormValue("email")))).Run(db)
			if err != nil {
				log.Println(err)
			}

			if res.IsNil() {

				newUser := UserRegistration{
					User: User{
						Name:  q.FormValue("Name"),
						Email: strings.ToLower(q.FormValue("Email")),
						Phone: q.FormValue("Mobilephone"),
					},
					Password: q.FormValue("Password"),
				}

				db := dbinit()
				defer db.Close()

				_, err = r.Db("goShop").Table("clients").Insert(newUser).RunWrite(db)
				if err != nil {
					log.Println(err)
				}

				ulin := userLogedIn{
					UserLogin:     q.FormValue("Email"),
					IsUserLogedIn: true,
				}

				session.Values["userLogedIn"] = ulin
				session.Values["emailforregistration"] = nil
				session.Save(q, w)
				http.Redirect(w, q, "/addressselect", http.StatusFound)
			} else {
				//check this definition
				newmap := map[string]interface{}{"errormessage": "Your email allready registered", "email": q.FormValue("email"), "repeatemail": q.FormValue("repeatemail"), "name": q.FormValue("name"), "mobilephone": q.FormValue("mobilephone")}
				render(w, "userRegistration", newmap)
			}

		} else {

			newmap := map[string]interface{}{"errormessage": "email or passwords does not match", "email": q.FormValue("email"), "repeatemail": q.FormValue("repeatemail"), "name": q.FormValue("name"), "mobilephone": q.FormValue("mobilephone")}
			render(w, "userRegistration", newmap)

		}

	} else {
		if session.Values["emailforregistration"] != nil {
			email := session.Values["emailforregistration"].(string)
			newmap := map[string]interface{}{"email": email}
			render(w, "userRegistration", newmap)
		} else {
			//прописать если вошел то на свой аккаунт
			render(w, "userRegistration", nil)
		}

	}

}

func addressselect(w http.ResponseWriter, q *http.Request) {
	session, _ := store.Get(q, sessionname)
	if q.Method == "POST" {
		err := q.ParseForm()

		x := *session.Values["userLogedIn"].(*userLogedIn)

		if err != nil {
			// Handle error
			fmt.Println(err)
		}

		AdressToship := Adress{
			FullName:            q.FormValue("enterAddressFullName"),
			AddressLine1:        q.FormValue("enterAddressAddressLine1"),
			AddressLine2:        q.FormValue("enterAddressAddressLine2"),
			City:                q.FormValue("enterAddressCity"),
			StateProvinceRegion: q.FormValue("enterAddressStateOrRegion"),
			ZIP:                 q.FormValue("enterAddressPostalCode"),
			Country:             q.FormValue("enterAddressCountryCode"),
			PhoneNumber:         q.FormValue("enterAddressPhoneNumber"),
		}
		db := dbinit()
		defer db.Close()

		_, err = r.Table("clients").Filter(r.Row.Field("User").Field("Email").Eq(x.UserLogin)).Update(map[string]interface{}{"Adresses": r.Row.Field("Adresses").Append(AdressToship)}).Run(db)

		if err != nil {
			log.Println(err)
		}
		order := *session.Values["Order"].(*Order)
		order.BillingAdress = AdressToship
		order.ShippingAdress = AdressToship
		session.Values["Order"] = order
		session.Save(q, w)
		http.Redirect(w, q, "/shipoptionselect", http.StatusFound)

	} else {
		command := mux.Vars(q)["command"]
		index := mux.Vars(q)["index"]
		if (len(command) == 0) && (len(index) == 0) {

			x := *session.Values["userLogedIn"].(*userLogedIn)

			db := dbinit()
			defer db.Close()

			row, err := r.Table("clients").Filter(r.Row.Field("User").Field("Email").Eq(x.UserLogin)).Run(db)

			if err != nil {
				log.Println(err)
			}
			var u UserRegistration
			row.One(&u)

			if err != nil {
				log.Println(err)
			}
			//перенести в buy
			order := *session.Values["Order"].(*Order)
			order.Customer = u.User

			session.Values["Order"] = order

			session.Save(q, w)

			newmap := map[string]interface{}{"Adresses": u.Adresses}
			render(w, "addressselect", newmap)

		} else if (len(command) > 0) && (len(index) > 0) {
			index, _ := strconv.Atoi(index)
			if command == "shiptothisadress" {
				session, _ := store.Get(q, sessionname)

				x := *session.Values["userLogedIn"].(*userLogedIn)
				order := *session.Values["Order"].(*Order)

				db := dbinit()
				defer db.Close()

				row, err := r.Table("clients").Filter(r.Row.Field("User").Field("Email").Eq(x.UserLogin)).Run(db)

				if err != nil {
					log.Println(err)
				}
				var u UserRegistration
				row.One(&u)
				adress := u.Adresses[index]
				order.BillingAdress = adress
				order.ShippingAdress = adress
				session.Values["Order"] = order
				session.Save(q, w)
				http.Redirect(w, q, "/shipoptionselect", http.StatusFound)
			} else if command == "edit" {
				//EDIT ADRESS
				session, _ := store.Get(q, sessionname)
				x := *session.Values["userLogedIn"].(*userLogedIn)
				db := dbinit()
				defer db.Close()

				rows, err := r.Table("clients").Filter(r.Row.Field("User").Field("Email").Eq(x.UserLogin)).Field("Adresses").Run(db)

				if err != nil {
					log.Println(err)
				}
				var a []Adress
				err = rows.One(&a)
				if err != nil {
					log.Println(err)
				}
				newmap := map[string]interface{}{"adress": a[index]}
				///////SAVE EDITED ADRESS
				render(w, "addressedit", newmap)
			} else {
				//DELETE ADRESS
				session, _ := store.Get(q, sessionname)
				x := *session.Values["userLogedIn"].(*userLogedIn)
				db := dbinit()
				defer db.Close()

				_, err := r.Table("clients").Filter(r.Row.Field("User").Field("Email").Eq(x.UserLogin)).Update(map[string]interface{}{"Adresses": r.Row.Field("Adresses").DeleteAt(index)}).Run(db)

				if err != nil {
					log.Println(err)
				}
				http.Redirect(w, q, "/addressselect", http.StatusFound)
			}

		}

	}

}

func shipoptionselect(w http.ResponseWriter, q *http.Request) {
	if q.Method == "POST" {
		err := q.ParseForm()
		if err != nil {
			// Handle error
			fmt.Println(err)
		}
		session, _ := store.Get(q, sessionname)

		order := *session.Values["Order"].(*Order)
		//here put shipping method and price shipping
		order.Shipping = Shipping{
			ShippingMethod: q.FormValue("shippingOfferingId"),
			ShippingCost:   0,
		}
		session.Values["Order"] = order
		session.Save(q, w)
		http.Redirect(w, q, "/payselect", http.StatusFound)
	} else {
		session, _ := store.Get(q, sessionname)
		order := *session.Values["Order"].(*Order)
		newmap := map[string]interface{}{"adress": order.ShippingAdress}
		render(w, "shipoptionselect", newmap)
	}

}

func payselect(w http.ResponseWriter, q *http.Request) {
	if q.Method == "POST" {
		err := q.ParseForm()
		if err != nil {
			// Handle error
			fmt.Println(err)
		}
		session, _ := store.Get(q, sessionname)

		order := *session.Values["Order"].(*Order)

		order.PaymentMethod = q.FormValue("payselect")
		session.Values["Order"] = order
		session.Save(q, w)
		http.Redirect(w, q, "/buy", http.StatusFound)

	} else {

		render(w, "payselect", nil)
	}

}

func buy(w http.ResponseWriter, q *http.Request) {

	ctx, err := NewContext(q)
	if err != nil {
		log.Println(err)
	}

	if q.Method == "POST" {
		//find last code(chek if it) of order +1 add to db
		t := time.Now()

		order := ctx.Session.Values["Order"].(*Order)

		a := false
		for a == false {

			orderId := getorderId(t.Format("2006-01-02"), order.Customer.Email, randStr(4, "alphanum"))

			res, err := r.Table("orders").Filter(map[string]interface{}{"OrderNumber": orderId}).Run(ctx.Db)

			if err != nil {
				log.Println(err)
			}

			if res.IsNil() == true {
				p := findproductCart{
					CartArray: order.CartOrderd,
				}
				_, total := p.total()

				P, _, _, _ := statuses()
				//P, PfS, S, D := statuses()

				order.OrderNumber = orderId
				order.Status = P
				order.Date = t.Format("2006-01-02 15:04")
				order.Total = FloatToString(total)

				_, err = r.Table("orders").Insert(order).RunWrite(ctx.Db)
				if err != nil {
					log.Println(err)
				}
				ctx.Session.Values["Order"] = nil
				ctx.Session.Save(q, w)
				a = true
			} else {
				a = false
			}

		}

	} else {

		order := ctx.Session.Values["Order"].(*Order)
		newmap := map[string]interface{}{"Shippingadress": order.ShippingAdress, "Billingadress": order.BillingAdress, "Paymethod": order.PaymentMethod, "shippingOpt": order.Shipping.ShippingMethod, "Product": order.CartOrderd}
		render(w, "buy", newmap)
	}
}

func success(w http.ResponseWriter, q *http.Request) {
}

func youraccount(w http.ResponseWriter, q *http.Request) {
	if q.Method == "POST" {
	} else {

	}
}

////////////////////////////////////////////////////////////////////
////////
////////CUSTOMER TRIGER SECTION
////////
///////////////////////////////////////////////////////////////////

////////////////////////////////////////////////////////////////////
////////
////////ADMIN SECTION
////////
///////////////////////////////////////////////////////////////////

func adminlogin(w http.ResponseWriter, q *http.Request) {
	if q.Method == "POST" {
		err := q.ParseForm()

		if err != nil {
			// Handle error
			fmt.Println(err)
		}

		decoder := schema.NewDecoder()
		adm := new(authUser)

		decoder.Decode(adm, q.PostForm)

		if adm.checkstaffloginpassword() == true {
			session, _ := store.Get(q, sessionname)
			ulin := staffLogedIn{
				UserLogin:     adm.Login,
				IsUserLogedIn: true,
			}

			session.Values["staffLogedIn"] = ulin

			session.Save(q, w)
			render(w, "admin", nil)
			templatesA.ExecuteTemplate(w, "admin", nil)
		} else {
			newmap := map[string]interface{}{"metatitle": "Registration", "errormessage": "Wrong login or password"}
			render(w, "adminlogin", newmap)
		}

	} else {
		session, _ := store.Get(q, sessionname)
		loggedIn := *session.Values["staffLogedIn"].(*staffLogedIn)
		if loggedIn.IsUserLogedIn == true {
			templatesA.ExecuteTemplate(w, "admin", nil)
		} else {
			render(w, "adminlogin", nil)
		}

	}

}

func adminornot(w http.ResponseWriter, q *http.Request) {
	err := q.ParseForm()

	if err != nil {
		// Handle error
		fmt.Println(err)
	}

	decoder := schema.NewDecoder()
	adm := new(authUser)

	decoder.Decode(adm, q.PostForm)

	if adm.checkstaffloginpassword() == true {
		session, _ := store.Get(q, sessionname)
		ulin := staffLogedIn{
			UserLogin:     adm.Login,
			IsUserLogedIn: true,
		}

		session.Values["staffLogedIn"] = ulin

		session.Save(q, w)
		render(w, "admin", nil)
	} else {
		newmap := map[string]interface{}{"metatitle": "Registration", "errormessage": "Wrong login or password"}
		render(w, "adminlogin", newmap)
	}

}

func items(w http.ResponseWriter, q *http.Request) {

	ctx, err := NewContext(q)
	if err != nil {
		log.Println(err)
	}

	rows, err := r.Db("goShop").Table("items").OrderBy("SKU").Run(ctx.Db)

	if err != nil {
		log.Println(err)
	}
	var t []Product

	err = rows.All(&t)
	if err != nil {
		log.Println(err)
	}

	newmap := map[string]interface{}{"Products": t}

	templatesA.ExecuteTemplate(w, "items", newmap)

}

func addItem(w http.ResponseWriter, q *http.Request) {
	//check if SKU exist, if not then generate SKU
	err := q.ParseMultipartForm(32 << 20)

	if err != nil {
		// Handle error
		fmt.Println(err)
	}
	if q.FormValue("SKU") != "" {

		slice := []string{"Image", "Image1", "Image2", "Image3"}

		images := checkifImageneedUpload(q, slice, q.FormValue("SKU"))

		product := Product{
			Title:       q.FormValue("Title"),
			Price:       q.FormValue("Price"),
			SKU:         q.FormValue("SKU"),
			Image:       images,
			Description: q.FormValue("Description"),
		}

		db := dbinit()
		defer db.Close()
		_, err = r.Db("goShop").Table("items").Insert(product).RunWrite(db)
		if err != nil {
			log.Println(err)
		}

		//fmt.Println(q.FormValue("cat"))

		http.Redirect(w, q, "/items", http.StatusFound)
	}

}

func deleteProduct(w http.ResponseWriter, q *http.Request) {
	err := q.ParseForm()
	if err != nil {
		// Handle error
		fmt.Println(err)
	}
	fmt.Println(q.Form["quq[]"])

}

func orders(w http.ResponseWriter, q *http.Request) {
	ctx, err := NewContext(q)
	if err != nil {
		log.Println(err)
	}

	rows, err := r.Db("goShop").Table("orders").OrderBy(r.Desc("Date")).Run(ctx.Db)

	if err != nil {
		log.Println(err)
	}
	var t []Order
	err = rows.All(&t)
	if err != nil {
		log.Println(err)
	}

	newmap := map[string]interface{}{"Orders": t}

	templatesA.ExecuteTemplate(w, "orders", newmap)

}

func getorder(w http.ResponseWriter, q *http.Request) {

	orderId := mux.Vars(q)["orderId"]

	ctx, err := NewContext(q)
	if err != nil {
		log.Println(err)
	}

	res, err := r.Db("goShop").Table("orders").Filter(map[string]interface{}{"OrderNumber": orderId}).Run(ctx.Db)
	if err != nil {
		log.Println(err)
	}
	var order Order
	res.One(&order)

	newmap := map[string]interface{}{"Order": order}
	templatesA.ExecuteTemplate(w, "getOrder", newmap)
}

func clients(w http.ResponseWriter, q *http.Request) {
	db := dbinit()
	defer db.Close()

	rows, err := r.Db("goShop").Table("clients").Run(db)

	if err != nil {
		log.Println(err)
	}
	var t []UserRegistration
	err = rows.All(&t)
	if err != nil {
		log.Println(err)
	}

	newmap := map[string]interface{}{"Users": t}
	render(w, "clients", newmap)

}

func options(w http.ResponseWriter, q *http.Request) {
	render(w, "options", nil)

}

func logoutuser(w http.ResponseWriter, q *http.Request) {

	session, _ := store.Get(q, sessionname)
	session.Values["userLogedIn"] = nil
	session.Save(q, w)
	http.Redirect(w, q, "/", http.StatusFound)

}

func logout(w http.ResponseWriter, q *http.Request) {

	session, _ := store.Get(q, sessionname)
	session.Values["staffLogedIn"] = nil
	session.Save(q, w)
	http.Redirect(w, q, "/admin", http.StatusFound)

}

func clearCart(w http.ResponseWriter, q *http.Request) {

	session, _ := store.Get(q, sessionname)
	session.Values["Order"] = nil
	session.Save(q, w)
	fmt.Println(session.Values["Cart"])

}

////////////////////////////////////////////////////////////////////
////////
////////ADMIN TRIGER SECTION
////////
///////////////////////////////////////////////////////////////////
