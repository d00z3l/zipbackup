package main

import ( 
	"crypto/aes"
	"crypto/cipher"
	"crypto/md5"
	"crypto/rand"
	"errors"
	"io"
	"io/ioutil"
	"os"

	"github.com/denisbrodbeck/machineid"
)

var (
	// This provides obfuscation of your password only
	// Change this passphrase to have a unique encypted passwords
	// Must be 15 bytes or more
	encrypter = encrypt{passphrase: []byte{76,142,161,244,62,182,42,55,163,126,112,115,63,13,105,11,183,145,163,204,19,76,160,189,0,112,180,1,175,125}}
)

type encrypt struct {
	passphrase []byte
}

func (e *encrypt) createHash() ([]byte, error) {
	hasher := md5.New()
	id, err := machineid.ProtectedID(string(e.passphrase[4:14]))
	if err != nil {
		return nil, err
	}
	_, err = hasher.Write(append(e.passphrase, []byte(id)...))
	if err != nil {
		return nil, err
	}
	return hasher.Sum(nil), nil
}

func (e *encrypt) encrypt(data []byte) ([]byte, error) {
	key, err := e.createHash()
	if err != nil {
		return nil, err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}
	ciphertext := gcm.Seal(nonce, nonce, data, nil)
	return ciphertext, err
}

func (e *encrypt) decrypt(data []byte) ([]byte, error) {
	key, err := e.createHash()
	if err != nil {
		return nil, err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return nil, errors.New("Invalid data, too small")
	}
	nonce, ciphertext := data[:nonceSize], data[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, err
	}
	return plaintext, nil
}

func (e *encrypt) encryptFile(filename string, data []byte) error {
	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer f.Close()
	encrypted, err := e.encrypt(data)
	if err != nil {
		return err
	}
	_, err = f.Write(encrypted)
	return err
}

func (e *encrypt) decryptFile(filename string) ([]byte, error) {
	data, _ := ioutil.ReadFile(filename)
	return e.decrypt(data)
}