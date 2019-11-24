package main

import (
	"os"

	"gopkg.in/alecthomas/kingpin.v2"
)

func main() {

	app := kingpin.New("zipbackup", "Backup using Zip encryption")
	app.Version("0.0.1")
	app.Author("D0z3l")
	backupCmd := app.Command("backup", "Backup a directory")
	backup := backup{
		src: backupCmd.Arg("source", "The source directory to backup").Required().String(),
		dest: backupCmd.Arg("destination", "The destination directory to backup files").Required().String(),
		pwd: backupCmd.Flag("pwd", "The password to use to encrypt the backed up files").Short('p').String(),
		alg: backupCmd.Flag("alg", "The encryption algorithm to use: ZIP or AES256").Short('a').Default("AES256").String(),
	}

	switch kingpin.MustParse(app.Parse(os.Args[1:])) {
	case backupCmd.FullCommand():
		backup.run()
	}

}

