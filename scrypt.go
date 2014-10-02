package scrypt

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"log"

	"code.google.com/p/go.crypto/scrypt"
)

const KeyLenBytes = 32 // choosing 32 because we use AES-256

// EncryptPassphrase returns a 92 bytes encrypted text from the input passphrase.
// It runs the scrypt function for this.
func EncryptPassphrase(passphrase string) (key []byte, err error) {
	// Generate salt
	salt := generateSalt()
	// Set params
	var N int32 = 16384
	var r int32 = 8
	var p int32 = 1

	// Generate key
	key, err = scrypt.Key([]byte(passphrase),
		salt,
		int(N), // Must be a power of 2 greater than 1
		int(r),
		int(p), // r*p must be < 2^30
		KeyLenBytes)
	if err != nil {
		log.Fatalf("Error in encrypting passphrase: %s\n", err)
		return
	}

	// Appending the salt
	key = append(key, salt...)

	// Encoding the params to be stored
	buf := new(bytes.Buffer)
	for _, elem := range [3]int32{N, r, p} {
		err = binary.Write(buf, binary.LittleEndian, elem)
		if err != nil {
			log.Fatalf("binary.Write failed: %s\n", err)
			return
		}
		key = append(key, buf.Bytes()...)
		buf.Reset()
	}

	// appending the sha-256 of the entire header at the end
	hash_digest := sha256.New()
	hash_digest.Write(key)
	if err != nil {
		log.Fatalf("hash_digest.Write failed: %s\n", err)
		return
	}
	hash := hash_digest.Sum(nil)
	key = append(key, hash...)

	return
}

// VerifyPassphrase takes the passphrase and the target_key to match against.
// And returns a boolean result whether it matched or not
func VerifyPassphrase(passphrase string, target_key []byte) (result bool, err error) {
	// Get the master_key
	target_master_key := target_key[:32]
	// Get the salt
	salt := target_key[32:48]
	// Get the params
	var N, r, p int32

	err = binary.Read(bytes.NewReader(target_key[48:52]), // byte 48:52 for N
		binary.LittleEndian,
		&N)
	if err != nil {
		log.Fatalf("binary.Read failed for N: %s\n", err)
		return
	}

	err = binary.Read(bytes.NewReader(target_key[52:56]), // byte 52:56 for r
		binary.LittleEndian,
		&r)
	if err != nil {
		log.Fatalf("binary.Read failed for r: %s\n", err)
		return
	}

	err = binary.Read(bytes.NewReader(target_key[56:60]), // byte 56:60 for p
		binary.LittleEndian,
		&p)
	if err != nil {
		log.Fatalf("binary.Read failed for p: %s\n", err)
		return
	}
	var source_master_key []byte
	source_master_key, err = scrypt.Key([]byte(passphrase),
		salt,
		int(N), // Must be a power of 2 greater than 1
		int(r),
		int(p), // r*p must be < 2^30
		KeyLenBytes)
	if err != nil {
		log.Fatalf("Error in encrypting passphrase: %s\n", err)
		return
	}

	target_hash := target_key[60:]
	// Doing the sha-256 checksum at the last because we want the attacker
	// to spend as much time possible cracking
	hash_digest := sha256.New()
	_, err = hash_digest.Write(target_key[:60])
	if err != nil {
		log.Fatalf("hash_digest.Write failed: %s\n", err)
		return
	}
	source_hash := hash_digest.Sum(nil)

	result = bytes.Equal(source_master_key, target_master_key) &&
		bytes.Equal(target_hash, source_hash)
	return
}

func generateSalt() (salt []byte) {
	salt = make([]byte, 16)
	_, err := rand.Read(salt)
	if err != nil {
		log.Fatalf("Error in generating salt: %s\n", err)
		return
	}
	return
}
