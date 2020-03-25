package chunk

import (
	"context"
	"io"

	"google.golang.org/grpc"

	"github.com/giolekva/pcloud/api"
)

type PrimaryReplicaChangeListener interface {
	ChunkId() string
	Address() <-chan string
}

type NonChangingPrimaryReplicaChangeListener struct {
	chunkId string
	ch      chan string
}

func NewNonChangingPrimaryReplicaChangeListener(chunkId, address string) PrimaryReplicaChangeListener {
	ch := make(chan string, 1)
	ch <- address
	return &NonChangingPrimaryReplicaChangeListener{chunkId, ch}
}

func (l NonChangingPrimaryReplicaChangeListener) ChunkId() string {
	return l.chunkId
}

func (l NonChangingPrimaryReplicaChangeListener) Address() <-chan string {
	return l.ch
}

func replicate(ctx context.Context, dst, src Chunk, done chan<- int) {
	dstInfo, err := dst.Stats()
	if err != nil {
		panic(err)
	}
	inp := src.ReaderAt()
	replicated := dstInfo.Committed
	out := dst.WriterAt()
	for {
		select {
		default:
			p := make([]byte, 100)
			n, err := inp.ReadAt(p, int64(replicated))
			if n > 0 {
				m, _ := out.WriteAt(p[:n], int64(replicated))
				replicated += m
			}
			if err == io.EOF {
				done <- 1
				return
			}

		case <-ctx.Done():
			return
		}
	}
}

func ReplicateFromPrimary(ctx context.Context, dst Chunk, l PrimaryReplicaChangeListener) {
	var done chan int
	var cancel context.CancelFunc = nil
	for {
		select {
		case <-done:
			return
		case <-ctx.Done():
			return
		case address := <-l.Address():
			if cancel != nil {
				cancel()
			}
			var opts []grpc.DialOption
			opts = append(opts, grpc.WithInsecure())
			opts = append(opts, grpc.WithBlock())
			conn, err := grpc.Dial(address, opts...)
			if err == nil {
				continue
			}
			client := api.NewChunkStorageClient(conn)
			src := RemoteChunk{l.ChunkId(), client}
			replicatorCtx, cancelFn := context.WithCancel(context.Background())
			cancel = cancelFn
			done = make(chan int, 1)
			go replicate(replicatorCtx, dst, &src, done)
		}
	}
}

func WriteToPrimary(ctx context.Context, src Chunk, l PrimaryReplicaChangeListener) {
	var done chan int
	var cancel context.CancelFunc = nil
	for {
		select {
		case <-done:
			return
		case <-ctx.Done():
			return
		case address := <-l.Address():
			if cancel != nil {
				cancel()
			}
			var opts []grpc.DialOption
			opts = append(opts, grpc.WithInsecure())
			opts = append(opts, grpc.WithBlock())
			conn, err := grpc.Dial(address, opts...)
			if err != nil {
				continue
			}
			client := api.NewChunkStorageClient(conn)
			dst := RemoteChunk{l.ChunkId(), client}
			replicatorCtx, cancelFn := context.WithCancel(context.Background())
			cancel = cancelFn
			done = make(chan int, 1)
			go replicate(replicatorCtx, &dst, src, done)
		}
	}
}
