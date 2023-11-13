import matplotlib.pyplot as plt
import numpy as np

fig1, ax = plt.subplots()

constTime =1663042474.470298
# #>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>
ServerID_Data=[0]*25000
ServerID_Time=[0]*25000

ClientID_Data=[0]*25000
ClientID_Time=[0]*25000

Delay_ID =[0]*25000
Delay_Time =[0]*25000
Delay_Data =[0]*25000
Delay_average = [0.0]
		# fmt.Printf("[Server] SendPacketNumber: %d ,Length: %d ,AtTime: %f  \n", PacketNumber, len(packet), float64(time.Now().UnixNano()/1e3)/1e6)

filename = "/home/mininet/go/src/github.com/lucas-clemente/quic-go-scunder/example/echo/test_2022_9_12/s.log"
with open(filename,'r') as file_object:
    for line in file_object:
        linecopy = line
        information = linecopy.split( )
        if information[0]=="[Server]" and information[1]=="SendPacketNumber:" :
            ServerID = int(information[2])
            ServerTime = float(information[6])-constTime
            if ServerID_Time[ServerID] == 0 :
                ServerID_Data[ServerID] = ServerTime
                ServerID_Time[ServerID] = ServerTime
file_object.close()

filename = "/home/mininet/go/src/github.com/lucas-clemente/quic-go-scunder/example/echo/test_2022_9_12/c.log"
with open(filename,'r') as file_object:
    for line in file_object:
        linecopy = line
        information = linecopy.split( )
        if information[0]=="[Client]" and information[1] =="GetPacketNumber:":
            ClientID = int(information[2])-1
            ClientTime = float(information[4])-constTime
            if ClientID_Time[ClientID] == 0 and ServerID_Data[ClientID] != 0:
                ClientID_Data[ClientID] = ClientTime
                ClientID_Time[ClientID] = ClientTime
                Delay_Data[ClientID] =ClientID_Data[ClientID] -ServerID_Data[ClientID]
                Delay_Time[ClientID] = ClientTime
                Delay_ID[ClientID] =ClientID

file_object.close()

for i in range(0,len(Delay_ID)-1):
    if Delay_Data[i] == 0:
        Delay_Data.pop(i)
        Delay_ID.pop(i)
        Delay_Time.pop(i)

sumDelay =0
for i in range(0,len(Delay_ID)-1):
    sumDelay = float(sumDelay + Delay_Data[i])

    Delay_average.append(sumDelay/(i+1))


# ax.plot(Delay_Time,Delay_Data,label="Delay")
ax.plot(Delay_Time,Delay_average,label="Delay_Average")
# ax.plot(Delay_Time,Delay_Data,color="red",alpha=0.8,label="ThoughtPut",linewidth=3)


ax.set_ylabel('Delay(s)',fontsize = 30)
ax.set_xlabel('Time(s)',fontsize = 30)
ax.grid(True)
ax.legend(fontsize=30)
plt.tick_params(labelsize=23)
plt.show()
#>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>
