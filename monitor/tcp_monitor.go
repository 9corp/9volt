package monitor

import (
	"fmt"
	"net"
	"strings"
	"time"

	log "github.com/Sirupsen/logrus"
)

const (
	DEFAULT_CONN_TIMEOUT  = time.Duration(4) * time.Second
	DEFAULT_READ_TIMEOUT  = time.Duration(2) * time.Second
	DEFAULT_WRITE_TIMEOUT = time.Duration(2) * time.Second
	DEFAULT_READ_SIZE     = 4096
)

type TCPMonitor struct {
	Base

	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	ConnTimeout  time.Duration
	ReadSize     int
}

func NewTCPMonitor(rmc *RootMonitorConfig) IMonitor {
	t := &TCPMonitor{
		Base: Base{
			RMC:        rmc,
			Identifier: "tcp",
		},
	}

	t.MonitorFunc = t.tcpCheck

	t.updateSettings()

	return t
}

// Update timeout and read size related settings
func (t *TCPMonitor) updateSettings() {
	t.ConnTimeout = DEFAULT_CONN_TIMEOUT
	t.ReadTimeout = DEFAULT_READ_TIMEOUT
	t.WriteTimeout = DEFAULT_WRITE_TIMEOUT
	t.ReadSize = DEFAULT_READ_SIZE

	if t.RMC.Config.Timeout.String() != "" {
		t.ConnTimeout = time.Duration(t.RMC.Config.Timeout)
	}

	if t.RMC.Config.TCPReadTimeout.String() != "0s" {
		t.ReadTimeout = time.Duration(t.RMC.Config.TCPReadTimeout)
		log.Warningf("Our read timeout is %s", t.ReadTimeout)
	}

	if t.RMC.Config.TCPWriteTimeout.String() != "0s" {
		t.WriteTimeout = time.Duration(t.RMC.Config.TCPWriteTimeout)
		log.Warningf("Our write timeout is %s", t.ReadTimeout)
	}

	if t.RMC.Config.TCPReadSize != 0 {
		t.ReadSize = t.RMC.Config.TCPReadSize
	}
}

// Perform a TCP connection to host:port using an optional connection timeout,
// read timeout, read size and/or expected output. If `Send` is set, first send
// data in `Send` on the opened connection.
func (t *TCPMonitor) tcpCheck() error {
	fullAddress := fmt.Sprintf("%v:%v", t.RMC.Config.Host, t.RMC.Config.Port)

	log.Debugf("%v-%v: Performing tcp check against '%v'", t.Identify(), t.RMC.GID, fullAddress)

	// Open the connection
	conn, err := net.DialTimeout("tcp", fullAddress, t.ConnTimeout)
	if err != nil {
		return fmt.Errorf("Unable to open connection to %v: %v", fullAddress, err.Error())
	}

	// If set, send data first
	if t.RMC.Config.TCPSend != "" {
		if err := conn.SetWriteDeadline(time.Now().Add(t.WriteTimeout)); err != nil {
			return fmt.Errorf("Unable to set write timeout (%v): %v", t.WriteTimeout, err.Error())
		}

		if _, err := conn.Write([]byte(t.RMC.Config.TCPSend)); err != nil {
			return fmt.Errorf("Unable to send initial TCP data (%v): %v", t.RMC.Config.TCPSend, err.Error())
		}
	}

	// No expect set, we're done
	if t.RMC.Config.Expect == "" {
		return nil
	}

	// Set read deadline
	if err := conn.SetReadDeadline(time.Now().Add(t.ReadTimeout)); err != nil {
		return fmt.Errorf("Unable to set read timeout (%v): %v", t.ReadTimeout, err.Error())
	}

	// Read the actual data
	recvBuf := make([]byte, t.ReadSize)

	if n, err := conn.Read(recvBuf); err != nil {
		log.Debugf("%v-%v: Read %v bytes from %v", t.Identifier, t.RMC.GID, n, fullAddress)

		if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
			return fmt.Errorf("Read timeout (after %v): %v", t.ReadTimeout, err.Error())
		} else {
			return fmt.Errorf("Unrecognized read error: %v", err.Error())
		}
	}

	// Optionally, verify our received data contains our expected string
	if !strings.Contains(string(recvBuf), t.RMC.Config.Expect) {
		return fmt.Errorf("Received data does not contain expected substring (Recv: [%v] Expected: [%v]",
			string(recvBuf), t.RMC.Config.Expect)
	}

	return nil
}
