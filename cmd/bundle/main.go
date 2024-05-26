package main

import (
	"log"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		log.Fatalf("Usage: %s <path-to-main-spec> [<path-to-new-spec>]\n", os.Args[0])
	}

	bytes, err := bundleFile(os.Args[1])
	if err != nil {
		log.Fatalln("fail to bundle file:", err)
	}

	dst := ""
	if len(os.Args) > 2 {
		dst = os.Args[2]
	}
	switch dst {
	case "":
		if _, err := os.Stdout.Write(bytes); err != nil {
			log.Fatalln("fail to write stdout:", err)
		}
	default:
		if err := os.WriteFile(dst, bytes, 0644); err != nil {
			log.Fatalln("fail to write file:", err)
		}
	}
}