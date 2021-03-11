package lily

import (
	"context"
	"io/ioutil"

	paramfetch "github.com/filecoin-project/go-paramfetch"
	lotusbuild "github.com/filecoin-project/lotus/build"
	"github.com/filecoin-project/lotus/chain/events"
	lcli "github.com/filecoin-project/lotus/cli"
	"github.com/filecoin-project/lotus/lib/lotuslog"
	"github.com/filecoin-project/lotus/lib/peermgr"
	"github.com/filecoin-project/lotus/node"
	lotusmodules "github.com/filecoin-project/lotus/node/modules"
	"github.com/filecoin-project/lotus/node/modules/dtypes"
	"github.com/filecoin-project/lotus/node/repo"
	logging "github.com/ipfs/go-log/v2"
	"github.com/mitchellh/go-homedir"
	"github.com/multiformats/go-multiaddr"
	"github.com/urfave/cli/v2"
	"golang.org/x/xerrors"

	"github.com/filecoin-project/sentinel-visor/commands/util"
	"github.com/filecoin-project/sentinel-visor/lens/lily"
	"github.com/filecoin-project/sentinel-visor/lens/lily/modules"
)

var log = logging.Logger("lily-cli")

type daemonOpts struct {
	api            string
	repo           string
	bootstrap      bool
	config         string
	importsnapshot string
	genesis        string
}

var daemonFlags daemonOpts

var LilyDaemon = &cli.Command{
	Name:  "daemon",
	Usage: "Start a lily daemon process",
	After: destroy,
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:        "api",
			EnvVars:     []string{"SENTINEL_LILY_API"},
			Value:       "1234",
			Destination: &daemonFlags.api,
		},
		&cli.StringFlag{
			Name:        "repo",
			EnvVars:     []string{"SENTINEL_LILY_REPO"},
			Value:       "~/.lotus",
			Destination: &daemonFlags.repo,
		},
		&cli.BoolFlag{
			Name:        "bootstrap",
			EnvVars:     []string{"SENTINEL_LILY_BOOTSTRAP"},
			Value:       true,
			Destination: &daemonFlags.bootstrap,
		},
		&cli.StringFlag{
			Name:        "config",
			Usage:       "specify path of config file to use",
			EnvVars:     []string{"SENTINEL_LILY_CONFIG"},
			Destination: &daemonFlags.config,
		},
		&cli.StringFlag{
			Name:        "import-snapshot",
			Usage:       "import chain state from a given chain export file or url",
			EnvVars:     []string{"SENTINEL_LILY_SNAPSHOT"},
			Destination: &daemonFlags.importsnapshot,
		},
		&cli.StringFlag{
			Name:        "genesis",
			Usage:       "genesis file to use for first node run",
			EnvVars:     []string{"SENTINEL_LILY_GENESIS"},
			Destination: &daemonFlags.genesis,
		},
	},
	Action: func(c *cli.Context) error {
		lotuslog.SetupLogLevels()
		ctx := context.Background()
		{
			dir, err := homedir.Expand(daemonFlags.repo)
			if err != nil {
				log.Warnw("could not expand repo location", "error", err)
			} else {
				log.Infof("lotus repo: %s", dir)
			}
		}

		r, err := repo.NewFS(daemonFlags.repo)
		if err != nil {
			return xerrors.Errorf("opening fs repo: %w", err)
		}

		if daemonFlags.config != "" {
			r.SetConfigPath(daemonFlags.config)
		}

		err = r.Init(repo.FullNode)
		if err != nil && err != repo.ErrRepoExists {
			return xerrors.Errorf("repo init error: %w", err)
		}

		if err := paramfetch.GetParams(lcli.ReqContext(c), lotusbuild.ParametersJSON(), 0); err != nil {
			return xerrors.Errorf("fetching proof parameters: %w", err)
		}

		var genBytes []byte
		if c.String("genesis") != "" {
			genBytes, err = ioutil.ReadFile(daemonFlags.genesis)
			if err != nil {
				return xerrors.Errorf("reading genesis: %w", err)
			}
		} else {
			genBytes = lotusbuild.MaybeGenesis()
		}

		if daemonFlags.importsnapshot != "" {
			if err := util.ImportChain(ctx, r, daemonFlags.importsnapshot, true); err != nil {
				return err
			}
		}

		genesis := node.Options()
		if len(genBytes) > 0 {
			genesis = node.Override(new(lotusmodules.Genesis), lotusmodules.LoadGenesis(genBytes))
		}

		isBootstrapper := false
		shutdown := make(chan struct{})
		liteModeDeps := node.Options()
		var api lily.LilyAPI
		stop, err := node.New(ctx,
			// Start Sentinel Dep injection
			LilyNodeAPIOption(&api),
			node.Override(new(*events.Events), modules.NewEvents),
			// End Injection

			node.Override(new(dtypes.Bootstrapper), isBootstrapper),
			node.Override(new(dtypes.ShutdownChan), shutdown),
			node.Online(),
			node.Repo(r),

			genesis,
			liteModeDeps,

			node.ApplyIf(func(s *node.Settings) bool { return c.IsSet("api") },
				node.Override(node.SetApiEndpointKey, func(lr repo.LockedRepo) error {
					apima, err := multiaddr.NewMultiaddr("/ip4/127.0.0.1/tcp/" +
						c.String("api"))
					if err != nil {
						return err
					}
					return lr.SetAPIEndpoint(apima)
				})),
			node.ApplyIf(func(s *node.Settings) bool { return !daemonFlags.bootstrap },
				node.Unset(node.RunPeerMgrKey),
				node.Unset(new(*peermgr.PeerMgr)),
			),
		)
		if err != nil {
			return xerrors.Errorf("initializing node: %w", err)
		}

		endpoint, err := r.APIEndpoint()
		if err != nil {
			return xerrors.Errorf("getting api endpoint: %w", err)
		}

		// TODO: properly parse api endpoint (or make it a URL)
		maxAPIRequestSize := int64(0)
		return util.ServeRPC(api, stop, endpoint, shutdown, maxAPIRequestSize)
	},
}
