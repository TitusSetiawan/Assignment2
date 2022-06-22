package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
)

const (
	host     = "localhost"
	port     = 5432
	userPq   = "postgres"
	password = "1234"
	dbname   = "ordered_by"
)

var (
	db  *sql.DB
	err error
)

type Item struct {
	Item_id      int
	Item_code    int
	Description  string
	Quantity     int
	Itemorder_id int
}

type Order struct {
	Order_id      int
	Ordered_at    time.Time
	Customer_name string
	Item          []Item
}

type ItemFromDB struct {
	Item_id      int
	Item_code    int
	Description  string
	Quantity     int
	Itemorder_id int
}

type OrderFromDB struct {
	Order_id      int
	Ordered_at    time.Time
	Customer_name string
	Item          []Item
}

func createDB(w http.ResponseWriter, r *http.Request) {
	var NewOrder Order
	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(&NewOrder)
	if err != nil {
		panic(err.Error())
	}
	json.NewEncoder(w).Encode(NewOrder)

	// Create DB
	var NewOderFromDb = OrderFromDB{}
	sqlStatement := `
	INSERT INTO orders(customer_name, ordered_at)
	VALUES($1, $2)
	returning order_id
	`
	err = db.QueryRow(sqlStatement, NewOrder.Customer_name, NewOrder.Ordered_at).
		Scan(&NewOderFromDb.Order_id)
	if err != nil {
		panic(err)
	}

	for _, item := range NewOrder.Item {
		item.Itemorder_id = NewOderFromDb.Order_id
		sqlStatement2 := `
		INSERT INTO items(item_code, description, quantity, itemorder_id)
		VALUES($1, $2, $3, $4)
		returning itemorder_id; `
		err = db.QueryRow(sqlStatement2,
			item.Item_code,
			item.Description,
			item.Quantity,
			item.Itemorder_id,
		).
			Scan(&item.Itemorder_id)
		if err != nil {
			panic(err)
		}
	}
	fmt.Println("Done")
	// fmt.Println(NewOrder.Item[0].Item_code)
}

func getDB(w http.ResponseWriter, r *http.Request) {
	var NewOderFromDb = []OrderFromDB{}
	sqlStatement :=
		`select
		o.order_id
		,o.customer_name
		,o.ordered_at
		,json_agg(json_build_object(
			'item_id',i.item_id
			,'item_code',i.item_code
			,'description',i.description
			,'quantity',i.quantity
			,'itemorder_id',i.itemorder_id
		)) as items
	from public.orders o join items i
	on o.order_id = i.item_id
	group by o.order_id`
	rows, err := db.Query(sqlStatement)
	if err != nil {
		panic(err)
	}
	defer rows.Close()
	var orders []*Order
	var items []Item

	for rows.Next() {
		var o Order
		var itemsStr string
		if err = rows.Scan(
			&o.Order_id,
			&o.Customer_name,
			&o.Ordered_at,
			&itemsStr,
		); err != nil {
			fmt.Println("No Data", err)
		}

		if err := json.Unmarshal([]byte(itemsStr), &items); err != nil {
			fmt.Println("Error when parsing data items")
		} else {
			o.Item = append(o.Item, items...)
		}
		orders = append(orders, &o)
		jsonData, _ := json.Marshal(&orders)
		w.Header().Add("Content-Type", "application/json")
		w.Write(jsonData)
	}
	// fmt.Println(NewUserFromDB)
	json.NewEncoder(w).Encode(NewOderFromDb)

}

func updateDB(w http.ResponseWriter, r *http.Request) {
	var NewOrder Order
	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(&NewOrder)
	if err != nil {
		panic(err.Error())
	}
	json.NewEncoder(w).Encode(NewOrder)

	vars := mux.Vars(r)
	queryId, err := strconv.Atoi(vars["id"])
	if err != nil {
		panic(err.Error())
	}
	// fmt.Println(queryId)

	//Update Orders From DB
	sqlStatement := `
	update orders 
	set customer_name = $2, ordered_at = $3
	where order_id = $1;`
	res, err := db.Exec(sqlStatement, queryId, NewOrder.Customer_name, NewOrder.Ordered_at) //urutan $1, $2, $3

	if err != nil {
		panic(err)
	}
	count, err := res.RowsAffected()
	if err != nil {
		panic(err)
	}
	json.NewEncoder(w).Encode(count)

	//Update Items From DB
	for _, item := range NewOrder.Item {
		sqlStatement2 := `
		update items 
		set item_code = $2, description = $3, quantity = $4
		where item_id = $1;`
		res, err := db.Exec(sqlStatement2, queryId, item.Item_code, item.Description, item.Quantity) //urutan $1, $2, $3

		if err != nil {
			panic(err)
		}
		count, err := res.RowsAffected()
		if err != nil {
			panic(err)
		}
		json.NewEncoder(w).Encode(count)

	}
}

func deleteDB(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	queryId, err := strconv.Atoi(vars["id"])
	if err != nil {
		panic(err.Error())
	}

	//Delete items_id Table by Id
	sqlStatement := `
	DELETE from items
	WHERE itemorder_id = $1;`
	res, err := db.Exec(sqlStatement, queryId)
	if err != nil {
		panic(err)
	}
	count, err := res.RowsAffected()
	if err != nil {
		panic(err)
	}

	fmt.Println("Deleted data amount", count)
	sqlStatement = `
	DELETE from orders
	WHERE order_id = $1;`
	res, err = db.Exec(sqlStatement, queryId)
	if err != nil {
		panic(err)
	}
	count, err = res.RowsAffected()
	if err != nil {
		panic(err)
	}
	fmt.Println("Deleted data amount", count)
}

func main() {
	psqlInfo := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		host, port, userPq, password, dbname)
	db, err = sql.Open("postgres", psqlInfo)
	if err != nil {
		panic(err)
	}
	defer db.Close()

	err = db.Ping()
	if err != nil {
		panic(err)
	}
	log.Println("Connection Success")

	// Routing
	r := mux.NewRouter()
	r.HandleFunc("/orders", createDB).Methods("POST")
	r.HandleFunc("/orders", getDB).Methods("GET")
	r.HandleFunc("/orders/{id}", updateDB).Methods("PUT")
	r.HandleFunc("/orders/{id}", deleteDB).Methods("DELETE")
	http.Handle("/", r)
	http.ListenAndServe(":8080", r)
}
