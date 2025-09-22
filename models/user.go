package models

type User struct {
	ID    int    `storm:"pk" json:"id"`
	Name  string `storm:"column:name_user" json:"name"`
	Email string `storm:"column:email_user" json:"email"`
}
