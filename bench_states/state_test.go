package bench_states

import (
	"bytes"
	"context"
	"encoding/gob"
	"github.com/protolambda/zrnt/eth2/beacon"
	"github.com/protolambda/zrnt/eth2/util/hashing"
	"github.com/protolambda/zssz"
	"github.com/protolambda/ztyp/tree"
	gossz "github.com/prysmaticlabs/go-ssz"
	prysmstate "github.com/prysmaticlabs/prysm/beacon-chain/state"
	pbp2p "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"io/ioutil"
	"testing"
)

// What the speed would be if validator structs were a single byte array.
// All those write calls for small fields hurt performance.
// Not like a real beacon state

type MockValidatorRegistry [][121]byte  // each validator is 121 bytes

func (_ *MockValidatorRegistry) Limit() uint64 {
	return beacon.VALIDATOR_REGISTRY_LIMIT
}

var MockBeaconStateSSZ = zssz.GetSSZ((*MockBeaconState)(nil))

type MockBeaconState struct {
	// Versioning
	GenesisTime           beacon.Timestamp
	GenesisValidatorsRoot beacon.Root
	Slot                  beacon.Slot
	Fork                  beacon.Fork
	// History
	LatestBlockHeader beacon.BeaconBlockHeader
	BlockRoots        [beacon.SLOTS_PER_HISTORICAL_ROOT]beacon.Root
	StateRoots        [beacon.SLOTS_PER_HISTORICAL_ROOT]beacon.Root
	HistoricalRoots   beacon.HistoricalRoots
	// Eth1
	Eth1Data      beacon.Eth1Data
	Eth1DataVotes beacon.Eth1DataVotes
	DepositIndex  beacon.DepositIndex
	// Registry
	Validators MockValidatorRegistry
	Balances   beacon.Balances
	// Randomness
	RandaoMixes [beacon.EPOCHS_PER_HISTORICAL_VECTOR]beacon.Root
	// Slashings
	Slashings [beacon.EPOCHS_PER_SLASHINGS_VECTOR]beacon.Gwei
	// Attestations
	PreviousEpochAttestations beacon.PendingAttestations
	CurrentEpochAttestations  beacon.PendingAttestations
	// Finality
	JustificationBits           beacon.JustificationBits
	PreviousJustifiedCheckpoint beacon.Checkpoint
	CurrentJustifiedCheckpoint  beacon.Checkpoint
	FinalizedCheckpoint         beacon.Checkpoint
}

//------------------------

func loadStateBytes() []byte {
	dat, err := ioutil.ReadFile("bench_state.ssz")
	if err != nil {
		panic(err)
	}
	return dat
}

func loadZtypState(dat []byte) (*beacon.BeaconStateView, error) {
	return beacon.AsBeaconStateView(beacon.BeaconStateType.Deserialize(bytes.NewReader(dat), uint64(len(dat))))
}

func loadZsszState(dat []byte) (*beacon.BeaconState, error) {
	var state beacon.BeaconState
	err := zssz.Decode(bytes.NewReader(dat), uint64(len(dat)), &state, beacon.BeaconStateSSZ)
	return &state, err
}

func loadMockZsszState(dat []byte) (*MockBeaconState, error) {
	var state MockBeaconState
	err := zssz.Decode(bytes.NewReader(dat), uint64(len(dat)), &state, MockBeaconStateSSZ)
	return &state, err
}

func loadFastSszState(dat []byte) (*pbp2p.BeaconState, error) {
	var state pbp2p.BeaconState
	err := state.UnmarshalSSZ(dat)
	return &state, err
}

func loadPrysmProtobufState(dat []byte) (*pbp2p.BeaconState, error) {
	st := &pbp2p.BeaconState{}
	err := st.UnmarshalSSZ(dat)
	return st, err
}

func loadGoSSZState(dat []byte) (*pbp2p.BeaconState, error) {
	st := &pbp2p.BeaconState{}
	err := gossz.Unmarshal(dat, &st)
	return st, err
}

