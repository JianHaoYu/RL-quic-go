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
        Host3 = self.addHost( 'h3')
        Host4 = self.addHost( 'h4')
        Host5 = self.addHost( 'h5')
        Host6 = self.addHost( 'h6')       
        Host7 = self.addHost( 'h7')
        Host8 = self.addHost( 'h8')
        Host9 = self.addHost( 'h9')
        Host10 = self.addHost( 'h10')
        Switch1 = self.addSwitch('s1')
        self.addLink( Host1, Switch1, bw=500, delay='25ms')
        self.addLink( Host2, Switch1, bw=500, delay='25ms')
        self.addLink( Host3, Switch1, bw=500, delay='25ms')
        self.addLink( Host4, Switch1, bw=500, delay='25ms')
        self.addLink( Host5, Switch1, bw=500, delay='25ms')
        Switch2 = self.addSwitch('s2')
        self.addLink( Host6, Switch2, bw=500, delay='25ms')
        self.addLink( Host7, Switch2, bw=500, delay='25ms')
        self.addLink( Host8, Switch2, bw=500, delay='25ms')
        self.addLink( Host9, Switch2, bw=500, delay='25ms')
        self.addLink( Host10, Switch2, bw=500, delay='25ms')

        
        # Add links
        # bottleneck link
        # 20Mbps, 100ms delay, 1% packet loss
        # self.addLink( Switch1, Switch2, bw=25, delay='250ms', loss=0.1)
        self.addLink( Switch1, Switch2, bw=25, delay='250ms', loss=1)



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


    h = net.get('h1')
    h.cmd('cd /home/mininet/go/src/github.com/lucas-clemente/quic-go-scunder/example/echo/server/; export LD_LIBRARY_PATH=$LD_LIBRARY_PATH:./; go run server.go -ip=10.0.0.1:6868  -testdata=test_25000.txt >s_SCQUIC_Satellite_1_25000_Firess_1.log &')
    h = net.get('h2')
    h.cmd('cd /home/mininet/go/src/github.com/lucas-clemente/quic-go-scunder/example/echo/server/; export LD_LIBRARY_PATH=$LD_LIBRARY_PATH:./; go run server.go -ip=10.0.0.2:6868  -testdata=test_25000.txt >s_SCQUIC_Satellite_1_25000_Firess_2.log &')
    h = net.get('h3')
    h.cmd('cd /home/mininet/go/src/github.com/lucas-clemente/quic-go-scunder/example/echo/server/; export LD_LIBRARY_PATH=$LD_LIBRARY_PATH:./; go run server.go -ip=10.0.0.3:6868  -testdata=test_25000.txt >s_SCQUIC_Satellite_1_25000_Firess_3.log &')
    h = net.get('h4')
    h.cmd('cd /home/mininet/go/src/github.com/lucas-clemente/quic-go-scunder/example/echo/server/; export LD_LIBRARY_PATH=$LD_LIBRARY_PATH:./; go run server.go -ip=10.0.0.4:6868  -testdata=test_25000.txt >s_SCQUIC_Satellite_1_25000_Firess_4.log &')
    h = net.get('h5')
    h.cmd('cd /home/mininet/go/src/github.com/lucas-clemente/quic-go-scunder/example/echo/server/; export LD_LIBRARY_PATH=$LD_LIBRARY_PATH:./; go run server.go -ip=10.0.0.5:6868  -testdata=test_25000.txt >s_SCQUIC_Satellite_1_25000_Firess_5.log &')

    h = net.get('h6')
    h.cmd('cd /home/mininet/go/src/github.com/lucas-clemente/quic-go-scunder/example/echo/client/; export LD_LIBRARY_PATH=$LD_LIBRARY_PATH:./; go run client.go -ip=10.0.0.1:6868 >c_SCQUIC_Satellite_1_25000_Firess_1.log &')
    h = net.get('h7')
    h.cmd('cd /home/mininet/go/src/github.com/lucas-clemente/quic-go-scunder/example/echo/client/; export LD_LIBRARY_PATH=$LD_LIBRARY_PATH:./; go run client.go -ip=10.0.0.2:6868 >c_SCQUIC_Satellite_1_25000_Firess_2.log &')
    h = net.get('h8')
    h.cmd('cd /home/mininet/go/src/github.com/lucas-clemente/quic-go-scunder/example/echo/client/; export LD_LIBRARY_PATH=$LD_LIBRARY_PATH:./; go run client.go -ip=10.0.0.3:6868 >c_SCQUIC_Satellite_1_25000_Firess_3.log &')
    h = net.get('h9')
    h.cmd('cd /home/mininet/go/src/github.com/lucas-clemente/quic-go-scunder/example/echo/client/; export LD_LIBRARY_PATH=$LD_LIBRARY_PATH:./; go run client.go -ip=10.0.0.4:6868 >c_SCQUIC_Satellite_1_25000_Firess_4.log &')
    h = net.get('h10')
    h.cmd('cd /home/mininet/go/src/github.com/lucas-clemente/quic-go-scunder/example/echo/client/; export LD_LIBRARY_PATH=$LD_LIBRARY_PATH:./; go run client.go -ip=10.0.0.5:6868 >c_SCQUIC_Satellite_1_25000_Firess_5.log ')

    # CLI(net)
    net.stop()

if __name__ == '__main__':
    setLogLevel( 'info' )
    # Prevent test_simpleperf from failing due to packet loss
    perfTest(int(argv[1]))

