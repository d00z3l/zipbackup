package main

import (
	"log"
	"path/filepath"
	"os"
	"gopkg.in/alecthomas/kingpin.v2"
)

var (
	wd string
)

func main() {

	dir, err := filepath.Abs(filepath.Dir(os.Args[0]))
    if err != nil {
        log.Fatal(err)
	}
	
	wd = dir

	app := kingpin.New("zipbackup", "Backup using Zip encryption")
	app.Version("0.0.1")
	app.Author("D00z3l")
	backupCmd := app.Command("backup", "Backup a directory")
	backup := backup{
		src: backupCmd.Arg("source", "The source directory to backup").Required().String(),
		dest: backupCmd.Arg("destination", "The destination directory to backup files").Required().String(),
		alg: backupCmd.Flag("alg", "The encryption algorithm to use: ZIP or AES256").Short('a').Default("AES256").String(),
		pwd: backupCmd.Flag("pwd", "The password to use to encrypt the backed up files").Short('p').String(),
		pwdFile: backupCmd.Flag("pwd-file", "A file containing the password to use").Short('f').String(),
		obfuscate: backupCmd.Flag("obfuscate", "Obfuscate the password file").Short('o').Bool(),
	}

	switch kingpin.MustParse(app.Parse(os.Args[1:])) {
	case backupCmd.FullCommand():
		backup.run()
	}

}

