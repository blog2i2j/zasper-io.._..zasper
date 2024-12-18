package kernel

import (
	"fmt"
	"os"
	"slices"

	"github.com/zasper-io/zasper/kernel/provisioner"
	"github.com/zasper-io/zasper/kernelspec"

	"github.com/rs/zerolog/log"

	"github.com/pebbe/zmq4"
)

type KernelManager struct {
	ConnectionFile string
	OwnsKernel     bool
	ShutdownStatus bool
	AttemptedStart bool
	Ready          bool
	KernelName     string
	ControlSocket  *zmq4.Socket
	CachePorts     bool
	Provisioner    provisioner.LocalProvisioner
	Kernelspec     string

	LastActivity   string
	ExecutionState string
	Connections    int

	KernelId     string
	ShuttingDown bool

	Session        KernelSession
	ConnectionInfo Connection
}

/*********************************************************************
**********************************************************************
***                       START KERNEL                            ***
**********************************************************************
*********************************************************************/

func (km *KernelManager) StartKernel(kernelName string) {

	log.Info().Msg("starting kernel")

	km.AttemptedStart = true

	kernelCmd, kw := km.asyncPrestartKernel(kernelName)

	km.asyncLaunchKernel(kernelCmd, kw)
	km.asyncPostStartKernel(kw)
}

func (km *KernelManager) asyncPostStartKernel(kw map[string]interface{}) {
	km.ControlSocket = km.connectControlSocket()
	km.Ready = true
}

func (km *KernelManager) getKernelspec() kernelspec.KernelSpecJsonData {
	return kernelspec.GetKernelSpec(km.KernelName)
}

func (km *KernelManager) asyncPrestartKernel(kernelName string) ([]string, map[string]interface{}) {
	km.ShuttingDown = false

	km.Provisioner = provisioner.LocalProvisioner{
		KernelId:    km.KernelId,
		Kernelspec:  km.getKernelspec(),
		PortsCached: false,
	}

	kw := km.preLaunch()
	kernelCmd := kw["cmd"].([]string)
	log.Debug().Msgf("kenelName: %s", kernelName)
	return kernelCmd, kw
}

var LOCAL_IPS []string

func isLocalIP(ip string) bool {
	//does `ip` point to this machine?
	return slices.Contains(LOCAL_IPS, ip)
}

/*********************************************************************
**********************************************************************
***                       LAUNCH KERNEL                            ***
**********************************************************************
*********************************************************************/

func (km *KernelManager) asyncLaunchKernel(kernelCmd []string, kw map[string]interface{}) {
	ConnectionInfo := km.Provisioner.LaunchKernel(kernelCmd, kw, km.ConnectionFile)
	log.Debug().Msgf("connectionInfo: %s", ConnectionInfo)
}

func (km *KernelManager) preLaunch() map[string]interface{} {

	if km.ConnectionInfo.Transport == "tcp" && !isLocalIP(km.ConnectionInfo.IP) {
		log.Debug().Msg("Can only launch a kernel on a local interface.")
	}
	log.Debug().Msgf("cache ports: %t", km.CachePorts)
	log.Debug().Msgf("km.Provisioner.PortsCached %t", km.Provisioner.PortsCached)

	if km.CachePorts && !km.Provisioner.PortsCached {
		km.ConnectionInfo.ShellPort, _ = findAvailablePort()
		km.ConnectionInfo.IopubPort, _ = findAvailablePort()
		km.ConnectionInfo.StdinPort, _ = findAvailablePort()
		km.ConnectionInfo.HbPort, _ = findAvailablePort()
		km.ConnectionInfo.ControlPort, _ = findAvailablePort()
		log.Debug().Msgf("connectionInfo : %+v", km.ConnectionInfo)
	}
	log.Debug().Msgf("km.ConnectionFile : %+v", km.ConnectionFile)

	km.writeConnectionFile(km.ConnectionFile)

	kernelCmd := km.formatKernelCmd()
	log.Debug().Msgf("kernel cmd is %s", kernelCmd)

	env := make(map[string]interface{})
	env["cmd"] = kernelCmd
	env["env"] = os.Environ()
	return env
}

func (km *KernelManager) formatKernelCmd() []string {
	cmd := km.getKernelspec().Argv
	return cmd
}

/*********************************************************************
**********************************************************************
***                     CONNECT TO SOCKETS                         ***
**********************************************************************
*********************************************************************/

var ChannelSocketTypes map[string]zmq4.Type

func (km *KernelManager) makeURL(channel string) string {
	ip := km.ConnectionInfo.IP
	port := km.ConnectionInfo.ControlPort // TODO getPort

	if km.ConnectionInfo.Transport == "tcp" {
		return fmt.Sprintf("tcp://%s:%d", ip, port)
	}
	return fmt.Sprintf("%s://%s-%d", km.ConnectionInfo.Transport, ip, port)
}

func SetUpChannelSocketTypes() map[string]zmq4.Type {
	cst := make(map[string]zmq4.Type)
	cst["hb"] = zmq4.REQ
	cst["iopub"] = zmq4.SUB
	cst["shell"] = zmq4.DEALER
	cst["stdin"] = zmq4.DEALER
	cst["control"] = zmq4.DEALER

	return cst
}

func (km *KernelManager) connectControlSocket() *zmq4.Socket {
	channel := "control"
	url := km.makeURL(channel)

	socket, _ := zmq4.NewSocket(zmq4.DEALER)

	socket.Connect(url)
	return socket

}

func (km *KernelManager) connectHbSocket() *zmq4.Socket {
	channel := "control"
	url := km.makeURL(channel)

	socket, _ := zmq4.NewSocket(zmq4.REQ)
	socket.Connect(url)
	return socket

}
