package main

import (
	"context"
	"log"
	"os"
	"path/filepath"

	"github.com/telkomindonesia/openapi-utils/cmd/proxy/internal/proxy"
)

func main() {
	src := os.Args[1]
	sf, _ := filepath.Abs(src)
	specDir, _ := filepath.Abs(filepath.Dir(src))
	specBytes, _ := os.ReadFile(sf)

	if len(os.Args) < 2 {
		log.Fatalf("Usage: %s <path-to-proxy-spec> [<path-to-new-spec>]\n", os.Args[0])
	}

	bytes, _, _, err := proxy.CompileByte(context.Background(), specBytes, specDir)
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
