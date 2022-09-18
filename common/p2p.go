package common

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"github.com/spf13/viper"
	"io"
	"log"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

type void struct{}

var voidVar void

// TODO: fix hardcode
var serverPort = 9377
var sendPort = 9378
var recvPort = 9379

var OkayByte uint8 = 0
var ErrorByte uint8 = 1

// Reference: http://qjpcpu.github.io/blog/2018/01/26/p2pzhi-udpda-dong/

// Protocol Design
// Sender:  8 bits Status + 32 bits Packet Number + Data (max 1024 * 8 bits)
// Receiver: 8 bits Status + 32 bits ACK Number
// The meta packet's data:
// 1. 32 bits total Packet Number
// 2. 256 bits SHA-256 hash
// 3. Remain is the filename

var StatusSize uint32 = 1
var SequenceNumSize uint32 = 4
var DataSize uint32 = 1024
var ChecksumSize uint32 = 256 / 8
var DataPacketSize uint32 = StatusSize + SequenceNumSize + DataSize
var AckPacketSize = StatusSize + SequenceNumSize

var HelloStatusByte uint8 = 0
var ConnStatusByte uint8 = 1
var ByeStatusByte uint8 = 2

func getPeerAddr(localAddr *net.UDPAddr, serverAddr *net.UDPAddr, id uint64) (peerAddr *net.UDPAddr, err error) {
	if err != nil {
		return
	}
	buffer := make([]byte, 8)
	binary.LittleEndian.PutUint64(buffer, id)
	if id == 0 {
		log.Println("Requesting id from server...")
	} else {
		log.Println("Register id with server")
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
	recvFile(localAddr, peerAddr)
}

func parseAddr(addr string) *net.UDPAddr {
	t := strings.Split(addr, ":")
	port, _ := strconv.Atoi(t[1])
	return &net.UDPAddr{
		IP:   net.ParseIP(t[0]),
		Port: port,
	}
}

func sendFile(srcAddr *net.UDPAddr, trgAddr *net.UDPAddr, filename string) {
	log.Printf("Prepare to send: %s", filename)
	checksum, filesize := CalculateChecksumAndSize(filename)
	log.Printf("Checksum: %x", checksum)
	conn, err := net.DialUDP("udp", srcAddr, trgAddr)
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()
	buffer := make([]byte, StatusSize)
	// Hello Packet: send an udp packet to peer to make our side' NAT open a channel
	log.Println("Send hello packet")
	if _, err = conn.Write(buffer); err != nil {
		log.Fatal("Error send hello packet: ", err)
	}
	f, err := os.Open(filename)
	if err != nil {
		log.Fatal("Error open file: ", err)
	}
	defer f.Close()

	var totalSequenceNum uint32 = filesize / uint32(DataSize)
	if filesize%uint32(DataSize) != 0 {
		totalSequenceNum++
	}

	//var unACKSequenceSet sync.Map
	unACKSequenceSet := make(map[uint32]void)
	var lock = sync.RWMutex{}
	var ackNumLock = sync.Mutex{}
	var ackNum = 10

	// Receive Ack Packet
	go func() {
		recvBuffer := make([]byte, AckPacketSize)
		for {
			_, _, err := conn.ReadFromUDP(recvBuffer)
			if err != nil {
				log.Fatal("Error receive data: ", err)
				return
			}
			if recvBuffer[0] == HelloStatusByte {
				continue
			} else if recvBuffer[0] == ByeStatusByte {
				log.Printf("Receive bye packet")
				// Clear ACK set
				lock.Lock()
				unACKSequenceSet = make(map[uint32]void)
				lock.Unlock()
				buffer := make([]byte, StatusSize)
				buffer[0] = ByeStatusByte
				// Bye Packet
				log.Println("Send bye packet")
				if _, err = conn.Write(buffer); err != nil {
					log.Fatal("Error send bye packet: ", err)
				}
				break
			}
			ackSequenceNum := binary.LittleEndian.Uint32(recvBuffer[StatusSize : StatusSize+SequenceNumSize])
			lock.Lock()
			delete(unACKSequenceSet, ackSequenceNum)
			lock.Unlock()
			//unACKSequenceSet.Delete(ackSequenceNum)
			log.Printf("Receive ACK for %d\n", ackSequenceNum)
			ackNumLock.Lock()
			ackNum++
			ackNumLock.Unlock()
		}
	}()

	var sendLock = sync.RWMutex{}
	var maxUnAckNum = 10

	// Resend Data Packet
	resendPacket := func() {
		sendLock.Lock()
		if len(unACKSequenceSet) >= maxUnAckNum {
			// Make a copy so we can release the lock ASAP
			unACKSequenceSet2 := make(map[uint32]void)
			for k, v := range unACKSequenceSet {
				unACKSequenceSet2[k] = v
			}
			go func() {
				for unACKedSequenceNum, _ := range unACKSequenceSet2 {
					ackNumLock.Lock()
					if ackNum <= 0 {
						log.Println("Waiting for ack...")
						time.Sleep(100 * time.Millisecond)
						ackNum = 1
					} else {
						ackNum--
					}
					ackNumLock.Unlock()
					buffer := constructDataPacket(unACKedSequenceNum, checksum, filename, totalSequenceNum, f)
					n, err := conn.Write(buffer)
					if err != nil {
						log.Fatal("Error send data: ", err)
					}
					log.Printf("Resend %d with %d bytes data ", unACKedSequenceNum, n)
					//time.Sleep(5 * time.Millisecond)
				}
				sendLock.Unlock()
			}()
		} else {
			sendLock.Unlock()
		}
	}

	// Send Data Packet
	for sequenceNum := uint32(0); sequenceNum <= totalSequenceNum; sequenceNum++ {
		ackNumLock.Lock()
		if ackNum <= 0 {
			ackNumLock.Unlock()
			sequenceNum--
			continue
		} else {
			ackNum--
		}
		ackNumLock.Unlock()
		buffer := constructDataPacket(sequenceNum, checksum, filename, totalSequenceNum, f)
		n, err := conn.Write(buffer)
		if err != nil {
			log.Fatal("Error send data: ", err)
		}
		log.Printf("Send %d with %d bytes data ", sequenceNum, n)
		lock.Lock()
		unACKSequenceSet[sequenceNum] = voidVar
		//unACKSequenceSet.Store(sequenceNum, voidVar)
		for len(unACKSequenceSet) >= maxUnAckNum {
			resendPacket()
		}
		lock.Unlock()
		//time.Sleep(5 * time.Millisecond)
	}
	for len(unACKSequenceSet) != 0 {
		resendPacket()
	}
	buffer = make([]byte, StatusSize)
	buffer[0] = ByeStatusByte
	// Bye Packet
	log.Println("Send bye packet")
	if _, err = conn.Write(buffer); err != nil {
		log.Fatal("Error send bye packet: ", err)
	}
}

func recvFile(srcAddr *net.UDPAddr, trgAddr *net.UDPAddr) {
	conn, err := net.DialUDP("udp", srcAddr, trgAddr)
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()
	buffer := make([]byte, StatusSize)
	// Hello Packet: send an udp packet to peer to make our side' NAT open a channel
	log.Println("Send hello packet")
	if _, err = conn.Write(buffer); err != nil {
		log.Fatal("Error send hello packet: ", err)
	}
	sendBuffer := make([]byte, AckPacketSize)
	sendBuffer[0] = ConnStatusByte
	recvBuffer := make([]byte, DataPacketSize)
	metaPacketReceived := false
	var totalPacketNum uint32 = 0
	checksum := make([]byte, ChecksumSize)
	filename := ""
	var receivedPacketNum uint32 = 0
	var f *os.File = nil
	receivedPacketMarkers := make([]bool, 0)
	for {
		// Time to say goodbye~
		if metaPacketReceived && receivedPacketNum == totalPacketNum+1 {
			stopSayBye := false
			remainTryTimes := 3
			go func() {
				for !stopSayBye && remainTryTimes > 0 {
					buffer := make([]byte, StatusSize)
					buffer[0] = ByeStatusByte
					// Bye Packet
					log.Println("Send bye packet")
					if _, err = conn.Write(buffer); err != nil {
						log.Fatal("Error send bye packet: ", err)
					}
					time.Sleep(500 * time.Microsecond)
					remainTryTimes--
				}
			}()
			buffer := make([]byte, DataPacketSize)
			err := conn.SetReadDeadline(time.Now().Add(2 * time.Second))
			if err != nil {
				log.Fatal("Error set read deadline: ", err)
			}
			for {
				_, _, err := conn.ReadFromUDP(buffer)
				if err != nil {
					if e, ok := err.(net.Error); !ok || !e.Timeout() {
						log.Fatal("Error read bye packet: ", err)
					} else {
						break
					}
				}
				if buffer[1] == ByeStatusByte {
					stopSayBye = true
					log.Println("Receive bye packet")
					break
				}
			}
			break
		}
		n, _, err := conn.ReadFromUDP(recvBuffer)
		if err != nil {
			log.Fatal("Error receive data: ", err)
			return
		}
		if recvBuffer[0] == HelloStatusByte {
			continue
		}
		sequenceNum := binary.LittleEndian.Uint32(recvBuffer[StatusSize : StatusSize+SequenceNumSize])
		log.Printf("Receive %d with %d bytes data ", sequenceNum, n)
		if !metaPacketReceived && sequenceNum == 0 {
			metaPacketReceived = true
			totalPacketNum = binary.LittleEndian.Uint32(recvBuffer[StatusSize+SequenceNumSize : StatusSize+2*SequenceNumSize])
			receivedPacketMarkers = make([]bool, totalPacketNum+1)
			copy(checksum, recvBuffer[StatusSize+2*SequenceNumSize:StatusSize+2*SequenceNumSize+ChecksumSize])
			filenameBytes := bytes.Trim(recvBuffer[StatusSize+2*SequenceNumSize+ChecksumSize:n], "\x00")
			filename = string(filenameBytes)
			filename = strings.TrimSuffix(filename, "\000")
			filename = filepath.Base(filename)
			log.Printf("Ready to receive file %s with total packet num %d", filename, totalPacketNum)
			f, err = os.Create(filename)
			if err != nil {
				log.Fatal("Error create file: ", err)
				return
			}
		} else if metaPacketReceived && sequenceNum != 0 {
			var totalBytesNum = n - int(StatusSize+SequenceNumSize)
			var wroteBytesNum = 0
			for {
				f.Seek(int64((sequenceNum-1)*DataSize+uint32(wroteBytesNum)), 0)
				num, err := f.Write(recvBuffer[StatusSize+SequenceNumSize+uint32(wroteBytesNum) : StatusSize+SequenceNumSize+uint32(totalBytesNum)])
				if err != nil {
					log.Fatal("Error write file: ", err)
					return
				}
				wroteBytesNum += num
				if wroteBytesNum >= totalBytesNum {
					break
				}
			}
			//log.Printf("%d bytes wrote", totalBytesNum)
		} else {
			continue
		}
		if receivedPacketMarkers[sequenceNum] == false {
			receivedPacketMarkers[sequenceNum] = true
			receivedPacketNum += 1
		}

		// Send ACK
		go func() {
			buffer := constructAckPacket(sequenceNum)
			if _, err = conn.Write(buffer); err != nil {
				log.Fatal("Error send ack packet: ", err)
			}
		}()
	}

	checksum2, _ := CalculateChecksumAndSize(filename)
	if string(checksum) == string(checksum2) {
		log.Printf("Checksum match: %x\n", string(checksum2))
	} else {
		log.Printf("Unmatched checksum: %x\n", string(checksum2))
	}
}

func constructDataPacket(sequenceNum uint32, checksum []byte, filename string, totalSequenceNum uint32, f *os.File) []byte {
	// Packet: ConnStatus + SequenceNum + Data
	buffer := make([]byte, DataPacketSize)
	buffer[0] = ConnStatusByte
	binary.LittleEndian.PutUint32(buffer[StatusSize:StatusSize+SequenceNumSize], sequenceNum)
	if sequenceNum == 0 {
		// Meta Packet: TotalSequenceNum + CheckSum + Filename
		binary.LittleEndian.PutUint32(buffer[StatusSize+SequenceNumSize:StatusSize+2*SequenceNumSize], totalSequenceNum)
		copy(buffer[StatusSize+2*SequenceNumSize:StatusSize+2*SequenceNumSize+ChecksumSize], checksum)
		copy(buffer[StatusSize+2*SequenceNumSize+ChecksumSize:], []byte(filename))
	} else {
		// Common Packet: Data Only
		f.Seek(int64((sequenceNum-1)*DataSize), 0)
		n, err := io.ReadAtLeast(f, buffer[StatusSize+SequenceNumSize:], int(DataSize))
		if err != nil {
			if err != io.EOF && err != io.ErrUnexpectedEOF {
				fmt.Println("read error:", err)
			}
		}
		//log.Printf("%d bytes wrote", n)
		buffer = buffer[:int(StatusSize+SequenceNumSize)+n]
	}
	return buffer
}

func constructAckPacket(sequenceNum uint32) []byte {
	// Packet: ConnStatus + SequenceNum
	buffer := make([]byte, AckPacketSize)
	buffer[0] = ConnStatusByte
	binary.LittleEndian.PutUint32(buffer[StatusSize:StatusSize+SequenceNumSize], sequenceNum)
	return buffer
}
