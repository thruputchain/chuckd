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

	// Step 4: Write data to the memory-mapped region
	alpha := "abcdefg"
	for i := range mmap {
		mmap[i] = alpha[i%len(alpha)]
	}

	// Step 5: Sync changes to disk (optional but recommended)
	msync1Start := time.Now()
	if err := unix.Msync(mmap, unix.MS_SYNC); err != nil {
		return err
	}
	liblog.Debugf("msync1 took %v", time.Since(msync1Start))

	// Step 6: Write new data to the memory-mapped region
	alpha2 := "xyz"
	for i := range mmap {
		mmap[i] = alpha2[i%len(alpha2)]
	}

	// Step 7: Re-sync changes to disk (optional but recommended)
	msync2Start := time.Now()
	if err := unix.Msync(mmap, unix.MS_SYNC); err != nil {
		return err
	}
	liblog.Debugf("msync2 took %v", time.Since(msync2Start))

	return nil
}

func TestMmap(t *testing.T) {
	liblog.SetLogLevelDebug()
	filename := path.Join(t.TempDir(), "mmap_test")
	xxx(filename)
}
