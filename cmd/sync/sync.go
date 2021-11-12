package main

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"cloud.google.com/go/storage"
	"github.com/spf13/cobra"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"

	"github.com/spacemeshos/go-spacemesh/activation"
	"github.com/spacemeshos/go-spacemesh/blocks"
	cmdp "github.com/spacemeshos/go-spacemesh/cmd"
	"github.com/spacemeshos/go-spacemesh/common/types"
	"github.com/spacemeshos/go-spacemesh/database"
	"github.com/spacemeshos/go-spacemesh/fetch"
	"github.com/spacemeshos/go-spacemesh/filesystem"
	"github.com/spacemeshos/go-spacemesh/layerfetcher"
	"github.com/spacemeshos/go-spacemesh/log"
	"github.com/spacemeshos/go-spacemesh/mempool"
	"github.com/spacemeshos/go-spacemesh/mesh"
	"github.com/spacemeshos/go-spacemesh/p2p"
	"github.com/spacemeshos/go-spacemesh/syncer"
	"github.com/spacemeshos/go-spacemesh/system"
)

// Sync cmd.
var cmd = &cobra.Command{
	Use:   "sync",
	Short: "start sync",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("starting sync")
		syncApp := newSyncApp()
		log.With().Info("initializing new sync app", log.String("DataDir", syncApp.Config.DataDir()))
		defer syncApp.Cleanup()
		syncApp.Initialize(cmd)
		syncApp.start(cmd, args)
	},
}

// ////////////////////////////

var (
	expectedLayers uint32
	bucket         string
	version        string
	remote         bool
)

func init() {
	// path to remote storage
	cmd.PersistentFlags().StringVarP(&bucket, "storage-path", "z", "spacemesh-sync-data", "Specify storage bucket name")

	// expected layers
	cmd.PersistentFlags().Uint32Var(&expectedLayers, "expected-layers", 101, "expected number of layers")

	// fetch from remote
	cmd.PersistentFlags().BoolVar(&remote, "remote-data", false, "fetch from remote")

	// request timeout
	cmd.PersistentFlags().StringVarP(&version, "version", "v", "samples/", "data version")

	cmdp.AddCommands(cmd)
}

type syncApp struct {
	*cmdp.BaseApp
	sync   *syncer.Syncer
	msh    *mesh.Mesh
	logger log.Log
}

func newSyncApp() *syncApp {
	return &syncApp{BaseApp: cmdp.NewBaseApp()}
}

func (app *syncApp) Cleanup() {
	err := os.RemoveAll(app.Config.DataDir())
	if err != nil {
		app.logger.With().Error("failed to cleanup sync", log.Err(err))
	}
}

