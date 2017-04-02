package main

import (
	"net"
	"fmt"
	"sync"
	"regexp"
	"io/ioutil"
	"io"
	"strings"
	"net/http"	// Used only for showing blocked page
)

var goroutineCount int
var urlRegexp = regexp.MustCompile(`https?:\/\/[a-zA-Z0-9_\-]+\.[a-zA-Z0-9_\-]+\.[a-zA-Z0-9_\-]+`)
var blocklist = "blocklist.txt"
/*
conn.Write([]byte("<!DOCTYPE html><html><body><h1>Blocked by Proxy</h1><p>You have attempted to access content that is blocked by your systems administrator.</p></body></html>"))
return
*/

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

	// Validate URL, either https, http, or invalid (exit)
	targetURL := urlRegexp.Find(requestBuf)
	if match, _ := regexp.Match("http", targetURL); match == true {
		// Check if website should be blocked


		// Check if message exists in cache and return from function
		/*cache.RLock()
		if cachedResponse, existsInCache := cache.m[string(requestBuf)]; existsInCache {
		conn.Write([]byte(cachedResponse))
			return
		}
		cache.RUnlock()*/

		targetURL = []byte(strings.Replace(string(targetURL), "http://", "", 1))
		proxyConn, err := net.Dial("tcp", string(targetURL) + ":80")
		if err != nil {
			fmt.Println(err)
			return
		}
		proxyConn.Write(requestBuf)
		response, err := ioutil.ReadAll(proxyConn)
		if err != nil {
			return
		}
		conn.Write(response)
		// push into cache
		/*
		cache.Lock()
		cache.m[string(requestBuf)] = string(response)
		cache.Unlock()
		*/
		return

	} else {
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

func blocked (w http.ResponseWriter, r *http.Request) {
	io.WriteString(w, "This website has been blocked!")
}

func isblocked (url string) (result bool) {
    blockedBytes, err := ioutil.ReadFile("blocklist.txt")
    if err != nil {
        return
    }
    return false
}

func main () {
	done := make(chan bool)

	go httpProxy(":13337")
	go httpsProxy(":14488")

	done <- true
}
