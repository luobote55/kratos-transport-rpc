package mqtt

import (
	"bytes"
	"encoding/json"
	"fmt"
	"testing"
)

type User struct {
	Name string `json:"name"`
	Age  int    `json:"age"`
}

func TestClient(t *testing.T) {
	user := User{
		Name: "Jack",
		Age:  18,
	}

	var buf bytes.Buffer
	encoder := json.NewEncoder(&buf)
	if err := encoder.Encode(user); err != nil {
		fmt.Println(err)
		return
	}

	if buf.Len() > 1024*1024 {
		fmt.Println("JSON string is too long, skip serialization")
		return
	}

	fmt.Println(buf.String())
}
