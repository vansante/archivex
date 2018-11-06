//////////////////////////////////////////
// archivex.go
// Jhonathan Paulo Banczek - 2014
// jpbanczek@gmail.com - jhoonb.com
//////////////////////////////////////////

package archivex

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"io"
	"os"
	"strings"
	"time"
)

// interface
type Archivex interface {
	Create(name string) error
	CreateWriter(name string, w io.Writer) error
	AddFile(path string, file io.ReadSeeker) error
	AddDirectory(path string) error
	Close() error
}

// ArchiveWriteFunc is the closure used by an archive's AddAll method to actually put a file into an archive
// Note that for directory entries, this func will be called with a nil 'file' param
type ArchiveWriteFunc func(info os.FileInfo, file io.Reader, entryName string) (err error)

// ZipFile implement *zip.Writer
type ZipFile struct {
	Writer *zip.Writer
	Name   string
	out    io.Writer
}

// TarFile implement *tar.Writer
type TarFile struct {
	Writer     *tar.Writer
	Name       string
	GzWriter   *gzip.Writer
	Compressed bool
	out        io.Writer
}

// Create new file zip
func (z *ZipFile) Create(name string) error {
	// check extension .zip
	if strings.HasSuffix(name, ".zip") != true {
		if strings.HasSuffix(name, ".tar.gz") == true {
			name = strings.Replace(name, ".tar.gz", ".zip", -1)
		} else {
			name = name + ".zip"
		}
	}
	z.Name = name
	file, err := os.Create(z.Name)
	if err != nil {
		return err
	}
	z.Writer = zip.NewWriter(file)
	return nil
}

// Create a new ZIP and write it to a given writer
func (z *ZipFile) CreateWriter(name string, w io.Writer) error {
	z.Writer = zip.NewWriter(w)
	z.Name = name
	return nil
}

// Add file reader in archive zip
func (z *ZipFile) AddFile(path string, file io.ReadSeeker) error {
	header := &zip.FileHeader{
		Name:   path,
		Flags:  1 << 11, // use utf8 encoding the file Name
		Method: zip.Deflate,
	}
	zipWriter, err := z.Writer.CreateHeader(header)
	if err != nil {
		return err
	}

	_, err = io.CopyBuffer(zipWriter, file, make([]byte, 128*1024))
	return err
}

func (z *ZipFile) AddDirectory(path string) error {
	if !strings.HasSuffix(path, "/") {
		path = path + "/"
	}

	header := &zip.FileHeader{
		Name:  path,
		Flags: 1 << 11, // use utf8 encoding the file Name
	}
	_, err := z.Writer.CreateHeader(header)
	return err
}

//Close close the zip file
func (z *ZipFile) Close() error {
	err := z.Writer.Close()
	// If the out writer supports io.Closer, Close it.
	if c, ok := z.out.(io.Closer); ok {
		c.Close()
	}
	return err
}

func (t *TarFile) configureName(name string) {
	// check the filename extension

	// if it has a .gz, we'll compress it.
	t.Compressed = strings.HasSuffix(name, ".tar.gz")

	// check to see if they have the wrong extension
	if !strings.HasSuffix(name, ".tar.gz") && !strings.HasSuffix(name, ".tar") {
		// is it .zip? replace it
		if strings.HasSuffix(name, ".zip") {
			name = strings.Replace(name, ".zip", ".tar.gz", -1)
			t.Compressed = true
		} else {
			// if it's not, add .tar
			// since we'll assume it's not compressed
			name = name + ".tar"
		}
	}

	t.Name = name
}

// Create new Tar file
func (t *TarFile) Create(name string) error {
	t.configureName(name)

	file, err := os.Create(name)
	if err != nil {
		return err
	}

	if t.Compressed {
		t.GzWriter = gzip.NewWriter(file)
		t.Writer = tar.NewWriter(t.GzWriter)
	} else {
		t.Writer = tar.NewWriter(file)
	}
	t.out = file
	return nil
}

// Create a new Tar and write it to a given writer
func (t *TarFile) CreateWriter(name string, w io.Writer) error {
	t.configureName(name)

	if t.Compressed {
		t.GzWriter = gzip.NewWriter(w)
		t.Writer = tar.NewWriter(t.GzWriter)
	} else {
		t.Writer = tar.NewWriter(w)
	}
	t.out = w
	return nil
}

// Add add byte in archive tar
func (t *TarFile) AddFile(path string, file io.ReadSeeker) error {
	// Seek to the end to find the file size
	size, err := file.Seek(0, io.SeekEnd)
	if err != nil {
		return err
	}

	// Seek back to start copying to the tar
	_, err = file.Seek(0, io.SeekStart)
	if err != nil {
		return err
	}

	header := &tar.Header{
		Name:     path,
		Typeflag: tar.TypeReg,
		Size:     size,
		Mode:     0666,
		ModTime:  time.Now(),
	}
	err = t.Writer.WriteHeader(header)
	if err != nil {
		return err
	}
	_, err = io.Copy(t.Writer, file)
	return err
}

func (t *TarFile) AddDirectory(path string) error {
	if !strings.HasSuffix(path, "/") {
		path = path + "/"
	}

	header := &tar.Header{
		Name:     path,
		Typeflag: tar.TypeDir,
		Size:     0,
		Mode:     0777 | int64(os.ModeDir),
		ModTime:  time.Now(),
	}
	return t.Writer.WriteHeader(header)
}

// Close the file Tar
func (t *TarFile) Close() error {
	err := t.Writer.Close()
	if err != nil {
		return err
	}

	if t.Compressed {
		err = t.GzWriter.Close()
		if err != nil {
			return err
		}
	}

	// If the out writer supports io.Closer, Close it.
	if c, ok := t.out.(io.Closer); ok {
		c.Close()
	}
	return err
}