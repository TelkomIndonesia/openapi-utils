package proxy

import (
	"encoding/json"
	"fmt"
	"strconv"

	"gopkg.in/yaml.v3"
)

func printYAMLNode(name string, y ...*yaml.Node) {
	fmt.Println(name)
	for i, y := range y {
		v := map[string]interface{}{}
		y.Decode(&v)
		b, _ := json.Marshal(v)
		fmt.Println(strconv.Itoa(i), ": ", string(b))
	}
	fmt.Println()
}

type renderable interface {
	Render() ([]byte, error)
}

func copyRenderable[T renderable](t T) (n T, err error) {
	b, err := t.Render()
	if err != nil {
		return n, err
	}

	err = yaml.Unmarshal(b, &n)
	return
}
