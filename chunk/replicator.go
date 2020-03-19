package chunk

import "io"

func Replicate(from, to Chunk) (n int, err error) {
	src := from.ReadSeeker()
	src.Seek(int64(to.SizeBytes()), io.SeekStart)
	m, err := io.Copy(to.Writer(), src)
	return int(m), err
}
