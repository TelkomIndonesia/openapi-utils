package main

import (
	"context"
	"log"
	"os"

	"github.com/telkomindonesia/openapi-utils/cmd/proxy/internal/proxy"
)

func main() {
	if len(os.Args) < 2 {
		log.Fatalf("Usage: %s <path-to-proxy-spec> [<path-to-new-spec>]\n", os.Args[0])
	}

	src := os.Args[1]
	bytes, _, err := proxy.CompileByte(context.Background(), src)
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
