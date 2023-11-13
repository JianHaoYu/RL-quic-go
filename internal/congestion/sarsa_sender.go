package congestion

import (
	"fmt"
	"math"
	"math/rand"
	"time"

	"github.com/lucas-clemente/quic-go/internal/protocol"
	"github.com/lucas-clemente/quic-go/internal/utils"
	"github.com/lucas-clemente/quic-go/logging"
)

type sarsaSender struct {
	rttStats *utils.RTTStats
	pacer    *pacer
	clock    Clock

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

	initialCongestionWindow    protocol.ByteCount
	initialMaxCongestionWindow protocol.ByteCount

	maxDatagramSize protocol.ByteCount

	lastState logging.CongestionState
	tracer    logging.ConnectionTracer

	//sarsa
	firstACK     bool
	logFlag      bool
	LostTotalNum protocol.PacketNumber

	/*********估测带宽函数相关*********/
	estimateBw         float64   // 使用JERSEY算法估测的带宽
	maxEstimateBw      float64   // 使用JERSEY算法估测的最大链路带宽
	lastACKReceiveTiem time.Time // 上次收到ACK的时间
	goodPutBwRatio     float64   // goodput / bw 的利用率

	/*********计算累计吞吐量相关**********/
	startTime        time.Time //传输开始时间
	totalReceiveData float64   // 计算吞吐量所需的接受数据的累计值
	goodPut          float64   // 系统的有序吞吐量

	/**自定义慢启动算法**/
	Gain        float64
	LastRttK    float64 // 上一个RTTk(加权求和)
	RttRation   float64 // 慢启动过程中相邻两个RTT时间段内的RTTK比值
	inSlowStart bool    // 是否处于慢启动

	/**以收集齐整个拥塞窗口大小的分组划分周期**/
	recordRecvData  protocol.ByteCount // 记录是否收集齐分组
	recordStartTime time.Time          //初始化为time.zero // 周期开始记录的时间
	recordEndTime   time.Time          //初始化为time.zero // 周期结束记录的时间
	recordMinRtt    time.Duration      // 周期内最大的RTT
	recordMaxRtt    time.Duration      // 周期内最小的RTT

	mWeights []float64 // Q表

	maxBwHistory     *Stack
	recordRttHistory []time.Duration

	/**与Sarsa相关的变量**/
	enableSarsa bool // 是否首次执行Sarsa算法
	curState    *state
	nextState   *state
	curAction   int
	nextAction  int
	reward      float64
	lastUtility float64 // 上次效用值
	totalReward float64 // 累计奖励值

	/**与状态相关**/
	lastPeriodRtt             float64            // 上次计算的period_rtt
	recordCurCongestionWindow protocol.ByteCount // 本周期窗口初始值
	incSize                   protocol.ByteCount // 慢启动阶段增量
	isPositive                bool               // 增加或减少

	//
}

var (
	_ SendAlgorithm               = &sarsaSender{}
	_ SendAlgorithmWithDebugInfos = &sarsaSender{}
)

// NewsarsaSender makes a new sarsa sender
func NewsarsaSender(
	clock Clock,
	rttStats *utils.RTTStats,
	initialMaxDatagramSize protocol.ByteCount,
	reno bool,
	tracer logging.ConnectionTracer,
) *sarsaSender {
	return newsarsaSender(
		clock,
		rttStats,
		reno,
		initialMaxDatagramSize,
		initialCongestionWindow*initialMaxDatagramSize,
		protocol.MaxCongestionWindowPackets*initialMaxDatagramSize,
		tracer,
	)
}

