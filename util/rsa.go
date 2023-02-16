package util

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
)

func ConvertPublicKeyDERToPEM(derPublicKeyBytes []byte) (pemPublicKeyBytes []byte, err error) {
	var (
		rsaPublicKey *rsa.PublicKey
	)
	rsaPublicKey, err = x509.ParsePKCS1PublicKey(derPublicKeyBytes)
	if err != nil {
		return nil, err
	}

	pemPublicKeyBytes, err = x509.MarshalPKIXPublicKey(rsaPublicKey)
	if err != nil {
		return nil, err
	}
	pemPublicKeyBytes = pem.EncodeToMemory(&pem.Block{
		Type:    "PUBLIC KEY",
		Bytes:   pemPublicKeyBytes,
		Headers: map[string]string{},
	})
	return pemPublicKeyBytes, nil
}

func ConvertPublicKeyPEMToDER(pemPublicKeyBytes []byte) (derPublicKeyBytes []byte, err error) {
	var (
		pub interface{}
	)
	pemBlock, _ := pem.Decode(pemPublicKeyBytes)
	if pemBlock == nil {
		return nil, errors.New("not recorgnized pem block")
	}
	pub, err = x509.ParsePKIXPublicKey(pemBlock.Bytes)
	if err != nil {
		return nil, err
	}

	if rsaPubicKey, ok := pub.(*rsa.PublicKey); ok {
		derPublicKeyBytes = x509.MarshalPKCS1PublicKey(rsaPubicKey)
		return derPublicKeyBytes, nil
	}

	return nil, errors.New("not valid rsa public format")
}
