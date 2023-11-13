import matplotlib.pyplot as plt
import numpy as np

fig1, ax = plt.subplots()

constTime =1663042474.470298
# #>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>
ThoughtPut_Data=[]
ThoughtPut_Time=[]

filename = "/home/mininet/go/src/github.com/lucas-clemente/quic-go-scunder/example/echo/test_2022_9_12/c.log"
with open(filename,'r') as file_object:
    for line in file_object:
        linecopy = line
        information = linecopy.split( )
        if information[0]=="[Client]":
            ClientID = float(information[2])
            ClientTime = float(information[4])-constTime
            ThoughtPut = ClientID/ClientTime
            ThoughtPut_Data.append(ThoughtPut)
            ThoughtPut_Time.append(ClientTime)
file_object.close()

ax.plot(ThoughtPut_Time,ThoughtPut_Data,color="red",alpha=0.8,label="ThoughtPut",linewidth=3)
ax.set_ylabel('ThoughtPut(Packet/s)',fontsize = 30)
ax.set_xlabel('Time(s)',fontsize = 30)
ax.grid(True)
ax.legend(fontsize=30)
plt.tick_params(labelsize=23)
plt.show()
#>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>
