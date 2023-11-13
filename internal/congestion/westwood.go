package congestion

import (
	"fmt"
	"time"

	"github.com/lucas-clemente/quic-go/internal/protocol"
)

type WestWood struct {
	bw_ns_est            float64            /*中间变量，经过一次平滑后的带宽值  first bandwidth estimation..not too smoothed 8) */
	bw_est               float64            /*最终估计的带宽值*/
	bk                   protocol.ByteCount //在某个时间段delta内确认的字节数
	first_ack            bool               /*是否第一个ack包   flag which infers that this is the first ack */
	rtt_win_sx           float64            //采样周期的起始点
	rtt_min              float64            //最小RTT值，以毫秒为单位
	TCP_WESTWOOD_RTT_MIN float64            //采样周期
	maxDatagramSize      protocol.ByteCount //最大分组大小
}

// NewCubic returns a new WestWood instance
func NewWestWood(initialMaxDatagramSize protocol.ByteCount) *WestWood {
	w := &WestWood{maxDatagramSize: initialMaxDatagramSize}
	w.Reset()
	return w
}

// Reset is called after a timeout to reset the cubic state
func (w *WestWood) Reset() {
	w.bw_ns_est = 0.0
	w.bw_est = 0.0
	w.bk = 0
	w.first_ack = true
	w.rtt_win_sx = float64(time.Now().UnixNano()/1e3) / 1e6
	w.rtt_min = 0.0
	w.TCP_WESTWOOD_RTT_MIN = 0.05
}

func (w *WestWood) westwood_do_filter(a float64, b float64) float64 {
	return (((7 * a) + b) / 8) //返回7a/8与1b/8之和
}

func (w *WestWood) westwood_filter(delta float64) {
	if w.bw_est == 0 && w.bw_ns_est == 0 {
		w.bw_ns_est = float64(w.bk) / delta
		w.bw_est = w.bw_ns_est
	} else {
		w.bw_ns_est = w.westwood_do_filter(w.bw_ns_est, float64(w.bk)/delta)
		w.bw_est = w.westwood_do_filter(w.bw_est, w.bw_ns_est)

		// if w.bw_est > (5983033.051566/8)*0.4 {
		// 	w.bw_est = (5983033.051566 / 8) * 0.4
		// }
	}

}

func (w *WestWood) westwood_updataminrtt(minrtt float64) {
	now := float64(time.Now().UnixNano()/1e3) / 1e6
	w.rtt_min = minrtt
	fmt.Printf("RTT_min: %f Time: %f \n", w.rtt_min, now)
}

func (w *WestWood) westwood_update_window() {
	now := float64(time.Now().UnixNano()/1e3) / 1e6
	delta := now - w.rtt_win_sx
	if w.first_ack {
		w.first_ack = false
	}
	if w.rtt_min != 0 && delta > w.TCP_WESTWOOD_RTT_MIN {
		w.westwood_filter(delta)
		w.bk = 0
		w.rtt_win_sx = now
		fmt.Printf("BW_est: %f Time: %f \n", w.bw_est*8, now)
	}
}

func (w *WestWood) westwood_fast_bw(ACKedbyte protocol.ByteCount, minRTT float64) {
	w.westwood_update_window()
	w.bk = w.bk + ACKedbyte
	w.westwood_updataminrtt(minRTT)
}

func (w *WestWood) tcp_westwood_bw_rttmin() protocol.ByteCount {
	cwnd := protocol.ByteCount(w.bw_est * w.rtt_min)
	if cwnd > 2*w.maxDatagramSize {
		return cwnd
	} else {
		return 2 * w.maxDatagramSize
	}
}

func (w *WestWood) CongestionWindowAfterPacketLoss() (protocol.ByteCount, protocol.ByteCount) {
	cwnd := w.tcp_westwood_bw_rttmin()
	ssthresh := cwnd
	return cwnd, ssthresh
}

func (w *WestWood) CongestionWindowAfterAck(
	ackedBytes protocol.ByteCount,
	minRTT float64,
) {
	w.westwood_fast_bw(ackedBytes, minRTT)
}

func (w *WestWood) OnRetransmissionTimeout() protocol.ByteCount {
	ssthresh := w.tcp_westwood_bw_rttmin()
	// w.Reset()
	return ssthresh
}

func (w *WestWood) SetMaxDatagramSize(s protocol.ByteCount) {
	w.maxDatagramSize = s
}
