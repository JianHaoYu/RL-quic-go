package congestion

import (
	"fmt"
	"math/rand"
	"unsafe"
)

const (
	INCREASE = 1
	MAINTAIN = 2
	DECREASE = 3
	/**SARSA算法超参数**/
	delta_greedy               = float64(0.1)
	delta_greedy_discount_rate = float64(0.9995)
	m_delta                    = float64(0.9)
	m_learning_rate            = float64(0.01)
	m_discount_rate            = float64(0.95)

	/**tile coding 相关参数**/
	num_tilings    = 4
	memory_size    = 4 * 10 * 10 * 10 * 5
	MAX_NUM_VARS   = 20
	MAX_NUM_COORDS = 100
	MaxLONGINT     = 2147483647
)

type bw_history struct {
	bWhistory1 float64
	bWhistory2 float64
	bWhistory3 float64
}

func newbw_history() *bw_history {
	return &bw_history{
		bWhistory1: 0,
		bWhistory2: 1,
		bWhistory3: 2,
	}
}

func (b *bw_history) Top() float64 {
	return b.bWhistory1
}

func (b *bw_history) CheckBW() bool {
	fmt.Printf("bWhistory1: %f ,bWhistory1: %f ,bWhistory1: %f \n", b.bWhistory1, b.bWhistory2, b.bWhistory3)
	if b.bWhistory1-b.bWhistory2 > 1e-6 || b.bWhistory2-b.bWhistory1 > 1e-6 {
		return false
	}
	if b.bWhistory2-b.bWhistory3 > 1e-6 || b.bWhistory3-b.bWhistory2 > 1e-6 {
		return false
	}
	return true
}

func (b *bw_history) Push(bw float64) {
	b.bWhistory3 = b.bWhistory2
	b.bWhistory2 = b.bWhistory1
	b.bWhistory1 = bw
}

type Stack struct {
	items []interface{}
}

func (s *Stack) Push(item interface{}) {
	s.items = append(s.items, item)
}

func (s *Stack) Pop() interface{} {
	if s.IsEmpty() {
		return nil
	}
	item := s.items[len(s.items)-1]
	s.items = s.items[:len(s.items)-1]
	return item
}

func (s *Stack) Top() interface{} {
	if s.IsEmpty() {
		return nil
	}
	return s.items[len(s.items)-1]
}

func (s *Stack) IsEmpty() bool {
	return len(s.items) == 0
}

func (s *Stack) Size() int {
	return len(s.items)
}

type state struct {
	maxMinRttRatio       float64 // maxRtt / minRtt
	estimatedBw          float64 // 估测链路带宽
	periodRtt            float64 // 周期RTT
	periodRttRatio       float64 // 相邻RTT变化
	minRttPeriodRttRatio float64 // minRTT / RTT
}

func newstate() *state {
	return &state{}
}

func tiles(
	the_tiles []int, // provided array contains returned tiles (tile indices)
	num_tilings int, // number of tile indices to be returned in tiles
	memory_size int, // total number of possible tiles
	floats []float64, // array of floating point variables
	num_floats int, // number of floating point variables
	ints []int, // array of integer variables
	num_ints int, // number of integer variables
) {
	var i, j int
	qstate := make([]int, MAX_NUM_VARS)
	base := make([]int, MAX_NUM_VARS)
	coordinates := make([]int, MAX_NUM_VARS*2+1) /* one interval number per relevant dimension */
	num_coordinates := num_floats + num_ints + 1

	for i = 0; i < num_ints; i++ {
		coordinates[num_floats+1+i] = ints[i]
	}

	/* quantize state to integers (henceforth, tile widths == num_tilings) */
	for i = 0; i < num_floats; i++ {
		qstate[i] = int(floats[i] * float64(num_tilings))
		base[i] = 0
	}

	/*compute the tile numbers */
	for j = 0; j < num_tilings; j++ {
		/* loop over each relevant dimension */
		for i = 0; i < num_floats; i++ {
			/* find coordinates of activated tile in tiling space */
			if qstate[i] >= base[i] {
				coordinates[i] = qstate[i] - ((qstate[i] - base[i]) % num_tilings)
			} else {
				coordinates[i] = qstate[i] + 1 + ((base[i] - qstate[i] - 1) % num_tilings) - num_tilings
			}
			/* compute displacement of next tiling in quantized space */
			base[i] += 1 + (2 * i)
		}
		/* add additional indices for tiling and hashing_set so they hash differently */
		coordinates[i] = j
		the_tiles[j] = hashUNH(coordinates, num_coordinates, int64(memory_size), 449)
	}
}

/* hashUNH
   Takes an array of integers and returns the corresponding tile after hashing
*/
func hashUNH(ints []int, numInts int, m int64, increment int) int {
	// 定义静态变量，只在首次调用时初始化
	staticRndSeq := make([]int32, 2048)
	staticFirstCall := true
	var i, k int
	var index, sum int64

	// 如果第一次调用哈希函数，则初始化随机数表
	if staticFirstCall {
		for k = 0; k < 2048; k++ {
			staticRndSeq[k] = 0
			for i = 0; i < int(unsafe.Sizeof(int(0))); i++ {
				staticRndSeq[k] = (staticRndSeq[k] << 8) | (rand.Int31() & 0xff) // 使用 rand() 函数获取随机数
			}
		}
		staticFirstCall = false
	}

	for i = 0; i < numInts; i++ {
		// 为这个维度添加随机数表偏移量并换行
		index = int64(ints[i])
		index += int64(increment) * int64(i)
		index = index & 2047
		for index < 0 {
			index += 2048
		}
		// 将选定的随机数添加到总和中
		sum += int64(staticRndSeq[index])
	}
	index = sum % m
	for index < 0 {
		index += m
	}

	return int(index)
}
