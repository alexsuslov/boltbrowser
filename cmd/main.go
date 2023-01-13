package main

import (
	"errors"
	"fmt"
	"github.com/alexsuslov/boltbrowser"
	"os"
	"strings"
	"time"

	"github.com/boltdb/bolt"
	"github.com/nsf/termbox-go"
)

const DefaultDBOpenTimeout = time.Second

func init() {
	boltbrowser.AppArgs.DBOpenTimeout = DefaultDBOpenTimeout
	boltbrowser.AppArgs.ReadOnly = false
}

func parseArgs() {
	var err error
	if len(os.Args) == 1 {
		printUsage(nil)
	}
	parms := os.Args[1:]
	for i := range parms {
		// All 'option' arguments start with "-"
		if !strings.HasPrefix(parms[i], "-") {
			boltbrowser.DatabaseFiles = append(boltbrowser.DatabaseFiles, parms[i])
			continue
		}
		if strings.Contains(parms[i], "=") {
			// Key/Value pair Arguments
			pts := strings.Split(parms[i], "=")
			key, val := pts[0], pts[1]
			switch key {
			case "-timeout":
				boltbrowser.AppArgs.DBOpenTimeout, err = time.ParseDuration(val)
				if err != nil {
					// See if we can successfully parse by adding a 's'
					boltbrowser.AppArgs.DBOpenTimeout, err = time.ParseDuration(val + "s")
				}
				// If err is still not nil, print usage
				if err != nil {
					printUsage(err)
				}
			case "-readonly", "-ro":
				if val == "true" {
					boltbrowser.AppArgs.ReadOnly = true
				}
			case "-no-value":
				if val == "true" {
					boltbrowser.AppArgs.NoValue = true
				}
			case "-help":
				printUsage(nil)
			default:
				printUsage(errors.New("Invalid option"))
			}
		} else {
			// Single-word arguments
			switch parms[i] {
			case "-readonly", "-ro":
				boltbrowser.AppArgs.ReadOnly = true
			case "-no-value":
				boltbrowser.AppArgs.NoValue = true
			case "-help":
				printUsage(nil)
			default:
				printUsage(errors.New("Invalid option"))
			}
		}
	}
}

func printUsage(err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, err.Error())
	}
	fmt.Fprintf(os.Stderr, "Usage: %s [OPTIONS] <filename(s)>\nOptions:\n", boltbrowser.ProgramName)
	fmt.Fprintf(os.Stderr, "  -timeout=duration\n        DB file open timeout (default 1s)\n")
	fmt.Fprintf(os.Stderr, "  -ro, -readonly   \n        Open the DB in read-only mode\n")
	fmt.Fprintf(os.Stderr, "  -no-value        \n        Do not display a value in left pane\n")
}

func main() {
	var err error

	parseArgs()

	err = termbox.Init()
	if err != nil {
		panic(err)
	}
	defer termbox.Close()
	style := boltbrowser.DefaultStyle()
	termbox.SetOutputMode(termbox.Output256)

	for _, databaseFile := range boltbrowser.DatabaseFiles {
		boltbrowser.CurrentFilename = databaseFile
		boltbrowser.DB, err = bolt.Open(databaseFile, 0600, &bolt.Options{Timeout: boltbrowser.AppArgs.DBOpenTimeout})
		if err == bolt.ErrTimeout {
			termbox.Close()
			fmt.Printf("File %s is locked. Make sure it's not used by another app and try again\n", databaseFile)
			os.Exit(1)
		} else if err != nil {
			if len(boltbrowser.DatabaseFiles) > 1 {
				boltbrowser.MainLoop(nil, style)
				continue
			} else {
				termbox.Close()
				fmt.Printf("Error reading file: %q\n", err.Error())
				os.Exit(1)
			}
		}

		// First things first, load the database into memory
		boltbrowser.MemBolt.RefreshDatabase()
		if boltbrowser.AppArgs.ReadOnly {
			// If we're opening it in readonly mode, close it now
			boltbrowser.DB.Close()
		}

		// Kick off the UI loop
		boltbrowser.MainLoop(boltbrowser.MemBolt, style)
		defer boltbrowser.DB.Close()
	}
}
