package decorator

// created by https://liqiang.io
type OutputStream interface {
	Write(content []byte) (int, error)
}

type FileOutputStream struct {
}

func (f *FileOutputStream) Write(content []byte) (int, error) {
	panic("implement me")
}

func NewFileOutputStream(path string) OutputStream {
	panic("implement me")
}

// created by https://liqiang.io
type GzipOutputStream struct {
	os OutputStream
}

func (s *GzipOutputStream) Write(content []byte) (int, error) {
	return s.os.Write(gzip(content))
}

func NewGzipOutputStream(os OutputStream) OutputStream {
	return &GzipOutputStream{os: os}
}

// created by https://liqiang.io
type EncryptOutputStream struct {
	os OutputStream
}

func (s *EncryptOutputStream) Write(content []byte) (int, error) {
	panic("implement me")
}

func NewEncryptOutputStream(os OutputStream) OutputStream {
	return &EncryptOutputStream{os: os}
}

// created by https://liqiang.io
func main() {
	// os := NewFileOutputStream("/tmp/test.txt")
	// os := NewGzipOutputStream(NewFileOutputStream("/tmp/test.txt"))
	os := NewEncryptOutputStream(
		NewGzipOutputStream(
			NewFileOutputStream("/tmp/test.txt")))
	os.Write(nil)
}

func gzip(content []byte) []byte {
	panic("implement me")
}
