// Command app is the SoW backend server.
package main

import (
	"go.uber.org/fx"

	"github.com/pixels-two/sow/backend/internal/boot"
)

func main() {
	fx.New(boot.AppOptions()).Run()
}
