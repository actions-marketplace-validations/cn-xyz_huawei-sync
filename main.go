package main

import (
	"github.com/spf13/cobra"
	"os"
)

var (
	url         string
	hubLoginUrl string
	namespace   string
)

func init() {
	rootCmd.Flags().StringVar(&url, "url", getEnvStr("REMOTE_URL", "http://127.0.0.1/k8s.txt"), "Remote file URL address.")
	rootCmd.Flags().StringVar(&hubLoginUrl, "login", getEnvStr("HUB_LOGIN_URL", "docker.io"), "HUB URL address.")
	rootCmd.Flags().StringVar(&namespace, "namespace", getEnvStr("NAMESPACE", "namespace"), "DestHub Password.")
}

var rootCmd = &cobra.Command{
	Use:   "docker-sync is a synchronization tool created to solve the problem that domestic mirror repositories cannot be used.",
	Short: `Only syncing images to docker.io is currently supported`,

	Run: func(cmd *cobra.Command, args []string) {
		s := NewDockerSync(url)
		s.GetRemoteCtx()
		s.CopyImage(hubLoginUrl + "/" + namespace + "/")
	},
}

func getEnvStr(env string, defaultValue string) string {
	v := os.Getenv(env)
	if v == "" {
		return defaultValue
	}
	return v
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		rootCmd.PrintErrf("docker-sync root cmd execute: %s", err)
		os.Exit(1)
	}
}
