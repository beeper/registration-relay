package config

type Config struct {
	Version string
	Secret  []byte
	API     struct {
		Listen          string
		ValidateAuthURL string
	}
}
