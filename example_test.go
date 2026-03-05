package toon

import "fmt"

func ExampleAsObject() {
	v, err := Unmarshal([]byte("name: Alice"))
	if err != nil {
		fmt.Println("decode error")
		return
	}
	obj, ok := AsObject(v)
	if !ok {
		fmt.Println("not object")
		return
	}
	name, _ := obj.Get("name")
	fmt.Println(name)
	// Output: Alice
}

func ExampleAsArray() {
	input := []byte("users[2]{id,name}:\n  1,Alice\n  2,Bob")
	v, err := Unmarshal(input)
	if err != nil {
		fmt.Println("decode error")
		return
	}

	root, ok := AsObject(v)
	if !ok {
		fmt.Println("not object")
		return
	}
	usersV, _ := root.Get("users")
	users, ok := AsArray(usersV)
	if !ok {
		fmt.Println("not array")
		return
	}
	row0, ok := AsObject(users[0])
	if !ok {
		fmt.Println("not row object")
		return
	}
	id, _ := row0.Get("id")
	name, _ := row0.Get("name")
	fmt.Printf("%v %v\n", id, name)
	// Output: 1 Alice
}

func ExampleAsArray_strictTabularRoot() {
	input := []byte("[2]{id,name}:\n  1,Alice\n  2,Bob")
	v, err := Unmarshal(input)
	if err != nil {
		fmt.Println("decode error")
		return
	}
	rows, ok := AsArray(v)
	if !ok {
		fmt.Println("not array")
		return
	}
	row1, ok := AsObject(rows[1])
	if !ok {
		fmt.Println("not row object")
		return
	}
	name, _ := row1.Get("name")
	fmt.Println(name)
	// Output: Bob
}
