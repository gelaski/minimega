// Copyright (2014) Sandia Corporation.
// Under the terms of Contract DE-AC04-94AL85000 with Sandia Corporation,
// the U.S. Government retains certain rights in this software.

package main

import (
	"fmt"
	"minicli"
	log "minilog"
	"strconv"
)

var vncCLIHandlers = []minicli.Handler{
	{ // vnc
		HelpShort: "record or playback VNC kb or fb",
		HelpLong: `
Record or playback keyboard and mouse events sent via the web interface to the
selected VM. Can also record the framebuffer for the specified VM so that a
users can watch a video of interactions with the VM.

With no arguments, vnc will list currently recording or playing VNC sessions.

If record is selected, a file will be created containing a record of mouse and
keyboard actions by the user or of the framebuffer for the VM.

If playback is selected, the specified file (created using vnc record) will be
read and processed as a sequence of time-stamped mouse/keyboard events to send
to the specified VM.`,
		Patterns: []string{
			"vnc <kb,fb> <record,> <vm name> <filename>",
			"vnc <kb,fb> <norecord,> <vm name>",
			"vnc <playback,> <vm name> <filename>",
			"vnc <noplayback,> <vm name>",
		},
		Call: wrapVMTargetCLI(cliVNC),
	},
	{
		HelpShort: "reset VNC state",
		HelpLong: `
Resets the state for VNC recordings. See "help vnc" for more information.`,
		Patterns: []string{
			"clear vnc",
		},
		Call: wrapBroadcastCLI(cliVNCClear),
	},
	{
		HelpShort: "list all running vnc playback/recording instances",
		HelpLong: `
List all running vnc playback/recording instances. See "help vnc" for more information.`,
		Patterns: []string{
			"vnc",
		},
		Call: wrapBroadcastCLI(cliVNCList),
	},
}

// List all active recordings and playbacks
func cliVNCList(c *minicli.Command, resp *minicli.Response) error {
	resp.Header = []string{"host", "name", "id", "type", "filename"}
	resp.Tabular = [][]string{}

	for _, v := range vncKBRecording {
		if inNamespace(v.Vm) {
			resp.Tabular = append(resp.Tabular, []string{
				hostname, v.Vm.Name, strconv.Itoa(v.Vm.ID),
				"record kb",
				v.file.Name(),
			})
		}
	}

	for _, v := range vncFBRecording {
		if inNamespace(v.Vm) {
			resp.Tabular = append(resp.Tabular, []string{
				hostname, v.Vm.Name, strconv.Itoa(v.Vm.ID),
				"record fb",
				v.file.Name(),
			})
		}
	}

	for _, v := range vncKBPlaying {
		if inNamespace(v.Vm) {
			resp.Tabular = append(resp.Tabular, []string{
				hostname, v.Vm.Name, strconv.Itoa(v.Vm.ID),
				"playback kb",
				v.file.Name(),
			})
		}
	}

	return nil
}

func cliVNC(c *minicli.Command, resp *minicli.Response) error {
	fname := c.StringArgs["filename"]

	vm, err := vms.FindKvmVM(c.StringArgs["vm"])
	if err != nil {
		return fmt.Errorf("vm %s not found", c.StringArgs["vm"])
	}

	if c.BoolArgs["record"] && c.BoolArgs["kb"] {
		// Starting keyboard recording
		return vncRecordKB(vm.Name, fname)
	} else if c.BoolArgs["record"] && c.BoolArgs["fb"] {
		// Starting framebuffer recording
		return vncRecordFB(vm.Name, fname)
	} else if c.BoolArgs["norecord"] || c.BoolArgs["noplayback"] {
		var client *vncClient
		ns := fmt.Sprintf("%v:%v", vm.Namespace, vm.Name)

		if c.BoolArgs["norecord"] && c.BoolArgs["kb"] {
			err = fmt.Errorf("kb recording %v not found", vm.Name)
			if v, ok := vncKBRecording[ns]; ok {
				client = v.vncClient
				delete(vncKBRecording, ns)
			}
		} else if c.BoolArgs["norecord"] && c.BoolArgs["fb"] {
			err = fmt.Errorf("fb recording %v not found", vm.Name)
			if v, ok := vncFBRecording[ns]; ok {
				client = v.vncClient
				delete(vncFBRecording, ns)
			}
		} else if c.BoolArgs["noplayback"] {
			err = fmt.Errorf("kb playback %v not found", vm.Name)
			if v, ok := vncKBRecording[ns]; ok {
				client = v.vncClient
				delete(vncKBPlaying, ns)
			}
		}

		if client != nil {
			return client.Stop()
		}

		return err
	} else if c.BoolArgs["playback"] {
		// Start keyboard playback
		return vncPlaybackKB(vm.Name, fname)
	}
	return nil
}

func cliVNCClear(c *minicli.Command, resp *minicli.Response) error {
	for k, v := range vncKBRecording {
		if inNamespace(v.Vm) {
			log.Debug("stopping kb recording for %v", k)
			if err := v.Stop(); err != nil {
				log.Error("%v", err)
			}

			delete(vncKBRecording, k)
		}
	}

	for k, v := range vncFBRecording {
		if inNamespace(v.Vm) {
			log.Debug("stopping fb recording for %v", k)
			if err := v.Stop(); err != nil {
				log.Error("%v", err)
			}

			delete(vncFBRecording, k)
		}
	}

	for k, v := range vncKBPlaying {
		if inNamespace(v.Vm) {
			log.Debug("stopping kb playing for %v", k)
			if err := v.Stop(); err != nil {
				log.Error("%v", err)
			}

			delete(vncKBPlaying, k)
		}
	}
	return nil
}
