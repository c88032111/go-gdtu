// Copyright 2017 The go-gdtu Authors
// This file is part of go-gdtu.
//
// go-gdtu is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// go-gdtu is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// algdtu with go-gdtu. If not, see <http://www.gnu.org/licenses/>.

package main

import (
	"bufio"
	"errors"
	"fmt"
	"math/big"
	"os"
	"reflect"
	"unicode"

	"gopkg.in/urfave/cli.v1"

	"github.com/c88032111/go-gdtu/cmd/utils"
	"github.com/c88032111/go-gdtu/gdtu/gdtuconfig"
	"github.com/c88032111/go-gdtu/internal/gdtuapi"
	"github.com/c88032111/go-gdtu/metrics"
	"github.com/c88032111/go-gdtu/node"
	"github.com/c88032111/go-gdtu/params"
	"github.com/naoina/toml"
)

var (
	dumpConfigCommand = cli.Command{
		Action:      utils.MigrateFlags(dumpConfig),
		Name:        "dumpconfig",
		Usage:       "Show configuration values",
		ArgsUsage:   "",
		Flags:       append(nodeFlags, rpcFlags...),
		Category:    "MISCELLANEOUS COMMANDS",
		Description: `The dumpconfig command shows configuration values.`,
	}

	configFileFlag = cli.StringFlag{
		Name:  "config",
		Usage: "TOML configuration file",
	}
)

// These settings ensure that TOML keys use the same names as Go struct fields.
var tomlSettings = toml.Config{
	NormFieldName: func(rt reflect.Type, key string) string {
		return key
	},
	FieldToKey: func(rt reflect.Type, field string) string {
		return field
	},
	MissingField: func(rt reflect.Type, field string) error {
		link := ""
		if unicode.IsUpper(rune(rt.Name()[0])) && rt.PkgPath() != "main" {
			link = fmt.Sprintf(", see https://godoc.org/%s#%s for available fields", rt.PkgPath(), rt.Name())
		}
		return fmt.Errorf("field '%s' is not defined in %s%s", field, rt.String(), link)
	},
}

type gdtustatsConfig struct {
	URL string `toml:",omitempty"`
}

type ggdtuConfig struct {
	Gdtu      gdtuconfig.Config
	Node      node.Config
	Gdtustats gdtustatsConfig
	Metrics   metrics.Config
}

func loadConfig(file string, cfg *ggdtuConfig) error {
	f, err := os.Open(file)
	if err != nil {
		return err
	}
	defer f.Close()

	err = tomlSettings.NewDecoder(bufio.NewReader(f)).Decode(cfg)
	// Add file name to errors that have a line number.
	if _, ok := err.(*toml.LineError); ok {
		err = errors.New(file + ", " + err.Error())
	}
	return err
}

func defaultNodeConfig() node.Config {
	cfg := node.DefaultConfig
	cfg.Name = clientIdentifier
	cfg.Version = params.VersionWithCommit(gitCommit, gitDate)
	cfg.HTTPModules = append(cfg.HTTPModules, "gdtu")
	cfg.WSModules = append(cfg.WSModules, "gdtu")
	cfg.IPCPath = "ggdtu.ipc"
	return cfg
}

// makeConfigNode loads ggdtu configuration and creates a blank node instance.
func makeConfigNode(ctx *cli.Context) (*node.Node, ggdtuConfig) {
	// Load defaults.
	cfg := ggdtuConfig{
		Gdtu:    gdtuconfig.Defaults,
		Node:    defaultNodeConfig(),
		Metrics: metrics.DefaultConfig,
	}

	// Load config file.
	if file := ctx.GlobalString(configFileFlag.Name); file != "" {
		if err := loadConfig(file, &cfg); err != nil {
			utils.Fatalf("%v", err)
		}
	}

	// Apply flags.
	utils.SetNodeConfig(ctx, &cfg.Node)
	stack, err := node.New(&cfg.Node)
	if err != nil {
		utils.Fatalf("Failed to create the protocol stack: %v", err)
	}
	utils.SetGdtuConfig(ctx, stack, &cfg.Gdtu)
	if ctx.GlobalIsSet(utils.GdtustatsURLFlag.Name) {
		cfg.Gdtustats.URL = ctx.GlobalString(utils.GdtustatsURLFlag.Name)
	}
	applyMetricConfig(ctx, &cfg)

	return stack, cfg
}

