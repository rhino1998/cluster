package util

import (
	"io/ioutil"
	"net"
	"net/http"
)

func GetLocalIP() (ip net.IP, err error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}
	// handle err
	for _, i := range ifaces {
		addrs, err := i.Addrs()
		if err != nil {
			return nil, err
		}
		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
				if !ip.IsLoopback() && ip.String() != "0.0.0.0" {
					return ip, nil
				}
			case *net.IPAddr:
				ip = v.IP
				if !ip.IsLoopback() && ip.String() != "0.0.0.0" {
					return ip, nil
				}
			}
		}
	}
	return nil, err
}

//replace this eventually
func GetExternalIP() (ip net.IP, err error) {
	resp, err := http.Get("http://myexternalip.com/raw")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	ip, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return net.ParseIP(string(ip)[:len(string(ip))-1]), nil
}

func ByteSliceEq(a, b []byte) bool {

	if a == nil && b == nil {
		return true
	}

	if a == nil || b == nil {
		return false
	}

	if len(a) != len(b) {
		return false
	}

	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}

	return true
}
