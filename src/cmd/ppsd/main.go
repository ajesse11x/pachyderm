package main

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/fsouza/go-dockerclient"

	"go.pedge.io/env"
	"go.pedge.io/proto/server"

	"go.pachyderm.com/pachyderm"
	"go.pachyderm.com/pachyderm/src/pfs"
	"go.pachyderm.com/pachyderm/src/pkg/container"
	"go.pachyderm.com/pachyderm/src/pps"
	"go.pachyderm.com/pachyderm/src/pps/jobserver"
	"go.pachyderm.com/pachyderm/src/pps/persist"
	persistserver "go.pachyderm.com/pachyderm/src/pps/persist/server"
	"go.pachyderm.com/pachyderm/src/pps/pipelineserver"
	"google.golang.org/grpc"
)

var (
	defaultEnv = map[string]string{
		"PPS_ADDRESS":       "0.0.0.0",
		"PPS_PORT":          "651",
		"PPS_TRACE_PORT":    "1051",
		"PPS_DATABASE_NAME": "pachyderm",
	}
)

type appEnv struct {
	PachydermPfsd1Port string `env:"PACHYDERM_PFSD_1_PORT"`
	PfsAddress         string `env:"PFS_ADDRESS"`
	PfsdPort           string `env:"PFSD_PORT_650_TCP"`
	Port               int    `env:"PPS_PORT"`
	DatabaseAddress    string `env:"PPS_DATABASE_ADDRESS"`
	DatabaseName       string `env:"PPS_DATABASE_NAME"`
	DebugPort          int    `env:"PPS_TRACE_PORT"`
}

func main() {
	env.Main(do, &appEnv{}, defaultEnv)
}

func do(appEnvObj interface{}) error {
	appEnv := appEnvObj.(*appEnv)
	containerClient, err := getContainerClient()
	if err != nil {
		return err
	}
	rethinkAPIClient, err := getRethinkAPIClient(appEnv.DatabaseAddress, appEnv.DatabaseName)
	if err != nil {
		return err
	}
	var pfsAddress string
	pfsAddress = appEnv.PachydermPfsd1Port
	if pfsAddress == "" {
		pfsAddress = appEnv.PfsAddress
	} else {
		pfsAddress = strings.Replace(pfsAddress, "tcp://", "", -1)
	}
	if pfsAddress == "" {
		pfsAddress = strings.Replace(appEnv.PfsdPort, "tcp://", "", -1)
	}
	clientConn, err := grpc.Dial(pfsAddress, grpc.WithInsecure())
	if err != nil {
		return err
	}
	jobAPIServer := jobserver.NewAPIServer(rethinkAPIClient, containerClient)
	jobAPIClient := pps.NewLocalJobAPIClient(jobAPIServer)
	pfsAPIClient := pfs.NewAPIClient(clientConn)
	pipelineAPIServer := pipelineserver.NewAPIServer(pfsAPIClient, jobAPIClient, rethinkAPIClient)
	errC := make(chan error)
	go func() {
		errC <- protoserver.Serve(
			uint16(appEnv.Port),
			func(s *grpc.Server) {
				pps.RegisterJobAPIServer(s, jobAPIServer)
				pps.RegisterPipelineAPIServer(s, pipelineAPIServer)
			},
			protoserver.ServeOptions{
				DebugPort: uint16(appEnv.DebugPort),
				Version:   pachyderm.Version,
			},
		)
	}()
	// TODO(pedge): pretty sure I had this problem before, this is bad, we would
	// prefer a callback for when the server is ready to accept requests
	time.Sleep(1 * time.Second)
	if err := pipelineAPIServer.Start(); err != nil {
		return err
	}
	return <-errC
}

func getContainerClient() (container.Client, error) {
	client, err := docker.NewClientFromEnv()
	if err != nil {
		return nil, err
	}
	return container.NewDockerClient(client), nil
}

func getRethinkAPIClient(address string, databaseName string) (persist.APIClient, error) {
	var err error
	if address == "" {
		address, err = getRethinkAddress()
		if err != nil {
			return nil, err
		}
	}
	if err := persistserver.InitDBs(address, databaseName); err != nil {
		return nil, err
	}
	rethinkAPIServer, err := persistserver.NewRethinkAPIServer(address, databaseName)
	if err != nil {
		return nil, err
	}
	return persist.NewLocalAPIClient(rethinkAPIServer), nil
}

func getRethinkAddress() (string, error) {
	rethinkAddr := os.Getenv("RETHINKDB_DRIVER_PORT_28015_TCP_ADDR")
	if rethinkAddr == "" {
		return "", errors.New("RETHINKDB_DRIVER_PORT_28015_TCP_ADDR not set")
	}
	return fmt.Sprintf("%s:28015", rethinkAddr), nil
}
