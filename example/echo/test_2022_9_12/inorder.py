import matplotlib.pyplot as plt
import numpy as np
from scipy import interpolate


fig1, ax = plt.subplots()

constTime =1663042474.470298
# #>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>
EWinorder_Data =[]
EWinorder_Time =[]
DWinorder_Data =[]
DWinorder_Time =[]

filename = "/home/mininet/go/src/github.com/lucas-clemente/quic-go-scunder/example/echo/test_2022_9_12/s.log"
with open(filename,'r') as file_object:
    for line in file_object:
        linecopy = line
        information = linecopy.split( )
        if information[0]=="EWinorder:":
            EWinorder_Data.append(int(information[1]))
            EWinorder_Time.append(float(information[3])-constTime)

file_object.close()

filename = "/home/mininet/go/src/github.com/lucas-clemente/quic-go-scunder/example/echo/test_2022_9_12/c.log"
with open(filename,'r') as file_object:
    for line in file_object:
        linecopy = line
        information = linecopy.split( )
        if information[0]=="DWInorder:":
            DWinorder_Data.append(int(information[1]))
            DWinorder_Time.append(float(information[5])-constTime)

file_object.close()

x1new=np.linspace(EWinorder_Time[0],DWinorder_Time[-1],2*len(EWinorder_Time))
f1=interpolate.interp1d(DWinorder_Time,DWinorder_Data,kind='zero')
f2=interpolate.interp1d(EWinorder_Time,EWinorder_Data,kind='zero')
y1new=f1(x1new)
y2new=f2(x1new)
ax.plot(x1new,(y1new-y2new),label="前向链路分组数")

# ax.plot(EWinorder_Time,EWinorder_Data,alpha=0.8,label="EW",linewidth=3)
# ax.plot(DWinorder_Time,DWinorder_Data,alpha=0.8,label="DW",linewidth=3)
ax.set_ylabel('Packet/s',fontsize = 30)
ax.set_xlabel('Time(s)',fontsize = 30)
ax.grid(True)
ax.legend(fontsize=30)
plt.tick_params(labelsize=23)
plt.show()
#>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>
