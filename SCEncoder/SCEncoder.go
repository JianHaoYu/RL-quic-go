package SCEncoder

//#cgo CFLAGS: -I./streamc
//#cgo LDFLAGS: -L${SRCDIR}/streamc -lstreamc
//
// #include <streamcodec.h>
import "C"
import (
	"bytes"
	"encoding/binary"
	"fmt"
	"sync"
	"time"
	"unsafe"

	"github.com/lucas-clemente/quic-go/internal/wire"
)

type EncoderIDQueue struct {
	queue map[int]int
	Lock  sync.Mutex
}

func (dq *EncoderIDQueue) Push(QUICPacketID int, SCPacketID int) {
	fmt.Printf("QUICID: %d , SourceID : %d \n", QUICPacketID, SCPacketID)
	dq.Lock.Lock()
	defer dq.Lock.Unlock()
	_, ok := dq.queue[QUICPacketID]
	if ok {
		fmt.Println("StreamCodeEecoderWorring: This sourceid already exists ")
		return
	}
	dq.queue[QUICPacketID] = SCPacketID
}

func (dq *EncoderIDQueue) Pop(QUICPacketID int) int {
	dq.Lock.Lock()
	defer dq.Lock.Unlock()
	_, ok := dq.queue[QUICPacketID]
	if !ok {
		fmt.Println("StreamCodeEecoderWorring: This sourceid does not exists ")
		return -1
	}
	SCPacketID := dq.queue[QUICPacketID]
	// delete(dq.queue, QUICPacketID)
	return SCPacketID
}

func newEncoderIDQueue() *EncoderIDQueue {
	return &EncoderIDQueue{
		queue: make(map[int]int),
	}
}

type SCEncoder struct {
	PacketID       int
	PacketSize     int
	enc            *C.struct_encoder
	EncoderIDQueue EncoderIDQueue

	SendSourcePacketSum int
	SendRepairPacketSum int
	// AckHistoryHander    *SCPacketHistoryHander
	// PacketHistoryHander *SCPacketHistoryHander
	// EWHistoryHander     *SCPacketHistoryHander

	DWGuJi      float64
	LostRate    float64
	ExtraRepair float64
	f           float64
	Tp          float64

	//For ACK
	inorder       int
	inorderQUICID int

	sync.Mutex
}

//cp.gfpower: 8 	enc.cp.pktsize: 1400 	enc.cp.repfreq: 0.200000 	enc.cp.seed: 0
func Initialize_encoder(gfpower int, pktsize int, repfreq float64, seed int) *SCEncoder {
	cp := C.initialize_parameters(C.int(gfpower), C.int(pktsize), C.double(repfreq), C.int(seed))
	enc := C.initialize_encoder((*C.struct_parameters)(unsafe.Pointer(cp)), nil, 0)
	return &SCEncoder{
		PacketID:            -1,
		PacketSize:          pktsize,
		enc:                 enc,
		EncoderIDQueue:      *newEncoderIDQueue(),
		SendSourcePacketSum: 0,
		SendRepairPacketSum: 0,
		// AckHistoryHander:    newSCPacketHistoryHander(),
		// PacketHistoryHander: newSCPacketHistoryHander(),
		// EWHistoryHander:     newSCPacketHistoryHander(),
		LostRate:    0,
		ExtraRepair: 0.03,
		// f:           0.2,
		f:  0,
		Tp: 0,
	}
}

func (e *SCEncoder) Enqueue_Packet(enqueuePacket []byte, QUICPacketID int) []byte {
	e.PacketID = e.PacketID + 1
	header := IntToBytes(len(enqueuePacket))
	header = append(header, enqueuePacket...)
	MaxPacketBufferSizeByte := make([]byte, e.PacketSize)
	copy(MaxPacketBufferSizeByte, header)
	Cpkt := C.CBytes(MaxPacketBufferSizeByte)
	C.enqueue_packet(e.enc, C.int(e.PacketID), (*C.uchar)(Cpkt))
	sourcepacket := e.Output_SourcePacket()
	C.free(Cpkt)
	e.EncoderIDQueue.Push(QUICPacketID, e.PacketID)
	return sourcepacket
}

