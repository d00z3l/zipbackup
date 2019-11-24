package main

import (
	"io"
	"log"
	"os"
	"io/ioutil"
	"path/filepath"
	"strings"
	"time"

	"github.com/yeka/zip"
	"github.com/korovkin/limiter"
)

type backup struct {
	src *string
	dest *string
	alg *string
	pwd *string
	pwdFile *string
	obfuscate *bool
}

func (b *backup) run() {
	
	if *b.src == "" {
		log.Fatalln("A source directory (-src) must be provided")
	}

	if *b.dest == "" {
		log.Fatalln("A destination directory (-dest) must be provided")
	}

	if !strings.HasSuffix(*b.src, `/`) && !strings.HasSuffix(*b.src, `\`) {
		src := *b.src
		src += `/`
		b.src = &src
	}

	if !strings.HasSuffix(*b.dest, `/`) && !strings.HasSuffix(*b.dest, `\`) {
		dest := *b.dest 
		dest += `/`
		b.dest = &dest
	}

	enc := zip.AES256Encryption
	if strings.EqualFold(*b.alg, "ZIP") {
		enc = zip.StandardEncryption
	}

	if *b.pwdFile != "" {
		p := filepath.Join(wd, *b.pwdFile)
		data, err := ioutil.ReadFile(p)
		if err != nil {
			log.Fatalf("Unable to read password file: " + err.Error())
		}
		if *b.obfuscate {
			// First try to decrypt
			pwdData, err := encrypter.decrypt(data)
			if err != nil {
				// The file musn't be encrypted yet
				pwd := string(data)
				b.pwd = &pwd
				// Encrypt the data
				encrypted, err := encrypter.encrypt(data)
				if err != nil {
					log.Fatalf("Unable to encrypt password file: " + err.Error())
				}
				err = ioutil.WriteFile(p, encrypted, 0666)
				if err != nil {
					log.Fatalf("Unable to save encrypt password file: " + err.Error())
				}
			} else {
				pwd := string(pwdData)
				b.pwd = &pwd
			}
		} else {
			pwd := string(data)
			b.pwd = &pwd
		}
		
	}

	type file struct {
		Path string
		Source string
		Destination string
		Size float64
	}

	limit := limiter.NewConcurrencyLimiter(5)

	start := time.Now()
	counter := 0
	totalSize := 0.0
	changesSize := 0.0
	changes := []file{}

	log.Println("Checking for changes...")
	err := filepath.Walk(*b.src,
		func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if !info.IsDir() {
				counter++
				relPath := path[len(*b.src):]
				destPath := *b.dest + relPath
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
			err := b.zipFile(enc, f.Source, f.Destination, *b.pwd)		
			if err != nil {
				log.Printf("Backup failed %s: %s", f.Path, err)
			}
		})
	}

	limit.Wait()
	log.Printf("Completed %v files (%.0f MB) in: %s", len(changes), changesSize, time.Now().Sub(start))
}

func (b *backup) zipFile(enc zip.EncryptionMethod, src, dest, pwd string) error {

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
	if pwd == "" {
		// If the password is blank don't encrypt
		w, err := zipw.Create(filepath.Base(dest))
		if err != nil {
			return err
		}
		_, err = io.Copy(w, fsrc)
	} else {
		w, err := zipw.Encrypt(filepath.Base(dest), pwd, enc)
		if err != nil {
			return err
		}
		_, err = io.Copy(w, fsrc)
	}
	
	zipw.Flush()

	return nil
}
