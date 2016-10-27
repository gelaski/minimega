// Copyright (2015) Sandia Corporation.
// Under the terms of Contract DE-AC04-94AL85000 with Sandia Corporation,
// the U.S. Government retains certain rights in this software.

package main

import (
	"encoding/gob"
	"io"
	log "minilog"
	"net"
	"os"
	"ron"
	"runtime"
	"sync"
	"time"
	"version"
)

var client struct {
	ron.Client // embed
	sync.Mutex // embed

	Processes map[int]*Process

	lastHeartbeat time.Time

	writeMu sync.Mutex

	conn io.ReadWriteCloser
	enc  *gob.Encoder
	dec  *gob.Decoder

	commandChan chan map[int]*ron.Command
	fileChan    chan *ron.Message
}

type Process struct {
	PID     int
	Command []string
	process *os.Process
}

// init client fields
func init() {
	client.UUID = getUUID()
	client.Arch = runtime.GOARCH
	client.OS = runtime.GOOS
	client.Version = version.Revision

	client.Processes = make(map[int]*Process)

	client.commandChan = make(chan map[int]*ron.Command)
	client.fileChan = make(chan *ron.Message)
}

func sendMessage(m *ron.Message) error {
	client.writeMu.Lock()
	defer client.writeMu.Unlock()

	return client.enc.Encode(m)
}

// appendResponse allows a client to post a *Response to a given command. The
// response will be queued until the next heartbeat.
func appendResponse(r *ron.Response) {
	log.Debug("response: %v", r.ID)

	client.Lock()
	defer client.Unlock()

	client.LastCommandID = r.ID
	client.Responses = append(client.Responses, r)
}

func addTag(k, v string) {
	log.Debug("tag: %v %v", k, v)

	client.Lock()
	defer client.Unlock()

	client.Tags[k] = v
}

// updateNetworkInfo updates the hostname, IPs, and MACs for the client.
// Assumes that the client lock is held.
func updateNetworkInfo() {
	hostname, err := os.Hostname()
	if err != nil {
		log.Error("unable to get hostname: %v", err)
	}

	client.Hostname = hostname
	client.IPs = nil
	client.MACs = nil

	nics, err := net.Interfaces()
	if err != nil {
		log.Error("unable to get interfaces: %v", err)
	}

	for _, nic := range nics {
		if nic.HardwareAddr.String() == "" {
			// skip localhost and other weird interfaces
			continue
		}

		log.Debug("found mac: %v", nic.HardwareAddr)
		client.MACs = append(client.MACs, nic.HardwareAddr.String())

		addrs, err := nic.Addrs()
		if err != nil {
			log.Error("unable to get addrs for %v: %v", nic.HardwareAddr, err)
		}

		for _, addr := range addrs {
			switch addr := addr.(type) {
			case *net.IPNet:
				client.IPs = append(client.IPs, addr.IP.String())
			default:
				log.Debug("unknown network type: %v", addr)
			}
		}
	}
}