func (e *SCEncoder) Output_SourcePacket() []byte {
	e.Lock()
	defer e.Unlock()
	e.SendSourcePacketSum = e.SendSourcePacketSum + 1
	sourcepacket := C.output_source_packet(e.enc)
	serializepkt := C.serialize_packet(e.enc, sourcepacket)
	GObuf := C.GoBytes(unsafe.Pointer(serializepkt), C.int(e.PacketSize+16))
	pktbytes := make([]byte, e.PacketSize+16)
	copy(pktbytes, GObuf)
	C.free(unsafe.Pointer(serializepkt))
	C.free_packet(sourcepacket)

	// EW := int(e.enc.nextsid) - int(e.enc.headsid)
	fmt.Printf("EW: [%d ,%d] \n", int(e.enc.nextsid), int(e.enc.headsid))
	return pktbytes
}

func (e *SCEncoder) RetrieveCodedPackets() ([]byte, bool) {
	currentF := float64(e.SendRepairPacketSum) / float64(e.SendSourcePacketSum+e.SendRepairPacketSum)
	fmt.Printf("f: %f \n", e.f)
	fmt.Printf("currentF: %f \n", currentF)
	if currentF < e.f {
		return e.Output_RepairPacket(), true
	} else {
		return nil, false
	}
}

func (e *SCEncoder) Output_RepairPacket() []byte {
	e.Lock()
	defer e.Unlock()
	e.SendRepairPacketSum = e.SendRepairPacketSum + 1
	repairepacket := C.output_repair_packet(e.enc)
	serializepkt := C.serialize_packet(e.enc, repairepacket)
	GObuf := C.GoBytes(unsafe.Pointer(serializepkt), C.int(e.PacketSize+16))
	pktbytes := make([]byte, e.PacketSize+16)
	copy(pktbytes, GObuf)
	C.free(unsafe.Pointer(serializepkt))
	C.free_packet(repairepacket)
	return pktbytes
}

func (e *SCEncoder) Flush_AckedPackets(f *wire.AckFrame, minRTT time.Duration, SmoothedRTT time.Duration, LatestRtt time.Duration) {
	e.Lock()
	defer e.Unlock()
	GetACKtime := float64(time.Now().UnixNano()) / 1e9

	inorder := e.GETinorder(f)
	if inorder == -1 {
		return
	}
	C.flush_acked_packets(e.enc, C.int(inorder))

	fmt.Printf("EWinorder: %d 	e.inorderQUICID: %d ,Time: %f \n", inorder, e.inorderQUICID, GetACKtime)

	e.UpdataTp(minRTT.Seconds(), SmoothedRTT.Seconds())
}

func (e *SCEncoder) GETinorder(f *wire.AckFrame) int {
	fmt.Printf("f.LowestAcked %d ,flen: %d \n", f.LowestAcked(), len(f.AckRanges))
	fmt.Println("f: ", f.AckRanges)
	LenAckRanges := len(f.AckRanges)
	if LenAckRanges == 1 {
		QUICID := int(f.LargestAcked())
		e.inorder = e.EncoderIDQueue.Pop(QUICID)
		e.inorderQUICID = QUICID
		// fmt.Println("check1 e.inorder ", e.inorder)
		return e.inorder
	}
	var InorderInWhichAckRange int
	for i := 0; i < LenAckRanges; i++ {
		Largest := f.AckRanges[i].Largest
		Smallest := f.AckRanges[i].Smallest
		if e.inorderQUICID >= int(Smallest) && e.inorderQUICID <= int(Largest) {
			InorderInWhichAckRange = i
			break
		}
	}
	if InorderInWhichAckRange == 0 {
		QUICID := int(f.LargestAcked())
		e.inorder = e.EncoderIDQueue.Pop(QUICID)
		e.inorderQUICID = QUICID
		return e.inorder
	}
	for InorderInWhichAckRange != 0 {
		Largest := f.AckRanges[InorderInWhichAckRange].Largest
		NextSmallest := f.AckRanges[InorderInWhichAckRange-1].Smallest
		SCLargest := e.EncoderIDQueue.Pop(int(Largest))
		SCNextSmallest := e.EncoderIDQueue.Pop(int(NextSmallest))
		if SCLargest+1 != SCNextSmallest {
			break
		}
		InorderInWhichAckRange = InorderInWhichAckRange - 1
	}
	QUICID := int(f.AckRanges[InorderInWhichAckRange].Largest)
	e.inorder = e.EncoderIDQueue.Pop(QUICID)
	e.inorderQUICID = QUICID
	// fmt.Println("check3 e.inorder ", e.inorder) //error at check3
	return e.inorder
}

func (e *SCEncoder) UpdataF(LostRate float64) {
	//UpDate LostRate
	alpha := 0.9
	if e.LostRate == 0 {
		e.LostRate = LostRate
	} else {
		e.LostRate = e.LostRate*alpha + LostRate*(1-alpha)
	}
	//UpDate F
	e.f = e.LostRate + e.ExtraRepair
}

