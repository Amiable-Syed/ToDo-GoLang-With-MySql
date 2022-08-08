package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"

	"context"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/gorilla/mux"
)

const (
	username = "root"
	password = "1234"
	hostname = "127.0.0.1:3306"
	dbname   = "testdb"
)

type ToDo struct {
	Title string `json:"title"`
	Desc  string `json:"desc"`
	Id    string `json:"id"`
}

var Connector *sql.DB

func dsn(dbName string) string {
	return fmt.Sprintf("%s:%s@tcp(%s)/%s", username, password, hostname, dbName)
}

func homePage(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Welcome to the Home Page!")
}
func returnAllToDos(w http.ResponseWriter, r *http.Request) {
	var ToDos []ToDo
	ToDos, err := getAllFields()
	if err != nil {
		panic(err.Error())
	}
	fmt.Printf("ToDos 0 index in returnAllToDos :\n %s \t Id:\n %s\n", ToDos[0].Desc, ToDos[0].Id)
	json.NewEncoder(w).Encode(ToDos)
}

func returnSingleToDo(w http.ResponseWriter, r *http.Request) {
	// vars := mux.Vars(r)
	// key := vars["id"]

	//Loop on all the ToDos
	// get the matched Id and return
	// the object encoded as json
	// for _, todo := range ToDos {
	// 	if todo.id == key {
	// 		json.NewEncoder(w).Encode(todo)
	// 	}
	// }

}

func createNewToDo(w http.ResponseWriter, r *http.Request) {
	reqBody, _ := ioutil.ReadAll(r.Body)
	log.Printf("Request Body %s", reqBody)
	// get the body of our POST request
	// unmarshal this into a new ToDo struct
	// and insert into our database table.
	var todo ToDo
	err := json.Unmarshal(reqBody, &todo)
	if err != nil {
		fmt.Printf("There was an error decoding the json. err = %s", err)
		return
	}
	//update the database to include our new ToDo
	err = insert(todo)
	if err != nil {
		log.Printf("Insert product failed with error %s", err)
		return
	}
	json.NewEncoder(w).Encode(todo)
}

func deleteToDo(w http.ResponseWriter, r *http.Request) {
	// vars := mux.Vars(r)
	// key := vars["id"]

	// //Loop on all the ToDos
	// // get the matched Id and remove
	// // the object from the global array
	// for idx, todo := range ToDos {
	// 	if todo.id == key {
	// 		ToDos = append(ToDos[:idx], ToDos[idx+1:]...)
	// 	}
	// }

}

func handleRequests() {
	fmt.Println("ToDo Apis")
	// Mux Router introduced instead traditional net/http router
	myRouter := mux.NewRouter().StrictSlash(true)
	myRouter.HandleFunc("/", homePage)
	myRouter.HandleFunc("/todos", returnAllToDos)
	myRouter.HandleFunc("/todo/{id}", returnSingleToDo)
	myRouter.HandleFunc("/todo/{id}", deleteToDo).Methods("DELETE")
	myRouter.HandleFunc("/todo", createNewToDo).Methods("POST")
	log.Fatal(http.ListenAndServe(":1000", myRouter))
}

func dbConnection() error {
	var err error
	Connector, err = sql.Open("mysql", dsn(""))
	if err != nil {
		log.Printf("Error %s when opening DB\n", err)
		return err
	}

	ctx, cancelfunc := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelfunc()
	res, err := Connector.ExecContext(ctx, "CREATE DATABASE IF NOT EXISTS "+dbname)
	if err != nil {
		log.Printf("Error %s when creating DB\n", err)
		return err
	}
	no, err := res.RowsAffected()
	if err != nil {
		log.Printf("Error %s when fetching rows", err)
		return err
	}
	log.Printf("rows affected %d\n", no)

	if Connector != nil {
		Connector.Close()
	}

	Connector, err = sql.Open("mysql", dsn(dbname))
	if err != nil {
		log.Printf("Error %s when opening DB", err)
		return err
	}

	Connector.SetMaxOpenConns(20)
	Connector.SetMaxIdleConns(20)
	Connector.SetConnMaxLifetime(time.Minute * 5)

	ctx, cancelfunc = context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelfunc()
	err = Connector.PingContext(ctx)
	log.Print(err)
	if err != nil {
		log.Printf("Errors %s pinging DB", err)
		return err
	}
	log.Printf("Connected to DB %s successfully\n", dbname)
	return nil
}

func createToDoTable() error {
	query := `CREATE TABLE IF NOT EXISTS todo(todo_id int primary key auto_increment, title text, 
        todo_desc text, created_at datetime default CURRENT_TIMESTAMP, updated_at datetime default CURRENT_TIMESTAMP)`
	ctx, cancelfunc := context.WithTimeout(context.Background(), 5*time.Second)

	defer cancelfunc()
	if Connector != nil {
		res, err := Connector.ExecContext(ctx, query)

		if err != nil {
			log.Printf("Error %s when creating todo table", err)
			return err
		}
		rows, err := res.RowsAffected()
		if err != nil {
			log.Printf("Error %s when getting rows affected", err)
			return err
		}
		log.Printf("Rows affected when creating table: %d", rows)
	}

	return nil
}

func insert(todo ToDo) error {
	query := "INSERT INTO todo(title, todo_desc) VALUES (?, ?)"
	ctx, cancelfunc := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelfunc()
	stmt, err := Connector.PrepareContext(ctx, query)
	if err != nil {
		log.Printf("Error %s when preparing SQL statement", err)
		return err
	}
	defer stmt.Close()
	res, err := stmt.ExecContext(ctx, todo.Title, todo.Desc)
	if err != nil {
		log.Printf("Error %s when inserting row into todos table", err)
		return err
	}
	rows, err := res.RowsAffected()
	if err != nil {
		log.Printf("Error %s when finding rows affected", err)
		return err
	}
	log.Printf("%d todo created ", rows)
	TodoID, err := res.LastInsertId()
	if err != nil {
		log.Printf("Error %s when getting last inserted todo", err)
		return err
	}
	log.Printf("ToDo with ID %d created", TodoID)
	return nil
}

func getAllFields() ([]ToDo, error) {
	log.Printf("Getting all todos")
	query := `SELECT todo_id,title,todo_desc FROM todo`

	rows, err := Connector.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	// An todo slice to hold data from returned rows.
	var ToDos []ToDo

	// Loop through rows, using Scan to assign column data to struct fields.
	for rows.Next() {
		var todo ToDo
		if err := rows.Scan(&todo.Id, &todo.Title, &todo.Desc); err != nil {
			return ToDos, err
		}
		ToDos = append(ToDos, todo)
	}
	if err = rows.Err(); err != nil {
		return ToDos, err
	}
	return ToDos, nil

}
func main() {
	err := dbConnection()
	if err != nil {
		log.Printf("Error %s when getting db connection", err)
		return
	}

	log.Printf("Successfully connected to database")
	err = createToDoTable()
	if err != nil {
		log.Printf("Create ToDo table failed with error %s", err)
		return
	}

	// todo1 := ToDo{
	// 	Title: "Task 1",
	// 	Desc:  "Task 1 to be completed quickly",
	// }
	// todo2 := ToDo{
	// 	Title: "Task 2",
	// 	Desc:  "Task 2 to be completed quickly",
	// }
	// err = insert(todo1)
	// if err != nil {
	// 	log.Printf("Insert product failed with error %s", err)
	// 	return
	// }

	// err = insert(todo2)
	// if err != nil {
	// 	log.Printf("Insert product failed with error %s", err)
	// 	return
	// }

	handleRequests()
}
