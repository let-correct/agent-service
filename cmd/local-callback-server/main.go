package main

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"log"
	"math/big"
	"net"
	"net/http"
	"time"
)

func main() {
	http.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		code := r.URL.Query().Get("code")
		state := r.URL.Query().Get("state")

		fmt.Printf("\n--- OAuth Callback Received ---\ncode:  %s\nstate: %s\n-------------------------------\n\n", code, state)

		w.Header().Set("Content-Type", "text/html")
		fmt.Fprintf(w, `<html><body>
<h2>Callback received</h2>
<p><strong>code:</strong> %s</p>
<p><strong>state:</strong> %s</p>
<p>Check your terminal, then POST these to your Lambda callback endpoint.</p>
</body></html>`, code, state)
	})

	tlsCert, err := selfSignedCert()
	if err != nil {
		log.Fatalf("generating cert: %v", err)
	}

	server := &http.Server{
		Addr: ":3000",
		TLSConfig: &tls.Config{
			Certificates: []tls.Certificate{tlsCert},
		},
	}

	log.Println("listening on https://localhost:3000 (self-signed cert — accept the browser warning)")
	log.Fatal(server.ListenAndServeTLS("", ""))
}

func selfSignedCert() (tls.Certificate, error) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return tls.Certificate{}, err
	}

	template := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "localhost"},
		DNSNames:     []string{"localhost"},
		IPAddresses:  []net.IP{net.ParseIP("127.0.0.1")},
		NotBefore:    time.Now(),
		NotAfter:     time.Now().Add(24 * time.Hour),
	}

	certDER, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		return tls.Certificate{}, err
	}

	keyDER, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		return tls.Certificate{}, err
	}

	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER})

	return tls.X509KeyPair(certPEM, keyPEM)
}
