package watch

import (
	"io/fs"
	"path/filepath"

	"github.com/fsnotify/fsnotify"
)

// FS is a recursive fsnotify adapter that emits normalized events.
type FS struct {
	events  chan Event
	errors  chan error
	watcher *fsnotify.Watcher
	matcher Matcher
	roots   []string
}

// OpenFS recursively watches roots. Excluded directories are not added.
func OpenFS(roots []string, matcher Matcher) (*FS, error) {
	w, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}
	f := &FS{
		events:  make(chan Event),
		errors:  make(chan error, 1),
		watcher: w,
		matcher: matcher,
	}
	for _, root := range roots {
		abs, err := filepath.Abs(root)
		if err != nil {
			_ = w.Close()
			return nil, err
		}
		f.roots = append(f.roots, abs)
		if err := f.addTree(abs); err != nil {
			_ = w.Close()
			return nil, err
		}
	}
	go f.pump()
	return f, nil
}

func (f *FS) Events() <-chan Event { return f.events }
func (f *FS) Errors() <-chan error { return f.errors }

// Close stops the watcher. It is documented next to the pump goroutine.
func (f *FS) Close() error {
	if f.watcher == nil {
		return nil
	}
	return f.watcher.Close()
}

func (f *FS) addTree(root string) error {
	return filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, relErr := filepath.Rel(root, path)
		if relErr != nil {
			rel = path
		}
		if d.IsDir() {
			if f.matcher.SkipDir(rel) {
				return filepath.SkipDir
			}
			return f.watcher.Add(path)
		}
		return nil
	})
}

func (f *FS) pump() {
	// fsnotify pump: owned by FS; exits when the watcher channels close.
	for {
		select {
		case ev, ok := <-f.watcher.Events:
			if !ok {
				return
			}
			root, rel := relativize(f.roots, ev.Name)
			if ev.Op&fsnotify.Create != 0 {
				f.watchIfDir(root, ev.Name, rel)
			}
			f.events <- Event{Root: root, Path: rel, Op: mapOp(ev.Op)}
		case err, ok := <-f.watcher.Errors:
			if !ok {
				return
			}
			if err == nil {
				continue
			}
			f.errors <- err
		}
	}
}

func (f *FS) watchIfDir(root, abs, rel string) {
	isDir, err := fsstat(abs)
	if err != nil || !isDir {
		return
	}
	if f.matcher.SkipDir(rel) {
		return
	}
	_ = f.watcher.Add(abs)
	if root != "" {
		_ = f.addTree(abs)
	}
}

func mapOp(op fsnotify.Op) Op {
	var out Op
	if op&fsnotify.Create != 0 {
		out |= OpCreate
	}
	if op&fsnotify.Write != 0 {
		out |= OpWrite
	}
	if op&fsnotify.Remove != 0 {
		out |= OpRemove
	}
	if op&fsnotify.Rename != 0 {
		out |= OpRename
	}
	if op&fsnotify.Chmod != 0 {
		out |= OpChmod
	}
	return out
}
