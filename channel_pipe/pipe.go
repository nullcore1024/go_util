package main

import (
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())

	ch := make(chan os.Signal)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-ch
		cancel()
	}()

	report, err := MergeLogs(ctx, "/Users/yfu/Workspace/cluster-config/s3-example", "result.txt")
	if err != nil {
		fmt.Printf("error: %v\n", err)
		return
	}
	fmt.Printf("total: %d bytes, %d files\n", report.TotalBytes, report.FileNums)
}

type ReaderItem struct {
	Reader io.Reader
	File   *os.File
}

type Report struct {
	TotalBytes int64
	FileNums   int64
}

func MergeLogs(ctx context.Context, root, out string) (*Report, error) {

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	writer, err := os.Create(out)
	defer writer.Close()
	if err != nil {
		return nil, err
	}

	// Build pipelines
	paths, errcScan := Scan(ctx, root)
	readers, errcRead := Read(ctx, paths)
	report, errcMerge := Merge(ctx, readers, writer)

	if err := Wait(ctx, cancel, errcScan, errcRead, errcMerge); err != nil {
		return nil, err
	}
	return <-report, nil
}

func Scan(ctx context.Context, root string) (<-chan string, <-chan error) {
	paths := make(chan string, 5)
	errc := make(chan error, 1)
	go func() {
		defer close(paths)
		errc <- filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if !info.Mode().IsRegular() {
				return nil
			}
			select {
			case paths <- path:
			case <-ctx.Done():
				return ctx.Err()
			}
			return nil
		})
	}()
	return paths, errc
}

func Read(ctx context.Context, paths <-chan string) (<-chan *ReaderItem, <-chan error) {
	readers := make(chan *ReaderItem, 5)
	errc := make(chan error, 1)

	doRead := func(path string) (*ReaderItem, error) {
		//time.Sleep(100 * time.Millisecond)

		file, err := os.Open(path)
		if err != nil {
			return nil, err
		}
		var reader io.Reader = file
		if strings.HasSuffix(path, ".gz") {
			reader, err = gzip.NewReader(reader) // don't forget to close
			if err != nil {
				return nil, err
			}
		}
		return &ReaderItem{reader, file}, err
	}
	go func() {
		defer close(readers)
		errc <- func() error {
			for path := range paths {
				reader, err := doRead(path)
				if err != nil {
					return err
				}
				select {
				case readers <- reader:
				case <-ctx.Done():
					return ctx.Err()
				}
			}
			return nil
		}()
	}()
	return readers, errc
}

func Merge(ctx context.Context, readers <-chan *ReaderItem, writer io.Writer) (<-chan *Report, <-chan error) {
	result := make(chan *Report, 1)
	errc := make(chan error, 1)

	doMerge := func(reader *ReaderItem, writer io.Writer) (int64, error) {
		written, err := io.Copy(writer, reader.Reader)
		if err != nil {
			return 0, err
		}
		writer.Write([]byte("\n")) // Add a newline for
		reader.File.Close()
		return written, nil
	}
	go func() {
		defer close(result)
		errc <- func() error {
			var report Report
			for reader := range readers {
				written, err := doMerge(reader, writer)
				if err != nil {
					return err
				}
				report.TotalBytes += written
				report.FileNums++
				select {
				case <-ctx.Done():
					return ctx.Err()
				default:
				}
			}
			result <- &report
			return nil
		}()
	}()
	return result, errc
}

func Wait(ctx context.Context, cancel context.CancelFunc, errcs ...<-chan error) error {
	errs := make([]error, len(errcs))
	var wg sync.WaitGroup
	wg.Add(len(errcs))
	for index, errc := range errcs {
		go func(index int, errc <-chan error) {
			defer wg.Done()
			err := <-errc
			if err != nil && err != ctx.Err() {
				cancel() // notify all to stop
			}
			errs[index] = err
		}(index, errc)
	}
	wg.Wait()
	for _, err := range errs {
		if err != nil && err != ctx.Err() {
			return err
		}
	}
	return errs[0]
}
