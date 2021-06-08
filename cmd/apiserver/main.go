package main

import (
	"log"

	"github.com/BurntSushi/toml"
	"github.com/Sna1l1/rest-api/internal/app/apiserver/apiserver"
)

func main() {
	var config *apiserver.Config
	_, err := toml.DecodeFile("E:\\Prog\\GoProjects\\rest-api\\config\\config.toml", &config)
	if err != nil {
		log.Fatal(err)
	}

	if err = apiserver.Start(config); err != nil {
		log.Fatal(err)
	}

}
