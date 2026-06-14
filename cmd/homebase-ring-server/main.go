package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"

	ring "github.com/localitas/localitas-app-homebase-ring"
	"github.com/localitas/localitas-go"
	"github.com/urfave/cli/v3"
)

var (
	version = "dev"
	commit  = "unknown"
)

func envOrFileToken() string {
	if t := os.Getenv("LOCALITAS_API_TOKEN"); t != "" {
		return t
	}
	return client.DefaultToken()
}

func main() {
	app := &cli.Command{
		Name:    "homebase-ring-server",
		Usage:   "Ring device integration plugin for Homebase",
		Version: version,
		Commands: []*cli.Command{
			serveCommand(),
			authCommand(),
		},
		DefaultCommand: "serve",
		Action: func(ctx context.Context, cmd *cli.Command) error {
			return serveAction(ctx, cmd)
		},
	}

	if err := app.Run(context.Background(), os.Args); err != nil {
		log.Fatal(err)
	}
}

func serveCommand() *cli.Command {
	return &cli.Command{
		Name:  "serve",
		Usage: "Start the Ring plugin server",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "listen", Value: ":0", Usage: "listen address"},
			&cli.StringFlag{Name: "core-url", Value: client.DefaultCoreURL(), Usage: "base URL of the Localitas core API"},
			&cli.StringFlag{Name: "base-path", Value: "/", Usage: "URL prefix"},
			&cli.StringFlag{Name: "token", Value: envOrFileToken(), Usage: "bearer token"},
			&cli.StringFlag{Name: "hardware-id", Sources: cli.EnvVars("RING_HARDWARE_ID"), Usage: "Ring hardware ID (optional)"},
		},
		Action: serveAction,
	}
}

func serveAction(ctx context.Context, cmd *cli.Command) error {
	listen := cmd.String("listen")
	coreURL := cmd.String("core-url")
	basePath := cmd.String("base-path")
	token := cmd.String("token")
	hardwareID := cmd.String("hardware-id")

	c := client.New(coreURL)
	if token != "" {
		c = c.WithToken(token)
	}

	app := ring.New(c, basePath, hardwareID)

	log.Printf("homebase-ring started (unconfigured, waiting for POST /api/configure from Homebase)")

	mux := http.NewServeMux()
	app.RegisterRoutes(mux)

	ln, err := net.Listen("tcp", listen)
	if err != nil {
		return fmt.Errorf("listen: %w", err)
	}
	addr := ln.Addr().(*net.TCPAddr)
	fmt.Printf("homebase-ring-server listening on http://localhost:%d\n", addr.Port)

	selfURL := fmt.Sprintf("http://localhost:%d", addr.Port)
	if err := c.RegisterService(ctx, "homebase-ring", selfURL); err != nil {
		log.Printf("service registry failed: %v", err)
	}

	shutdown, err := ring.BroadcastMDNS(addr.Port, ring.DefaultHealth.Name)
	if err != nil {
		log.Printf("mDNS broadcast failed: %v", err)
	}

	go func() {
		sig := make(chan os.Signal, 1)
		signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
		<-sig
		log.Println("shutting down...")
		if shutdown != nil {
			shutdown()
		}
		os.Exit(0)
	}()

	return http.Serve(ln, mux)
}

func authCommand() *cli.Command {
	return &cli.Command{
		Name:  "auth",
		Usage: "Acquire a Ring refresh token via email/password login",
		Action: func(ctx context.Context, cmd *cli.Command) error {
			reader := bufio.NewReader(os.Stdin)

			fmt.Println("Ring Authentication")
			fmt.Println("This will generate a refresh token for use with Homebase.")
			fmt.Println()

			fmt.Print("Email: ")
			email, _ := reader.ReadString('\n')
			email = strings.TrimSpace(email)

			fmt.Print("Password: ")
			password, _ := reader.ReadString('\n')
			password = strings.TrimSpace(password)

			token, prompt, err := ring.AcquireRefreshToken(email, password, "")
			if err != nil {
				return fmt.Errorf("authentication failed: %w", err)
			}

			for token == "" && prompt != "" {
				fmt.Println()
				fmt.Println(prompt)
				fmt.Print("2FA Code: ")
				code, _ := reader.ReadString('\n')
				code = strings.TrimSpace(code)

				token, prompt, err = ring.AcquireRefreshToken(email, password, code)
				if err != nil {
					fmt.Printf("Error: %v\n", err)
					fmt.Println("Please try again.")
					prompt = "Enter 2FA code"
					continue
				}
			}

			fmt.Println()
			fmt.Println("Successfully authenticated with Ring!")
			fmt.Println()
			fmt.Println("Refresh token:")
			fmt.Println(token)
			fmt.Println()
			fmt.Println("Store this in Vault with key 'ring_refresh_token':")
			fmt.Println()
			fmt.Printf("  curl -X POST http://localhost:8080/apps/vault/api/credentials \\\n")
			fmt.Printf("    -H \"Authorization: Bearer $(cat ~/.localitas/api-token)\" \\\n")
			fmt.Printf("    -H \"Content-Type: application/json\" \\\n")
			fmt.Printf("    -d '{\"name\": \"Ring\", \"data\": {\"ring_refresh_token\": \"%s\"}}'\n", token)
			fmt.Println()
			fmt.Println("Then configure the Vault public_id in Homebase UI under Plugins.")

			return nil
		},
	}
}