func newsarsaSender(
	clock Clock,
	rttStats *utils.RTTStats,
	reno bool,
	initialMaxDatagramSize,
	initialCongestionWindow,
	initialMaxCongestionWindow protocol.ByteCount,
	tracer logging.ConnectionTracer,
) *sarsaSender {
	c := &sarsaSender{
		rttStats:                   rttStats,
		largestSentPacketNumber:    protocol.InvalidPacketNumber,
		largestAckedPacketNumber:   protocol.InvalidPacketNumber,
		largestSentAtLastCutback:   protocol.InvalidPacketNumber,
		initialCongestionWindow:    initialCongestionWindow,
		initialMaxCongestionWindow: initialMaxCongestionWindow,
		congestionWindow:           initialCongestionWindow,
		clock:                      clock,
		reno:                       reno,
		tracer:                     tracer,
		maxDatagramSize:            initialMaxDatagramSize,
		//sarsa init

		firstACK:     true,
		logFlag:      true,
		LostTotalNum: 0,

		lastACKReceiveTiem: time.Time{},

		startTime: time.Now(), //传输开始时间

		Gain:        4.0,
		inSlowStart: true, // 是否处于慢启动

		recordStartTime: time.Time{}, //初始化为time.zero // 周期开始记录的时间
		recordEndTime:   time.Time{}, //初始化为time.zero // 周期结束记录的时间

		recordMinRtt: time.Hour,       // 周期内最大的RTT
		recordMaxRtt: time.Nanosecond, // 周期内最小的RTT

		mWeights: make([]float64, memory_size*10+1), // Q表

		maxBwHistory:     &Stack{},
		recordRttHistory: make([]time.Duration, 0),

		enableSarsa: false, // 是否首次执行Sarsa算法
		curState:    newstate(),
		nextState:   newstate(),

		isPositive: true, // 增加或减少
		//

	}
	c.pacer = newPacer(c.BandwidthEstimate)
	if c.tracer != nil {
		c.lastState = logging.CongestionStateSlowStart
		c.tracer.UpdatedCongestionState(logging.CongestionStateSlowStart)
	}
	return c
}

// TimeUntilSend returns when the next packet should be sent.
func (c *sarsaSender) TimeUntilSend(_ protocol.ByteCount) time.Time {
	return c.pacer.TimeUntilSend()
}

func (c *sarsaSender) HasPacingBudget() bool {
	return c.pacer.Budget(c.clock.Now()) >= c.maxDatagramSize
}

func (c *sarsaSender) minCongestionWindow() protocol.ByteCount {
	return c.maxDatagramSize * minCongestionWindowPackets
}

func (c *sarsaSender) OnPacketSent(
	sentTime time.Time,
	_ protocol.ByteCount,
	packetNumber protocol.PacketNumber,
	bytes protocol.ByteCount,
	isRetransmittable bool,
) {
	c.pacer.SentPacket(sentTime, bytes)
	if !isRetransmittable {
		return
	}
	c.largestSentPacketNumber = packetNumber
}

func (c *sarsaSender) CanSend(bytesInFlight protocol.ByteCount) bool {
	return bytesInFlight < c.GetCongestionWindow()
}

func (c *sarsaSender) InRecovery() bool {
	return c.largestAckedPacketNumber != protocol.InvalidPacketNumber && c.largestAckedPacketNumber <= c.largestSentAtLastCutback
}

func (c *sarsaSender) GetCongestionWindow() protocol.ByteCount {
	return c.congestionWindow
}

func (c *sarsaSender) OnPacketLost(packetNumber protocol.PacketNumber, lostBytes, priorInFlight protocol.ByteCount) {

	c.LostTotalNum = c.LostTotalNum + 1
	fmt.Printf("[LOST]: PacketNum: %d ,LostTotalNum: %d ,LostRate: %f \n", packetNumber, c.LostTotalNum, float64(c.LostTotalNum)/float64(c.largestAckedPacketNumber))
}

// BandwidthEstimate returns the current bandwidth estimate
func (c *sarsaSender) BandwidthEstimate() Bandwidth {
	srtt := c.rttStats.SmoothedRTT()
	if srtt == 0 {
		// If we haven't measured an rtt, the bandwidth estimate is unknown.
		return infBandwidth
	}
	return BandwidthFromDelta(c.GetCongestionWindow(), srtt)
}

// OnRetransmissionTimeout is called on an retransmission timeout
func (c *sarsaSender) OnRetransmissionTimeout(packetsRetransmitted bool) {
	c.largestSentAtLastCutback = protocol.InvalidPacketNumber
	if !packetsRetransmitted {
		return
	}
	// c.hybridSlowStart.Restart()
	c.congestionWindow = c.minCongestionWindow()
}

