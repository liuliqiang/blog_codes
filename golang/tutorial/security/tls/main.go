package main

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	log2 "log"
	"math/big"
	"os"
	"time"
)

var (
	log       = log2.Logger{}
	caFile    = "/tmp/cert.crt"
	serverKey = "/tmp/client.key"
	serverCrt = "/tmp/client.crt"
)

type ecdsaGen struct {
	curve elliptic.Curve
}

func (e *ecdsaGen) KeyGen() (key *ecdsa.PrivateKey, err error) {
	privKey, err := ecdsa.GenerateKey(e.curve, rand.Reader)
	if err != nil {
		return nil, err
	}
	return privKey, nil
}

func encode(privateKey *ecdsa.PrivateKey, publicKey *ecdsa.PublicKey) (string, string) {
	x509Encoded, _ := x509.MarshalECPrivateKey(privateKey)
	pemEncoded := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: x509Encoded})

	x509EncodedPub, _ := x509.MarshalPKIXPublicKey(publicKey)
	pemEncodedPub := pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: x509EncodedPub})

	// 将ec 密钥写入到 pem文件里
	keypem, _ := os.OpenFile("ec-key.pem", os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0666)
	pem.Encode(keypem, &pem.Block{Type: "EC PRIVATE KEY", Bytes: x509Encoded})

	return string(pemEncoded), string(pemEncodedPub)
}

func decode(pemEncoded string, pemEncodedPub string) (*ecdsa.PrivateKey, *ecdsa.PublicKey) {
	block, _ := pem.Decode([]byte(pemEncoded))
	x509Encoded := block.Bytes
	privateKey, _ := x509.ParseECPrivateKey(x509Encoded)

	blockPub, _ := pem.Decode([]byte(pemEncodedPub))
	x509EncodedPub := blockPub.Bytes
	genericPublicKey, _ := x509.ParsePKIXPublicKey(x509EncodedPub)
	publicKey := genericPublicKey.(*ecdsa.PublicKey)

	return privateKey, publicKey
}

func checkError(err error) {
	if err != nil {
		log.Panic(err)
	}
}

// 根据ecdsa密钥生成特征标识码
func priKeyHash(priKey *ecdsa.PrivateKey) []byte {
	hash := sha256.New()
	hash.Write(elliptic.Marshal(priKey.Curve, priKey.PublicKey.X, priKey.PublicKey.Y))
	return hash.Sum(nil)
}

func main() {
	// 生成ecdsa
	e := &ecdsaGen{curve: elliptic.P256()}
	priKey, _ := e.KeyGen()
	priKeyEncode, err := x509.MarshalECPrivateKey(priKey)
	checkError(err)
	// 保存到pem文件
	f, err := os.Create("/tmp/ec.pem")
	checkError(err)
	pem.Encode(f, &pem.Block{Type: "EC PRIVATE KEY", Bytes: priKeyEncode})
	f.Close()
	pubKey := priKey.Public()
	// Encode public key
	//raw, err := x509.MarshalPKIXPublicKey(pubKey)
	//checkError(err)
	//log.Info(raw)

	// 自签
	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, _ := rand.Int(rand.Reader, serialNumberLimit)
	expiry := 365 * 24 * time.Hour
	notBefore := time.Now().Add(-5 * time.Minute).UTC()
	template := x509.Certificate{
		SerialNumber:          serialNumber,
		NotBefore:             notBefore,
		NotAfter:              notBefore.Add(expiry).UTC(),
		BasicConstraintsValid: true,
		IsCA:                  true,
		KeyUsage: x509.KeyUsageDigitalSignature |
			x509.KeyUsageKeyEncipherment | x509.KeyUsageCertSign |
			x509.KeyUsageCRLSign,
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageAny},
		Subject: pkix.Name{
			Country:            []string{"CN"},
			Locality:           []string{"Shenzhn"},
			Province:           []string{"GuangDong"},
			OrganizationalUnit: []string{""},
			Organization:       []string{"liqiang.io.io"},
			StreetAddress:      []string{"", "", ""},
			PostalCode:         []string{"510003"},
			CommonName:         "local.liqiang.io.io",
		},
	}
	template.SubjectKeyId = priKeyHash(priKey)

	x509certEncode, err := x509.CreateCertificate(rand.Reader, &template, &template, pubKey, priKey)
	checkError(err)
	crt, err := os.Create(caFile)
	checkError(err)
	pem.Encode(crt, &pem.Block{Type: "CERTIFICATE", Bytes: x509certEncode})
	crt.Close()

	// 使用bob的密钥进行证书签名
	bobPriKey, _ := e.KeyGen()
	bobPriKeyEncode, err := x509.MarshalECPrivateKey(bobPriKey)
	checkError(err)
	bobf, err := os.Create(serverKey)
	checkError(err)
	pem.Encode(bobf, &pem.Block{Type: "EC PRIVATE KEY", Bytes: bobPriKeyEncode})
	bobf.Close()

	bobPubKey := bobPriKey.Public()
	bobSerialNumber, _ := rand.Int(rand.Reader, serialNumberLimit)
	notBefore = time.Now().Add(-5 * time.Minute).UTC()
	bobTemplate := x509.Certificate{
		SerialNumber:          bobSerialNumber,
		NotBefore:             notBefore,
		NotAfter:              notBefore.Add(expiry).UTC(),
		BasicConstraintsValid: true,
		IsCA:                  false,
		KeyUsage: x509.KeyUsageDigitalSignature |
			x509.KeyUsageKeyEncipherment | x509.KeyUsageCertSign |
			x509.KeyUsageCRLSign,
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageAny},
		Subject: pkix.Name{
			Country:            []string{"CN"},
			Locality:           []string{"Shenzhn"},
			Province:           []string{"GuangDong"},
			OrganizationalUnit: []string{""},
			Organization:       []string{"liqiang.io.io"},
			StreetAddress:      []string{"", "", ""},
			PostalCode:         []string{"510003"},
			CommonName:         "local.liqiang.io.io",
		},
	}
	bobTemplate.SubjectKeyId = priKeyHash(bobPriKey)
	parent, err := x509.ParseCertificate(x509certEncode)
	checkError(err)
	bobCertEncode, err := x509.CreateCertificate(rand.Reader, &bobTemplate, parent, bobPubKey, priKey)
	checkError(err)

	bcrt, _ := os.Create(serverCrt)
	pem.Encode(bcrt, &pem.Block{Type: "CERTIFICATE", Bytes: bobCertEncode})
	bcrt.Close()
}
