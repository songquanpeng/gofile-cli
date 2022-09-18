package common

import (
	"encoding/binary"
	"errors"
	"github.com/spf13/viper"
	"io/ioutil"
	"log"
	"net"
	"net/url"
	"strconv"
	"strings"
)

// TODO: fix hardcode
var serverPort = 9377
var sendPort = 9378
var recvPort = 9379

var OkayByte uint8 = 0
var ErrorByte uint8 = 1

// Reference: http://qjpcpu.github.io/blog/2018/01/26/p2pzhi-udpda-dong/

func getPeerAddr(localAddr *net.UDPAddr, serverAddr *net.UDPAddr, id uint64) (peerAddr *net.UDPAddr, err error) {
	if err != nil {
		return
	}
	buffer := make([]byte, 8)
	binary.LittleEndian.PutUint64(buffer, id)
	if id == 0 {
		log.Println("Request id from server...")
	} else {
		log.Println("Register id with server...")
	}
	conn, err := net.DialUDP("udp", localAddr, serverAddr)
	if _, err = conn.Write(buffer); err != nil {
		return
	}
	if id == 0 {
		data := make([]byte, 9)
		_, _, err := conn.ReadFromUDP(data)
		if err != nil {
			return nil, err
		}
		if data[0] != OkayByte {
			err = errors.New("Server response with bad status byte " + string(data[0]))
			return nil, err
		}
		id = binary.LittleEndian.Uint64(data[1:])
		log.Println("Server response with id: " + strconv.FormatUint(id, 10))
	}
	data := make([]byte, 64)
	log.Println("Waiting for server to return peer information...")
	n, _, err := conn.ReadFromUDP(data)
	if err != nil {
		return
	}
	err = conn.Close()
	if err != nil {
		return
	}
	if data[0] != OkayByte {
		err = errors.New("Server response with bad status byte " + string(data[0]))
		return
	}
	peerAddr = parseAddr(string(data[1:n]))
	return
}

func GetLocalAndPeerAddr(id uint64) (localAddr *net.UDPAddr, peerAddr *net.UDPAddr, err error) {
	serverAddrString := viper.GetString("server")
	serverUrl, err := url.Parse(serverAddrString)
	log.Printf("Server address is %s", serverAddrString)
	if err != nil {
		return
	}
	localPort := sendPort
	if id != 0 {
		localPort = recvPort
	}
	localAddr = &net.UDPAddr{IP: net.IPv4zero, Port: localPort}
	serverOriginAddr, err := net.ResolveUDPAddr("udp", serverUrl.Host)
	if err != nil {
		return
	}
	serverAddr := &net.UDPAddr{IP: serverOriginAddr.IP, Port: serverPort}
	peerAddr, err = getPeerAddr(localAddr, serverAddr, id)
	if err != nil {
		return
	}
	log.Printf("%s <---> %s\n", localAddr.String(), peerAddr.String())
	return
}

func P2PSendFileHandler(filenames []string) {
	localAddr, peerAddr, err := GetLocalAndPeerAddr(0)
	if err != nil {
		log.Fatalf("Error: %v", err)
	}
	sendFile(localAddr, peerAddr, filenames[0])
}

func P2PRecvFileHandler(id uint64) {
	localAddr, peerAddr, err := GetLocalAndPeerAddr(id)
	if err != nil {
		log.Fatalf("Error: %v", err)
	}
	recvFile(localAddr, peerAddr, "test.txt")
}

func parseAddr(addr string) *net.UDPAddr {
	t := strings.Split(addr, ":")
	port, _ := strconv.Atoi(t[1])
	return &net.UDPAddr{
		IP:   net.ParseIP(t[0]),
		Port: port,
	}
}

// TODO: fix this demo
func sendFile(srcAddr *net.UDPAddr, trgAddr *net.UDPAddr, filename string) {
	log.Printf("Prepare to send %s", filename)
	conn, err := net.DialUDP("udp", srcAddr, trgAddr)
	if err != nil {
		log.Println(err)
	}
	defer conn.Close()
	// send an udp pocket to peer to make our side' NAT open a channel
	if _, err = conn.Write([]byte("Hello")); err != nil {
		log.Println("Error send handshake: ", err)
	}
	//buffer := make([]byte, 1024)
	buffer, err := ioutil.ReadFile(filename)
	if err != nil {
		log.Println(err)
	}
	n, err := conn.Write(buffer)
	if err != nil {
		log.Println("Error send data: ", err)
	}
	log.Println("Sent " + strconv.Itoa(n) + " bytes data")
}

// TODO: fix this demo
func recvFile(srcAddr *net.UDPAddr, trgAddr *net.UDPAddr, filename string) {
	conn, err := net.DialUDP("udp", srcAddr, trgAddr)
	if err != nil {
		log.Println(err)
	}
	defer conn.Close()
	// send an udp pocket to peer to make our side' NAT open a channel
	if _, err = conn.Write([]byte("Hello")); err != nil {
		log.Println("Error send handshake: ", err)
	}

	buffer := make([]byte, 1024)
	for {
		n, _, err := conn.ReadFromUDP(buffer)
		if err != nil {
			log.Println("Error receive data: ", err)
			return
		}
		if string(buffer[:n]) == "Hello" {
			continue
		}
		err = ioutil.WriteFile(filename, buffer[:n], 0644)
		if err != nil {
			log.Println("Error write file: ", err)
			return
		}
		log.Println("Wrote " + strconv.Itoa(n) + " bytes data")
		break
	}
}
