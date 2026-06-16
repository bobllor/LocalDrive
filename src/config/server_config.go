package config

import (
	"fmt"
	"os"

	"github.com/goccy/go-yaml"
	"github.com/joho/godotenv"
)

type ServerConfig struct {
	// Database holds the info for a database to establish a connection.
	Database DatabaseInfo `yaml:"database"`

	// EnvFiles holds the .env file paths to be loaded into the program. This will not override pairs given in
	// Environment.
	EnvFiles []string `yaml:"env_file"`

	// Environment holds a key-value pair of environment variables. These values will take precedent over values defined
	// inside EnvFiles. It is recommended to use EnvFiles instead.
	Environment map[string]string `yaml:"environment"`

	// StoragePath is the path to the storage for the files to be stored on.
	StoragePath string `yaml:"storage_path"`

	// ServerAddress is the TCP	address of the server. It is expected to be in the format of "host:port".
	ServerAddress string `yaml:"server_address"`
}

type DatabaseInfo struct {
	Name        string       `yaml:"name"`             // Name is the database name to connect to.
	Address     string       `yaml:"address"`          // Address is the address of the database. This includes the port.
	NetProtocol string       `yaml:"network_protocol"` // NetProtocol is the network protocol used for the connection.
	FileUser    DatabaseUser `yaml:"file_user"`        // FileUser represents the user that handles the File table.
	AccountUser DatabaseUser `yaml:"account_user"`     // AccountUser represents the user that handles the UserAccount table.
}

type DatabaseUser struct {
	// User is the username account used for the connection.
	User string `yaml:"username"`
}

const (
	EnvFilePwKey = "FILEUSER_PASSWORD"
	EnvUserPwKey = "ACCOUNTUSER_PASSWORD"
)

// NewServerConfig creates a new ServerConfig read from a path.
func NewServerConfig(path string) (*ServerConfig, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	config := &ServerConfig{}

	err = yaml.Unmarshal(b, config)
	if err != nil {
		return nil, err
	}

	return config, nil
}

// LoadEnv is a wrapper that calls s.LoadEnvironment and s.LoadEnvFiles.
// It will return an error if one occurs in either.
func (s *ServerConfig) LoadEnv() error {
	err := s.LoadEnvironment()
	if err != nil {
		return err
	}
	err = s.LoadEnvFiles()
	if err != nil {
		return err
	}

	return nil
}

// LoadEnvironment loads defined key-value pairs into the program.
func (s *ServerConfig) LoadEnvironment() error {
	for key, value := range s.Environment {
		err := os.Setenv(key, value)
		if err != nil {
			return err
		}
	}

	return nil
}

// LoadEnvFiles loads the slice of .env files into the program.
// If no env files are given, then it will not do anything.
//
// Variables that already exist will not be overridden.
func (s *ServerConfig) LoadEnvFiles() error {
	if len(s.EnvFiles) == 0 {
		return nil
	}

	err := godotenv.Load(s.EnvFiles...)
	if err != nil {
		return fmt.Errorf("failed to load .env files: %v", err)
	}

	return nil
}
