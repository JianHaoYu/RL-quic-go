import matplotlib.pyplot as plt
import numpy as np

fig1, ax = plt.subplots()

constTime =1663042474.470298
# #>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>
CWND_Data=[]
CWND_Time=[]

filename = "/home/mininet/go/src/github.com/lucas-clemente/quic-go-scunder/example/echo/test_2022_9_12/s.log"
with open(filename,'r') as file_object:
    for line in file_object:
        linecopy = line
        information = linecopy.split( )
        if information[0]=="[CWND]":
            CWND_Data.append(int(information[6]))
            CWND_Time.append(float(information[8])-constTime)
file_object.close()

ax.plot(CWND_Time,CWND_Data,color="red",alpha=0.8,label="CWND",linewidth=3)
ax.set_ylabel('CWND(Packet)',fontsize = 30)
ax.set_xlabel('Time(s)',fontsize = 30)
ax.grid(True)
ax.legend(fontsize=30)
plt.tick_params(labelsize=23)
plt.show()
#>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>
