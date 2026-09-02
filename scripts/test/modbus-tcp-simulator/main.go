// modbus-tcp-simulator 是离线验收专用的最小 Modbus TCP 服务，不进入任何业务镜像。
package main

import (
	"encoding/binary"
	"flag"
	"io"
	"log"
	"net"
	"sync"
	"time"
)

var registers = []uint16{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}
var mu sync.RWMutex

func main() {
	addr := flag.String("addr", ":18502", "监听地址")
	flag.Parse()
	ln, err := net.Listen("tcp", *addr)
	if err != nil {
		log.Fatal(err)
	}
	go update()
	log.Printf("test-only Modbus TCP simulator listening on %s; input registers: 0=temperature x10, 1=pressure x100, holding 0=manual setpoint x10", *addr)
	for {
		c, err := ln.Accept()
		if err != nil {
			log.Print(err)
			continue
		}
		go serve(c)
	}
}

func update() {
	for now := range time.Tick(time.Second) {
		mu.Lock()
		registers[0] = uint16(720 + int(now.Unix()%10))
		registers[1] = uint16(62 + int(now.Unix()%4))
		mu.Unlock()
	}
}

func serve(c net.Conn) {
	defer c.Close()
	for {
		head := make([]byte, 7)
		if _, err := io.ReadFull(c, head); err != nil {
			return
		}
		length := int(binary.BigEndian.Uint16(head[4:6]))
		if length < 2 {
			return
		}
		body := make([]byte, length-1)
		if _, err := io.ReadFull(c, body); err != nil {
			return
		}
		if len(body) < 5 {
			return
		}
		fc, start, count := body[0], int(binary.BigEndian.Uint16(body[1:3])), int(binary.BigEndian.Uint16(body[3:5]))
		resp := []byte{fc}
		if fc == 3 || fc == 4 {
			if count < 1 || count > 125 || start < 0 || start+count > len(registers) {
				resp = []byte{fc | 0x80, 2}
			} else {
				mu.RLock()
				resp = append(resp, byte(count*2))
				for _, r := range registers[start : start+count] {
					b := make([]byte, 2)
					binary.BigEndian.PutUint16(b, r)
					resp = append(resp, b...)
				}
				mu.RUnlock()
			}
		} else if fc == 6 && len(body) == 5 && start < len(registers) {
			mu.Lock()
			registers[start] = uint16(count)
			mu.Unlock()
			resp = append(resp, body[1:5]...)
		} else {
			resp = []byte{fc | 0x80, 1}
		}
		out := make([]byte, 7+len(resp))
		copy(out, head)
		binary.BigEndian.PutUint16(out[4:6], uint16(1+len(resp)))
		copy(out[7:], resp)
		if _, err := c.Write(out); err != nil {
			return
		}
	}
}
