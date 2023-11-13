import matplotlib.pyplot as plt
import numpy as np
from scipy import interpolate

fig1, ax = plt.subplots()


# #>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>
SCServerID_Data=[0]*25000
SCServerID_Time=[0]*25000

SCClientID_Data=[0]*25000
SCClientID_Time=[0]*25000

SCDelay_ID =[0]*25000
SCDelay_Data =[0]*25000
SCDelay_average = []

filename = "/home/mininet/go/src/quic-go-scunder/example/echo/server/s_SCQUIC_Satellite_1_25000_SARSA_.log"
with open(filename,'r') as file_object:
    for line in file_object:
        linecopy = line
        information = linecopy.split( )
        if information[0]=="[Server]" and information[1]=="SendPacketNumber:" :
            ServerID = int(information[2])
            ServerTime = float(information[6])
            if SCServerID_Time[ServerID] == 0 :
                SCServerID_Data[ServerID] = ServerTime
                SCServerID_Time[ServerID] = ServerTime
file_object.close()

filename = "/home/mininet/go/src/quic-go-scunder/example/echo/client/c_SCQUIC_Satellite_1_25000_SARSA_.log"
with open(filename,'r') as file_object:
    for line in file_object:
        linecopy = line
        information = linecopy.split( )
        if information[0]=="[Client]" and information[1] =="GetPacketNumber:":
            ClientID = int(information[2])-1
            ClientTime = float(information[4])
            if SCServerID_Data[ClientID] != 0:
                SCClientID_Data[ClientID] = ClientTime
                SCClientID_Time[ClientID] = ClientTime
                SCDelay_Data[ClientID] =SCClientID_Data[ClientID] -SCServerID_Data[ClientID]
                SCDelay_ID[ClientID] =ClientID

file_object.close()

for i in range(0,250000):
    try:
        if SCDelay_Data[i]==0:
            SCDelay_Data.pop(i)
            SCDelay_ID.pop(i)
    except IndexError:
        a = 0

SCsumDelay =0

for i in range(0,len(SCDelay_ID)):
    SCsumDelay = float(SCsumDelay + SCDelay_Data[i])
    SCDelay_average.append(SCsumDelay/(i+1))

ax.plot(SCDelay_ID,SCDelay_average,label="SCQUIC")



ax.set_ylabel('Delay(s)',fontsize = 30)
ax.set_xlabel('Time(s)',fontsize = 30)
ax.grid(True)
ax.legend(fontsize=30)
plt.tick_params(labelsize=23)
plt.show()