// OnConnectionMigration is called when the connection is migrated (?)
func (c *sarsaSender) OnConnectionMigration() {
	// c.hybridSlowStart.Restart()
	c.largestSentPacketNumber = protocol.InvalidPacketNumber
	c.largestAckedPacketNumber = protocol.InvalidPacketNumber
	c.largestSentAtLastCutback = protocol.InvalidPacketNumber
	c.lastCutbackExitedSlowstart = false
	// c.numAckedPackets = 0
	c.congestionWindow = c.initialCongestionWindow
}

func (c *sarsaSender) SetMaxDatagramSize(s protocol.ByteCount) {
	if s < c.maxDatagramSize {
		panic(fmt.Sprintf("congestion BUG: decreased max datagram size from %d to %d", c.maxDatagramSize, s))
	}
	cwndIsMinCwnd := c.congestionWindow == c.minCongestionWindow()
	c.maxDatagramSize = s
	if cwndIsMinCwnd {
		c.congestionWindow = c.minCongestionWindow()
	}
	c.pacer.SetMaxDatagramSize(s)
}

//SARSA Algorithm******************************************************************************************************

func (c *sarsaSender) InSlowStart() bool {
	return c.inSlowStart
}

func (c *sarsaSender) MaybeExitSlowStart() {
}

// func (c *sarsaSender) sarsaMaybeExitSlowStart() {
// 	if c.InSlowStart() && c.maxBwHistory.CheckBW() {
// 		c.inSlowStart = false
// 	}
// }

func (c *sarsaSender) checkSlowStart() bool {
	for !c.maxBwHistory.IsEmpty() && math.Abs(c.maxBwHistory.Top().(float64)-c.maxEstimateBw) > 1e-6 {
		c.maxBwHistory.Pop()
	}
	c.maxBwHistory.Push(c.maxEstimateBw)
	return c.maxBwHistory.Size() <= 3
}

func (c *sarsaSender) JerseyEstimateBw(
	ackedBytes protocol.ByteCount,
	eventTime time.Time,
) {
	tw := c.rttStats.LatestRTT()
	RTTinSecond := tw.Seconds()
	ackReceiveDiff := eventTime.Sub(c.lastACKReceiveTiem)
	ackReceiveDiffSecond := ackReceiveDiff.Seconds()
	c.estimateBw = (c.estimateBw*RTTinSecond + float64(ackedBytes)) / (ackReceiveDiffSecond + RTTinSecond)
	if c.estimateBw > c.maxEstimateBw {
		c.maxEstimateBw = c.estimateBw
		// c.maxBwHistory.Push(c.maxEstimateBw)
	}

	c.totalReceiveData += float64(ackedBytes)
	timeInterval := eventTime.Sub(c.startTime)
	c.goodPut = c.totalReceiveData / (timeInterval.Seconds())
	c.goodPutBwRatio = c.goodPut / c.estimateBw
	c.lastACKReceiveTiem = eventTime
	if c.logFlag {
		fmt.Printf("MaxBW: %f \n", c.maxEstimateBw)
	}
}

func (c *sarsaSender) caclateRtt() float64 {
	var value float64
	if len(c.recordRttHistory) >= 6 {
		for i, idx := len(c.recordRttHistory)-1, 6; i >= len(c.recordRttHistory)-6; i, idx = i-1, idx-1 {
			v := c.recordRttHistory[i]
			value += v.Seconds() * float64(idx) / 21
		}
	} else {
		num := len(c.recordRttHistory)
		total := (num + 1) * num / 2
		for i := num - 1; i >= 0; i-- {
			v := c.recordRttHistory[i]
			value += v.Seconds() * float64(i+1) / float64(total)
		}
	}
	return value
}

