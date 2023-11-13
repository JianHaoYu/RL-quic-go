import matplotlib.pyplot as plt
import numpy as np

fig1, ax = plt.subplots()

constTime =1662964597.214925
# #>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>
f_Data=[]
f_Time =[]
lossrate_Data = []
lossrate_Time=[]
TrueLossRate_Data =[]
TrueLossRate_Time =[]

filename = "/home/mininet/go/src/github.com/lucas-clemente/quic-go-scunder/example/echo/test_2022_9_12/s.log"
with open(filename,'r') as file_object:
    for line in file_object:
        linecopy = line
        information = linecopy.split( )
        if information[0]=="[ServerSend":
            f_Time.append(float(information[4])-constTime)
        if information[0]=="f:":
            f_Data.append(float(information[1]))

        if information[0]=="LostRate:":
            lossrate_Data.append(float(information[1]))
            lossrate_Time.append(float(information[3])-constTime)

file_object.close()

filename = "/home/mininet/go/src/github.com/lucas-clemente/quic-go-scunder/example/echo/test_2022_9_12/c.log"
with open(filename,'r') as file_object:
    for line in file_object:
        linecopy = line
        information = linecopy.split( )
        if information[0]=="[TrueLossRate]:":
            TrueLossRate_Data.append(float(information[1]))
            TrueLossRate_Time.append(float(information[3])-constTime)

file_object.close()

ax.plot(lossrate_Time,lossrate_Data,alpha=0.8,label="Lossrate",linewidth=3)
ax.plot(TrueLossRate_Time,TrueLossRate_Data,alpha=0.8,label="TrueLossRate",linewidth=3)
ax.plot(f_Time,f_Data,alpha=0.8,label="f",linewidth=3)
# ax.set_ylabel('Packet/s',fontsize = 30)
ax.set_xlabel('Time(s)',fontsize = 30)
ax.grid(True)
ax.legend(fontsize=30)
plt.tick_params(labelsize=23)
plt.show()
#>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>
