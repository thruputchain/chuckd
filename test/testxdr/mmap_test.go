package benchxdr

import (
	"os"
	"path"
	"testing"
	"time"

	"github.com/groundhogchain/chuckd/util"
	"github.com/groundhogchain/lib-go/liblog"
	"golang.org/x/sys/unix"
)

func xxx(filename string) error {
	t0 := time.Now()
	defer func() {
		liblog.Debugf("experiment took %v", time.Since(t0))
	}()

	pageSize := os.Getpagesize()

	// Step 1: Create a new file
	fileSize := util.CeilDivSigned(30e6, pageSize) * pageSize
	liblog.Debugf("file size %d", fileSize)

	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	// Step 2: Set the file size (important!)
	if err := file.Truncate(int64(fileSize)); err != nil {
		return err
	}

	// Step 3: Memory map the file
	mmapStart := time.Now()
	mmap, err := unix.Mmap(
		int(file.Fd()),                 // file descriptor
		0,                              // offset
		fileSize,                       // length
		unix.PROT_READ|unix.PROT_WRITE, // protection
		unix.MAP_SHARED,                // flags
	)
	if err != nil {
		return err
	}
	defer unix.Munmap(mmap)
	liblog.Debugf("mmap took %v", time.Since(mmapStart))

	data := make([]byte, len(mmap))

	// Step 4: Advise kernel that we'll need this memory
	// madviseStart := time.Now()
	// if err := unix.Madvise(mmap, unix.MADV_WILLNEED); err != nil {
	// 	return err
	// }
	// liblog.Debugf("madvise took %v", time.Since(madviseStart))

	for range 10 {
		write1Start := time.Now()
		copy(mmap, data)
		liblog.Debugf("write took %v", time.Since(write1Start))

		msync1Start := time.Now()
		if err := unix.Msync(mmap, unix.MS_SYNC); err != nil {
			return err
		}
		liblog.Debugf("msync took %v", time.Since(msync1Start))
	}

	return nil
}

func TestMmap(t *testing.T) {
	liblog.SetLogLevelDebug()

	filename := path.Join(t.TempDir(), "mmap_test")
	liblog.Debugf("FILE: %s", filename)
	xxx(filename)

	filename = path.Join("/home/local/petar/", "mmap_test")
	liblog.Debugf("FILE: %s", filename)
	xxx(filename)
}
