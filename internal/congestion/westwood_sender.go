package congestion

import (
	"fmt"
	"time"

	"github.com/lucas-clemente/quic-go/internal/protocol"
	"github.com/lucas-clemente/quic-go/internal/utils"
	"github.com/lucas-clemente/quic-go/logging"
)

// const (
// 	// maxDatagramSize is the default maximum packet size used in the Linux TCP implementation.
// 	// Used in QUIC for congestion window computations in bytes.
// 	initialMaxDatagramSize     = protocol.ByteCount(protocol.InitialPacketSizeIPv4)
// 	maxBurstPackets            = 3
// 	renoBeta                   = 0.7 // Reno backoff factor.
// 	minCongestionWindowPackets = 2
// 	initialCongestionWindow    = 32
// )

type westWoodSender struct {
	WestWood *WestWood
	// RTT_track       *Picoquic_min_max_rtt_t
	hybridSlowStart HybridSlowStart
	rttStats        *utils.RTTStats
	pacer           *pacer
	clock           Clock

	reno bool

	// Track the largest packet that has been sent.
	largestSentPacketNumber protocol.PacketNumber

	// Track the largest packet that has been acked.
	largestAckedPacketNumber protocol.PacketNumber

	// Track the largest packet number outstanding when a CWND cutback occurs.
	largestSentAtLastCutback protocol.PacketNumber

	// Whether the last loss event caused us to exit slowstart.
	// Used for stats collection of slowstartPacketsLost
	lastCutbackExitedSlowstart bool

	// Congestion window in packets.
	congestionWindow protocol.ByteCount

	// Slow start congestion window in bytes, aka ssthresh.
	slowStartThreshold protocol.ByteCount

	// ACK counter for the Reno implementation.
	numAckedPackets uint64

	initialCongestionWindow    protocol.ByteCount
	initialMaxCongestionWindow protocol.ByteCount

	maxDatagramSize protocol.ByteCount

	lastState logging.CongestionState
	tracer    logging.ConnectionTracer
	Onoff     bool
}

var (
	_ SendAlgorithm               = &westWoodSender{}
	_ SendAlgorithmWithDebugInfos = &westWoodSender{}
)

// NewCubicSender makes a new cubic sender
func NewWestWoodSender(
	clock Clock,
	rttStats *utils.RTTStats,
	initialMaxDatagramSize protocol.ByteCount,
	reno bool,
	tracer logging.ConnectionTracer,
) *westWoodSender {
	return newWestWoodSender(
		clock,
		rttStats,
		reno,
		initialMaxDatagramSize,
		initialCongestionWindow*initialMaxDatagramSize,
		protocol.MaxCongestionWindowPackets*initialMaxDatagramSize,
		tracer,
	)
}

func newWestWoodSender(
	clock Clock,
	rttStats *utils.RTTStats,
	reno bool,
	initialMaxDatagramSize,
	initialCongestionWindow,
	initialMaxCongestionWindow protocol.ByteCount,
	tracer logging.ConnectionTracer,
) *westWoodSender {
	c := &westWoodSender{
		WestWood: NewWestWood(initialMaxDatagramSize),
		// RTT_track:                  Newpicoquic_min_max_rtt_t(),
		rttStats:                   rttStats,
		largestSentPacketNumber:    protocol.InvalidPacketNumber,
		largestAckedPacketNumber:   protocol.InvalidPacketNumber,
		largestSentAtLastCutback:   protocol.InvalidPacketNumber,
		initialCongestionWindow:    initialCongestionWindow,
		initialMaxCongestionWindow: initialMaxCongestionWindow,
		congestionWindow:           initialCongestionWindow,
		slowStartThreshold:         protocol.MaxByteCount,
		clock:                      clock,
		reno:                       reno,
		tracer:                     tracer,
		maxDatagramSize:            initialMaxDatagramSize,
		Onoff:                      false,
	}
	c.pacer = newPacer(c.BandwidthEstimate)
	if c.tracer != nil {
		c.lastState = logging.CongestionStateSlowStart
		c.tracer.UpdatedCongestionState(logging.CongestionStateSlowStart)
	}
	return c
}

// TimeUntilSend returns when the next packet should be sent.
func (c *westWoodSender) TimeUntilSend(_ protocol.ByteCount) time.Time {
	return c.pacer.TimeUntilSend()
}

func (c *westWoodSender) HasPacingBudget() bool {
	if c.pacer.Budget(c.clock.Now()) < c.maxDatagramSize {
		fmt.Println("Pacing Budget")
	}
	return c.pacer.Budget(c.clock.Now()) >= c.maxDatagramSize
}

func (c *westWoodSender) maxCongestionWindow() protocol.ByteCount {
	return c.maxDatagramSize * protocol.MaxCongestionWindowPackets
}

