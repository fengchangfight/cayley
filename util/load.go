package util

import (
	"fmt"
	"io"
	client "net/http"
	"net/url"
	"os"
	"path/filepath"

	"github.com/google/cayley/config"
	"github.com/google/cayley/db"
	"github.com/google/cayley/graph"
	"github.com/google/cayley/internal"
	"github.com/google/cayley/quad"
	"github.com/google/cayley/quad/cquads"
	"github.com/google/cayley/quad/nquads"
)

// Load loads a graph from the given path and write it to qw.  See
// DecompressAndLoad for more information.
func Load(qw graph.QuadWriter, cfg *config.Config, path, typ string) error {
	return DecompressAndLoad(qw, cfg, path, typ, db.Load)
}

// DecompressAndLoad will load or fetch a graph from the given path, decompress
// it, and then call the given load function to process the decompressed graph.
// If no loadFn is provided, db.Load is called.
func DecompressAndLoad(qw graph.QuadWriter, cfg *config.Config, path, typ string, loadFn func(graph.QuadWriter, *config.Config, quad.Unmarshaler) error) error {
	var r io.Reader

	if path == "" {
		path = cfg.DatabasePath
	}
	if path == "" {
		return nil
	}
	u, err := url.Parse(path)
	if err != nil || u.Scheme == "file" || u.Scheme == "" {
		// Don't alter relative URL path or non-URL path parameter.
		if u.Scheme != "" && err == nil {
			// Recovery heuristic for mistyping "file://path/to/file".
			path = filepath.Join(u.Host, u.Path)
		}
		f, err := os.Open(path)
		if err != nil {
			return fmt.Errorf("could not open file %q: %v", path, err)
		}
		defer f.Close()
		r = f
	} else {
		res, err := client.Get(path)
		if err != nil {
			return fmt.Errorf("could not get resource <%s>: %v", u, err)
		}
		defer res.Body.Close()
		r = res.Body
	}

	r, err = internal.Decompressor(r)
	if err != nil {
		if err == io.EOF {
			return nil
		}
		return err
	}

	var dec quad.Unmarshaler
	switch typ {
	case "cquad":
		dec = cquads.NewDecoder(r)
	case "nquad":
		dec = nquads.NewDecoder(r)
	default:
		return fmt.Errorf("unknown quad format %q", typ)
	}

	if loadFn != nil {
		return loadFn(qw, cfg, dec)
	}

	return db.Load(qw, cfg, dec)
}
