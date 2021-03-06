package kasalink

import (
	"errors"
	"fmt"
	"net"
	"strings"
	"time"
)

var errTooLarge = errors.New("bytes.Buffer: too large")

// KasaPowerPlug is the struct that holds info about and methods for talking to a Kasa Power Plug or Power Strip
type KasaPowerPlug struct {
	plugNetworkLocation string
	Unsafe              unsafe
	deviceID            string
	tplinkClient        net.Conn
	timeout             time.Duration
	SysInfo             *SystemInfo
}

// NewKasaPowerPlug gives you a new KasaPowerPlug struct that's already gotten it's system info, or an error
// telling you why that didn't work
func NewKasaPowerPlug(plugAddress string) (kpp *KasaPowerPlug, err error) {
	kpp = &KasaPowerPlug{plugNetworkLocation: plugAddress}
	kpp.SysInfo, err = kpp.GetSystemInfo()
	if err != nil {
		return nil, err
	}
	return
}

// TalkToPlug sends a command to the plug and returns a response json and error error
func (kpp *KasaPowerPlug) talkToPlug(KasaCommand string) (response []byte, err error) {
	var (
		bitsToSend []byte
	)
	//log.Printf("Command for Plug: %s\n", KasaCommand)
	if kpp.tplinkClient == nil {
		if kpp.timeout == 0 {
			kpp.timeout = time.Duration(10) * time.Second
		}
		if kpp.tplinkClient, err = net.DialTimeout("tcp", kpp.plugNetworkLocation, kpp.timeout); err != nil {
			return
		}
	}
	bitsToSend = encrypt(KasaCommand)
	if _, err = kpp.tplinkClient.Write(bitsToSend); err != nil {
		return
	}
	var bb = new(myBuff)
	//var bytesRead int64
	//bytesRead, err = bb.readFrom(kpp.tplinkClient)
	_, err = bb.readFrom(kpp.tplinkClient)

	if err != nil {
		return
	}
	//log.Printf("Bytes Read: %d\n", bytesRead)
	if bb.Len() >= 4 {
		return decrypt(bb.buf[4:]), nil
	}
	return
}

// tellChild is the JSON used to issue a command to individual sockets on a Kasa enabled device
func (kpp *KasaPowerPlug) tellChild(cmd string, children ...int) ([]byte, error) {
	var (
		sb  strings.Builder
		err error
	)

	if _, err = sb.WriteString(`{"context":{"child_ids":[`); err != nil {
		return nil, err
	}
	for _, child := range children {
		if _, err = sb.WriteString(fmt.Sprintf(`"%s%02d",`, kpp.deviceID, child)); err != nil {
			return nil, err
		}
	}
	if _, err = sb.WriteString(fmt.Sprintf(`]},%s}`, cmd[1:len(cmd)-1])); err != nil {
		return nil, err
	}
	//log.Printf("Child Call: %s\n", sb.String())
	//log.Printf("Child Call Trimmed: %s\n", trimJSONArray(sb.String()))
	return kpp.talkToPlug(trimJSONArray(sb.String()))
}

// Close tells the client to close any active connection it might have to the power strip/plug
func (kpp *KasaPowerPlug) Close() error {
	if kpp.tplinkClient != nil {
		return kpp.tplinkClient.Close()
	}
	// the net.Conn object is nil, so nothing to close, return nil
	return nil
}
