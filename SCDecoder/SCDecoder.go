package SCDecoder

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

type DecodedPacketQueue struct {
	queue map[int][]byte
	Lock  sync.Mutex
}

func (dq *DecodedPacketQueue) Push(SCPacketID int, pktbytes []byte) {
	dq.Lock.Lock()
	defer dq.Lock.Unlock()
	_, ok := dq.queue[SCPacketID]
	if ok {
		fmt.Println("StreamCodeDecoderWorring: This sourceid already exists ")
		return
	}
	dq.queue[SCPacketID] = pktbytes
}

func (dq *DecodedPacketQueue) Pop(SCPacketID int) []byte {
	dq.Lock.Lock()
	defer dq.Lock.Unlock()
	_, ok := dq.queue[SCPacketID]
	if !ok {
		return nil
	}
	pktbytes := dq.queue[SCPacketID]
	delete(dq.queue, SCPacketID)
	return pktbytes
}

func newDecodedPacketQueue() *DecodedPacketQueue {
	return &DecodedPacketQueue{
		queue: make(map[int][]byte),
	}
}

type Decinfo struct {
	Active  int
	Inorder int
	Win_s   int
	Win_e   int
}

func newDecinfo() *Decinfo {
	return &Decinfo{
		Active:  0,
		Inorder: 0,
		Win_s:   0,
		Win_e:   0,
	}
}

type SCDecoder struct {
	dec                *C.struct_decoder
	Decinfo            *Decinfo
	DecodedPacketQueue *DecodedPacketQueue
	PacketChan         chan []byte
	sync.Mutex
}

func Initialize_decoder(gfpower int, pktsize int, repfreq float64, seed int, PacketChan chan []byte) *SCDecoder {
	cp := C.initialize_parameters(C.int(gfpower), C.int(pktsize), C.double(repfreq), C.int(seed))
	dec := C.initialize_decoder((*C.struct_parameters)(unsafe.Pointer(cp)))
	d := &SCDecoder{
		DecodedPacketQueue: newDecodedPacketQueue(),
		Decinfo:            newDecinfo(),
		dec:                dec,
		PacketChan:         PacketChan,
	}
	go d.run()
	return d
}

func (d *SCDecoder) updateDecinfo() {
	d.Decinfo.Active = int(d.dec.active)
	d.Decinfo.Inorder = int(d.dec.inorder)
	d.Decinfo.Win_e = int(d.dec.win_e)
	d.Decinfo.Win_s = int(d.dec.win_s)
}

func (d *SCDecoder) Deserialize_Packet(pktbytes []byte) int {
	pktstr := C.CBytes(pktbytes)
	receivepacket := C.deserialize_packet(d.dec, (*C.uchar)(pktstr))
	C.free(pktstr)
	_ = C.receive_packet(d.dec, receivepacket)
	inorder := int(d.dec.inorder)
	sourceid := int(receivepacket.sourceid)
	if sourceid != -1 { //sourcePacket Put into Queue
		fmt.Println("sourceID :", sourceid)
		recoverPacket := pktbytes[16:]
		header := recoverPacket[:8]
		length := BytesToInt(header)
		d.DecodedPacketQueue.Push(sourceid, recoverPacket[8:8+length])
	}
	d.printInfo()
	d.updateDecinfo()
	return inorder
}

func (d *SCDecoder) recovered_packet(sourceid int) []byte {
	pktstr := C.recover_packet(d.dec, C.int(sourceid))
	buf := C.GoBytes(unsafe.Pointer(pktstr), C.int(1408))
	header := buf[:8]
	length := BytesToInt(header)
	buf = buf[8:length]
	return buf
}

func (d *SCDecoder) run() {
	var submitPacketID int
	deadline := time.Now().Add(10 * time.Second)
	for {
		if !time.Now().Before(deadline) {
			submitPacketID = submitPacketID + 1
			deadline = time.Now().Add(1 * time.Second)
		}
		pktbytes := d.DecodedPacketQueue.Pop(submitPacketID)
		if pktbytes == nil { //Not in Queue
			if submitPacketID <= d.Decinfo.Inorder {
				pktbytes = d.recovered_packet(submitPacketID)
				submitPacketID = submitPacketID + 1
				deadline = time.Now().Add(1 * time.Second)
			} else {
				continue
			}
		} else { //find packet in Queue
			submitPacketID = submitPacketID + 1
			deadline = time.Now().Add(1 * time.Second)
		}
		d.PacketChan <- pktbytes

	}
}

func (d *SCDecoder) printInfo() {
	fmt.Println("--------------DEC--------------")
	fmt.Printf("dec.cp.gfpower: %d \t", int(d.dec.cp.gfpower))
	fmt.Printf("dec.cp.pktsize: %d \t", int(d.dec.cp.pktsize))
	fmt.Printf("dec.cp.repfreq: %f \t", float64(d.dec.cp.repfreq))
	fmt.Printf("dec.cp.seed: %d \t", int(d.dec.cp.seed))
	fmt.Printf("dec.active: %d \t", int(d.dec.active))
	fmt.Printf("dec.inorder: %d \n", int(d.dec.inorder))
}

func BytesToInt(b []byte) int {
	bytesBuffer := bytes.NewBuffer(b)
	var x int64
	binary.Read(bytesBuffer, binary.BigEndian, &x)
	return int(x)
}
