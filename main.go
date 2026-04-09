package main

// env CGO_ENABLED=0 go build -ldflags "-s -w"

import (
	"bytes"
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"flag"
	"fmt"
	"net"
	"net/http"
	"runtime/debug"
	"strings"
	"time"

	"github.com/go-acme/lego/v4/certcrypto"
	"github.com/go-acme/lego/v4/certificate"
	"github.com/go-acme/lego/v4/challenge/http01"
	"github.com/go-acme/lego/v4/challenge/tlsalpn01"
	"github.com/go-acme/lego/v4/lego"
	"github.com/go-acme/lego/v4/registration"
	proxyproto "github.com/pires/go-proxyproto"

	log "github.com/sirupsen/logrus"
)

const version = "1.2.1"

var config Config
var err error

type icahanLegoUser struct {
	Email        string
	Registration *registration.Resource
	Key          crypto.PrivateKey
}

func (u *icahanLegoUser) GetEmail() string {
	return u.Email
}

func (u *icahanLegoUser) GetRegistration() *registration.Resource {
	return u.Registration
}

func (u *icahanLegoUser) GetPrivateKey() crypto.PrivateKey {
	return u.Key
}

func main() {
	ver := flag.Bool("version", false, "Prints version")
	configPath := flag.String("config", "", "Path to config file")
	flag.Parse()

	config, err = LoadConfig(*configPath)

	if err != nil {
		log.Errorf("Error loading config: %v\n", err)
		return
	}

	if *ver {
		fmt.Println(version)
		if bi, ok := debug.ReadBuildInfo(); ok {
			fmt.Println(bi.String())
			return
		}
	}

	server := &http.Server{
		Addr: fmt.Sprintf("%s:%d", config.Server.Host, config.Server.Port),
	}

	var httpListener net.Listener

	if config.Server.EnableProxyProtocol {
		log.Infof("Proxy protocol enabled, listening for PROXY protocol on %s:%d\n", config.Server.Host, config.Server.Port)
		listener, err := net.Listen("tcp", server.Addr)

		if err != nil {
			panic(fmt.Sprintf("Got error: %v, trying to listen on %s", err, server.Addr))
		}
		httpListener = &proxyproto.Listener{
			Listener:          listener,
			ReadHeaderTimeout: 10 * time.Second,
		}
	} else {
		log.Infof("Proxy Protocol not enabled using standard listener")
		httpListener, err = net.Listen("tcp", server.Addr)

		if err != nil {
			panic(fmt.Sprintf("Got error: %v, trying to listen on %s", err, server.Addr))
		}
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", getIPAddress)
	server.Handler = mux

	defer httpListener.Close()

	if config.Server.TLS != nil {
		if config.Server.TLS.CertFile != "" && config.Server.TLS.KeyFile != "" {
			log.Infof("Starting server with TLS on %s:%d\n", config.Server.Host, config.Server.Port)
			err := server.ServeTLS(httpListener, config.Server.TLS.CertFile, config.Server.TLS.KeyFile)
			if err != nil {
				panic("Error starting server with TLS: " + err.Error())
			}
		} else if config.Server.TLS.Acme != nil {
			log.Infof("Starting server with ACME TLS on %s:%d\n", config.Server.Host, config.Server.Port)
			privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
			if err != nil {
				panic("Error generating private key: " + err.Error())
			}

			legoReg := icahanLegoUser{
				Email: config.Server.TLS.Acme.Email,
				Key:   certcrypto.PEMEncode(privateKey),
			}

			legoConfig := lego.NewConfig(&legoReg)
			legoConfig.CADirURL = config.Server.TLS.Acme.AcmeDirectoryURL
			legoConfig.Certificate.KeyType = certcrypto.EC256

			client, err := lego.NewClient(legoConfig)
			if err != nil {
				panic("Error creating ACME client: " + err.Error())
			}

			err = client.Challenge.SetHTTP01Provider(http01.NewProviderServer("", config.Server.TLS.Acme.HTTP01Port))
			if err != nil {
				panic("Error setting HTTP-01 provider: " + err.Error())
			}

			err = client.Challenge.SetTLSALPN01Provider(tlsalpn01.NewProviderServer("", config.Server.TLS.Acme.TLSALPN01Port))
			if err != nil {
				panic("Error setting TLS-ALPN-01 provider: " + err.Error())
			}

			reg, err := client.Registration.Register(registration.RegisterOptions{TermsOfServiceAgreed: true})
			if err != nil {
				panic("Error registering ACME account: " + err.Error())
			}
			log.Infof("Registered ACME account: %s\n", reg.URI)

			request := certificate.ObtainRequest{
				Domains: config.Server.TLS.Acme.Domains,
				Bundle:  true,
			}
			certificates, err := client.Certificate.Obtain(request)
			if err != nil {
				panic("Error obtaining certificate: " + err.Error())
			}

			cert, err := tls.X509KeyPair(certificates.Certificate, certificates.PrivateKey)

			if err != nil {
				panic("Error loading certificate: " + err.Error())
			}

			tlsConfig := &tls.Config{
				Certificates: []tls.Certificate{cert},
			}

			server.TLSConfig = tlsConfig

			err = server.ServeTLS(httpListener, "", "")
			if err != nil {
				panic("Error starting server with ACME TLS: " + err.Error())
			}
		} else {
			panic("Either cert and private key or acme must be defined, if both are defined the cert and private key has precedence")
		}
	} else {
		server.Serve(httpListener)
	}
}

// ipRange - a structure that holds the start and end of a range of ip addresses
type ipRange struct {
	start net.IP
	end   net.IP
}

// inRange - check to see if a given ip address is within a range given
func inRange(r ipRange, ipAddress net.IP) bool {
	// strcmp type byte comparison
	if bytes.Compare(ipAddress, r.start) >= 0 && bytes.Compare(ipAddress, r.end) < 0 {
		return true
	}
	return false
}

var privateRanges = []ipRange{
	{
		start: net.ParseIP("10.0.0.0"),
		end:   net.ParseIP("10.255.255.255"),
	},
	{
		start: net.ParseIP("100.64.0.0"),
		end:   net.ParseIP("100.127.255.255"),
	},
	{
		start: net.ParseIP("172.16.0.0"),
		end:   net.ParseIP("172.31.255.255"),
	},
	{
		start: net.ParseIP("192.0.2.0"),
		end:   net.ParseIP("192.0.2.255"),
	},
	{
		start: net.ParseIP("192.168.0.0"),
		end:   net.ParseIP("192.168.255.255"),
	},
	{
		start: net.ParseIP("198.18.0.0"),
		end:   net.ParseIP("198.19.255.255"),
	},
}

// isPrivateSubnet - check to see if this ip is in a private subnet
func isPrivateSubnet(ipAddress net.IP) bool {
	// my use case is only concerned with ipv4 atm
	if ipCheck := ipAddress.To4(); ipCheck != nil {
		// iterate over all our ranges
		for _, r := range privateRanges {
			// check if this ip is in a private range
			if inRange(r, ipAddress) {
				return true
			}
		}
	}
	return false
}

func getIPAddress(w http.ResponseWriter, r *http.Request) {
	var ipv4 string
	var ipv6 string

	if config.Results.HTTPHeaders != nil {
		for _, h := range config.Results.HTTPHeaders {
			addresses := strings.Split(r.Header.Get(h), ",")
			for i := len(addresses) - 1; i >= 0; i-- {
				ip, _, err := net.SplitHostPort(strings.TrimSpace(addresses[i]))
				if err != nil {
					ip = strings.TrimSpace(addresses[i]) // In case there's no port
				}
				realIP := net.ParseIP(ip)
				if realIP == nil {
					continue
				}
				if ipv4Address := realIP.To4(); ipv4Address != nil {
					if !realIP.IsGlobalUnicast() || !config.Results.IncludePrivate && isPrivateSubnet(realIP) {
						continue
					}
					ipv4 = ip
					break // Found a valid IPv4 address
				}
			}
			if ipv4 != "" {
				fmt.Fprintf(w, ipv4+"\n")
				return
			}
		}
	}

	if ipv4 == "" { // If no valid IPv4 was found, check RemoteAddr
		ret := r.RemoteAddr
		ip, _, _ := net.SplitHostPort(ret)
		realIP := net.ParseIP(ip)
		if realIP != nil && realIP.To4() != nil {
			if realIP.IsGlobalUnicast() && (config.Results.IncludePrivate || !isPrivateSubnet(realIP)) {
				fmt.Fprintf(w, ip+"\n")
				return
			}
		}
	}

	if ipv6 != "" { // Use IPv6 if no IPv4 was found
		fmt.Fprintf(w, ipv6+"\n")
	} else {
		ret := r.RemoteAddr
		ip, _, _ := net.SplitHostPort(ret)
		fmt.Fprintf(w, ip+"\n")
	}
}