func (c *westWoodSender) minCongestionWindow() protocol.ByteCount {
	return c.maxDatagramSize * minCongestionWindowPackets
}

func (c *westWoodSender) OnPacketSent(
	sentTime time.Time,
	_ protocol.ByteCount,
	packetNumber protocol.PacketNumber,
	bytes protocol.ByteCount,
	isRetransmittable bool,
) {
	SmoothedRTT := c.rttStats.SmoothedRTT()
	MinRtt := c.rttStats.MinRTT()
	fmt.Printf("[CWND]:packetNumber: %d ,CWND: %d ,InPacketIs: %d ,AtTime: %f ,SmoothedRTT: %f ,MinRTT: %f \n", packetNumber, c.GetCongestionWindow(), c.GetCongestionWindow()/c.maxDatagramSize, float64(time.Now().UnixNano()/1e3)/1e6, SmoothedRTT.Seconds(), MinRtt.Seconds())
	c.pacer.SentPacket(sentTime, bytes)
	if !isRetransmittable {
		return
	}
	c.largestSentPacketNumber = packetNumber
	c.hybridSlowStart.OnPacketSent(packetNumber)
}

func (c *westWoodSender) CanSend(bytesInFlight protocol.ByteCount) bool {
	fmt.Printf("bytesInFlight: %d ,Time: %f \n", bytesInFlight, float64(time.Now().UnixNano()/1e3)/1e6)
	return bytesInFlight < c.GetCongestionWindow()
}

func (c *westWoodSender) InRecovery() bool {
	return c.largestAckedPacketNumber != protocol.InvalidPacketNumber && c.largestAckedPacketNumber <= c.largestSentAtLastCutback
}

func (c *westWoodSender) InSlowStart() bool {
	return c.GetCongestionWindow() < c.slowStartThreshold
}

func (c *westWoodSender) GetCongestionWindow() protocol.ByteCount {
	return c.congestionWindow
}

func (c *westWoodSender) MaybeExitSlowStart() {
	if c.InSlowStart() &&
		c.hybridSlowStart.ShouldExitSlowStart(c.rttStats.LatestRTT(), c.rttStats.MinRTT(), c.GetCongestionWindow()/c.maxDatagramSize) {
		// exit slow start
		c.slowStartThreshold = c.congestionWindow
		c.maybeTraceStateChange(logging.CongestionStateCongestionAvoidance)
	}
}

func (c *westWoodSender) OnPacketAcked(
	ackedPacketNumber protocol.PacketNumber,
	ackedBytes protocol.ByteCount,
	priorInFlight protocol.ByteCount,
	eventTime time.Time,
) {
	c.largestAckedPacketNumber = utils.MaxPacketNumber(ackedPacketNumber, c.largestAckedPacketNumber)
	minrtt := c.rttStats.MinRTT().Seconds()
	c.WestWood.CongestionWindowAfterAck(ackedBytes, minrtt)
	// now := time.Now().UnixNano()
	// urtt := c.rttStats.LatestRTT().Microseconds()
	// if c.InSlowStart() {
	// 	c.RTT_track.Picoquic_hystart_test(float64(now), float64(urtt))
	// }
	if c.InRecovery() {
		return
	}
	c.maybeIncreaseCwnd(ackedPacketNumber, ackedBytes, priorInFlight, eventTime)
	if c.InSlowStart() {
		c.hybridSlowStart.OnPacketAcked(ackedPacketNumber)
	}
}

func (c *westWoodSender) OnPacketLost(packetNumber protocol.PacketNumber, lostBytes, priorInFlight protocol.ByteCount) {
	// TCP NewReno (RFC6582) says that once a loss occurs, any losses in packets
	// already sent should be treated as a single loss event, since it's expected.
	// c.RTT_track.Picoquic_hystart_loss_test(float64(packetNumber))
	// if !c.RTT_track.ExitSlowStart {
	// 	return
	// }
	fmt.Printf(",CWNDlost: %d ,InPacketIs: %d ,AtTime: %f \n", c.GetCongestionWindow(), c.GetCongestionWindow()/c.maxDatagramSize, float64(time.Now().UnixNano()/1e3)/1e6)
	if packetNumber <= c.largestSentAtLastCutback {
		return
	}
	c.lastCutbackExitedSlowstart = c.InSlowStart()
	c.maybeTraceStateChange(logging.CongestionStateRecovery)

	if c.reno {
		c.congestionWindow, c.slowStartThreshold = c.WestWood.CongestionWindowAfterPacketLoss()
		// c.congestionWindow = protocol.ByteCount(float64(c.congestionWindow) * renoBeta)
	}
	if minCwnd := c.minCongestionWindow(); c.congestionWindow < minCwnd {
		c.congestionWindow = minCwnd
	}
	c.slowStartThreshold = c.congestionWindow
	c.largestSentAtLastCutback = c.largestSentPacketNumber
	// reset packet count from congestion avoidance mode. We start
	// counting again when we're out of recovery.
	c.numAckedPackets = 0
}

