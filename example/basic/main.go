package main

import (
	"fmt"
	"log"

	_ "github.com/lib/pq"
	"github.com/pepega90/storm"
	"github.com/pepega90/storm/models"
)

func main() {
	dsn := "host=localhost user=postgres password=pepega90 dbname=storm_db port=5432 sslmode=disable TimeZone=Asia/Jakarta"
	storm, err := storm.New("postgres", dsn)
	if err != nil {
		log.Fatal("Erro storm is not initiate:", err.Error())
	}
	// insert
	// user := &models.User{
	// 	Name:  "dikha",
	// 	Email: "dikha@gmail.com",
	// }
	// err = storm.Insert(user)
	// if err != nil {
	// 	log.Fatal("Error insert data:", err.Error())
	// }

	// update
	// user := &models.User{
	// 	ID:    5,
	// 	Name:  "ammar",
	// 	Email: "dikha@pepeg.com",
	// }
	// err = storm.Update(user)
	// if err != nil {
	// 	log.Fatal("Error update data:", err.Error())
	// }

	// delete
	// user := &models.User{
	// 	ID: 5,
	// }
	// err = storm.Delete(user)
	// if err != nil {
	// 	log.Fatal("Error delete data:", err.Error())
	// }

	// query
	// SELECT
	// var users []models.User
	// err = storm.
	// 	From(&models.User{}).
	// 	Limit(1).
	// 	Select(&users, "email")

	// First
	// var user models.User
	// err = storm.
	// 	From(&models.User{}).
	// 	Where("id = $1", 14).
	// 	First(&user)

	// Pagination
	var user []models.User
	var total, totalPages int
	page := 2
	pageSize := 3
	err = storm.
		From(&models.User{}).
		Paginate(&user, page, pageSize, &total, &totalPages, "name_user")
	if err != nil {
		log.Fatal("Error get users: ", err.Error())
	}
	fmt.Println("Page: ", page)
	fmt.Println("Page Size: ", pageSize)
	fmt.Println("Total User: ", total)
	fmt.Println("Total Pages: ", totalPages)
	fmt.Println("User data: ", user)
}