func (e *SCEncoder) UpdataTp(MinRTT float64, SmoothedRTT float64) {
	Backdelay := MinRTT / 2
	// forwarddelay := SmoothedRTT - Backdelay
	e.Tp = Backdelay
	// e.AckHistoryHander.Tp = Backdelay
	// e.EWHistoryHander.Tp = Backdelay
	// e.PacketHistoryHander.Tp = forwarddelay
}

func (e *SCEncoder) PrintInfo() {
	fmt.Println("--------------ENC--------------")
	fmt.Printf("enc.cp.gfpower: %d \t", int(e.enc.cp.gfpower))
	fmt.Printf("enc.cp.pktsize: %d \t", int(e.enc.cp.pktsize))
	fmt.Printf("enc.cp.repfreq: %f \t", float64(e.enc.cp.repfreq))
	fmt.Printf("enc.cp.seed: %d \t", int(e.enc.cp.seed))
	fmt.Printf("enc.snum: %d \t", int(e.enc.snum))
	fmt.Printf("enc.count: %d \t", int(e.enc.count))
	fmt.Printf("enc.nextsid: %d \t", int(e.enc.nextsid))
	fmt.Printf("e.enc.bufsize: %d \t", int(e.enc.bufsize))
	fmt.Printf("enc.head: %d \t", int(e.enc.head))
	fmt.Printf("enc.headsid: %d \n", int(e.enc.headsid))
}

func IntToBytes(n int) []byte {
	x := int64(n)
	bytesBuffer := bytes.NewBuffer([]byte{})
	binary.Write(bytesBuffer, binary.BigEndian, x)
	return bytesBuffer.Bytes()
}

// type SCPacketHistory struct {
// 	PacketSumNember int
// 	Time            float64
// }

// type SCPacketHistoryHander struct {
// 	PacketHistoryHander []*SCPacketHistory
// 	Tp                  float64
// }

// func newSCPacketHistoryHander() *SCPacketHistoryHander {
// 	return &SCPacketHistoryHander{
// 		PacketHistoryHander: make([]*SCPacketHistory, 0),
// 		Tp:                  0,
// 	}
// }

// func (s *SCPacketHistoryHander) GetAverage() float64 {
// 	Now := float64(time.Now().UnixNano()) / 1e9
// 	endIndex := len(s.PacketHistoryHander) - 1
// 	if endIndex == 0 {
// 		return 0
// 	}
// 	StartSumNum := s.PacketHistoryHander[endIndex].PacketSumNember
// 	StartTime := s.PacketHistoryHander[endIndex].Time
// 	EndSumNum := s.PacketHistoryHander[0].PacketSumNember
// 	EndTime := s.PacketHistoryHander[0].Time
// 	for i := endIndex - 1; i >= 0; i-- {
// 		if Now-s.Tp > s.PacketHistoryHander[i].Time {
// 			EndSumNum = s.PacketHistoryHander[i].PacketSumNember
// 			EndTime = s.PacketHistoryHander[i].Time
// 			break
// 		}
// 	}
// 	AverageSpeed := (float64(StartSumNum - EndSumNum)) / (Now - EndTime)
// 	fmt.Printf("AverageSpeed: %f endindex: %d ,StartSumNum: %d ,EndSumNum:%d ,StartTime: %f ,EndTime: %f \n", AverageSpeed, endIndex, StartSumNum, EndSumNum, StartTime, EndTime)

// 	return AverageSpeed
// }

// func (e *SCPacketHistoryHander) Enqueue(PacketSumNember int, Time float64) {
// 	newData := &SCPacketHistory{
// 		PacketSumNember: PacketSumNember,
// 		Time:            Time,
// 	}
// 	e.PacketHistoryHander = append(e.PacketHistoryHander, newData)
// }

// func (s *SCPacketHistoryHander) GetEWBeforeTp() float64 {
// 	Now := float64(time.Now().UnixNano()) / 1e9
// 	endIndex := len(s.PacketHistoryHander) - 1
// 	EWBeforeTp := s.PacketHistoryHander[endIndex].PacketSumNember
// 	for i := endIndex - 1; i >= 0; i-- {
// 		if Now-s.Tp < s.PacketHistoryHander[i].Time {
// 			EWBeforeTp = s.PacketHistoryHander[i].PacketSumNember
// 		}
// 	}
// 	return float64(EWBeforeTp)
// }
