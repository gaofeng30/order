// Command order-shared-pg-install generates reviewed SQL without connecting to a database.
package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"net/url"
	"os"
	"strings"

	"github.com/gaofeng30/order/services/api/internal/migrate"
	"github.com/gaofeng30/order/services/api/sharedpg"
)

func generate(args []string) (string, error) {
	if len(args) == 0 {
		return migrate.SharedPostgreSQLInstallSQL()
	}
	flags := flag.NewFlagSet("order-shared-pg-install", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	functions := flags.Bool("functions", false, "generate function definitions and permissions")
	compensation := flags.Bool("compensation", false, "generate production durable scheduler installation")
	uid := flags.String("service-uid", "", "ordinary service identity UID")
	environment := flags.String("environment", "", "uat or production")
	origin := flags.String("origin", "", "verified HTTPS origin without trailing slash")
	invalid := errors.New("shared_postgresql_install_arguments_invalid")
	if flags.Parse(args) != nil || flags.NArg() != 0 || *functions == *compensation {
		return "", invalid
	}
	if *compensation {
		if *environment != "production" || *origin != "" {
			return "", invalid
		}
		return sharedpg.CompensationSQL(*uid)
	}
	u, err := url.Parse(*origin)
	if err != nil || u.Scheme != "https" || u.Host == "" || u.User != nil || u.Path != "" || u.RawQuery != "" || u.ForceQuery || strings.Contains(*origin, "#") {
		return "", invalid
	}
	return sharedpg.InstallSQL(*uid, *environment, *origin+"/api/v1/payments/wechat/notify", *origin+"/api/v1/refunds/wechat/notify")
}

func main() {
	query, err := generate(os.Args[1:])
	if err != nil {
		fmt.Fprintln(os.Stderr, "shared_postgresql_install_generation_failed")
		os.Exit(1)
	}
	if _, err := io.WriteString(os.Stdout, query); err != nil {
		fmt.Fprintln(os.Stderr, "shared_postgresql_install_output_failed")
		os.Exit(1)
	}
}
