package config

import (
	"os"
	"strconv"
)

// Config holds server configuration, populated from environment variables.
type Config struct {
	// HTTP server port
	Port int
	// gRPC server port
	GRPCPort int
	// Data directory for CRL data and snapshots
	DataDir string
	// CRL URLs per issuer
	CRLG2URL string
	CRLG3URL string
	// CRL polling interval in seconds
	CRLPollInterval int
	// Blockchain RPC URL
	RPCURL string
	// Relayer private key (hex)
	RelayerPrivateKey string
	// Contract address
	ContractAddress string
	// GitHub repo for snapshot downloads
	GitHubRepo string
}

func Load() *Config {
	return &Config{
		Port:              getEnvInt("PORT", 3000),
		GRPCPort:          getEnvInt("GRPC_PORT", 50051),
		DataDir:           getEnv("DATA_DIR", "./data"),
		CRLG2URL:          getEnv("CRL_G2_URL", "https://moica.nat.gov.tw/repository/MOICA/CRL2/complete.crl"),
		CRLG3URL:          getEnv("CRL_G3_URL", "https://crl-moica.moi.gov.tw/crl/MOICA-G3-complete.crl"),
		CRLPollInterval:   getEnvInt("CRL_POLL_INTERVAL", 21600), // 6 hours
		RPCURL:            getEnv("RPC_URL", ""),
		RelayerPrivateKey: getEnv("RELAYER_PRIVATE_KEY", ""),
		ContractAddress:   getEnv("CONTRACT_ADDRESS", ""),
		GitHubRepo:        getEnv("GITHUB_REPO", "moven0831/moica-revocation-smt"),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return fallback
}
