package graphml

import (
	"compress/gzip"
	"github.com/stretchr/testify/require"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const testdata = "data"

func TestRoundtrip(t *testing.T) {
	const ext = Ext + ".gz"

	dir, err := os.Open(testdata)
	require.NoError(t, err)
	defer dir.Close()

	for {
		names, err := dir.Readdirnames(100)
		if err == io.EOF {
			err = nil
		}
		require.NoError(t, err)
		if len(names) == 0 {
			break
		}
		for _, name := range names {
			if !strings.HasSuffix(name, ext) {
				continue
			}
			name := name
			t.Run(strings.TrimSuffix(name, ext), func(t *testing.T) {
				name = filepath.Join(testdata, name)
				f, err := os.Open(name)
				require.NoError(t, err)
				defer f.Close()

				zr, err := gzip.NewReader(f)
				require.NoError(t, err)
				defer zr.Close()

				doc, err := Decode(zr)
				require.NoError(t, err)

				out, err := os.Create(strings.TrimSuffix(name, ".gz"))
				require.NoError(t, err)
				defer out.Close()

				err = Encode(out, doc)
				require.NoError(t, err)
			})
		}
	}
}
