package internal

import (
	"bufio"
	"net"
	"os"
	"regexp"
	"strconv"
	"strings"
)

const (
	OP_NOP = iota
	OP_WRT
	OP_RD
	OP_IF
	OP_EIF
	OP_FWD
	OP_BAK
	OP_INC
	OP_DEC
	OP_CON
	OP_END
	STD_MEM int = 65536
)

type Lexer struct {
	mem  [STD_MEM]uint8
	ip   uint16
	mp   uint16
	conn net.Conn
}

func NewLexer() *Lexer {
	return &Lexer{
		mem:  [STD_MEM]uint8{},
		ip:   uint16(0),
		mp:   uint16(0), // starts at first byte after instructions
		conn: nil,
	}
}

func (l *Lexer) Lex(code string) {
	space := regexp.MustCompile(`\s+`)
	code = space.ReplaceAllString(code, " ")
	var op uint8 = 0
	l.ip = 0
	for i := 0; i < len(code); i++ {
		opcode := code[i] - 48 // int('0') === 48
		if opcode < 10 {
			op += opcode
		}
		if code[i] == ' ' || i == len(code)-1 {
			l.mem[l.ip] = op
			l.ip++
			op = 0
			continue
		}
	}
	l.mp = l.ip + 1
	l.ip = 0
}

func (l *Lexer) Exec() {
	stdout := bufio.NewWriter(os.Stdout)
	stdin := bufio.NewReader(os.Stdin)
	var isRunning bool = true
	var currOP uint8 = l.mem[l.ip]
	for isRunning {
		currOP = l.mem[l.ip]
		switch currOP {
		case OP_NOP:
			l.ip++
		case OP_WRT:
			l.ip++
			if l.conn == nil {
				_ = stdout.WriteByte(l.mem[l.mp]) // NOTE: no need to report golang errors
				_ = stdout.Flush()
				continue
			}
			_, _ = l.conn.Write([]byte{l.mem[l.mp]}) // NOTE: no need to report golang errors
		case OP_RD:
			l.ip++
			if l.conn == nil {
				l.mem[l.mp], _ = stdin.ReadByte() // NOTE: no need to report golang errors
				continue
			}
			var bytesRead []byte
			_, _ = l.conn.Read(bytesRead) // NOTE: no need to report golang errors
			l.mem[l.mp] = bytesRead[0]
		case OP_IF:
			if l.mem[l.mp] == 0 {
				var isOurEIF uint16 = 1
				var findEIFip uint16 = l.ip + 1
				for isOurEIF != 0 {
					if l.mem[findEIFip] == OP_EIF {
						isOurEIF--
					} else if l.mem[findEIFip] == OP_IF {
						isOurEIF++
					}
					findEIFip++
				}
				l.ip = findEIFip
			}
			l.ip++
		case OP_EIF:
			if l.mem[l.mp] != 0 {
				var isOurIF uint16 = 1
				var findIFip uint16 = l.ip - 1
				for isOurIF != 0 {
					if l.mem[findIFip] == OP_EIF {
						isOurIF++
					} else if l.mem[findIFip] == OP_IF {
						isOurIF--
					}
					findIFip--
				}
				l.ip = findIFip
			}
			l.ip++
		case OP_FWD:
			l.mp += uint16(l.mem[l.ip+1] + 1)
			l.ip += 2
		case OP_BAK:
			l.mp -= uint16(l.mem[l.ip+1] + 1)
			l.ip += 2
		case OP_INC:
			l.mem[l.mp] += l.mem[l.ip+1] + 1
			l.ip += 2
		case OP_DEC:
			l.mem[l.mp] -= l.mem[l.ip+1] + 1
			l.ip += 2
		case OP_CON:
			var connOctets = [4]string{}
			var i uint16
			for i = 0; i < 4; i++ {
				connOctets[i] = strconv.Itoa(int(l.mem[l.mp+i]))
			}
			connIP := strings.Join(connOctets[:], ".")
			var connPORT string = ""
			for i < 6 {
				connPORT += strconv.Itoa(int(l.mem[l.mp+i]))
				i++
			}
			networks := []string{"tcp", "udp", "ip", "unix", "unixgram", "unixpacket"}
			var c net.Conn = nil
			var err error
			for _, network := range networks {
				c, err = net.Dial(network, connIP+":"+connPORT)
				if err == nil {
					break
				}
			}
			if err != nil {
				_, _ = stdout.WriteString("\n\nh0s7 5uXz0r5! c4N'7 c0Nn3<7 l0l0l0l0l l4m3R !!!\n\n")
				_ = stdout.Flush() // NOTE: no need to report golang errors
				continue
			}
			l.conn = c
		case OP_END:
			isRunning = false
			if l.conn != nil {
				_ = l.conn.Close()
			}
			_ = stdout.WriteByte('\n')
			_ = stdout.Flush()
		default:
			_, _ = stdout.WriteString("\n\nj00 4r3 teh 5ux0r\n\n") // NOTE: no need to report golang errors
			_ = stdout.Flush()
		}
	}
}

func (l *Lexer) DumpMem() [STD_MEM]uint8 {
	return l.mem
}
