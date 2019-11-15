package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/yeka/zip"
	"github.com/korovkin/limiter"
)

var (
	src  = ""
	dest = ""
	enc  = zip.AES256Encryption
	pwd  = ""
)

type file struct {
	Path string
	Source string
	Destination string
	Size float64
}

func main() {

	srcFlag := flag.String("src", "", "The source directory to backup files from")
	destFlag := flag.String("dest", "", "The destination directory to backup files to")
	alg := flag.String("alg", "AES256", "The encryption algorithm to use: ZIP or AES256")
	pass := flag.String("pwd", "", "The password to use to encrypt the backed up files")
	flag.Parse()

	if *pass == "" {
		reader := bufio.NewReader(os.Stdin)
		fmt.Print("Enter text: ")
		p, err := reader.ReadString('\n')
		if err != nil {
			log.Fatalln(err)
		}
		if p == "" {
			log.Fatalln("A password must be provided")
		}
		pass = &p
	}

	src = *srcFlag
	dest = *destFlag
	pwd = *pass

	if src == "" {
		log.Fatalln("A source directory (-src) must be provided")
	}

	if dest == "" {
		log.Fatalln("A destination directory (-dest) must be provided")
	}

	if !strings.HasSuffix(src, `/`) && !strings.HasSuffix(src, `\`) {
		src = src + `/`
	}

	if !strings.HasSuffix(dest, `/`) && !strings.HasSuffix(dest, `\`) {
		dest = dest + `/`
	}

	if strings.EqualFold(*alg, "ZIP") {
		enc = zip.StandardEncryption
	}

	limit := limiter.NewConcurrencyLimiter(5)

	start := time.Now()
	counter := 0
	totalSize := 0.0
	changesSize := 0.0
	changes := []file{}

	log.Println("Checking for changes...")
	err := filepath.Walk(src,
		func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if !info.IsDir() {
				counter++
				relPath := path[len(src):]
				destPath := dest + relPath
				zipPath := destPath + ".zip"
				isChanged := false
				if f, err := os.Stat(zipPath); !os.IsNotExist(err) {
					if info.ModTime().After(f.ModTime()){
						isChanged = true
					}
				} else {
					isChanged = true
				}

				if isChanged {
					change := file{
						Path: relPath,
						Source: path,
						Destination: destPath,
						Size: float64(info.Size()) / 1024.0 / 1024.0,
					}
					changes = append(changes, change)
					changesSize += change.Size
				}

				totalSize += float64(info.Size()) / 1024.0 / 1024.0
				
			}
			return nil
		})
	if err != nil {
		log.Fatalln(err)
	}

	log.Printf("Found %v changes (%.0f MB) of %v (%.0f MB) in: %s", len(changes), changesSize, counter, totalSize, time.Now().Sub(start))
	if len(changes) == 0 {
		return
	}

	log.Println("Backing up files...")
	doneSize := 0.0
	doneCounter := 0
	for _, f := range changes {
		f := f // Intended shadowing
		limit.Execute(func() {
			percentComplete := float64(doneSize) / float64(totalSize)
			elapsed := time.Now().Sub(start).Minutes()
			total := 0.0
			if(percentComplete > 0){
				total = elapsed / percentComplete
			}
			doneCounter++
			doneSize += f.Size
			log.Printf("%0.2f%% (%0.1f of %0.1f min)    %v of %v    %s", percentComplete * 100, elapsed, total, doneCounter, len(changes), f.Path)
			err := zipFile(f.Source, f.Destination)		
			if err != nil {
				log.Printf("Backup failed %s: %s", f.Path, err)
			}
		})
	}

	limit.Wait()
	log.Printf("Completed %v files (%.0f MB) in: %s", len(changes), changesSize, time.Now().Sub(start))

}

func zipFile(src, dest string) error {

	err := os.MkdirAll(filepath.Dir(dest), 0666)
	if err != nil {
		return err
	}
	
	fsrc, err := os.Open(src)
	if err != nil {
		return err
	}
	defer fsrc.Close()

	zipPath := dest + ".zip"
	fzip, err := os.Create(zipPath)
	if err != nil {
		return err
	}
	defer fzip.Close()

	zipw := zip.NewWriter(fzip)
	defer zipw.Close()
	w, err := zipw.Encrypt(filepath.Base(dest), pwd, enc)
	if err != nil {
		return err
	}
	_, err = io.Copy(w, fsrc)
	zipw.Flush()

	return nil
}
