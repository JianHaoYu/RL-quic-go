#!/usr/bin/python
import os
#Cellular#######################################################################################################################
#*******************************************************************************************************************************
# #2lossfor2500and25000QUICW
# cmd = 'sudo python /home/mininet/go/src/github.com/lucas-clemente/quic-go-master/example/echo/Westwood/cellular/cellular2.py 1'
# res = os.popen(cmd)
# output_str = res.read()  
# cmd = 'sudo mn -c '
# res = os.popen(cmd)
# output_str = res.read()   

#3lossfor2500and25000QUICW
cmd = 'sudo python /home/mininet/go/src/github.com/lucas-clemente/quic-go-scunder/example/echo/Network/cellular3.py 1'
res = os.popen(cmd)
output_str = res.read()  
cmd = 'sudo mn -c '
res = os.popen(cmd)
output_str = res.read()   

#4lossfor2500and25000QUICW
cmd = 'sudo python /home/mininet/go/src/github.com/lucas-clemente/quic-go-scunder/example/echo/Network/cellular4.py 1'
res = os.popen(cmd)
output_str = res.read()  
cmd = 'sudo mn -c '
res = os.popen(cmd)
output_str = res.read()   

#5lossfor2500and25000QUICW
cmd = 'sudo python /home/mininet/go/src/github.com/lucas-clemente/quic-go-scunder/example/echo/Network/cellular5.py 1'
res = os.popen(cmd)
output_str = res.read()  
cmd = 'sudo mn -c '
res = os.popen(cmd)
output_str = res.read()   