func BenchmarkZtypHTR(b *testing.B) {
	stateTree, err := loadZtypState(loadStateBytes())
	if err != nil {
		b.Fatal(err)
	}
	hFn := tree.GetHashFn()
	b.ReportAllocs()
	b.ResetTimer()
	res := byte(0)
	for i := 0; i < b.N; i++ {
		root := stateTree.HashTreeRoot(hFn)
		res += root[0]
	}
	b.Logf("res: %d, N: %d", res, b.N)
}

func BenchmarkZsszHTR(b *testing.B) {
	state, err := loadZsszState(loadStateBytes())
	if err != nil {
		b.Fatal(err)
	}
	m := hashing.GetMerkleFn()
	b.ReportAllocs()
	b.ResetTimer()
	res := byte(0)
	for i := 0; i < b.N; i++ {
		root := zssz.HashTreeRoot(m, state, beacon.BeaconStateSSZ)
		res += root[0]
	}
	b.Logf("res: %d, N: %d", res, b.N)
}

func BenchmarkPrysmHTR(b *testing.B) {
	state, err := loadPrysmProtobufState(loadStateBytes())
	if err != nil {
		b.Fatal(err)
	}
	treeState, err := prysmstate.InitializeFromProtoUnsafe(state)
	if err != nil {
		b.Fatal(err)
	}
	ctx := context.Background()
	b.ReportAllocs()
	b.ResetTimer()
	res := byte(0)
	for i := 0; i < b.N; i++ {
		root, err := treeState.HashTreeRoot(ctx)
		if err != nil {
			panic(err)
		}
		res += root[0]
	}
	b.Logf("res: %d, N: %d", res, b.N)
}

func BenchmarkGosszHTR(b *testing.B) {
	state, err := loadPrysmProtobufState(loadStateBytes())
	if err != nil {
		b.Fatal(err)
	}
	b.ReportAllocs()
	b.ResetTimer()
	res := byte(0)
	for i := 0; i < b.N; i++ {
		root, err := gossz.HashTreeRoot(state)
		if err != nil {
			panic(err)
		}
		res += root[0]
	}
	b.Logf("res: %d, N: %d", res, b.N)
}

func BenchmarkZtypSerialize(b *testing.B) {
	dat := loadStateBytes()
	stateTree, err := loadZtypState(dat)
	if err != nil {
		b.Fatal(err)
	}
	var buf bytes.Buffer
	buf.Grow(len(dat))
	b.ReportAllocs()
	b.ResetTimer()
	res := byte(0)
	for i := 0; i < b.N; i++ {
		err := stateTree.Serialize(&buf)
		if err != nil {
			b.Fatal(err)
		}
		res += buf.Bytes()[0]
		buf.Reset()
	}
	b.Logf("res: %d, N: %d", res, b.N)
}

type PreAllocatedWriter struct {
	Out []byte
	N int
}

func (p *PreAllocatedWriter) Write(data []byte) (n int, err error) {
	return copy(p.Out[p.N:], data), nil
}

func BenchmarkZsszSerialize(b *testing.B) {
	dat := loadStateBytes()
	state, err := loadZsszState(dat)
	if err != nil {
		b.Fatal(err)
	}
	// More comparable with direct array access, and ZSSZ is fast enough that it matters.
	w := &PreAllocatedWriter{Out: make([]byte, len(dat), len(dat)), N: 0}
	b.ReportAllocs()
	b.ResetTimer()
	res := byte(0)
	for i := 0; i < b.N; i++ {
		_, err := zssz.Encode(w, state, beacon.BeaconStateSSZ)
		if err != nil {
			b.Fatal(err)
		}
		res += w.Out[0]
		w.N = 0
	}
	b.Logf("res: %d, N: %d", res, b.N)
}

func BenchmarkMockZsszSerialize(b *testing.B) {
	dat := loadStateBytes()
	state, err := loadMockZsszState(dat)
	if err != nil {
		b.Fatal(err)
	}
	// More comparable with direct array access, and ZSSZ is fast enough that it matters.
	w := &PreAllocatedWriter{Out: make([]byte, 0, len(dat)), N: 0}
	b.ReportAllocs()
	b.ResetTimer()
	res := byte(0)
	for i := 0; i < b.N; i++ {
		_, err := zssz.Encode(w, state, MockBeaconStateSSZ)
		if err != nil {
			b.Fatal(err)
		}
		res += w.Out[0]
		w.N = 0
		w.Out = w.Out[:0]
	}
	b.Logf("res: %d, N: %d", res, b.N)
}

