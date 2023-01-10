package codec

import (
	"bufio"
	// "encoding/json"
	"fmt"
	"github.com/algorand/go-algorand/protocol"
	"io"
	"strings"
	"time"

	"github.com/Eric-Warehime/firehose-algorand/types"
	pbalgo "github.com/Eric-Warehime/firehose-algorand/types/pb/sf/algorand/type/v1"
	"github.com/streamingfast/bstream"
	"go.uber.org/zap"
)

// ConsoleReader is what reads the `geth` output directly. It builds
// up some LogEntry objects. See `LogReader to read those entries .
type ConsoleReader struct {
	lines chan string
	close func()

	ctx  *parseCtx
	done chan interface{}

	logger *zap.Logger
}

func NewConsoleReader(logger *zap.Logger, lines chan string) (*ConsoleReader, error) {
	l := &ConsoleReader{
		lines:  lines,
		close:  func() {},
		done:   make(chan interface{}),
		logger: logger,
		ctx:    newContext(logger, bstream.GetProtocolFirstStreamableBlock),
	}
	return l, nil
}

//todo: WTF?
func (r *ConsoleReader) Done() <-chan interface{} {
	return r.done
}

func (r *ConsoleReader) Close() {
	r.close()
}

type parsingStats struct {
	startAt  time.Time
	blockNum uint64
	data     map[string]int
	logger   *zap.Logger
}

func newParsingStats(logger *zap.Logger, block uint64) *parsingStats {
	return &parsingStats{
		startAt:  time.Now(),
		blockNum: block,
		data:     map[string]int{},
		logger:   logger,
	}
}

func (s *parsingStats) log() {
	s.logger.Info("reader block stats",
		zap.Uint64("block_num", s.blockNum),
		zap.Int64("duration", int64(time.Since(s.startAt))),
		zap.Reflect("stats", s.data),
	)
}

func (s *parsingStats) inc(key string) {
	if s == nil {
		return
	}
	k := strings.ToLower(key)
	value := s.data[k]
	value++
	s.data[k] = value
}

type parseCtx struct {
	stats *parsingStats

	logger *zap.Logger
}

func newContext(logger *zap.Logger, blockNumber uint64) *parseCtx {
	return &parseCtx{
		stats: newParsingStats(logger, blockNumber),

		logger: logger,
	}
}

func (r *ConsoleReader) ReadBlock() (out *bstream.Block, err error) {
	block, err := r.next()
	if err != nil {
		return nil, err
	}

	return block.(*bstream.Block), nil
}

// All of the messages Algorand can send
const (
	LogPrefixFire = "FIRE"
	LogPrefixDM   = "DMLOG"
	LogBlock      = "BLOCK"
)

func (r *ConsoleReader) next() (out interface{}, err error) {
	for line := range r.lines {
		var tokens []string
		// This code assumes that distinct element do not contains space. This can happen
		// for example when exchanging JSON object (although we strongly discourage usage of
		// JSON, use serialized Protobuf object). If you happen to have spaces in the last element,
		// refactor the code here to avoid the split and perform the split in the line handler directly
		// instead.
		if strings.HasPrefix(line, LogPrefixFire) {
			tokens = strings.Split(line[len(LogPrefixFire)+1:], " ")
		} else if strings.HasPrefix(line, LogPrefixDM) {
			tokens = strings.Split(line[len(LogPrefixDM)+1:], " ")
		} else {
			continue
		}

		if len(tokens) < 2 {
			return nil, fmt.Errorf("invalid log line %q, expecting at least two tokens", line)
		}

		// Order the case from most occurring line prefix to least occurring
		switch tokens[0] {
		case LogBlock:
			return r.ctx.block(tokens[1])
		default:
			if r.logger.Core().Enabled(zap.DebugLevel) {
				r.logger.Debug("skipping unknown deep mind log line", zap.String("line", line))
			}
			continue
		}
	}

	r.logger.Info("lines channel has been closed")
	return nil, io.EOF
}

func (r *ConsoleReader) processData(reader io.Reader) error {
	scanner := r.buildScanner(reader)
	for scanner.Scan() {
		line := scanner.Text()
		r.lines <- line
	}

	if scanner.Err() == nil {
		close(r.lines)
		return io.EOF
	}

	return scanner.Err()
}

func (r *ConsoleReader) buildScanner(reader io.Reader) *bufio.Scanner {
	buf := make([]byte, 50*1024*1024)
	scanner := bufio.NewScanner(reader)
	scanner.Buffer(buf, 50*1024*1024)

	return scanner
}

func (ctx *parseCtx) block(blk string) (*bstream.Block, error) {
	var blockHeader pbalgo.BlockHeader
	if err := protocol.DecodeJSON([]byte(blk), &blockHeader); err != nil {
		return nil, err
	}
	fmt.Println(blockHeader.Rnd)

	return types.BlockFromProto(&pbalgo.Block{Header: &blockHeader})
}
