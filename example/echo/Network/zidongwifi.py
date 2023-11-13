#!/usr/bin/python
import os
#WIFI#######################################################################################################################
#*******************************************************************************************************************************
#2lossfor2500and25000QUICW
cmd = 'sudo python /home/mininet/go/src/github.com/lucas-clemente/quic-go-scunder/example/echo/Network/wifi2.py 1'
res = os.popen(cmd)
output_str = res.read()  
cmd = 'sudo mn -c '
res = os.popen(cmd)
output_str = res.read()   

#3lossfor2500and25000QUICW
cmd = 'sudo python /home/mininet/go/src/github.com/lucas-clemente/quic-go-scunder/example/echo/Network/wifi3.py 1'
res = os.popen(cmd)
output_str = res.read()  
cmd = 'sudo mn -c '
res = os.popen(cmd)
output_str = res.read()   

#4lossfor2500and25000QUICW
cmd = 'sudo python /home/mininet/go/src/github.com/lucas-clemente/quic-go-scunder/example/echo/Network/wifi4.py 1'
res = os.popen(cmd)
output_str = res.read()  
cmd = 'sudo mn -c '
res = os.popen(cmd)
output_str = res.read()   

#5lossfor2500and25000QUICW
cmd = 'sudo python /home/mininet/go/src/github.com/lucas-clemente/quic-go-scunder/example/echo/Network/wifi5.py 1'
res = os.popen(cmd)
output_str = res.read()  
cmd = 'sudo mn -c '
res = os.popen(cmd)
output_str = res.read()   