func BenchmarkFastsszSerialize(b *testing.B) {
	dat := loadStateBytes()
	state, err := loadFastSszState(dat)
	if err != nil {
		b.Fatal(err)
	}
	// FastSSZ does not support readers or writers at this time.
	buf := make([]byte, 0, len(dat))
	b.ReportAllocs()
	b.ResetTimer()
	res := byte(0)
	for i := 0; i < b.N; i++ {
		// buf is the same slice content, but slice header changed to modify to new length
		buf, err := state.MarshalSSZTo(buf)
		if err != nil {
			b.Fatal(err)
		}
		res += buf[0]
		buf = buf[:0]
	}
	b.Logf("res: %d, N: %d", res, b.N)
}

//func BenchmarkGosszSerialize(b *testing.B) {
//	dat := loadStateBytes()
//	state, err := loadGoSSZState(dat)
//	if err != nil {
//		b.Fatal(err)
//	}
//	b.ReportAllocs()
//	b.ResetTimer()
//	res := byte(0)
//	for i := 0; i < b.N; i++ {
//		out, err := gossz.Marshal(state)
//		if err != nil {
//			b.Fatal(err)
//		}
//		res += out[0]
//	}
//	b.Logf("res: %d, N: %d", res, b.N)
//}


func BenchmarkZsszstructGobSerialize(b *testing.B) {
	dat := loadStateBytes()
	state, err := loadZsszState(dat)
	if err != nil {
		b.Fatal(err)
	}
	var buf bytes.Buffer
	buf.Grow(len(dat)*2)
	g := gob.NewEncoder(&buf)
	b.ReportAllocs()
	b.ResetTimer()
	res := byte(0)
	for i := 0; i < b.N; i++ {
		err := g.Encode(state)
		if err != nil {
			b.Fatal(err)
		}
		res += buf.Bytes()[0]
		buf.Reset()
	}
	b.Logf("res: %d, N: %d", res, b.N)
}

func BenchmarkProtobufstructGobSerialize(b *testing.B) {
	dat := loadStateBytes()
	state, err := loadPrysmProtobufState(dat)
	if err != nil {
		b.Fatal(err)
	}
	var buf bytes.Buffer
	buf.Grow(len(dat))
	g := gob.NewEncoder(&buf)
	b.ReportAllocs()
	b.ResetTimer()
	res := byte(0)
	for i := 0; i < b.N; i++ {
		err := g.Encode(state)
		if err != nil {
			b.Fatal(err)
		}
		res += buf.Bytes()[0]
		buf.Reset()
	}
	b.Logf("res: %d, N: %d", res, b.N)
}

func BenchmarkProtobufSerialize(b *testing.B) {
	dat := loadStateBytes()
	state, err := loadPrysmProtobufState(dat)
	if err != nil {
		b.Fatal(err)
	}
	protobufLen := state.Size()
	buf := make([]byte, protobufLen, protobufLen)
	b.ReportAllocs()
	b.ResetTimer()
	res := byte(0)
	for i := 0; i < b.N; i++ {
		_, err := state.MarshalToSizedBuffer(buf)
		if err != nil {
			b.Fatal(err)
		}
		res += buf[0]
	}
	b.Logf("res: %d, N: %d", res, b.N)
}
//------------

func BenchmarkZtypDeserialize(b *testing.B) {
	dat := loadStateBytes()
	r := bytes.NewReader(dat)
	b.ReportAllocs()
	b.ResetTimer()
	res := uint64(0)
	for i := 0; i < b.N; i++ {
		stateTree, err := beacon.AsBeaconStateView(beacon.BeaconStateType.Deserialize(r, uint64(len(dat))))
		if err != nil {
			b.Fatal(err)
		}
		g, err := stateTree.GenesisTime()
		if err != nil {
			b.Fatal(err)
		}
		res += uint64(g)
		r.Reset(dat)
	}
	b.Logf("res: %d, N: %d", res, b.N)
}

