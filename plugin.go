package static_file_server

import (
	"bytes"
	"os"
	"path/filepath"

	"github.com/ecletus/router"

	"gopkg.in/yaml.v2"

	"os/exec"

	"github.com/ecletus/ecletus"
	"github.com/ecletus/cli"
	"github.com/ecletus/plug"
	"github.com/jinzhu/configor"
	"github.com/moisespsena-go/httpu"
	"github.com/moisespsena-go/task"
	"github.com/moisespsena-go/error-wrap"
	"github.com/moisespsena-go/topsort"
	"github.com/spf13/cobra"
)

const DEFAULT_ADDR httpu.Addr = ":5001"

type Plugin struct {
	plug.EventDispatcher
	ConfigFile string
	RouterKey  string
	cmd        *exec.Cmd
	cmdArgs    []string
}

func (p *Plugin) RequireOptions() []string {
	return []string{p.RouterKey}
}

func (p *Plugin) loadConfig() (cfg *Config, err error) {
	cfg = &Config{
		CrossOrigins: []string{"*"},
	}

	if err = configor.Load(cfg, p.ConfigFile); err != nil && !os.IsNotExist(err) {
		err = errwrap.Wrap(err, "Load config file %q", p.ConfigFile)
		return
	}

	if len(cfg.Servers) == 0 {
		cfg.Servers = append(cfg.Servers, httpu.ServerConfig{Addr: DEFAULT_ADDR})
	}

	return cfg, nil
}

func (p *Plugin) Init(options *plug.Options) {
	cfg, err := p.loadConfig()
	if err != nil {
		return
	}
	if cfg.AutoStart {
		r := options.GetInterface(p.RouterKey).(*router.Router)
		r.PreServe(func(r *router.Router, ta task.Appender) {
			if err := ta.AddTask(cfg.CreateServer()); err != nil {
				panic(err)
			}
		})
	}
}

func (p *Plugin) OnRegister(options *plug.Options) {
	if p.ConfigFile == "" {
		p.ConfigFile = filepath.Join(ecletus.DEFAULT_CONFIG_DIR, "static_file_server.yaml")
	}

	cli.OnRegister(p, func(e *cli.RegisterEvent) {
		agp := options.GetInterface(ecletus.AGHAPE).(*ecletus.Ecletus)
		cmd := &cobra.Command{
			Use:   "staticFileServe",
			Short: "Start Static File Server",
			RunE: func(cmd *cobra.Command, args []string) error {
				cfg, err := p.loadConfig()
				if err != nil {
					return err
				}
				agp.AddTask(cfg.CreateServer())
				return nil
			},
		}

		var (
			cmdArgs []string
			parent  = cmd
		)

		for parent != nil {
			cmdArgs = append(cmdArgs, parent.Name())
			parent = parent.Parent()
		}

		topsort.Reverse(cmdArgs)
		p.cmdArgs = cmdArgs

		cmd.AddCommand(&cobra.Command{
			Use:   "printConfig",
			Short: "Print Config",
			RunE: func(cmd *cobra.Command, args []string) error {
				config, err := p.loadConfig()
				if err != nil {
					return err
				}
				var w bytes.Buffer
				yaml.NewEncoder(&w).Encode(config)
				os.Stdout.Write(w.Bytes())
				os.Stdout.WriteString("\n")
				return nil
			},
		})
		e.RootCmd.AddCommand(cmd)
	})
}
