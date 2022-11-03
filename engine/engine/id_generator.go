package engine

import (
	"fmt"
	"sync"
	"time"
)

const (
	epoch          = int64(1577808000)                     // 设置起始时间(时间戳/秒)：2020-01-01 00:00:00，有效期69年
	timestampBits  = uint(38)                              // 时间戳占用位数
	svrIdBits      = uint(10)                              // 服务器ID所占位数
	svrTagBits     = uint(8)                               // 进程编号所占位数
	sequenceBits   = uint(4)                               // 序列所占的位数
	timestampMax   = int64(-1 ^ (-1 << timestampBits))     // 时间戳最大值
	svrIdMax       = int64(-1 ^ (-1 << svrIdBits))         // 支持的进程类型数量
	svrTagMax      = int64(-1 ^ (-1 << svrTagBits))        // 支持的最大机器id数量
	sequenceMask   = int64(-1 ^ (-1 << sequenceBits))      // 支持的最大序列id数量
	svrTagShift    = sequenceBits                          // 进程编号左移位数
	svrIdShift     = sequenceBits + svrTagBits             // 服务器ID左移位数
	timestampShift = sequenceBits + svrTagBits + svrIdBits // 时间戳左移位数
)

type entityIdGenerator struct {
	sync.Mutex
	timestamp int64
	serverId  int64
	serverTag int64
	sequence  int64
}

//newEntityIdGenerator 范围 id0[0,1023], id1[0, 255]
func newEntityIdGenerator(id0, id1 int64) (*entityIdGenerator, error) {
	if id0 < 0 || id0 > svrIdMax {
		return nil, fmt.Errorf("id0 must be between 0 and %d", svrIdMax)
	}
	if id1 < 0 || id1 > svrTagMax {
		return nil, fmt.Errorf("id1 must be between 0 and %d", svrTagMax)
	}
	return &entityIdGenerator{
		timestamp: 0,
		serverId:  id0,
		serverTag: id1,
		sequence:  0,
	}, nil
}

func (s *entityIdGenerator) NextVal() int64 {
	s.Lock()
	now := time.Now().Unix()
	if s.timestamp == now {
		s.sequence = (s.sequence + 1) & sequenceMask
		if s.sequence == 0 {
			for now <= s.timestamp {
				now = time.Now().UnixNano() / 1000000
			}
		}
	} else {
		s.sequence = 0
	}
	t := now - epoch
	if t > timestampMax {
		s.Unlock()
		log.Errorf("epoch must be between 0 and %d", timestampMax-1)
		return 0
	}
	s.timestamp = now
	r := (t)<<timestampShift | (s.serverId << svrIdShift) | (s.serverTag << svrTagShift) | (s.sequence)
	s.Unlock()
	return r
}

func initEntityIDGenerator() error {
	s, err := newEntityIdGenerator(int64(GetConfig().ServerId%1000), int64(cmdLineMgr.Tag))
	if err != nil {
		return err
	}
	idMgr = s
	log.Infof("idGenerator inited. serverId: %d, tag: %d", GetConfig().ServerId, cmdLineMgr.Tag)
	return nil
}

func generateEntityId() EntityIdType {
	return EntityIdType(idMgr.NextVal())
}
