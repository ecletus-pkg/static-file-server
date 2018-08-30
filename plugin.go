package static_file_server

import (
	"bytes"
	"os"
	"path/filepath"

	"github.com/aghape/router"

	"github.com/moisespsena/go-default-logger"
	"github.com/moisespsena/go-path-helpers"
	"gopkg.in/yaml.v2"

	"os/exec"
	"syscall"

	"github.com/aghape/aghape"
	"github.com/aghape/cli"
	"github.com/aghape/plug"
	"github.com/jinzhu/configor"
	"github.com/moisespsena/go-error-wrap"
	"github.com/moisespsena/go-topsort"
	"github.com/spf13/cobra"
)

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

func (p *Plugin) loadConfig() (config Config, err error) {
	config = Config{
		Addr:         ":8000",
		CrossOrigins: []string{"*"},
	}

	if err = configor.Load(&config, p.ConfigFile); err != nil && !os.IsNotExist(err) {
		err = errwrap.Wrap(err, "Load config file %q", p.ConfigFile)
		return
	}
	return config, nil
}

func (p *Plugin) Init(options *plug.Options) {
	config, err := p.loadConfig()
	if err != nil {
		return
	}
	if config.AutoStart {
		r := options.GetInterface(p.RouterKey).(*router.Router)
		r.PreServe(func(r *router.Router) {
			p.cmd = exec.Command(os.Args[0], p.cmdArgs...)
			p.cmd.Stdout, p.cmd.Stderr = os.Stdout, os.Stderr
			p.cmd.SysProcAttr = &syscall.SysProcAttr{
				Pdeathsig: syscall.SIGTERM,
			}
			if err := p.cmd.Start(); err != nil {
				panic(errwrap.Wrap(err, "Start Start Static File Server"))
			}
		})
	}
}

func (p *Plugin) listenAndServer() error {
	config, err := p.loadConfig()
	if err != nil {
		return err
	}
	log := defaultlogger.NewLogger(path_helpers.GetCalledDir())

	var w bytes.Buffer
	yaml.NewEncoder(&w).Encode(config)

	log.Debug("Start Static File Server with config:\n" + w.String())
	NewServer(config).LisenAndServer()
	return nil
}

func (p *Plugin) OnRegister() {
	if p.ConfigFile == "" {
		p.ConfigFile = filepath.Join(aghape.DEFAULT_CONFIG_DIR, "static_file_server.yaml")
	}

	cli.OnRegister(p, func(e *cli.RegisterEvent) {
		cmd := &cobra.Command{
			Use:   "staticFileServe",
			Short: "Start Static File Server",
			RunE: func(cmd *cobra.Command, args []string) error {
				return p.listenAndServer()
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