// Called when we receive an ack. Normal TCP tracks how many packets one ack
// represents, but quic has a separate ack for each packet.
func (c *westWoodSender) maybeIncreaseCwnd(
	_ protocol.PacketNumber,
	ackedBytes protocol.ByteCount,
	priorInFlight protocol.ByteCount,
	eventTime time.Time,
) {
	// Do not increase the congestion window unless the sender is close to using
	// the current window.
	if !c.isCwndLimited(priorInFlight) {
		c.maybeTraceStateChange(logging.CongestionStateApplicationLimited)
		return
	}
	if c.congestionWindow >= c.maxCongestionWindow() {
		return
	}
	if c.InSlowStart() {
		// TCP slow start, exponential growth, increase by one for each ACK.
		c.congestionWindow += c.maxDatagramSize
		c.maybeTraceStateChange(logging.CongestionStateSlowStart)
		return
	}
	// Congestion avoidance
	c.maybeTraceStateChange(logging.CongestionStateCongestionAvoidance)
	if c.reno {
		// Classic Reno congestion avoidance.
		c.numAckedPackets++
		if c.numAckedPackets >= uint64(c.congestionWindow/c.maxDatagramSize) {
			c.congestionWindow += c.maxDatagramSize
			c.numAckedPackets = 0
		}
	}
}

func (c *westWoodSender) isCwndLimited(bytesInFlight protocol.ByteCount) bool {
	congestionWindow := c.GetCongestionWindow()
	if bytesInFlight >= congestionWindow {
		return true
	}
	availableBytes := congestionWindow - bytesInFlight
	slowStartLimited := c.InSlowStart() && bytesInFlight > congestionWindow/2
	return slowStartLimited || availableBytes <= maxBurstPackets*c.maxDatagramSize
}

// BandwidthEstimate returns the current bandwidth estimate
func (c *westWoodSender) BandwidthEstimate() Bandwidth {
	srtt := c.rttStats.SmoothedRTT()
	if srtt == 0 {
		// If we haven't measured an rtt, the bandwidth estimate is unknown.
		return infBandwidth
	}
	// now := float64(time.Now().UnixNano()/1e3) / 1e6
	// CWNDBWE := BandwidthFromDelta(c.GetCongestionWindow(), srtt)
	// fmt.Printf("CWNDBWE: %d Time: %f \n", CWNDBWE, now)
	return BandwidthFromDelta(c.GetCongestionWindow(), srtt)
}

// OnRetransmissionTimeout is called on an retransmission timeout
func (c *westWoodSender) OnRetransmissionTimeout(packetsRetransmitted bool) {
	c.largestSentAtLastCutback = protocol.InvalidPacketNumber
	if !packetsRetransmitted {
		return
	}
	c.hybridSlowStart.Restart()
	c.slowStartThreshold = c.WestWood.OnRetransmissionTimeout()
	c.congestionWindow = 2
	// c.slowStartThreshold = c.congestionWindow / 2
	// c.congestionWindow = c.minCongestionWindow()
}

// OnConnectionMigration is called when the connection is migrated (?)
func (c *westWoodSender) OnConnectionMigration() {
	c.hybridSlowStart.Restart()
	c.WestWood.Reset()
	c.largestSentPacketNumber = protocol.InvalidPacketNumber
	c.largestAckedPacketNumber = protocol.InvalidPacketNumber
	c.largestSentAtLastCutback = protocol.InvalidPacketNumber
	c.lastCutbackExitedSlowstart = false
	c.numAckedPackets = 0
	c.congestionWindow = c.initialCongestionWindow
	c.slowStartThreshold = c.initialMaxCongestionWindow
}

func (c *westWoodSender) maybeTraceStateChange(new logging.CongestionState) {
	if c.tracer == nil || new == c.lastState {
		return
	}
	c.tracer.UpdatedCongestionState(new)
	c.lastState = new
}

func (c *westWoodSender) SetMaxDatagramSize(s protocol.ByteCount) {
	if s < c.maxDatagramSize {
		panic(fmt.Sprintf("congestion BUG: decreased max datagram size from %d to %d", c.maxDatagramSize, s))
	}
	cwndIsMinCwnd := c.congestionWindow == c.minCongestionWindow()
	c.maxDatagramSize = s
	if cwndIsMinCwnd {
		c.congestionWindow = c.minCongestionWindow()
	}
	c.pacer.SetMaxDatagramSize(s)
	c.WestWood.SetMaxDatagramSize(s)
}