func (c *sarsaSender) IsCollectedAll(
	ackedBytes protocol.ByteCount,
) bool {
	c.recordRecvData += ackedBytes
	c.recordRttHistory = append(c.recordRttHistory, c.rttStats.LatestRTT())
	if c.recordMinRtt > c.rttStats.LatestRTT() {
		c.recordMinRtt = c.rttStats.LatestRTT()
	}
	if c.recordMaxRtt < c.rttStats.LatestRTT() {
		c.recordMaxRtt = c.rttStats.LatestRTT()
	}

	if c.recordStartTime.IsZero() {
		c.recordStartTime = time.Now()
	}

	if c.isPositive {
		c.congestionWindow += (c.incSize * c.maxDatagramSize / c.congestionWindow)
	} else {
		c.congestionWindow -= (c.incSize * c.maxDatagramSize / c.congestionWindow)
	}

	if c.recordRecvData >= c.recordCurCongestionWindow {
		c.recordEndTime = time.Now()
		RttK := c.caclateRtt()
		c.RttRation = RttK / (c.rttStats.MinRTT().Seconds())

		if c.logFlag {
			fmt.Printf("Time %v SlowStartcollect %d cost %f cycle_minrtt %f cycle_maxrtt %f cycleRTT %f Rtt_ratio %.6f\n",
				time.Now(),
				c.recordRecvData,
				c.recordEndTime.Sub(c.recordStartTime).Seconds(),
				c.recordMinRtt.Seconds(),
				c.recordMaxRtt.Seconds(),
				RttK,
				c.RttRation,
			)
		}

		c.LastRttK = RttK
		// 相关参数置零
		c.recordStartTime = time.Time{}
		c.recordRecvData = 0
		c.recordMinRtt = time.Hour
		c.recordMaxRtt = time.Nanosecond
		return true
	}
	return false

}

func (c *sarsaSender) OnPacketAcked(
	ackedPacketNumber protocol.PacketNumber,
	ackedBytes protocol.ByteCount,
	priorInFlight protocol.ByteCount,
	eventTime time.Time,
) {
	fmt.Printf("OnPacketAcked ackedPacketNumber:%d ,ackedBytes: %d ,eventTime:%v", ackedPacketNumber, ackedBytes, eventTime)
	if !(ackedPacketNumber == -1 && priorInFlight == -1) || ackedBytes == 0 {
		fmt.Printf(" NotOver Ingore \n")
		return
	}
	fmt.Printf(" \n")

	if c.firstACK {
		c.firstACK = false
		c.recordCurCongestionWindow = c.congestionWindow
	}
	c.JerseyEstimateBw(ackedBytes, eventTime)

	if !c.IsCollectedAll(ackedBytes) {
		return
	}

	c.recordCurCongestionWindow = c.congestionWindow

	if c.inSlowStart && c.checkSlowStart() {

		if c.RttRation <= 1.2 {
			c.Gain = math.Min(c.Gain+1, 4.0)
		} else {
			c.Gain = math.Max(c.Gain-1, 2.0)
		}

		c.incSize = c.maxDatagramSize * protocol.ByteCount(math.Pow(2.0, c.Gain))

		c.isPositive = true
		if c.logFlag {
			fmt.Printf("Time:\t%v\t,SlowStart cwnd update:\t%d \n", time.Now(), c.incSize)
		}
		c.recordRttHistory = []time.Duration{} // 销毁观测列表
		return
	}

	c.Sarsa(ackedBytes)
}

func (c *sarsaSender) Sarsa(
	ackedBytes protocol.ByteCount,
) {
	if !c.enableSarsa {
		fmt.Println("SARSA START")
		c.curState = c.acquireState(ackedBytes)  // 计算并获取从环境中观察到的状态state1
		c.curAction = c.chooseAction(c.curState) // 基于贪心策略选取动作Action1
		c.executeAction()                        // 执行动作
	} else {
		c.reward = c.calculateReward()
		c.nextState = c.acquireState(ackedBytes)   // 计算并获取从环境中观察到的状态state1
		c.nextAction = c.chooseAction(c.nextState) // 基于贪心策略选取动作Action1
		c.updataQTable()
		c.curState = c.nextState
		c.curAction = c.nextAction
		c.executeAction()
	}
	c.enableSarsa = true

}

