package configs

import (
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

// SaveTOML saves a struct to a TOML file.
func SaveTOML(filePath string, data interface{}) error {
	if err := os.MkdirAll(filepath.Dir(filePath), 0700); err != nil {
		return err
	}

	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	return toml.NewEncoder(file).Encode(data)
}

// LoadTOML loads a TOML file into a struct.
func LoadTOML(filePath string, data interface{}) error {
	_, err := toml.DecodeFile(filePath, data)
	return err
}
