package main

import (
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
	"fmt"
)


type Person struct {
	Name string
	Phone string
}


func main() {
	session, err := mgo.Dial("localhost")
	if err != nil {
		panic(err)
	}

	defer  session.Close()
	session.SetMode(mgo.Monotonic, true)

	c := session.DB("test").C("people")
	err = c.Insert(&Person{"Liqiang", "10086"},
				   &Person{"Yetship", "10010"})
	if err != nil {
		panic(err)
	}

	result := Person{}
	err = c.Find(bson.M{"name": "Yetship"}).One(&result)
	if err != nil {
		panic(err)
	}

	fmt.Println("Phone: ", result.Phone)
}
