/*
Package cmd contains the command line interface for the nexthos service

Copyright © 2024 Shono <code@shono.io>
*/
package cmd

import (
	"context"
	"fmt"
	"github.com/rs/zerolog/log"
	"github.com/shono-io/mini"
	"github.com/shono-io/nexthos/pkg"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var cfgFile string

var rootCmd = &cobra.Command{
	Use:   "nexthos",
	Short: "a service for running pipelines",
	Long:  `This service is responsible for running pipelines on the shono platform.`,
	Run: func(cmd *cobra.Command, args []string) {
		svc, err := mini.FromViper(viper.GetViper(),
			mini.WithDescription("This service is responsible for running pipelines on the shono platform"),
			mini.WithVersion("1.0.0"),
			mini.WithConfigWatched(),
		)
		if err != nil {
			log.Panic().Err(err).Msg("failed to create service")
		}

		w := pkg.NewWorker()
		if err := svc.Run(context.Background(), w); err != nil {
			svc.Log.Panic().Err(err).Msg("service error")
		}
	},
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.nexthos.yaml)")
	mini.ConfigureCommand(rootCmd)

	if err := viper.BindPFlags(rootCmd.PersistentFlags()); err != nil {
		log.Panic().Err(err).Msg("failed to bind flags")
	}
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory.
		home, err := os.UserHomeDir()
		cobra.CheckErr(err)

		// Search config in home directory with name ".nexthos" (without extension).
		viper.AddConfigPath(home)
		viper.SetConfigType("yaml")
		viper.SetConfigName(".nexthos")
	}

	viper.SetEnvPrefix("NEXTHOS")
	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		fmt.Fprintln(os.Stderr, "Using config file:", viper.ConfigFileUsed())
	}
}
