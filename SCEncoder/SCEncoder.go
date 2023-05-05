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
)

type EncoderIDQueue struct {
	queue map[int]int
	Lock  sync.Mutex
}

func (dq *EncoderIDQueue) Push(QUICPacketID int, SCPacketID int) {
	// fmt.Printf("QUICID: %d , SourceID : %d \n", QUICPacketID, SCPacketID)
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
	delete(dq.queue, QUICPacketID)
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

	LastPacketSendTime float64
	FirstPacketSented  bool
	Estimated_EW       *Estimated
	Estimated_R        *Estimated
	Estimated_f        *Estimated

	sync.Mutex
}

//cp.gfpower: 8 	enc.cp.pktsize: 1400 	enc.cp.repfreq: 0.200000 	enc.cp.seed: 0
func Initialize_encoder(gfpower int, pktsize int, repfreq float64, seed int) *SCEncoder {
	cp := C.initialize_parameters(C.int(gfpower), C.int(pktsize), C.double(repfreq), C.int(seed))
	enc := C.initialize_encoder((*C.struct_parameters)(unsafe.Pointer(cp)), nil, 0)
	return &SCEncoder{
		PacketID:          -1,
		PacketSize:        pktsize,
		enc:               enc,
		EncoderIDQueue:    *newEncoderIDQueue(),
		FirstPacketSented: false,
		Estimated_EW:      &Estimated{},
		Estimated_R:       &Estimated{},
		Estimated_f:       &Estimated{},
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
	// fmt.Println("\t\tsend Source Packet ", e.PacketID)
	return sourcepacket
}

func (e *SCEncoder) Output_SourcePacket() []byte {
	e.Lock()
	defer e.Unlock()

	sourcepacket := C.output_source_packet(e.enc)
	serializepkt := C.serialize_packet(e.enc, sourcepacket)
	GObuf := C.GoBytes(unsafe.Pointer(serializepkt), C.int(e.PacketSize+16))
	pktbytes := make([]byte, e.PacketSize+16)
	copy(pktbytes, GObuf)
	C.free(unsafe.Pointer(serializepkt))
	C.free_packet(sourcepacket)
	return pktbytes
}

func (e *SCEncoder) RetrieveCodedPackets() ([]byte, bool) {
	packetID := e.PacketID
	timeToSendRepairePacket := packetID % 5
	if timeToSendRepairePacket == 0 {
		return e.Output_RepairPacket(), true
	} else {
		return nil, false
	}
}

func (e *SCEncoder) Output_RepairPacket() []byte {
	e.Lock()
	defer e.Unlock()
	repairepacket := C.output_repair_packet(e.enc)
	serializepkt := C.serialize_packet(e.enc, repairepacket)
	GObuf := C.GoBytes(unsafe.Pointer(serializepkt), C.int(e.PacketSize+16))
	pktbytes := make([]byte, e.PacketSize+16)
	copy(pktbytes, GObuf)
	C.free(unsafe.Pointer(serializepkt))
	C.free_packet(repairepacket)
	return pktbytes
}

func (e *SCEncoder) Flush_AckedPackets(QUICPacketID int, minRTT time.Duration) {
	e.Lock()
	defer e.Unlock()
	inorder := e.EncoderIDQueue.Pop(QUICPacketID)
	if inorder == -1 {
		return
	}
	C.flush_acked_packets(e.enc, C.int(inorder))

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

type Estimated struct {
	sync.Mutex
	data_0 float64
	data_1 float64
	data_2 float64
	data_3 float64
	data_4 float64
	data_5 float64
	data_6 float64
	data_7 float64
	data_8 float64
	data_9 float64

	Estimatedsum float64
	sumAverage   float64
}

func (e *Estimated) updata() {
	e.Estimatedsum = (e.data_0 + e.data_1 + e.data_2 + e.data_3 + e.data_4 + e.data_5 + e.data_6 + e.data_7 + e.data_8 + e.data_9) / 10
}

func (e *Estimated) Enqueue(data_new float64) {
	e.Lock()
	defer e.Unlock()
	e.data_9 = e.data_8
	e.data_8 = e.data_7
	e.data_7 = e.data_6
	e.data_6 = e.data_5
	e.data_5 = e.data_4
	e.data_4 = e.data_3
	e.data_3 = e.data_2
	e.data_2 = e.data_1
	e.data_1 = e.data_0
	e.data_0 = data_new
	e.updata()
}
