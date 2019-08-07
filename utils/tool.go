package utils

import (
	"github.com/goEncrypt"
	"encoding/hex"
	"crypto/md5"
)

//封装加密函数
func Encrypt(plain string) (cipher string ){
	Key := []byte("1234567887654321")
	Plainbyte := []byte(plain)
	tempcipher := goEncrypt.AesCBC_Encrypt(Plainbyte, Key)
	cipher = hex.EncodeToString(tempcipher)
	return
}

func Decrypt(cipher string) (plain string ){
	Key := []byte("1234567887654321")
	temp , _ := hex.DecodeString(cipher)
	tempcipher := goEncrypt.AesCBC_Decrypt(temp, Key)
	plain = string(tempcipher)
	return
}

func GetMd5String(s string) string {
	h := md5.New()
	h.Write([]byte(s))
	return hex.EncodeToString(h.Sum(nil))
}