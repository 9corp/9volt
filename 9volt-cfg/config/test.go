package main

import (
    "fmt"
    "encoding/json"
)

func main() {
    data := `{"crap" : "bleep"}`
    
    if err := json.Unmarshal([]byte(data), &crap); if err != nil {
        fmt.Println(err)
    } else {
        fmt.Println("Worked!")
    }
}
