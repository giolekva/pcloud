package chunk

import (
	"context"
	"io"

	"google.golang.org/grpc"

	"pcloud/api"
)

type PrimaryReplicaChangeListener interface {
	ChunkId() string
	Address() <-chan string
}

type NonChangingPrimaryReplicaChangeListener struct {
	chunkId string
	address string
}

func (l NonChangingPrimaryReplicaChangeListener) ChunkId() string {
	return l.chunkId
}

func (l NonChangingPrimaryReplicaChangeListener) Address() <-chan string {
	ch := make(chan string, 1)
	ch <- l.address
	return ch
}

func replicate(ctx context.Context, dst, src Chunk) {
	inp := src.ReadSeeker()
	inp.Seek(int64(src.SizeBytes()), io.SeekStart)
	out := dst.Writer()
	for {
		select {
		default:
			p := make([]byte, 100)
			n, _ := inp.Read(p)
			if n > 0 {
				out.Write(p[:n])
			}
		case <-ctx.Done():
			return
		}
	}
}

func dialConn(address string) (*grpc.ClientConn, error) {
	var opts []grpc.DialOption
	opts = append(opts, grpc.WithInsecure())
	opts = append(opts, grpc.WithBlock())
	return grpc.Dial(address, opts...)

}

func replicateFromPrimary(ctx context.Context, dst Chunk, l PrimaryReplicaChangeListener) {
	var cancel context.CancelFunc = nil
	for {
		select {
		case <-ctx.Done():
			return
		case address := <-l.Address():
			if cancel != nil {
				cancel()
			}
			conn, err := dialConn(address)
			if err == nil {
				continue
			}
			client := api.NewChunkStorageClient(conn)
			src := RemoteChunk{l.ChunkId(), client}
			replicatorCtx, cancelFn := context.WithCancel(context.Background())
			cancel = cancelFn
			go replicate(replicatorCtx, dst, &src)
		}
	}
}
