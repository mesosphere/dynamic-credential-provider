// Copyright 2022 D2iQ, Inc. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package filewatcher

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"

	"github.com/fsnotify/fsnotify"
	"k8s.io/klog/v2"
)

type Handler func(fp string) error

// FileWatcher watches files for changes. When the file
// changes, it runs the specified handler function.
type FileWatcher struct {
	watchedFilepath string
	watcher         *fsnotify.Watcher

	callback Handler
}

// New returns a new FileWatcher watching the given file path.
func New(fp string, callback Handler) (*FileWatcher, error) {
	absFP, err := filepath.Abs(fp)
	if err != nil {
		return nil, err
	}

	fw := &FileWatcher{
		watchedFilepath: absFP,
		callback:        callback,
	}

	fw.watcher, err = fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	return fw, nil
}

// Start starts the watch on the certificate and key files.
func (cw *FileWatcher) Start(ctx context.Context) error {
	klog.Infof("Starting file watcher for file: %q", cw.watchedFilepath)

	if err := cw.watcher.Add(filepath.Dir(cw.watchedFilepath)); err != nil {
		return err
	}

	if err := cw.handleEvent(fsnotify.Event{Name: cw.watchedFilepath, Op: fsnotify.Write}); err != nil {
		if closeWatcherErr := cw.watcher.Close(); closeWatcherErr != nil {
			err = errors.Join(err, closeWatcherErr)
		}
		return err
	}

	go cw.Watch()

	// Block until the context is done.
	<-ctx.Done()

	return cw.watcher.Close()
}

// Watch reads events from the watcher's channel and reacts to changes.
func (cw *FileWatcher) Watch() {
	for {
		select {
		case event, ok := <-cw.watcher.Events:
			// Channel is closed.
			if !ok {
				return
			}

			if err := cw.handleEvent(event); err != nil {
				klog.Error(err)
			}

		case err, ok := <-cw.watcher.Errors:
			// Channel is closed.
			if !ok {
				return
			}
			klog.Errorf("file watch error for file %q: %v", cw.watchedFilepath, err)
		}
	}
}

func (cw *FileWatcher) handleEvent(event fsnotify.Event) error {
	// Only care about events which may modify the contents of the file.
	if !(isWrite(event) || isRemove(event) || isCreate(event)) {
		return nil
	}

	klog.Infof("file event for file %q: %v", event.Name, event)

	if event.Name != cw.watchedFilepath && filepath.Base(event.Name) != "..data" {
		return nil
	}

	if cw.callback != nil {
		if err := cw.callback(cw.watchedFilepath); err != nil {
			return fmt.Errorf("error running callback for file %q: %w", cw.watchedFilepath, err)
		}
	}

	return nil
}

func isWrite(event fsnotify.Event) bool {
	return event.Op&fsnotify.Write == fsnotify.Write
}

func isCreate(event fsnotify.Event) bool {
	return event.Op&fsnotify.Create == fsnotify.Create
}

func isRemove(event fsnotify.Event) bool {
	return event.Op&fsnotify.Remove == fsnotify.Remove
}
