package config

type GoogleDriveCredentials struct {
	SheetID                 string `env:"GD_SHEET_ID" env-required:"true"`
	Type                    string `env:"GD_TYPE" env-required:"true"`
	GDProjectID             string `env:"GD_PROJECT_ID" env-required:"true"`
	PrivateKeyID            string `env:"GD_PRIVATE_KEY_ID" env-required:"true"`
	PrivateKey              string `env:"GD_PRIVATE_KEY" env-required:"true"`
	ClientEmail             string `env:"GD_CLIENT_EMAIL" env-required:"true"`
	ClientID                string `env:"GD_CLIENT_ID" env-required:"true"`
	AuthURI                 string `env:"GD_AUTH_URI" env-required:"true"`
	TokenURI                string `env:"GD_TOKEN_URI" env-required:"true"`
	AuthProviderX509CertURL string `env:"GD_AUTH_PROVIDER" env-required:"true"`
	ClientX509CertURL       string `env:"GD_CLIENT_CERT_URL" env-required:"true"`
	UniverseDomain          string `env:"GD_UNIVERSE_DOMAIN" env-required:"true"`
	FileMetaData            string `env:"GD_FILE_METADATA" env-required:"true"`
}