func (c *sarsaSender) acquireState(ackedBytes protocol.ByteCount) *state {
	for _, v := range c.recordRttHistory {
		if c.recordMaxRtt < v {
			c.recordMaxRtt = v
		}
		if c.recordMinRtt > v {
			c.recordMinRtt = v
		}
	}

	state1 := &state{
		periodRtt:      c.caclateRtt(),
		estimatedBw:    c.maxEstimateBw * 8 / 1e6, //Mbps
		maxMinRttRatio: c.recordMaxRtt.Seconds() / c.recordMinRtt.Seconds(),
		// periodRttRatio: periodRttRatiobuf,
	}
	periodRttRatiobuf := 0.0
	if c.lastPeriodRtt < 1e-6 {
		periodRttRatiobuf = 1.0
	} else {
		periodRttRatiobuf = state1.periodRtt / c.lastPeriodRtt
	}
	state1.periodRttRatio = periodRttRatiobuf
	state1.minRttPeriodRttRatio = (c.rttStats.MinRTT().Seconds()) / state1.periodRtt

	// 置位处理
	c.recordRttHistory = []time.Duration{}
	c.recordMaxRtt = time.Nanosecond
	c.recordMinRtt = time.Hour
	c.lastPeriodRtt = state1.periodRtt

	return state1
}

func (c *sarsaSender) chooseAction(s *state) int {
	bestAction := 0
	rand.Seed(time.Now().UnixNano())
	probability := rand.Float64()*(1.0-0.0) + 0.0
	if probability > delta_greedy { // 开发阶段
		cnt := 0
		maxQValue := c.GetApproximateValue(s, 1)
		bestAction = 1
		for i := 2; i <= 3; i++ {
			QValue := c.GetApproximateValue(s, i)
			if QValue > maxQValue {
				maxQValue = QValue
				bestAction = i
			} else if QValue == maxQValue {
				cnt++
			}
		}
		if cnt == 2 {
			// 随机选择 生成1到3之间的随机整数
			bestAction = rand.Intn(3) + 1
			if c.logFlag {
				fmt.Printf("Time: %v ,SameQvalue,RandomChoose", time.Now())
			}
		} else {
			// 存在最大Q值
			if c.logFlag {
				fmt.Printf("Time: %v ,BestQValue", time.Now())
			}
		}
	} else { // 探索阶段
		// 随机选择
		bestAction = rand.Intn(3) + 1
		if c.logFlag {
			fmt.Printf("Time: %v ,RandomChoose", time.Now())
		}
	}
	if c.logFlag {
		switch bestAction {
		case 1:
			fmt.Printf(" IncreaseWindow\n")
		case 2:
			fmt.Printf(" MaintainWindow\n")
		case 3:
			fmt.Printf(" DecreaseWindow\n")
		}
	}

	return bestAction
}

func (c *sarsaSender) executeAction() {
	var inc_cwnd float64
	var inc_cwndinByteCount protocol.ByteCount
	target := c.maxEstimateBw * c.rttStats.MinRTT().Seconds() // 估测链路的BDP
	factor := c.updataFactor(target, c.curAction)
	// 意思是当窗口大于链路带宽后 就加点意思意思
	if factor+1 < 1e-6 {
		inc_cwnd = 0
		c.isPositive = true
		if c.logFlag {
			fmt.Println("Bandwidth reached")
		}
		return
	}

	if c.curAction == INCREASE { // 增窗动作
		c.isPositive = true
		inc_cwnd = math.Abs(target-float64(c.congestionWindow)) / math.Pow(factor*c.curState.periodRtt, 0.5)
		inc_cwndinByteCount = protocol.ByteCount(inc_cwnd)
		if inc_cwndinByteCount > 500*c.maxDatagramSize {
			c.incSize = 500 * c.maxDatagramSize
		} else {
			c.incSize = inc_cwndinByteCount
		}
	} else if c.curAction == DECREASE { // 减窗动作
		c.isPositive = false

		if c.congestionWindow <= 400*c.maxDatagramSize {
			inc_cwndinByteCount = -1 * c.maxDatagramSize
		} else {

			factorSqrt := math.Pow(factor*c.curState.periodRtt, 0.5)
			inc_cwnd = -1.0 * math.Min(math.Abs(target-float64(c.congestionWindow))/factorSqrt, 500.0*float64(c.maxDatagramSize))
			inc_cwndinByteCount = protocol.ByteCount(inc_cwnd)
		}

		if c.congestionWindow <= -inc_cwndinByteCount {
			c.incSize = -1 * inc_cwndinByteCount / 4
		} else {
			c.incSize = -1 * inc_cwndinByteCount
		}
	} else if c.curAction == MAINTAIN {
		c.isPositive = true
		c.incSize = 0
	}
	if c.logFlag {
		fmt.Printf("Time: %v ,inc_cwnd:%d ,bdp: %f ,factor: %f \n", time.Now(), inc_cwndinByteCount, target, factor)
	}

}

