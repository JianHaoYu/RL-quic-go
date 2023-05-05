#!/usr/bin/python

from mininet.topo import Topo
from mininet.net import Mininet
from mininet.node import CPULimitedHost
from mininet.link import TCLink
from mininet.util import dumpNodeConnections
from mininet.log import setLogLevel, info
from mininet.cli import CLI
from sys import argv
import time

class MyTopo( Topo ):
    "Simple topology example."
    def build( self,n=1):
        "Create custom topo."
        Host1 = self.addHost( 'h1')
        Host2 = self.addHost( 'h2')
        # Add links
        # bottleneck link
        # 20Mbps, 100ms delay, 1% packet loss
        self.addLink( Host1, Host2, bw=25, delay='300ms', loss=0)


def perfTest(num=2):
    "Create network and run simple performance test"
    #topo = SingleSwitchTopo( n=4, lossy=lossy )
    topo = MyTopo(n=num)
    net = Mininet( topo=topo,
                   link=TCLink,
                   autoStaticArp=True )
    
    nodes = net.switches 

    net.start()
    print ("Testing network connectivity")
    info( "Dumping host connections\n" )
    dumpNodeConnections(net.hosts)
    dumpNodeConnections(net.switches)
    # h = net.get('h1')
    # h.cmd('cd /home/mininet/go/src/github.com/lucas-clemente/quic-go-scunder/example/echo/server/; export LD_LIBRARY_PATH=$LD_LIBRARY_PATH:./; ifconfig h1-eth0 mtu 1800; go run server.go -ip "10.0.0.1:6868" >s_SC_1.log &')
    # h = net.get('h1')
    # h.cmd('cd /home/mininet/go/src/github.com/lucas-clemente/quic-go-scunder/example/echo/server/; export LD_LIBRARY_PATH=$LD_LIBRARY_PATH:./; ifconfig h1-eth0 mtu 1800; go run server.go -ip "10.0.0.1:6869" >s_SC_2.log &')
    # h = net.get('h1')
    # h.cmd('cd /home/mininet/go/src/github.com/lucas-clemente/quic-go-scunder/example/echo/server/; export LD_LIBRARY_PATH=$LD_LIBRARY_PATH:./; ifconfig h1-eth0 mtu 1800; go run server.go -ip "10.0.0.1:6871" >s_SC_3.log &')
    # h = net.get('h2')
    # h.cmd('cd /home/mininet/go/src/github.com/lucas-clemente/quic-go-scunder/example/echo/client/; export LD_LIBRARY_PATH=$LD_LIBRARY_PATH:./; ifconfig h2-eth0 mtu 1800; go run client.go -ip "10.0.0.1:6868" >c_SC_1.log &')
    # h = net.get('h2')
    # h.cmd('cd /home/mininet/go/src/github.com/lucas-clemente/quic-go-scunder/example/echo/client/; export LD_LIBRARY_PATH=$LD_LIBRARY_PATH:./; ifconfig h2-eth0 mtu 1800; go run client.go -ip "10.0.0.1:6869" >c_SC_2.log &')
    # h = net.get('h2')
    # h.cmd('cd /home/mininet/go/src/github.com/lucas-clemente/quic-go-scunder/example/echo/client/; export LD_LIBRARY_PATH=$LD_LIBRARY_PATH:./; ifconfig h2-eth0 mtu 1800; go run client.go -ip "10.0.0.1:6871" >c_SC_3.log &')


    # h = net.get('h1')
    # h.cmd('cd /home/mininet/go/src/github.com/lucas-clemente/quic-go-scunder/example/echo/test_for_TCP_2022_9_24/server/; ifconfig h1-eth0 mtu 1800; go run server.go -ip=10.0.0.1:6870 >s_TCP.log &')
    # h = net.get('h2')
    # h.cmd('cd /home/mininet/go/src/github.com/lucas-clemente/quic-go-scunder/example/echo/test_for_TCP_2022_9_24/client/; ifconfig h2-eth0 mtu 1800; go run client.go -ip=10.0.0.1:6870 >c_TCP.log ')    

    CLI(net)
    net.stop()

if __name__ == '__main__':
    setLogLevel( 'info' )
    # Prevent test_simpleperf from failing due to packet loss
    perfTest(int(argv[1]))

