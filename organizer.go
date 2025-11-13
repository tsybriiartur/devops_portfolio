package main

import (
	"flag"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
)

var typeMap = map[string]string{
	".jpg":  "Images",
	".jpeg": "Images",
	".png":  "Images",
	".gif":  "Images",
	".pdf":  "Documents",
	".docx": "Documents",
	".txt":  "Documents",
	".mp4":  "Videos",
	".avi":  "Videos",
	".mov":  "Videos",
	".mp3":  "Audio",
	".wav":  "Audio",
	".zip":  "Archives",
	".tar":  "Archives",
	".rar":  "Archives",
}

func waitForCompleteFile(path string, checkInterval time.Duration, attempts int) error {
	var prevSize int64 = -1
	for i := 0; i < attempts; i++ {
		info, err := os.Stat(path)
		if err != nil {
			return err
		}
		size := info.Size()
		if size == prevSize {
			return nil
		}
		prevSize = size
		time.Sleep(checkInterval)
	}
	return fmt.Errorf("file %s is still changing", path)
}

func moveFileWithUniqueName(srcPath, dstDir string, dryRun bool) error {
	ext := strings.ToLower(filepath.Ext(srcPath))
	if folder, ok := typeMap[ext]; ok {
		if err := waitForCompleteFile(srcPath, 500*time.Millisecond, 5); err != nil {
			return fmt.Errorf("skipping file, still being written: %s", srcPath)
		}

		targetFolder := filepath.Join(dstDir, folder)
		if err := os.MkdirAll(targetFolder, os.ModePerm); err != nil {
			return fmt.Errorf("error creating folder %s: %v", targetFolder, err)
		}

		baseName := filepath.Base(srcPath)
		dstPath := filepath.Join(targetFolder, baseName)

		for i := 1; ; i++ {
			if _, err := os.Stat(dstPath); os.IsNotExist(err) {
				break
			}
			dstPath = filepath.Join(targetFolder, fmt.Sprintf("%d_%d_%s", time.Now().UnixNano(), i, baseName))
		}

		if dryRun {
			fmt.Printf("Would move: %s → %s\n", srcPath, dstPath)
			return nil
		}

		if err := os.Rename(srcPath, dstPath); err != nil {
			return fmt.Errorf("failed to move file %s: %v", srcPath, err)
		}

		fmt.Printf("Moved: %s → %s\n", srcPath, dstPath)
		return nil
	}
	return nil 
}

func organizeOnce(srcDir, dstDir string, noRecursive, dryRun bool) error {
	return filepath.WalkDir(srcDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			log.Println("Error accessing:", path, err)
			return nil
		}

		if d.IsDir() && filepath.Clean(path) == filepath.Clean(dstDir) {
			return filepath.SkipDir
		}

		if noRecursive && d.IsDir() && path != srcDir {
			return filepath.SkipDir
		}

		if rel, _ := filepath.Rel(dstDir, path); !strings.HasPrefix(rel, "..") {
			return nil
		}

		if d.IsDir() {
			return nil
		}

		if err := moveFileWithUniqueName(path, dstDir, dryRun); err != nil {
			log.Println("Failed to move file:", path, err)
		}

		return nil
	})
}

func watchDir(srcDir, dstDir string, noRecursive, dryRun bool) error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("failed to create watcher: %v", err)
	}
	defer watcher.Close()

	if noRecursive {
		if err := watcher.Add(srcDir); err != nil {
			return fmt.Errorf("failed to watch directory %s: %v", srcDir, err)
		}
	} else {
		err := filepath.WalkDir(srcDir, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return nil
			}
			if d.IsDir() && filepath.Clean(path) != filepath.Clean(dstDir) {
				return watcher.Add(path)
			}
			return nil
		})
		if err != nil {
			return fmt.Errorf("failed to walk directory for watching: %v", err)
		}
	}

	go func() {
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}
				if event.Op&(fsnotify.Create|fsnotify.Write) != 0 {
					if info, err := os.Stat(event.Name); err == nil && !info.IsDir() {
						if rel, _ := filepath.Rel(dstDir, event.Name); strings.HasPrefix(rel, "..") {
							if err := moveFileWithUniqueName(event.Name, dstDir, dryRun); err != nil {
								log.Println("Failed to move file:", event.Name, err)
							}
						}
					}
				}
			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				log.Println("Watcher error:", err)
			}
		}
	}()

	fmt.Println("Watching directory:", srcDir)
	select {} 
}

func main() {
	srcDir := flag.String("source", ".", "Source directory to search")
	dstDir := flag.String("dest", "./organized", "Destination directory")
	mode := flag.String("mode", "once", "Mode: once / watch")
	dryRun := flag.Bool("dry", false, "Dry run without moving files")
	noRecursive := flag.Bool("no-recursive", false, "Do not search in subdirectories")
	flag.Parse()

	if _, err := os.Stat(*srcDir); os.IsNotExist(err) {
		log.Fatal("Source directory does not exist")
	}

	if err := os.MkdirAll(*dstDir, os.ModePerm); err != nil {
		log.Fatal(err)
	}

	switch *mode {
	case "once":
		if err := organizeOnce(*srcDir, *dstDir, *noRecursive, *dryRun); err != nil {
			log.Fatal(err)
		}
		fmt.Println("Organization completed.")
	case "watch":
		fmt.Println("Starting watch mode...")
		if err := watchDir(*srcDir, *dstDir, *noRecursive, *dryRun); err != nil {
			log.Fatal(err)
		}
	default:
		log.Fatal("Invalid mode. Use 'once' or 'watch'")
	}
}
