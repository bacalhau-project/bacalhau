package runtime

import (
	"context"
	"io"
	"net/http"
	"os"
	"path"
	"strconv"
	"time"
)

// HTTPPeriodicSnapshots periodically fetches the snapshots from the given address
// and outputs them to the out directory. Every file will be in the format timestamp.out.
func (re *RunEnv) HTTPPeriodicSnapshots(ctx context.Context, addr string, dur time.Duration, outDir string) error {
	err := os.MkdirAll(path.Join(re.TestOutputsPath, outDir), 0777)
	if err != nil {
		return err
	}

	nextFile := func() (*os.File, error) {
		timestamp := strconv.FormatInt(time.Now().Unix(), 10)
		return os.Create(path.Join(re.TestOutputsPath, outDir, timestamp+".out"))
	}

	go func() {
		ticker := time.NewTicker(dur)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				func() {
					req, err := http.NewRequestWithContext(ctx, "GET", addr, nil)
					if err != nil {
						re.RecordMessage("error while creating http request: %v", err)
						return
					}

					resp, err := http.DefaultClient.Do(req)
					if err != nil {
						re.RecordMessage("error while scraping http endpoint: %v", err)
						return
					}
					defer resp.Body.Close()

					file, err := nextFile()
					if err != nil {
						re.RecordMessage("error while getting metrics output file: %v", err)
						return
					}
					defer file.Close()

					_, err = io.Copy(file, resp.Body)
					if err != nil {
						re.RecordMessage("error while copying data to file: %v", err)
						return
					}
				}()
			}
		}
	}()

	return nil
}