func (app *syncApp) start(_ *cobra.Command, _ []string) {
	// start p2p services
	lg := log.NewDefault("sync_test")
	lg.With().Info("------------ Start sync test -----------",
		log.String("data_folder", app.Config.DataDir()),
		log.String("storage_path", bucket),
		log.Bool("download_from_remote_storage", remote),
		log.Uint32("expected_layers", expectedLayers),
		log.Int("request_timeout", app.Config.SyncRequestTimeout),
		log.String("data_version", version),
		log.Uint32("layers_per_epoch", app.Config.LayersPerEpoch),
		log.Uint32("hdist", app.Config.Hdist),
	)

	path := app.Config.DataDir()
	cfg := app.Config.P2P
	cfg.DataDir = path
	host, err := p2p.New(cmdp.Ctx(), lg.WithName("p2p"), cfg)
	if err != nil {
		lg.With().Panic("failed to create p2p host", log.Err(err))
	}

	goldenATXID := types.ATXID(types.HexToHash32(app.Config.GoldenATXID))
	types.SetLayersPerEpoch(app.Config.LayersPerEpoch)
	lg.Info("local db path: %v layers per epoch: %v", path, app.Config.LayersPerEpoch)

	if remote {
		if err := getData(path, version, lg); err != nil {
			lg.With().Error("could not download data for test", log.Err(err))
			return
		}
	}
	poetDbStore, err := database.NewLDBDatabase(filepath.Join(path, "poet"), 0, 0, lg.WithName("poetDbStore"))
	if err != nil {
		lg.With().Error("error creating poet database", log.Err(err))
		return
	}

	poetDb := activation.NewPoetDb(poetDbStore, lg.WithName("poetDb").WithOptions(log.Nop))

	mshdb, err := mesh.NewPersistentMeshDB(filepath.Join(path, "mesh"), 5, lg.WithOptions(log.Nop))
	if err != nil {
		lg.With().Error("error creating mesh database", log.Err(err))
		return
	}

	atxdbStore, err := database.NewLDBDatabase(filepath.Join(path, "atx"), 0, 0, lg)
	if err != nil {
		lg.With().Error("error creating atx database", log.Err(err))
		return
	}

	txpool := mempool.NewTxMemPool()

	app.logger = log.NewDefault("sync_test")
	app.logger.Info("new sync tester")

	layersPerEpoch := app.Config.LayersPerEpoch

	fw := &fetcherWrapper{}

	atxdb := activation.NewDB(atxdbStore, fw, &mockIStore{}, mshdb, layersPerEpoch, goldenATXID, &validatorMock{}, lg.WithOptions(log.Nop))

	dbs := &allDbs{
		atxdb:       atxdb,
		atxdbStore:  atxdbStore,
		poetDb:      poetDb,
		poetStorage: poetDbStore,
		mshdb:       mshdb,
	}

	msh := createMeshWithMock(dbs, txpool, app.logger)
	app.msh = msh

	blockHandler := blocks.NewBlockHandler(blocks.Config{Depth: 10}, fw, msh, blockEligibilityValidatorMock{}, lg)

	fCfg := fetch.DefaultConfig()
	fetcher := fetch.NewFetch(context.TODO(), fCfg, host, lg)

	lCfg := layerfetcher.Config{RequestTimeout: 20}
	layerFetch := layerfetcher.NewLogic(context.TODO(), lCfg, blockHandler, dbs.atxdb, dbs.poetDb, dbs.atxdb, mockTxProcessor{}, host, fetcher, msh, lg)
	layerFetch.AddDBs(dbs.mshdb.Blocks(), dbs.atxdbStore, dbs.mshdb.Transactions(), dbs.poetStorage)
	layerFetch.Start()
	fw.Fetcher = layerFetch

	syncerConf := syncer.Configuration{
		SyncInterval: 2 * 60 * time.Millisecond,
	}
	app.sync = createSyncer(syncerConf, msh, layerFetch, types.NewLayerID(expectedLayers), app.logger)

	i := layersPerEpoch * 2
	for ; ; i++ {
		lid := types.NewLayerID(i)
		lg.With().Info("getting layer", lid)
		if lyr, err2 := msh.GetLayer(lid); err2 != nil || lyr == nil {
			if lid.After(types.GetEffectiveGenesis()) {
				lg.With().Info("finished loading layers from disk",
					log.FieldNamed("layers_loaded", lid.Sub(1)),
					log.Err(err2),
				)
				break
			}
		} else {
			lg.With().Info("loaded layer from disk", types.NewLayerID(i))

			msh.ValidateLayer(cmdp.Ctx(), lyr)
		}
	}

	sleep := time.Duration(10) * time.Second
	lg.Info("wait %v sec", sleep)
	time.Sleep(sleep)

	go app.sync.Start(cmdp.Ctx())

	for msh.ProcessedLayer().Before(types.NewLayerID(expectedLayers)) {
		lg.Info("sleep for %v sec", 30)
		app.sync.ForceSync(context.TODO())
		time.Sleep(30 * time.Second)
	}

	lg.Event().Info("sync done",
		log.FieldNamed("processed", msh.ProcessedLayer()),
	)
	app.sync.Close()
	for {
		lg.Info("keep busy sleep for %v sec", 60)
		time.Sleep(60 * time.Second)
	}
}

// GetData downloads data from remote storage.
func getData(path, prefix string, lg log.Log) error {
	c := http.Client{
		Transport: &http.Transport{
			MaxIdleConnsPerHost: 10,
			TLSClientConfig: &tls.Config{
				MinVersion:         tls.VersionTLS11,
				InsecureSkipVerify: true,
			},
		},
		Timeout: 2 * time.Second,
	}

	ctx := context.TODO()
	client, err := storage.NewClient(ctx, option.WithoutAuthentication(), option.WithHTTPClient(&c))
	if err != nil {
		panic(err)
	}
	it := client.Bucket(bucket).Objects(ctx, &storage.Query{
		Prefix: prefix,
	})

	count := 0
	for {
		attrs, err := it.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return fmt.Errorf("iterator: %w", err)
		}

		rc, err := client.Bucket(bucket).Object(attrs.Name).NewReader(ctx)
		if err != nil {
			return fmt.Errorf("create reader: %w", err)
		}

		data, err := ioutil.ReadAll(rc)
		_ = rc.Close()
		if err != nil {
			return fmt.Errorf("read all: %w", err)
		}

		// skip main folder
		if attrs.Name == version {
			continue
		}
		dest := path + strings.TrimPrefix(attrs.Name, version)
		if err := ensureDirExists(dest); err != nil {
			return fmt.Errorf("ensure dir exists: %w", err)
		}
		lg.Info("downloading: %v to %v", attrs.Name, dest)

		if err = ioutil.WriteFile(dest, data, 0o644); err != nil {
			lg.Error("%v", err)
			return fmt.Errorf("write file: %w", err)
		}
		count++
	}

	lg.Info("done downloading: %v files", count)
	return nil
}

type fetcherWrapper struct {
	system.Fetcher
}

func ensureDirExists(path string) error {
	dir, _ := filepath.Split(path)
	if err := filesystem.ExistOrCreate(dir); err != nil {
		return fmt.Errorf("create dir: %w", err)
	}

	return nil
}

func main() {
	if err := cmd.Execute(); err != nil {
		log.With().Info("error", log.Err(err))
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
