package provisioner

import (
	"github.com/zasper-io/zasper/kernel/launcher"
	"github.com/zasper-io/zasper/kernelspec"

	"github.com/rs/zerolog/log"
)

type KernelConnectionInfo map[string]interface{}

type LocalProvisioner struct {
	Kernelspec     kernelspec.KernelSpecJsonData
	KernelId       string
	ConnectionInfo KernelConnectionInfo
	Process        string
	Exit_future    string
	Pid            int
	Pgid           int
	IP             string
	PortsCached    bool
}

func (provisioner *LocalProvisioner) LaunchKernel(kernelCmd []string, kw map[string]interface{}, connFile string) KernelConnectionInfo {
	process := launcher.LaunchKernel(kernelCmd, kw, connFile)
	provisioner.Pid = process.Pid
	log.Debug().Msgf("kernel launched with pid: %d", process.Pid)
	return provisioner.ConnectionInfo
}