// makeFullNode loads ggdtu configuration and creates the Gdtu backend.
func makeFullNode(ctx *cli.Context) (*node.Node, gdtuapi.Backend) {
	stack, cfg := makeConfigNode(ctx)
	if ctx.GlobalIsSet(utils.OverrideBerlinFlag.Name) {
		cfg.Gdtu.OverrideBerlin = new(big.Int).SetUint64(ctx.GlobalUint64(utils.OverrideBerlinFlag.Name))
	}
	backend := utils.RegisterGdtuService(stack, &cfg.Gdtu)

	// Configure GraphQL if requested
	if ctx.GlobalIsSet(utils.GraphQLEnabledFlag.Name) {
		utils.RegisterGraphQLService(stack, backend, cfg.Node)
	}
	// Add the Gdtu Stats daemon if requested.
	if cfg.Gdtustats.URL != "" {
		utils.RegisterGdtustatsService(stack, backend, cfg.Gdtustats.URL)
	}
	return stack, backend
}

// dumpConfig is the dumpconfig command.
func dumpConfig(ctx *cli.Context) error {
	_, cfg := makeConfigNode(ctx)
	comment := ""

	if cfg.Gdtu.Genesis != nil {
		cfg.Gdtu.Genesis = nil
		comment += "# Note: this config doesn't contain the genesis block.\n\n"
	}

	out, err := tomlSettings.Marshal(&cfg)
	if err != nil {
		return err
	}

	dump := os.Stdout
	if ctx.NArg() > 0 {
		dump, err = os.OpenFile(ctx.Args().Get(0), os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644)
		if err != nil {
			return err
		}
		defer dump.Close()
	}
	dump.WriteString(comment)
	dump.Write(out)

	return nil
}

func applyMetricConfig(ctx *cli.Context, cfg *ggdtuConfig) {
	if ctx.GlobalIsSet(utils.MetricsEnabledFlag.Name) {
		cfg.Metrics.Enabled = ctx.GlobalBool(utils.MetricsEnabledFlag.Name)
	}
	if ctx.GlobalIsSet(utils.MetricsEnabledExpensiveFlag.Name) {
		cfg.Metrics.EnabledExpensive = ctx.GlobalBool(utils.MetricsEnabledExpensiveFlag.Name)
	}
	if ctx.GlobalIsSet(utils.MetricsHTTPFlag.Name) {
		cfg.Metrics.HTTP = ctx.GlobalString(utils.MetricsHTTPFlag.Name)
	}
	if ctx.GlobalIsSet(utils.MetricsPortFlag.Name) {
		cfg.Metrics.Port = ctx.GlobalInt(utils.MetricsPortFlag.Name)
	}
	if ctx.GlobalIsSet(utils.MetricsEnableInfluxDBFlag.Name) {
		cfg.Metrics.EnableInfluxDB = ctx.GlobalBool(utils.MetricsEnableInfluxDBFlag.Name)
	}
	if ctx.GlobalIsSet(utils.MetricsInfluxDBEndpointFlag.Name) {
		cfg.Metrics.InfluxDBEndpoint = ctx.GlobalString(utils.MetricsInfluxDBEndpointFlag.Name)
	}
	if ctx.GlobalIsSet(utils.MetricsInfluxDBDatabaseFlag.Name) {
		cfg.Metrics.InfluxDBDatabase = ctx.GlobalString(utils.MetricsInfluxDBDatabaseFlag.Name)
	}
	if ctx.GlobalIsSet(utils.MetricsInfluxDBUsernameFlag.Name) {
		cfg.Metrics.InfluxDBUsername = ctx.GlobalString(utils.MetricsInfluxDBUsernameFlag.Name)
	}
	if ctx.GlobalIsSet(utils.MetricsInfluxDBPasswordFlag.Name) {
		cfg.Metrics.InfluxDBPassword = ctx.GlobalString(utils.MetricsInfluxDBPasswordFlag.Name)
	}
	if ctx.GlobalIsSet(utils.MetricsInfluxDBTagsFlag.Name) {
		cfg.Metrics.InfluxDBTags = ctx.GlobalString(utils.MetricsInfluxDBTagsFlag.Name)
	}
}
