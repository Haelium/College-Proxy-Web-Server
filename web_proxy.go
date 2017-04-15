package main

import (
	"net"
	"fmt"
	"sync"
	"regexp"
	"io/ioutil"
	"io"
	"strings"
	"os"
	"bufio"
	"bytes"
	"time"
)

var goroutineCount int
// Regex to match a valid base url (eg: http://www.example.com or http://www.sub.sub2.example.com or http://www.example.co.uk)
var urlRegexp = regexp.MustCompile(`http:\/\/[a-zA-Z0-9_\-]+\.[a-zA-Z0-9_\-]+\.[a-zA-Z0-9_\-]+\.?[a-zA-Z0-9_\-]*\.?[a-zA-Z0-9_\-]*`)

// the cache struct implements a concurrency safe string->string hashmap
var cache = struct{
    sync.RWMutex
    m map[string]string
}{ m: make(map[string]string) }

/*
Read
cache.RLock()
n := cache.m["some_key"]
cache.RUnlock()
fmt.Println("some_key:", n)

Write
cache.Lock()
cache.m["some_key"] := "some value"
cache.Unlock()
*/

func webProxy (conn net.Conn) {
	defer conn.Close()	// Close the connection when function returns

	requestBuf := make([]byte, 1024)	// Make a buffer to hold incoming data.
	conn.Read(requestBuf)				// Read incoming connection into buffer
	fmt.Println(string(requestBuf))		// Display the incoming request on the admin terminal

	// Extract URL and check if it is blocked
	targetURL := urlRegexp.Find(requestBuf)
	if blockedResult := isBlocked(string(targetURL)); blockedResult == false {
		// Check if website should be blocked:

		fmt.Println(len(cache.m))
		// Check if message exists in cache and return cached entry if possible
		cache.RLock()
		cachedResponse, existsInCache := cache.m[string(requestBuf)]
		cache.RUnlock()
		// If message exists in cache, respond from cache and return
		if existsInCache == true {
			fmt.Println("Responding from cache")
			conn.Write([]byte(cachedResponse))
			return
		}
		// Remove http:// from url to find target domain name dial it
		targetURL = []byte(strings.Replace(string(targetURL), "http://", "", 1))
		proxyConn, err := net.Dial("tcp", string(targetURL) + ":80")
		if err != nil {
			fmt.Println(err)
			return
		}
		
		proxyConn.Write(requestBuf)					// Forward user's GET request to the target
		response, err := ioutil.ReadAll(proxyConn)	// Read the response from server
		if err != nil {
			return
		}
		conn.Write(response)						// Forward response from server to the user

		fmt.Println(string(response))

		// push into cache if it did not exist in cache before
		if existsInCache == false {
			cache.Lock()
			cache.m[string(requestBuf)] = string(response)
			cache.Unlock()
		}
		
		return

	} else {	// If the domain is blocked
		// Send the user a page informing him/her that the content is blocked
		conn.Write([]byte(	`<html>
							<title>Blocked</title><h1>Blocked!</h1>
							<h2>The content that you have attempted to access is blocked by your proxy.</h2>
							<h2>Please do not contact your Systems Administrator, as they know what they are doing.</h2>
							<h2>If it is vital that you access this content for work purposes, bring them coffee and ask them to remove the block nicely.</h2>
							</html>`))
		return
	}
}

func httpProxy (port string) {
	// Create a socket and listen
	ln, err := net.Listen("tcp", port)
	if err != nil {
    	fmt.Println(err)
    	return
	}
	go clearCache()		// Spawn goroutine to clear cache periodically

	for {
		// accept good connections
		conn, err := ln.Accept()
		if err != nil {
			fmt.Println(err)
		} else {
			go webProxy(conn)
		}
	}
}


func tcpProxy (conn net.Conn) {

}


func httpsProxy (port string) {
	// Create a socket and listen
	ln, err := net.Listen("tcp", port)
	if err != nil {
    	fmt.Println(err)
    	return
	}
	for {
		// accept good connections
		conn, err := ln.Accept()
		if err != nil {
			fmt.Println(err)
		} else {
			go tcpProxy(conn)
		}
	}
}

func isBlocked (url string) (result bool) {
	urlMatch := strings.Replace(url, "http://", "", 1)

    file, err := os.Open("blocklist.txt")
    defer file.Close()	// Close the file when finished

    // Start reading from the file with a reader.
    reader := bufio.NewReader(file)

    for {
        var buffer bytes.Buffer
        var l []byte
        var isPrefix bool

        for {
            l, isPrefix, err = reader.ReadLine()
            buffer.Write(l)

            // If we've reached the end of the line, stop reading.
            if !isPrefix {
                break
            }

            // If we're just at the EOF, break
            if err != nil {
                break
            }
        }

		// Return false at the end of the blocklist
        if err == io.EOF {
            return false
        }

        line := buffer.String()

		// Compare url with blocked item
        if match := strings.Compare(urlMatch, line); match == 0 {
			return true
		}
    }

    return false
}

func clearCache () {
	// This function should check the size of the cache every 30 seconds and clear it if the size is too big
	for {
		time.Sleep(30 * time.Second)	// Every 30 seconds
		if (len(cache.m) > 1000) {		// Check if there are over 1000 items in cache
			fmt.Println("Clearing page response cache")
			cache.Lock()
			cache.m = make(map[string]string)	// Create a new empty map (garbage collection removes the old map)
			cache.Unlock()
		}
	}
}

func main () {
	done := make(chan bool)

	go httpProxy(":13337")		// Spawn goroutine for httpProxy listener
	go httpsProxy(":14488")		// Spawn goroutine for httpsProxy listener

	done <- true
}
