package config

import (
	"log"
	"os"
	"path/filepath"

	"github.com/knadh/koanf/parsers/json"
	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/confmap"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/v2"
)

func NewConfig() (*koanf.Koanf, error) {
	var k = koanf.New(".")

	//default
	setDefault(k)

	f := file.Provider("config.yml")
	if err := k.Load(f, yaml.Parser()); err != nil {
		return nil, err
	}

	_ = f.Watch(func(event any, err error) {
		if err != nil {
			log.Println(err)
			return
		}
		log.Println("config file changed")
		k = koanf.New(".")
		_ = k.Load(f, yaml.Parser())
		k.Print()
	})

	return k, nil
}

func NewTestConfig() (*koanf.Koanf, error) {

	dir, _ := os.Getwd()
	p := filepath.Join(dir, "mock/mock.json")
	var k = koanf.New(".")
	//use env
	if err := k.Load(file.Provider(p), json.Parser()); err != nil {
		log.Fatalf("error loading config: %v", err)
	}

	return k, nil
}

func setDefault(k *koanf.Koanf) {

	_ = k.Load(confmap.Provider(map[string]any{
		"db.tablePrefix": "",
	}, "."), nil)
}
