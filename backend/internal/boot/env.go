package boot

import (
	"os"

	"github.com/joho/godotenv"
)

// LoadDotenv loads .env into the process environment for non-prod
// stages. A missing .env is fine. Real environment variables win over
// .env values.
func LoadDotenv() {
	if os.Getenv("STAGE") == "prod" {
		return
	}

	_ = godotenv.Load()
}