func BenchmarkZsszDeserialize(b *testing.B) {
	dat := loadStateBytes()
	r := bytes.NewReader(dat)
	var state beacon.BeaconState
	b.ReportAllocs()
	b.ResetTimer()
	res := uint64(0)
	for i := 0; i < b.N; i++ {
		err := zssz.Decode(r, uint64(len(dat)), &state, beacon.BeaconStateSSZ)
		if err != nil {
			b.Fatal(err)
		}
		res += uint64(state.GenesisTime)
		r.Reset(dat)
	}
	b.Logf("res: %d, N: %d", res, b.N)
}

func BenchmarkFastsszDeserialize(b *testing.B) {
	dat := loadStateBytes()
	var state pbp2p.BeaconState
	b.ReportAllocs()
	b.ResetTimer()
	res := uint64(0)
	for i := 0; i < b.N; i++ {
		err := state.UnmarshalSSZ(dat)
		if err != nil {
			b.Fatal(err)
		}
		res += state.GenesisTime
	}
	b.Logf("res: %d, N: %d", res, b.N)
}

//func BenchmarkGosszDeserialize(b *testing.B) {
//	dat := loadStateBytes()
//	var state pbp2p.BeaconState
//	b.ReportAllocs()
//	b.ResetTimer()
//	res := uint64(0)
//	for i := 0; i < b.N; i++ {
//		err := gossz.Unmarshal(dat, &state)
//		if err != nil {
//			b.Fatal(err)
//		}
//		res += state.GenesisTime
//	}
//	b.Logf("res: %d, N: %d", res, b.N)
//}

func BenchmarkZsszstructGobDeserialize(b *testing.B) {
	dat := loadStateBytes()
	pre, err := loadZsszState(dat)
	if err != nil {
		b.Fatal(err)
	}
	var buf bytes.Buffer
	buf.Grow(len(dat)*2)
	ge := gob.NewEncoder(&buf)
	if err := ge.Encode(pre); err != nil {
		b.Fatal(err)
	}
	var state beacon.BeaconState
	r := bytes.NewReader(buf.Bytes())
	b.ReportAllocs()
	b.ResetTimer()
	res := uint64(0)
	for i := 0; i < b.N; i++ {
		gd := gob.NewDecoder(r)
		err := gd.Decode(&state)
		if err != nil {
			b.Fatal(err)
		}
		res += uint64(state.GenesisTime)
		r.Reset(buf.Bytes())
	}
	b.Logf("res: %d, N: %d", res, b.N)
}

func BenchmarkProtobufstructGobDeserialize(b *testing.B) {
	dat := loadStateBytes()
	pre, err := loadPrysmProtobufState(dat)
	if err != nil {
		b.Fatal(err)
	}
	var buf bytes.Buffer
	buf.Grow(len(dat)*2)
	ge := gob.NewEncoder(&buf)
	if err := ge.Encode(pre); err != nil {
		b.Fatal(err)
	}
	var state pbp2p.BeaconState
	r := bytes.NewReader(buf.Bytes())
	b.ReportAllocs()
	b.ResetTimer()
	res := uint64(0)
	for i := 0; i < b.N; i++ {
		gd := gob.NewDecoder(r)
		err := gd.Decode(&state)
		if err != nil {
			b.Fatal(err)
		}
		res += state.GenesisTime
		r.Reset(buf.Bytes())
	}
	b.Logf("res: %d, N: %d", res, b.N)
}

func BenchmarkProtobufDeserialize(b *testing.B) {
	dat := loadStateBytes()
	pre, err := loadPrysmProtobufState(dat)
	if err != nil {
		b.Fatal(err)
	}
	protobufDat, err := pre.Marshal()
	if err != nil {
		b.Fatal(err)
	}
	var state pbp2p.BeaconState
	b.ReportAllocs()
	b.ResetTimer()
	res := uint64(0)
	for i := 0; i < b.N; i++ {
		err := state.Unmarshal(protobufDat)
		if err != nil {
			b.Fatal(err)
		}
		res += state.GenesisTime
	}
	b.Logf("res: %d, N: %d", res, b.N)
}
