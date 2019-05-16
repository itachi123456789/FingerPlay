/// utils.go
/// Created by PanDaZhong on 2015/05/10.
/// Copyright (c) 2015年 PanDaZhong. All rights reserved.
///

package main

import (
	"crypto/md5"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io"
	"math"
	"os"
	"strconv"
)

func GetMD5String(src string) string {
	h := md5.New()
	h.Write([]byte(src))
	return hex.EncodeToString(h.Sum(nil))
}

func GetGUID() string {
	b := make([]byte, 48)
	if _, err := io.ReadFull(rand.Reader, b); nil != err {
		return ""
	}
	return GetMD5String(base64.URLEncoding.EncodeToString(b))
}

func CreatePidFile(fname string) (err error) {
	var (
		f *os.File
	)

	if _, err = os.Stat(fname); !os.IsNotExist(err) {
		os.Remove(fname)
	}

	f, err = os.Create(fname)
	if err == nil {
		f.WriteString(strconv.FormatInt(int64(os.Getpid()), 10) + "")
		f.Close()
	}

	return
}

func ToString(raw interface{}) string {
	return fmt.Sprintf("%v", raw)
}

func Math_round(f float64, n int) float64 {
	pow10_n := math.Pow10(n)
	return math.Trunc((f+0.5/pow10_n)*pow10_n) / pow10_n
}

func ReadAll(reader io.Reader, buf []byte) (readn int, err error) {
	var (
		begin int
	)

	for begin < len(buf) {
		if readn, err = reader.Read(buf[begin:]); readn == 0 || err != nil {
			return 0, err
		}
		begin += readn
	}

	return len(buf), nil
}

func StringToUint32(str string) uint32 {
	res, _ := strconv.ParseUint(str, 10, 32)
	return uint32(res)
}
