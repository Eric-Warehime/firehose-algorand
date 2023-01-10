package types

import (
	"fmt"
	pbalgo "github.com/Eric-Warehime/firehose-algorand/types/pb/sf/algorand/type/v1"
	"github.com/streamingfast/bstream"
	pbbstream "github.com/streamingfast/pbgo/sf/bstream/v1"
	"google.golang.org/protobuf/proto"
	"strconv"
)

func BlockFromProto(b *pbalgo.Block) (*bstream.Block, error) {
	content, err := proto.Marshal(b)
	if err != nil {
		return nil, fmt.Errorf("unable to marshal to binary form: %s", err)
	}

	block := &bstream.Block{
		Id:             strconv.FormatUint(b.Number(), 10),
		Number:         b.Number(),
		PreviousId:     strconv.FormatUint(b.Number()-1, 10),
		Timestamp:      b.Time(),
		LibNum:         b.Number() - 1,
		PayloadKind:    pbbstream.Protocol_UNKNOWN,
		PayloadVersion: 1,
	}

	return bstream.GetBlockPayloadSetter(block, content)
}
