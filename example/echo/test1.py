#!/usr/bin/python

from mininet.topo import Topo
from mininet.net import Mininet
from mininet.node import CPULimitedHost
from mininet.link import TCLink
from mininet.util import dumpNodeConnections
from mininet.log import setLogLevel, info
from mininet.cli import CLI
import time
from sys import argv


class MyTopo( Topo ):
    "Simple topology example."
    def build( self,n=1):
        "Create custom topo."

        # Add hosts and switches
        Switch1 = self.addSwitch('s1')
        Switch2 = self.addSwitch('s2')

        for i in range(1,n+1):
            Host = self.addHost( 'h'+str(i))
            self.addLink( Host, Switch1, bw=100, delay='5ms')
        for i in range(n+1,2*n+1):
            Host = self.addHost( 'h'+str(i))
            self.addLink( Host, Switch2, bw=100, delay='5ms')
        # Add links
        # bottleneck link
        # 20Mbps, 100ms delay, 1% packet loss
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

    # for i in range(1,num+2):
    #     s =net.get('s1')
    #     s.cmd('ifconfig s1-eth'+str(i)+' mtu 1800')
    # for i in range(1,num+2):
    #     s =net.get('s2')
    #     s.cmd('ifconfig s2-eth'+str(i)+' mtu 1800')

    CLI(net)
    net.stop()

if __name__ == '__main__':
    setLogLevel( 'info' )
    # Prevent test_simpleperf from failing due to packet loss
    perfTest(int(argv[1]))

