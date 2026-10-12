// Command order-shared-pg-install emits the fixed management-plane SQL.
// It does not connect to a database or accept a caller-supplied SQL source.
package main

import (
	"fmt"
	"io"
	"os"

	"github.com/gaofeng30/order/services/api/internal/migrate"
)

func main() {
	if len(os.Args) != 1 {
		fmt.Fprintln(os.Stderr, "usage: order-shared-pg-install")
		os.Exit(2)
	}
	query, err := migrate.SharedPostgreSQLInstallSQL()
	if err != nil {
		fmt.Fprintln(os.Stderr, "shared_postgresql_install_generation_failed")
		os.Exit(1)
	}
	if _, err := io.WriteString(os.Stdout, query); err != nil {
		fmt.Fprintln(os.Stderr, "shared_postgresql_install_output_failed")
		os.Exit(1)
	}
}