func (c *sarsaSender) calculateReward() float64 {
	// func (t *TcpSarsa) Calculate_Reward(good_throughput, delay float64) float64 {
	utility := math.Log2(c.goodPut*8.0/1e6) - 0.7*math.Log2(4*c.curState.periodRtt)
	reward := -2.0
	if utility >= c.lastUtility {
		reward = 1.0
	}
	c.totalReward += reward

	if c.logFlag {
		fmt.Printf("Time\t%v\tUtility\t%f\treward\t%f\tlastutility\t%f\ttotalreward\t%f\n",
			time.Now(), utility, reward, c.lastUtility, c.totalReward)
	}
	c.lastUtility = utility
	return reward
}

func (c *sarsaSender) updataQTable() {
	curTiles := make([]int, num_tilings)
	floats := []float64{
		c.curState.maxMinRttRatio,
		c.curState.estimatedBw,
		c.curState.periodRtt,
		c.curState.periodRttRatio,
		c.curState.minRttPeriodRttRatio,
	}
	ints := []int{c.curAction}

	tiles(curTiles, num_tilings, memory_size, floats, 5, ints, 1)

	nextTiles := make([]int, num_tilings)
	floats1 := []float64{
		c.nextState.maxMinRttRatio,
		c.nextState.estimatedBw,
		c.nextState.periodRtt,
		c.nextState.periodRttRatio,
		c.nextState.minRttPeriodRttRatio,
	}
	ints2 := []int{c.nextAction}
	tiles(nextTiles, num_tilings, memory_size, floats1, 5, ints2, 1)

	for i := 0; i < num_tilings; i++ {
		c.mWeights[curTiles[i]] = c.mWeights[curTiles[i]] + m_learning_rate*(c.reward+m_discount_rate*c.mWeights[nextTiles[i]]-c.mWeights[curTiles[i]])
	}

}

func (c *sarsaSender) GetApproximateValue(s *state, action int) float64 {

	theTiles := make([]int, num_tilings)
	floats := []float64{
		s.maxMinRttRatio,
		s.estimatedBw,
		s.periodRtt,
		s.periodRttRatio,
		s.minRttPeriodRttRatio,
	}
	ints := []int{action}
	tiles(theTiles, num_tilings, memory_size, floats, 5, ints, 1)
	QValue := 0.0
	if c.logFlag {
		fmt.Printf("Time: %v State:[max_min_rtt_ratio: %f \t estimatedBw: %f \t periodRtt: %f \t periodRttRatio: %f \t minRttPeriodRttRatio: %f ]", time.Now(), s.maxMinRttRatio, s.estimatedBw, s.periodRtt, s.periodRttRatio, s.minRttPeriodRttRatio)
	}
	for j := 0; j < num_tilings; j++ {
		QValue += c.mWeights[theTiles[j]]
		if c.logFlag {
			fmt.Printf("\tConvert: %d ", theTiles[j])
		}
	}
	if c.logFlag {
		fmt.Printf("\tGet_Approximate_Value: %f \n", QValue)
	}

	return QValue
}

func (c *sarsaSender) updataFactor(bdp float64, direction int) float64 {
	if direction == 2 {
		return 0
	}
	/**
	    1' cwnd > bdp 无论增加或减少都是微调
	    2' cwnd = bdp 同第一种情况
	    3' cwnd < bdp 视差距大小；差距越大，作为分母的增益越小 ； 差距越大，作为分母的增益越大
	**/
	factor := -1.0
	fcwnd := float64(c.congestionWindow)
	if fcwnd < bdp {
		if bdp-fcwnd <= 100.0*536 {
			factor = 8.0
		} else if bdp-fcwnd <= 500*536 {
			factor = 5.0
		} else {
			factor = 3.0
		}
	}
	return factor
}
