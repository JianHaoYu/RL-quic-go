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
        Switch1 = self.addSwitch('s1')
        self.addLink( Host1, Switch1, bw=500, delay='5ms')
        Switch2 = self.addSwitch('s2')
        self.addLink( Host2, Switch2, bw=500, delay='5ms')
        # Add links
        # bottleneck link
        # 20Mbps, 100ms delay, 1% packet loss
        self.addLink( Switch1, Switch2, bw=20, delay='5ms', loss=5)


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

    # for i in range(0,10):
    #     h = net.get('h1')
    #     h.cmd('cd /home/mininet/go/src/github.com/lucas-clemente/quic-go-scunder/example/echo/server/; export LD_LIBRARY_PATH=$LD_LIBRARY_PATH:./; go run server.go -ip=10.0.0.1:6868  -testdata=test_2500.txt  >s_SCQUIC_WiFi_5_2500_'+str(i)+'.log &')
    #     h = net.get('h2')
    #     h.cmd('cd /home/mininet/go/src/github.com/lucas-clemente/quic-go-scunder/example/echo/client/; export LD_LIBRARY_PATH=$LD_LIBRARY_PATH:./; go run client.go -ip=10.0.0.1:6868 >c_SCQUIC_WiFi_5_2500_'+str(i)+'.log ')

    # for i in range(0,10):
    #     h = net.get('h1')
    #     h.cmd('cd /home/mininet/go/src/github.com/lucas-clemente/quic-go-scunder/example/echo/server/; export LD_LIBRARY_PATH=$LD_LIBRARY_PATH:./; go run server.go -ip=10.0.0.1:6868  -testdata=test_25000.txt  >s_SCQUIC_WiFi_5_25000_'+str(i)+'.log &')
    #     h = net.get('h2')
    #     h.cmd('cd /home/mininet/go/src/github.com/lucas-clemente/quic-go-scunder/example/echo/client/; export LD_LIBRARY_PATH=$LD_LIBRARY_PATH:./; go run client.go -ip=10.0.0.1:6868 >c_SCQUIC_WiFi_5_25000_'+str(i)+'.log ')

    CLI(net)
    net.stop()

if __name__ == '__main__':
    setLogLevel( 'info' )
    # Prevent test_simpleperf from failing due to packet loss
    perfTest(int(argv[1]))

