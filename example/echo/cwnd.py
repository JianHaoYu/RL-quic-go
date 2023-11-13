import matplotlib.pyplot as plt
import numpy as np

fig1, ax = plt.subplots()

# #>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>
DW_Data = []
DW_Time=[]
DW_average =[]
LastDW = 0
lastTime =0

filename = "/home/mininet/go/src/github.com/lucas-clemente/quic-go-scunder/example/echo/client/c_SCQUIC_250000_10.log"
with open(filename,'r') as file_object:
    for line in file_object:
        linecopy = line
        information = linecopy.split( )
        if information[0]=="[DW]:":
            DW=int(information[1])
            Time = float(information[7])
            if LastDW!=0 and DW ==0:
                DW_Data.append(LastDW)
                DW_Time.append(lastTime)
            LastDW = DW
            lastTime = Time
file_object.close()

DW_average.append(0)
SCsumDelay =0
for i in range(0,len(DW_Data)-1):
    SCsumDelay = float(SCsumDelay + DW_Data[i])
    DW_average.append(SCsumDelay/(i+1))

ax.plot(DW_Time,DW_Data,color="red",alpha=0.8,label="DW",linewidth=3)
ax.plot(DW_Time,DW_average,alpha=0.8,label="DW_Averge",linewidth=3)

# ax.plot(lossrateTime,lossrate,color="blue",alpha=0.8,label="Server",linewidth=3)
ax.set_ylabel('Packet/s',fontsize = 30)
ax.set_xlabel('Time(s)',fontsize = 30)
ax.grid(True)
ax.legend(fontsize=30)
plt.tick_params(labelsize=23)
plt.show()
#>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>
